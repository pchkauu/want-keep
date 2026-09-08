//go:build integration

package matching_test

import (
	"testing"

	"github.com/google/uuid"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestSeparateRetainsIndependentTransferCompleteness(t *testing.T) {
	for _, undo := range []bool{false, true} {
		name := "counterpart"
		if undo {
			name = "undo"
		}
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			gate, connection := f.admit(), f.connection(f.p)
			from, to := f.create(money.RUB, "5000"), f.create(money.RUB, "0")
			purchase := f.revision(uuid.NewString(), from, "-1000", money.RUB, 1)
			if _, err := f.write(purchase, request()); err != nil {
				t.Fatal(err)
			}
			proof := ledger.Correspondence{Kind: "transfer", Namespace: "synthetic:transfer", Reference: "separate-side", FromAccountID: from, ToAccountID: to}
			outgoing := f.revision(uuid.NewString(), from, "-1000", money.RUB, 1)
			outgoing.Type, outgoing.Correspondence = ledger.Transfer, &proof
			in := f.sourceInput(&outgoing, outgoing.OperationID)
			if _, err := f.importSource(gate, connection, in); err != nil {
				t.Fatal(err)
			}
			g, found, err := f.store.MatchingForOperation(testContext, f.p, outgoing.OperationID)
			if err != nil || !found || g.State != matching.Clarification {
				t.Fatal(g, found, err)
			}
			c := f.client(f.p)
			c.result("/matching/"+g.ID+"/resolve", map[string]any{"decision": "separate", "expectedRevision": g.Revision, "expectedRevisions": f.versions(outgoing.OperationID), "reason": "Different outgoing payment"})
			decision := f.current(outgoing.OperationID).DecisionID
			waiting, found, err := f.store.MatchingForOperation(testContext, f.p, outgoing.OperationID)
			if err != nil || !found || waiting.ID == g.ID || waiting.State != matching.WaitingSide {
				t.Fatal(waiting, found, err)
			}
			rejected, err := f.store.MatchingRejected(testContext, f.p, []string{purchase.OperationID, outgoing.OperationID})
			if err != nil || !rejected {
				t.Fatal("duplicate rejection was lost", err)
			}
			funding, err := f.store.AccountFunding(testContext, f.p, from)
			if err != nil {
				t.Fatal(err)
			}
			if _, known := funding.Value(); known {
				t.Fatal("missing side became complete funding")
			}
			f.balance(from, "owned", "3000")
			f.balance(to, "owned", "0")
			versions := f.count("operation_revisions")
			if out, err := f.importSource(gate, connection, in); err != nil || !out.Duplicate {
				t.Fatal(out, err)
			}
			if f.count("operation_revisions") != versions {
				t.Fatal("source replay added a decision")
			}
			if undo {
				c.result("/transactions/"+outgoing.OperationID+"/undo", map[string]any{"decisionId": decision, "expectedRevisions": f.versions(outgoing.OperationID), "reason": "Reconsider duplicate evidence"})
				restored, found, err := f.store.MatchingForOperation(testContext, f.p, outgoing.OperationID)
				if err != nil || !found || restored.ID != g.ID || restored.State != matching.Clarification {
					t.Fatal(restored, found, err)
				}
				f.balance(from, "owned", "4000")
				rejected, err = f.store.MatchingRejected(testContext, f.p, []string{purchase.OperationID, outgoing.OperationID})
				if err != nil || rejected {
					t.Fatal("undo left the rejection active", err)
				}
				return
			}
			incoming := f.revision(uuid.NewString(), to, "1000", money.RUB, 1)
			incoming.Type, incoming.Correspondence = ledger.Transfer, &proof
			if _, err := f.importSource(gate, connection, f.sourceInput(&incoming, incoming.OperationID)); err != nil {
				t.Fatal(err)
			}
			linked, found, err := f.store.MatchingForOperation(testContext, f.p, outgoing.OperationID)
			if err != nil || !found || linked.State != matching.Linked || linked.ID != waiting.ID {
				t.Fatal(linked, found, err)
			}
			f.balance(from, "owned", "3000")
			f.balance(to, "owned", "1000")
			if components, err := f.current(outgoing.OperationID).Components(); err != nil || len(components) != 0 {
				t.Fatal("principal became expense", components, err)
			}
		})
	}
}
