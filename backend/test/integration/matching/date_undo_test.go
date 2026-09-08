//go:build integration

package matching_test

import (
	"testing"

	"github.com/google/uuid"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestMatchingUndoRetainsIndependentDate(t *testing.T) {
	for _, kind := range []matching.Kind{matching.Transfer, matching.Payment} {
		t.Run(string(kind), func(t *testing.T) {
			f := newFixture(t)
			from, to := f.create(money.RUB, "5000"), f.create(money.RUB, "0")
			a, b := f.revision(uuid.NewString(), from, "-1000", money.RUB, 1), f.revision(uuid.NewString(), to, "1000", money.RUB, 1)
			b.Type = ledger.Income
			if kind == matching.Payment {
				b = f.revision(b.OperationID, from, "-1000", money.RUB, 1)
			}
			for _, r := range []ledger.Revision{a, b} {
				if _, err := f.write(r, request()); err != nil {
					t.Fatal(err)
				}
			}
			f.link(kind, a.OperationID, a.OperationID, b.OperationID)
			linked := f.current(a.OperationID)
			c := f.client(f.p)
			changed := "2026-09-06T12:00:00Z"
			c.result("/transactions/"+a.OperationID+"/corrections", map[string]any{"expectedRevision": linked.Revision, "reason": "Correct independent date", "occurredAt": changed})
			corrected := f.current(a.OperationID)
			if corrected.FieldVersions[ledger.MatchingField] != linked.FieldVersions[ledger.MatchingField] {
				t.Fatal("date superseded association")
			}
			if kind == matching.Payment && f.current(b.OperationID).ContributionAt(0) != corrected.OccurredAt {
				t.Fatal("dependent evidence time was not refreshed")
			}
			c.result("/transactions/"+a.OperationID+"/undo", map[string]any{"decisionId": linked.DecisionID, "expectedRevisions": f.versions(a.OperationID, b.OperationID), "reason": "Undo association while retaining date"})
			if f.current(a.OperationID).OccurredAt.String() != changed {
				t.Fatal("association undo lost date correction")
			}
			revisions := f.versions(a.OperationID)
			if kind == matching.Payment {
				revisions = f.versions(a.OperationID, b.OperationID)
			}
			c.result("/transactions/"+a.OperationID+"/undo", map[string]any{"decisionId": corrected.DecisionID, "expectedRevisions": revisions, "reason": "Undo independent date afterward"})
			if f.current(a.OperationID).OccurredAt != a.OccurredAt {
				t.Fatal("date undo lost its original value")
			}
		})
	}
}
