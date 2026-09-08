//go:build integration

package matching_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestSourceConflictReconciliationUsesAllCurrentEvidence(t *testing.T) {
	for _, mode := range []string{"restore_amount", "both_amounts", "restore_proof"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			gate := f.admit()
			connection := f.connection(f.p)
			from, to := f.create(money.RUB, "5000"), f.create(money.RUB, "0")
			proof := ledger.Correspondence{Kind: "transfer", Namespace: "synthetic:transfer", Reference: "current", FromAccountID: from, ToAccountID: to}
			a, b := f.revision(uuid.NewString(), from, "-1000", money.RUB, 1), f.revision(uuid.NewString(), to, "1000", money.RUB, 1)
			a.Type, b.Type = ledger.Transfer, ledger.Transfer
			a.Correspondence, b.Correspondence = &proof, &proof
			for _, r := range []*ledger.Revision{&a, &b} {
				if _, err := f.importSource(gate, connection, f.sourceInput(r, r.OperationID)); err != nil {
					t.Fatal(err)
				}
			}
			review, outbox := f.count("ledger_review_requests"), f.count("outbox")
			oldA, oldB := f.current(a.OperationID), f.current(b.OperationID)
			a.Postings[0].Money = cash("-1100", money.RUB)
			if mode == "restore_proof" {
				a.Postings[0].Money = cash("-1000", money.RUB)
				different := proof
				different.Reference = "changed"
				a.Correspondence = &different
			}
			input := f.sourceInput(&a, a.OperationID)
			input.Classification = "correction"
			input.ExpectedRevision = 1
			input.PayloadHash = strings.Repeat("b", 64)
			if _, err := f.importSource(gate, connection, input); err != nil {
				t.Fatal(err)
			}
			f.balance(from, "owned", "4000")
			f.balance(to, "owned", "1000")
			for _, id := range []string{from, to} {
				balance, err := f.store.Balance(testContext, f.p, id, "owned")
				if err != nil || !slices.Contains(balance.Coverage.Reasons(), "matching_unresolved") {
					t.Fatal("conflict quality", balance, err)
				}
			}
			if f.current(a.OperationID).Revision != oldA.Revision+1 || f.current(b.OperationID).Revision != oldB.Revision+1 || f.count("ledger_review_requests") != review+2 || f.count("outbox") != outbox+2 {
				t.Fatal("material conflict work not atomic")
			}
			// Replaying the source is not a new material version.
			if _, err := f.importSource(gate, connection, input); err != nil {
				t.Fatal(err)
			}
			if f.count("ledger_review_requests") != review+2 || f.count("outbox") != outbox+2 {
				t.Fatal("conflict replay creates work")
			}
			if mode == "both_amounts" {
				b.Postings[0].Money = cash("1100", money.RUB)
				input = f.sourceInput(&b, b.OperationID)
				input.ExpectedRevision = 1
			} else {
				a.Postings[0].Money = cash("-1000", money.RUB)
				a.Correspondence = &proof
				input = f.sourceInput(&a, a.OperationID)
				input.ExpectedRevision = 2
			}
			input.Classification = "correction"
			input.PayloadHash = strings.Repeat("c", 64)
			if _, err := f.importSource(gate, connection, input); err != nil {
				t.Fatal(err)
			}
			g, _, err := f.store.MatchingForOperation(testContext, f.p, a.OperationID)
			if err != nil || g.State != matching.Linked {
				t.Fatal("confirmed source recovery", g, err)
			}
			if mode == "both_amounts" {
				f.balance(from, "owned", "3900")
				f.balance(to, "owned", "1100")
			} else {
				f.balance(from, "owned", "4000")
				f.balance(to, "owned", "1000")
			}
			for _, id := range []string{from, to} {
				balance, err := f.store.Balance(testContext, f.p, id, "owned")
				if err != nil || slices.Contains(balance.Coverage.Reasons(), "matching_unresolved") {
					t.Fatal("recovery quality", balance, err)
				}
			}
			if f.count("ledger_review_requests") != review+4 || f.count("outbox") != outbox+4 {
				t.Fatal("recovery material work")
			}
		})
	}
}
