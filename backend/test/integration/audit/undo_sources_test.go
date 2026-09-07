//go:build integration

package audit_test

import (
	"context"
	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"strings"
	"testing"
	"time"
)

func TestExclusionUndoPreservesEarlierIndependentDecision(t *testing.T) {
	f := newFixture(t)
	gate := f.admit()
	connection := f.connection(f.p)
	account := f.create(money.RUB, "5000")
	c := f.client(f.p)
	r := f.revision(uuid.NewString(), account, "-500", money.RUB, 1)
	r.Merchant = "Bank merchant"
	r.FeeKnowledge = ledger.KnownFees
	in := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: "raiffeisen", ExternalAccountID: "synthetic-business", Product: "current", Log: "statement", RecordID: "merchant"}, PayloadHash: strings.Repeat("a", 64), EvidenceRef: "synthetic:source", Classification: "new", Operation: &r}
	if _, err := f.importSource(gate, connection, in); err != nil {
		t.Fatal(err)
	}
	merchant := "Confirmed merchant"
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		return f.ledgerService().CompleteReview(ctx, f.p, journal.ReviewInput{OperationID: r.OperationID, Revision: 1, State: "reviewed", Rationale: "Synthetic confirmed merchant", Correction: &ledger.Correction{Merchant: &merchant}})
	}); err != nil {
		t.Fatal(err)
	}
	view := c.transaction(r.OperationID)
	if view.Revision != 2 || view.Merchant == nil || *view.Merchant != merchant {
		t.Fatal("review not applied")
	}
	c.call("POST", "/transactions/"+r.OperationID+"/exclude", uuid.NewString(), map[string]any{"expectedRevision": view.Revision, "reason": "Mistaken duplicate"}, 202)
	view = c.transaction(r.OperationID)
	view = c.undo(view, *view.DecisionId)
	if view.Merchant == nil {
		t.Fatal("merchant disappeared")
	}
	if *view.Merchant != merchant {
		t.Fatalf("undo of accounting-only exclusion also reset unrelated earlier merchant: got %q want %q", *view.Merchant, merchant)
	}
}

func TestUndoReconcilesFieldsAgainstFrozenDecisionEvidence(t *testing.T) {
	for _, updated := range []bool{false, true} {
		name := "unchanged source keeps decision"
		if updated {
			name = "new source updates unprotected field"
		}
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			gate, connection := f.admit(), f.connection(f.p)
			c := f.client(f.p)
			account := f.create(money.RUB, "5000")
			raw := f.revision(uuid.NewString(), account, "-500", money.RUB, 1)
			raw.State, raw.Merchant = ledger.Pending, "Bank merchant"
			raw.OccurredAt, _ = calendar.ParseInstant(f.now.Time().Add(-48 * time.Hour).UTC().Format(time.RFC3339Nano))
			zone, _ := calendar.ParseTimezone("Europe/Moscow")
			raw, _ = raw.InTimezone(zone)
			input := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: "raiffeisen", ExternalAccountID: "synthetic-business", Product: "current", Log: "statement", RecordID: "frozen-source"}, PayloadHash: strings.Repeat("a", 64), EvidenceRef: "synthetic:source", Classification: "new", Operation: &raw}
			if _, err := f.importSource(gate, connection, input); err != nil {
				t.Fatal(err)
			}
			view := c.correct(c.transaction(raw.OperationID), map[string]any{"occurredAt": f.now.String()})
			dateDecision := *view.DecisionId
			merchant := "Confirmed merchant"
			if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
				return f.ledgerService().CompleteReview(ctx, f.p, journal.ReviewInput{OperationID: raw.OperationID, Revision: uint64(view.Revision), State: "reviewed", Rationale: "Synthetic confirmed merchant", Correction: &ledger.Correction{Merchant: &merchant}})
			}); err != nil {
				t.Fatal(err)
			}
			reviewed := c.transaction(raw.OperationID)
			raw.State = ledger.Posted
			raw.PostedAt, _ = calendar.ParseInstant(f.now.Time().Add(-24 * time.Hour).UTC().Format(time.RFC3339Nano))
			if updated {
				raw.Merchant = "New bank merchant"
			}
			input.Classification, input.ExpectedRevision, input.PayloadHash = "correction", 1, strings.Repeat("b", 64)
			if _, err := f.importSource(gate, connection, input); err != nil {
				t.Fatal(err)
			}
			basis, err := f.store.DecisionSourceFact(testContext, f.p, raw.OperationID, *reviewed.DecisionId)
			if err != nil || basis == nil || basis.Merchant != "Bank merchant" {
				t.Fatal("decision evidence drifted", basis, err)
			}
			view = c.undo(c.transaction(raw.OperationID), dateDecision)
			expected := merchant
			if updated {
				expected = raw.Merchant
			}
			if view.Merchant == nil || *view.Merchant != expected || view.State != "posted" || view.SourceConflict {
				t.Fatal("incorrect reconciled decision", view)
			}
			f.balance(account, "owned", "4500")
			f.balance(account, "locked", "0")
		})
	}
}
