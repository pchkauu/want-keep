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

func TestMatchingUndoPreservesCurrentBankLifecycle(t *testing.T) {
	for _, state := range []ledger.State{ledger.Posted, ledger.Cancelled, ledger.Reversed} {
		t.Run(string(state), func(t *testing.T) {
			f := newFixture(t)
			gate, connection := f.admit(), f.connection(f.p)
			from, to := f.create(money.RUB, "5000"), f.create(money.RUB, "0")
			proof := ledger.Correspondence{Kind: "transfer", Namespace: "synthetic:transfer", Reference: "lifecycle-undo", FromAccountID: from, ToAccountID: to}
			a, b := f.revision(uuid.NewString(), from, "-1000", money.RUB, 1), f.revision(uuid.NewString(), to, "1000", money.RUB, 1)
			a.Type, b.Type = ledger.Transfer, ledger.Transfer
			a.State, b.State = ledger.Pending, ledger.Pending
			if state == ledger.Reversed {
				a.State = ledger.Posted
				a.PostedAt = f.now
			}
			a.Correspondence, b.Correspondence = &proof, &proof
			for _, r := range []*ledger.Revision{&a, &b} {
				if _, err := f.importSource(gate, connection, f.sourceInput(r, r.OperationID)); err != nil {
					t.Fatal(err)
				}
			}
			linked := f.current(a.OperationID)
			a.State = state
			if state == ledger.Posted {
				a.PostedAt = f.now
			}
			in := f.sourceInput(&a, a.OperationID)
			in.Classification, in.ExpectedRevision, in.PayloadHash = "correction", 1, strings.Repeat("b", 64)
			if _, err := f.importSource(gate, connection, in); err != nil {
				t.Fatal(err)
			}
			current := f.current(a.OperationID)
			if current.Revision <= linked.Revision || current.FieldVersions[ledger.MatchingField] != linked.FieldVersions[ledger.MatchingField] {
				t.Fatal("lifecycle overwrote association provenance or lost its revision")
			}
			c := f.client(f.p)
			c.result("/transactions/"+a.OperationID+"/undo", map[string]any{"decisionId": linked.DecisionID, "expectedRevisions": f.versions(a.OperationID, b.OperationID), "reason": "Undo association, keep bank state"})
			after := f.current(a.OperationID)
			if after.State != state || after.PostedAt != a.PostedAt {
				t.Fatal("undo changed current bank state/time")
			}
			g, found, err := f.store.MatchingForOperation(testContext, f.p, a.OperationID)
			if err != nil || !found || g.State != matching.WaitingSide {
				t.Fatal(g, found, err)
			}
			g, found, err = f.store.MatchingForOperation(testContext, f.p, b.OperationID)
			if err != nil || !found || g.State != matching.Clarification {
				t.Fatal(g, found, err)
			}
			owned := "5000"
			if state == ledger.Posted {
				owned = "4000"
			}
			f.balance(from, "owned", owned)
			f.balance(from, "locked", "0")
			f.balance(to, "owned", "0")
			if f.current(b.OperationID).State != ledger.Pending {
				t.Fatal("undo changed the other source lifecycle")
			}
		})
	}
}
