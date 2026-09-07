//go:build integration

package audit_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func TestPurchaseMonthCorrectionRetainsPostingAndRecalculatesAfterUndo(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	r := f.revision(uuid.NewString(), f.create(money.RUB, "5000"), "-500", money.RUB, 1)
	r.OccurredAt = instant("2026-08-31T20:59:59.123456789Z")
	r.PostedAt = instant("2026-09-02T10:00:00Z")
	r.Origin = "source"
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	view := c.correct(c.transaction(r.OperationID), map[string]any{"occurredAt": "2026-09-01T10:00:00.123456789Z"})
	if *view.ExpenseMonth != "2026-09" || *view.PostedAt != "2026-09-02T10:00:00Z" {
		t.Fatal("date contract")
	}
	view = c.undo(view, *view.DecisionId)
	if *view.ExpenseMonth != "2026-08" || view.OccurredAt != "2026-08-31T20:59:59.123456789Z" {
		t.Fatal("month not restored")
	}
	view = c.correct(view, map[string]any{"occurredAt": "2026-07-31T20:59:59Z"})
	f.balance(r.Postings[0].AccountID, "owned", "5000")
	view = c.undo(view, *view.DecisionId)
	f.balance(r.Postings[0].AccountID, "owned", "4500")
}
func TestCommandRetentionDoesNotDeleteDecisionHistory(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	r := c.expense(f.create(money.RUB, "5000"), money.RUB, "500")
	r = c.correct(r, map[string]any{"note": "Long lived history"})
	before := f.count("ledger_decisions")
	u, err := url.Parse(f.dsn)
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("want_keep_maintenance", "synthetic-maintenance")
	maintenance, err := storage.Open(testContext, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer maintenance.Close()
	future := instant("2028-01-01T00:00:00Z")
	if _, err = maintenance.CleanupCommandDetails(testContext, future, 1000); err != nil {
		t.Fatal(err)
	}
	if _, err = maintenance.CleanupCommandTombstones(testContext, future, 1000); err != nil {
		t.Fatal(err)
	}
	if f.count("command_details") != 0 || f.count("command_tombstones") != 0 || f.count("ledger_decisions") != before {
		t.Fatal("retention affected history")
	}
	h := decode[generated.TransactionHistoryPage](t, c.call("GET", "/transactions/"+r.Id+"/history", "", nil, 200))
	if len(h.Items) != 2 || h.Items[0].DecisionId == nil {
		t.Fatal("history disappeared")
	}
	// A financial audit reference prevents a new effect even after its command tombstone expired.
	var oldKey string
	if err = f.admin.QueryRow(testContext, `SELECT command_id FROM want_keep.operation_revisions WHERE household_id=$1 AND operation_id=$2 AND revision=2`, f.family.ID, r.Id).Scan(&oldKey); err != nil {
		t.Fatal(err)
	}
	e := commands.NewExecutor(f.store, f.store, func() calendar.Instant { return future })
	called := false
	_, err = e.Execute(testContext, f.p, commands.Request{ID: oldKey, Kind: "transactions.corrections", PayloadHash: request().PayloadHash}, func(context.Context) (command.Result, error) { called = true; return command.Result{}, nil })
	if err == nil || called {
		t.Fatal("expired key replayed financial effect")
	}
	stored, _, err := f.store.CurrentLedgerRevision(testContext, f.p, r.Id)
	if err != nil || stored.Accounting() != ledger.IncludedInAccounting {
		t.Fatal(err)
	}
}
