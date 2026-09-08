//go:build integration

package matching_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	application "github.com/pchkauu/want-keep/backend/internal/matching/application"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (f *fixture) matching() *application.Service { return f.writer.(*application.Service) }
func (f *fixture) current(id string) ledger.Revision {
	f.t.Helper()
	r, found, err := f.store.CurrentLedgerRevision(testContext, f.p, id)
	if err != nil || !found {
		f.t.Fatal(found, err)
	}
	return r
}
func (f *fixture) link(kind matching.Kind, primary string, ids ...string) command.Result {
	f.t.Helper()
	members := []matching.Member{}
	for _, id := range ids {
		r := f.current(id)
		members = append(members, matching.Member{OperationID: id, Revision: r.Revision})
	}
	c, err := f.executor.Execute(testContext, f.p, request(), func(ctx context.Context) (command.Result, error) {
		return f.matching().Link(ctx, f.p, application.LinkInput{Kind: kind, PrimaryID: primary, Members: members, Reason: "Synthetic confirmation"})
	})
	if err != nil {
		f.t.Fatal(err)
	}
	result, ok := c.Result()
	if !ok {
		f.t.Fatal(c.ErrorCode())
	}
	return result
}

func TestAmbiguousPaymentWaitsThenLinksAndUndoes(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "5000")
	a, b := uuid.NewString(), uuid.NewString()
	if _, err := f.write(f.revision(a, account, "-300", money.RUB, 1), request()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.write(f.revision(b, account, "-300", money.RUB, 1), request()); err != nil {
		t.Fatal(err)
	}
	f.balance(account, "available", "4700")
	g, found, err := f.store.MatchingForOperation(testContext, f.p, b)
	if err != nil || !found || g.State != matching.Clarification || len(g.Candidates) != 1 {
		t.Fatalf("case=%+v found=%v err=%v", g, found, err)
	}
	result := f.link(matching.Payment, a, a, b)
	if result.ResourceID != a {
		t.Fatal("unstable payment primary")
	}
	f.balance(account, "available", "4700")
	f.restart()
	current := f.current(a)
	_, err = f.executor.Execute(testContext, f.p, request(), func(ctx context.Context) (command.Result, error) {
		return f.ledgerService().Undo(ctx, f.p, current.DecisionID, []journal.ExpectedRevision{{OperationID: a, Revision: current.Revision}, {OperationID: b, Revision: f.current(b).Revision}}, "Undo matching")
	})
	if err != nil {
		t.Fatal(err)
	}
	f.balance(account, "available", "4700")
	g, found, err = f.store.MatchingForOperation(testContext, f.p, b)
	if err != nil || !found || g.State != matching.Clarification {
		t.Fatal(g, found, err)
	}
	if f.current(a).Participation.GroupID != "" || f.current(b).Participation.State != "waiting" {
		t.Fatal("undo did not restore waiting evidence")
	}
}

func TestSeparatePaymentContributesOnce(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "5000")
	a, b := uuid.NewString(), uuid.NewString()
	for _, id := range []string{a, b} {
		if _, err := f.write(f.revision(id, account, "-300", money.RUB, 1), request()); err != nil {
			t.Fatal(err)
		}
	}
	g, _, err := f.store.MatchingForOperation(testContext, f.p, b)
	if err != nil {
		t.Fatal(err)
	}
	key := request()
	for range 2 {
		_, err = f.executor.Execute(testContext, f.p, key, func(ctx context.Context) (command.Result, error) {
			return f.matching().Separate(ctx, f.p, g.ID, g.Revision, []matching.Member{{OperationID: b, Revision: 1}}, "Different purchase")
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	f.balance(account, "available", "4400")
}

func TestExistingTransferPreservesSideDatesAndFee(t *testing.T) {
	f := newFixture(t)
	from, to, fees := f.create(money.RUB, "5000"), f.create(money.RUB, "0"), f.create(money.BTC, "1")
	a, b := f.revision(uuid.NewString(), from, "-1000", money.RUB, 1), f.revision(uuid.NewString(), to, "1000", money.RUB, 1)
	b.Type = ledger.Income
	a.Postings = append(a.Postings, ledger.Posting{AccountID: fees, Money: cash("-0.00001", money.BTC), Role: ledger.Fee, Funding: ledger.OwnFunds})
	for _, r := range []ledger.Revision{a, b} {
		if _, err := f.write(r, request()); err != nil {
			t.Fatal(err)
		}
	}
	f.link(matching.Transfer, a.OperationID, a.OperationID, b.OperationID)
	f.balance(from, "available", "4000")
	f.balance(to, "available", "1000")
	f.balance(fees, "available", "0.99999")
	for _, id := range []string{a.OperationID, b.OperationID} {
		cs, err := f.current(id).Components()
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cs {
			if c.Kind != "fee" {
				t.Fatal("transfer principal classified as income/expense")
			}
		}
	}
}

func TestExactProofAutomaticallyLinksAndReplays(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.USDC, "5000")
	a, b := f.revision(uuid.NewString(), account, "-0.123456789123456789", money.USDC, 1), f.revision(uuid.NewString(), account, "-0.123456789123456789", money.USDC, 1)
	proof := ledger.Correspondence{Kind: "payment", Namespace: "synthetic:payment", Reference: "confirmed-payment-1"}
	a.Correspondence = &proof
	b.Correspondence = &proof
	for _, r := range []ledger.Revision{a, b} {
		r.Postings[0].Funding = ledger.OwnFunds
		r.Origin = "source"
		if _, err := f.write(r, request()); err != nil {
			t.Fatal(err)
		}
	}
	g, found, err := f.store.MatchingForOperation(testContext, f.p, a.OperationID)
	if err != nil || !found || g.State != matching.Linked {
		t.Fatal(g, found, err)
	}
	f.balance(account, "available", "4999.876543210876543211")
	if f.count("ledger_review_requests") != f.count("operation_revisions") {
		t.Fatal("review coverage missing")
	}
}
