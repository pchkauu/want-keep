//go:build integration

package audit_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestDecisionEvidenceUsesLatestSourceWithoutDeletingHistory(t *testing.T) {
	f := newFixture(t)
	gate, connection := f.admit(), f.connection(f.p)
	c := f.client(f.p)
	r := f.revision(uuid.NewString(), f.create(money.RUB, "5000"), "-500", money.RUB, 1)
	in := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: "raiffeisen", ExternalAccountID: "synthetic-business", Product: "current", Log: "statement", RecordID: "long-history"}, EvidenceRef: "synthetic:history", Classification: "new", Operation: &r, PayloadHash: fmt.Sprintf("%064x", 1)}
	if _, err := f.importSource(gate, connection, in); err != nil {
		t.Fatal(err)
	}
	view := c.correct(c.transaction(r.OperationID), map[string]any{"merchant": "Protected merchant"})
	for n := 2; n <= 101; n++ {
		r.Merchant = fmt.Sprintf("Bank merchant %d", n)
		in.Classification, in.ExpectedRevision, in.PayloadHash = "correction", uint64(n-1), fmt.Sprintf("%064x", n)
		if _, err := f.importSource(gate, connection, in); err != nil {
			t.Fatal(err)
		}
	}
	view = c.correct(view, map[string]any{"note": "Still correctable"})
	d, err := f.store.Decision(testContext, f.p, *view.DecisionId)
	if err != nil || len(d.Evidence) != 1 || d.Evidence[0].Revision != 101 {
		t.Fatal("decision did not select latest evidence", d.Evidence, err)
	}
	c.undo(view, d.ID)
	if f.count("ledger_source_facts") != 101 {
		t.Fatal("source history removed")
	}
}

func TestCompoundDecisionRetainsMoreThanOneHundredEvidenceReferences(t *testing.T) {
	f := newFixture(t)
	gate, connection := f.admit(), f.connection(f.p)
	account := f.create(money.RUB, "5000")
	changes := []journal.Change{}
	expected := []journal.ExpectedRevision{}
	note := "Compound correction"
	for i := 0; i < 51; i++ {
		r := f.revision(uuid.NewString(), account, "-1", money.RUB, 1)
		in := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: "raiffeisen", ExternalAccountID: "synthetic-business", Product: "current", Log: "statement", RecordID: fmt.Sprint(i)}, EvidenceRef: "synthetic:compound", Classification: "new", Operation: &r, PayloadHash: fmt.Sprintf("%064x", i+1)}
		if _, err := f.importSource(gate, connection, in); err != nil {
			t.Fatal(err)
		}
		if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
			return f.ledgerService().CompleteReview(ctx, f.p, journal.ReviewInput{OperationID: r.OperationID, Revision: 1, State: "reviewed", Rationale: "Verified synthetic fact"})
		}); err != nil {
			t.Fatal(err)
		}
		changes = append(changes, journal.Change{OperationID: r.OperationID, Expected: 1, Correction: ledger.Correction{Note: &note}})
		expected = append(expected, journal.ExpectedRevision{OperationID: r.OperationID, Revision: 2})
	}
	result, err := f.executor.Execute(testContext, f.p, request(), func(ctx context.Context) (command.Result, error) {
		return f.ledgerService().ApplyChanges(ctx, f.p, changes, note)
	})
	if err != nil || result.Status() != command.Succeeded {
		t.Fatal("compound correction failed", result.ErrorCode(), err)
	}
	r, _, err := f.store.CurrentLedgerRevision(testContext, f.p, changes[0].OperationID)
	if err != nil {
		t.Fatal(err)
	}
	d, err := f.store.Decision(testContext, f.p, r.DecisionID)
	if err != nil || len(d.Evidence) != 102 {
		t.Fatal("compound evidence lost", len(d.Evidence), err)
	}
	result, err = f.executor.Execute(testContext, f.q, request(), func(ctx context.Context) (command.Result, error) {
		return f.ledgerService().Undo(ctx, f.q, d.ID, expected, "Undo compound correction")
	})
	if err != nil || result.Status() != command.Succeeded {
		t.Fatal("compound undo failed", result.ErrorCode(), err)
	}
}
