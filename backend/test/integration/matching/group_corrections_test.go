//go:build integration

package matching_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestLinkedDateCorrectionAndUndo(t *testing.T) {
	f := newFixture(t)
	from, to := f.create(money.RUB, "5000"), f.create(money.RUB, "0")
	c := f.client(f.p)
	a, b := f.revision(uuid.NewString(), from, "-1000", money.RUB, 1), f.revision(uuid.NewString(), to, "1000", money.RUB, 1)
	b.Type = ledger.Income
	for _, r := range []ledger.Revision{a, b} {
		if _, err := f.write(r, request()); err != nil {
			t.Fatal(err)
		}
	}
	f.link(matching.Transfer, a.OperationID, a.OperationID, b.OperationID)
	c.result("/transactions/"+a.OperationID+"/corrections", map[string]any{"expectedRevision": f.current(a.OperationID).Revision, "reason": "Correct transfer day", "occurredAt": "2026-09-06T12:00:00.123456789Z"})
	after := c.transaction(a.OperationID)
	if f.current(a.OperationID).CashDate.String() != "2026-09-06" {
		t.Fatal("calendar date not normalized")
	}
	c.result("/transactions/"+a.OperationID+"/undo", map[string]any{"decisionId": *after.DecisionId, "expectedRevisions": f.versions(a.OperationID), "reason": "Restore day"})
	if f.current(a.OperationID).OccurredAt != a.OccurredAt {
		t.Fatal("date undo")
	}
}

func TestIndependentNotePreservesSourceConflict(t *testing.T) {
	f := newFixture(t)
	gate := f.admit()
	connection := f.connection(f.p)
	from, to := f.create(money.RUB, "5000"), f.create(money.RUB, "0")
	proof := ledger.Correspondence{Kind: "transfer", Namespace: "synthetic:transfer", Reference: "review-conflict", FromAccountID: from, ToAccountID: to}
	a, b := f.revision(uuid.NewString(), from, "-1000", money.RUB, 1), f.revision(uuid.NewString(), to, "1000", money.RUB, 1)
	a.Type, b.Type = ledger.Transfer, ledger.Transfer
	a.Correspondence, b.Correspondence = &proof, &proof
	for _, r := range []*ledger.Revision{&a, &b} {
		if _, err := f.importSource(gate, connection, f.sourceInput(r, r.OperationID)); err != nil {
			t.Fatal(err)
		}
	}
	a.Postings[0].Money = cash("-1100", money.RUB)
	in := f.sourceInput(&a, a.OperationID)
	in.Classification = "correction"
	in.ExpectedRevision = 1
	in.PayloadHash = strings.Repeat("b", 64)
	if _, err := f.importSource(gate, connection, in); err != nil {
		t.Fatal(err)
	}
	before, found, err := f.store.MatchingForOperation(testContext, f.p, a.OperationID)
	if err != nil || !found || before.State != matching.Conflict {
		t.Fatal(before, found, err)
	}
	c := f.client(f.p)
	c.result("/transactions/"+b.OperationID+"/corrections", map[string]any{"expectedRevision": f.current(b.OperationID).Revision, "reason": "Independent note", "note": "Still investigating"})
	after, _, err := f.store.MatchingForOperation(testContext, f.p, a.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	av := c.transaction(a.OperationID)
	funding, err := f.store.AccountFunding(testContext, f.p, from)
	if err != nil {
		t.Fatal(err)
	}
	_, known := funding.Value()
	if !av.SourceConflict || known {
		t.Fatal("conflict no longer gates funding")
	}
	if after.State != matching.Conflict {
		t.Fatalf("nonfinancial note silently cleared source conflict: %s", after.State)
	}
}

func TestMixedExclusionAndUndoPreserveExistingExclusion(t *testing.T) {
	f := newFixture(t)
	from, to := f.create(money.RUB, "5000"), f.create(money.RUB, "0")
	c := f.client(f.p)
	a, b := f.revision(uuid.NewString(), from, "-1000", money.RUB, 1), f.revision(uuid.NewString(), to, "1000", money.RUB, 1)
	b.Type = ledger.Income
	for _, r := range []ledger.Revision{a, b} {
		if _, err := f.write(r, request()); err != nil {
			t.Fatal(err)
		}
	}
	c.result("/transactions/"+b.OperationID+"/exclude", map[string]any{"expectedRevision": f.current(b.OperationID).Revision, "reason": "Ignore incoming"})
	f.link(matching.Transfer, a.OperationID, a.OperationID, b.OperationID)
	c.result("/transactions/"+a.OperationID+"/exclude", map[string]any{"expectedRevision": f.current(a.OperationID).Revision, "relatedRevisions": f.versions(b.OperationID), "reason": "Ignore remaining side"})
	f.balance(from, "owned", "5000")
	f.balance(to, "owned", "0")
	after := c.transaction(a.OperationID)
	c.result("/transactions/"+a.OperationID+"/undo", map[string]any{"decisionId": *after.DecisionId, "expectedRevisions": f.versions(a.OperationID, b.OperationID), "reason": "Restore only new exclusion"})
	f.balance(from, "owned", "4000")
	f.balance(to, "owned", "0")
	if f.current(b.OperationID).Accounting() != ledger.ExcludedFromAccounting {
		t.Fatal("prior independent exclusion removed")
	}
}

func TestIndependentNotePreservesComponentCarriers(t *testing.T) {
	f := newFixture(t)
	from, to := f.create(money.RUB, "5000"), f.create(money.RUB, "0")
	c := f.client(f.p)
	bank := f.revision("f0000000-0000-4000-8000-000000000001", from, "-1000", money.RUB, 1)
	bank.OccurredAt = instant("2026-09-05T12:00:00Z")
	bank.RecordedAt = instant("2026-09-05T13:00:00Z")
	manual := f.revision("10000000-0000-4000-8000-000000000001", from, "-1000", money.RUB, 1)
	manual.Type = ledger.Transfer
	manual.Origin = "manual"
	manual.OccurredAt = instant("2026-09-06T12:00:00Z")
	manual.RecordedAt = instant("2026-09-06T13:00:00Z")
	manual.Postings = append(manual.Postings, ledger.Posting{AccountID: to, Money: cash("1000", money.RUB), Role: ledger.Principal})
	for _, r := range []ledger.Revision{bank, manual} {
		if _, err := f.write(r, request()); err != nil {
			t.Fatal(err)
		}
	}
	f.link(matching.Transfer, bank.OperationID, bank.OperationID, manual.OperationID)
	if !f.current(bank.OperationID).Contributes(0) {
		t.Fatal("setup outgoing is not bank carrier")
	}
	c.result("/transactions/"+bank.OperationID+"/corrections", map[string]any{"expectedRevision": f.current(bank.OperationID).Revision, "reason": "Independent note", "note": "Keep bank evidence"})
	after := f.current(bank.OperationID)
	if !after.Contributes(0) || after.ContributionAt(0) != bank.OccurredAt {
		t.Fatal("note moved outgoing carrier/date to manual transfer")
	}
}
