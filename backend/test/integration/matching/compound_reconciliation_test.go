//go:build integration

package matching_test

import (
	"context"
	"strings"
	"testing"

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
)

func TestCompoundReconciliationCommitsWholeGroup(t *testing.T) {
	for _, mode := range []string{"correction", "source_update"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			gate, connection := f.admit(), f.connection(f.p)
			reconciler := reconciliationapp.NewService(f.store, f.store, journal.NewWriter(f.store, f.store), gate, func() calendar.Instant { return f.now }, uuid.NewString)
			f.writer = matchingapp.NewService(f.store, journal.NewWriterWithReconciliation(f.store, f.store, reconciler), func() calendar.Instant { return f.now }, uuid.NewString)
			ids := []string{}
			for i, observed := range []string{"3900", "1100"} {
				in := f.input(connection)
				in.ExternalID = []string{"review-from", "review-to"}[i]
				in.Observation.Amounts, _ = account.CashAmounts(cash(observed, money.RUB))
				result := f.importAccount(gate, in)
				ids = append(ids, result.Account.ID)
				opening, _ := account.CashAmounts(cash([]string{"5000", "0"}[i], money.RUB))
				date, _ := calendar.ParseDate("2026-08-01")
				if c, err := f.executor.Execute(testContext, f.p, request(), func(ctx context.Context) (command.Result, error) {
					return f.service().CorrectOpening(ctx, f.p, result.Account.ID, accounts.Correction{ExpectedRevision: result.Account.Revision, Date: date, Amounts: opening, Reason: "Confirmed opening"})
				}); err != nil || c.Status() != command.Succeeded {
					t.Fatal(c.ErrorCode(), err)
				}
			}
			a, b := f.revision(uuid.NewString(), ids[0], "-1000", money.RUB, 1), f.revision(uuid.NewString(), ids[1], "1000", money.RUB, 1)

			a.RecordedAt, b.RecordedAt = f.now, f.now
			var c *client
			if mode == "source_update" {
				a.Type, b.Type = ledger.Transfer, ledger.Transfer
				proof := &ledger.Correspondence{Kind: "transfer", Namespace: "synthetic:transfer", Reference: "compound-source", FromAccountID: ids[0], ToAccountID: ids[1]}
				a.Correspondence, b.Correspondence = proof, proof
				for _, r := range []*ledger.Revision{&a, &b} {
					if _, err := f.importSource(gate, connection, f.sourceInput(r, r.OperationID)); err != nil {
						t.Fatal(err)
					}
				}
				a.Postings[0].Money = cash("-1100", money.RUB)
				b.Postings[0].Money = cash("1100", money.RUB)
				for _, r := range []*ledger.Revision{&a, &b} {
					input := f.sourceInput(r, r.OperationID)
					input.Classification, input.ExpectedRevision, input.PayloadHash = "correction", 1, strings.Repeat("b", 64)
					if _, err := f.importSource(gate, connection, input); err != nil {
						t.Fatal(err)
					}
				}
			} else {
				b.Type = ledger.Income
				for _, r := range []ledger.Revision{a, b} {
					if _, err := f.write(r, request()); err != nil {
						t.Fatal(err)
					}
				}
				f.link(matching.Transfer, a.OperationID, a.OperationID, b.OperationID)
				c = f.client(f.p)
				left, right := c.transaction(a.OperationID), c.transaction(b.OperationID)
				left.Postings[0].Money.Amount, right.Postings[0].Money.Amount = "-1100", "1100"
				c.result("/transactions/"+a.OperationID+"/corrections", map[string]any{"expectedRevision": left.Revision, "reason": "Correct both native sides", "principal": left.Postings, "relatedChanges": []any{map[string]any{"transactionId": b.OperationID, "expectedRevision": right.Revision, "principal": right.Postings}}})
			}
			for _, id := range ids {
				current, found, err := f.store.ActiveReconciliation(testContext, f.p, id)
				if err != nil || !found || current.Result != reconciliation.Balanced {
					t.Errorf("compound correction leaves stale comparison for %s: %s found=%v err=%v", id, current.Result, found, err)
				}
			}
			if mode == "correction" {
				corrected := c.transaction(a.OperationID)
				c.result("/transactions/"+a.OperationID+"/undo", map[string]any{"decisionId": *corrected.DecisionId, "expectedRevisions": f.versions(a.OperationID, b.OperationID), "reason": "Restore both original sides"})
				for _, id := range ids {
					current, found, err := f.store.ActiveReconciliation(testContext, f.p, id)
					if err != nil || !found || current.Result != reconciliation.Discrepant {
						t.Fatal("undo comparison", current.Result, found, err)
					}
				}
			}
			if f.count("account_observations") != 2 || f.count("reconciliation_resolutions") != 0 {
				t.Fatal("compound change rewrote source or invented an adjustment")
			}
		})
	}
}
