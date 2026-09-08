//go:build integration

package matching_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matchingapp "github.com/pchkauu/want-keep/backend/internal/matching/application"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reconciliationapp "github.com/pchkauu/want-keep/backend/internal/reconciliation/application"
	reconciliation "github.com/pchkauu/want-keep/backend/internal/reconciliation/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func (f *fixture) historical(accountID string, at calendar.Instant) (account.Amounts, reporting.Coverage) {
	f.t.Helper()
	values, coverage, _, err := f.store.HistoricalProjection(testContext, f.p, accountID, at)
	if err != nil {
		f.t.Fatal(err)
	}
	return values, coverage
}

func TestHistoricalMatchingRetainsIncompleteCoverage(t *testing.T) {
	f := newFixture(t)
	id := f.create(money.RUB, "1000")
	a, b := f.revision(uuid.NewString(), id, "-100", money.RUB, 1), f.revision(uuid.NewString(), id, "-100", money.RUB, 1)
	a.RecordedAt, b.RecordedAt = f.now, f.now
	for _, r := range []ledger.Revision{a, b} {
		if _, err := f.write(r, request()); err != nil {
			t.Fatal(err)
		}
	}
	values, coverage := f.historical(id, f.now)
	owned, known := values.Owned.Value()
	if !known || owned.Amount() != "900" || coverage.State() != reporting.Partial || !slices.Contains(coverage.Reasons(), "matching_unresolved") {
		t.Fatalf("pending duplicate projected as complete: owned=%s known=%v coverage=%s reasons=%v", owned.Amount(), known, coverage.State(), coverage.Reasons())
	}
	f.link(matching.Payment, a.OperationID, a.OperationID, b.OperationID)
	values, coverage = f.historical(id, f.now)
	owned, known = values.Owned.Value()
	if !known || owned.Amount() != "900" || coverage.State() != reporting.Complete {
		t.Fatalf("linked evidence changed balance: %s %v %s", owned.Amount(), known, coverage.State())
	}
}

func TestHistoricalMatchingUsesComponentLifecycleAtSourceTime(t *testing.T) {
	f := newFixture(t)
	id := f.create(money.RUB, "1000")
	gate, connection := f.admit(), f.connection(f.p)
	proof := &ledger.Correspondence{Kind: "payment", Namespace: "synthetic:payment", Reference: "historical-state"}
	a, b := f.revision(uuid.NewString(), id, "-100", money.RUB, 1), f.revision(uuid.NewString(), id, "-100", money.RUB, 1)
	a.State, b.State = ledger.Pending, ledger.Pending
	a.Correspondence, b.Correspondence = proof, proof
	for _, r := range []*ledger.Revision{&a, &b} {
		if _, err := f.importSource(gate, connection, f.sourceInput(r, r.OperationID)); err != nil {
			t.Fatal(err)
		}
	}
	asOf := f.now
	f.now = instant(f.now.Time().Add(time.Hour).Format(time.RFC3339Nano))
	a.State, a.PostedAt = ledger.Posted, f.now
	in := f.sourceInput(&a, a.OperationID)
	in.Classification, in.ExpectedRevision, in.PayloadHash = "correction", 1, strings.Repeat("b", 64)
	if _, err := f.importSource(gate, connection, in); err != nil {
		t.Fatal(err)
	}
	values, coverage := f.historical(id, asOf)
	owned, ok := values.Owned.Value()
	locked, lk := values.Locked.Value()
	if !ok || !lk || owned.Amount() != "1000" || locked.Amount() != "100" || coverage.State() != reporting.Complete {
		t.Fatalf("future posting changed historical hold: owned=%s/%v locked=%s/%v coverage=%s reasons=%v", owned.Amount(), ok, locked.Amount(), lk, coverage.State(), coverage.Reasons())
	}
	values, _ = f.historical(id, f.now)
	owned, ok = values.Owned.Value()
	locked, lk = values.Locked.Value()
	if !ok || !lk || owned.Amount() != "900" || locked.Amount() != "0" {
		t.Fatalf("current posting not applied once: %s/%v %s/%v", owned.Amount(), ok, locked.Amount(), lk)
	}
}

func TestMatchingWriterRefreshesBankReconciliation(t *testing.T) {
	f := newFixture(t)
	gate, connection := f.admit(), f.connection(f.p)
	reconciler := reconciliationapp.NewService(f.store, f.store, journal.NewWriter(f.store, f.store), gate, func() calendar.Instant { return f.now }, uuid.NewString)
	f.writer = matchingapp.NewService(f.store, journal.NewWriterWithReconciliation(f.store, f.store, reconciler), func() calendar.Instant { return f.now }, uuid.NewString)
	in := f.input(connection)
	in.Observation.Amounts, _ = account.CashAmounts(cash("900", money.RUB))
	result := f.importAccount(gate, in)
	id := result.Account.ID
	values, _ := account.CashAmounts(cash("1000", money.RUB))
	date, _ := calendar.ParseDate("2026-08-01")
	if c, err := f.executor.Execute(testContext, f.p, request(), func(ctx context.Context) (command.Result, error) {
		return f.service().CorrectOpening(ctx, f.p, id, accounts.Correction{ExpectedRevision: result.Account.Revision, Date: date, Amounts: values, Reason: "Confirmed opening"})
	}); err != nil || c.Status() != command.Succeeded {
		t.Fatal(c.ErrorCode(), err)
	}
	a, b := f.revision(uuid.NewString(), id, "-100", money.RUB, 1), f.revision(uuid.NewString(), id, "-100", money.RUB, 1)
	a.RecordedAt, b.RecordedAt = f.now, f.now
	for _, r := range []ledger.Revision{a, b} {
		if _, err := f.write(r, request()); err != nil {
			t.Fatal(err)
		}
	}
	pending, found, err := f.store.ActiveReconciliation(testContext, f.p, id)
	if err != nil || !found || pending.Result != reconciliation.Incomplete {
		t.Fatalf("matching gap disappeared: result=%s found=%v err=%v", pending.Result, found, err)
	}
	c := f.client(f.p)
	c.result("/transactions/"+a.OperationID+"/links", map[string]any{"kind": "receipt_match", "expectedRevisions": f.versions(a.OperationID, b.OperationID), "reason": "One confirmed payment"})
	current, found, err := f.store.ActiveReconciliation(testContext, f.p, id)
	if err != nil || !found || current.Result != reconciliation.Balanced {
		t.Fatalf("matching did not refresh final bank comparison: result=%s found=%v err=%v", current.Result, found, err)
	}
	if f.count("account_observations") != 1 || f.count("reconciliation_resolutions") != 0 {
		t.Fatal("matching rewrote source or invented an adjustment")
	}
}
