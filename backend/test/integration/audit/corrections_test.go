//go:build integration

package audit_test

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (c *client) expense(account string, asset money.Asset, amount string) generated.Transaction {
	c.f.t.Helper()
	in := map[string]any{"type": "expense", "accountId": account, "amount": map[string]any{"asset": asset, "amount": amount}, "occurredAt": "2026-09-07T10:00:00Z", "payer": map[string]any{"state": "known", "memberId": c.f.members[0].ID}, "allocation": map[string]any{"mode": "unresolved", "reason": "Synthetic allocation"}}
	result := decode[generated.CommandSucceeded](c.f.t, c.call("POST", "/transactions", uuid.NewString(), in, 202))
	if result.Status != "succeeded" {
		c.f.t.Fatal(result)
	}
	return c.transaction(result.Result.Id)
}
func (c *client) transaction(id string) generated.Transaction {
	c.f.t.Helper()
	return decode[generated.Transaction](c.f.t, c.call("GET", "/transactions/"+id, "", nil, 200))
}
func (c *client) correct(r generated.Transaction, fields map[string]any) generated.Transaction {
	c.f.t.Helper()
	fields["expectedRevision"] = r.Revision
	fields["reason"] = "Synthetic correction"
	result := decode[generated.CommandSucceeded](c.f.t, c.call("POST", "/transactions/"+r.Id+"/corrections", uuid.NewString(), fields, 202))
	if result.Status != "succeeded" {
		c.f.t.Fatal(result)
	}
	return c.transaction(r.Id)
}
func (c *client) undo(r generated.Transaction, id string) generated.Transaction {
	c.f.t.Helper()
	in := map[string]any{"decisionId": id, "reason": "Synthetic undo", "expectedRevisions": []any{map[string]any{"transactionId": r.Id, "expectedRevision": r.Revision}}}
	result := decode[generated.CommandSucceeded](c.f.t, c.call("POST", "/transactions/"+r.Id+"/undo", uuid.NewString(), in, 202))
	if result.Status != "succeeded" {
		c.f.t.Fatal(result)
	}
	return c.transaction(r.Id)
}

func TestCorrectionsSelectiveUndoAndHistory(t *testing.T) {
	f := newFixture(t)
	a, b := f.client(f.p), f.client(f.q)
	account := f.create(money.RUB, "5000")
	r := a.expense(account, money.RUB, "500")
	principal := append([]generated.Posting(nil), r.Postings...)
	principal[0].Money.Amount = "-700"
	r = a.correct(r, map[string]any{"principal": principal})
	decision := *r.DecisionId
	f.balance(account, "owned", "4300")
	r = b.correct(r, map[string]any{"merchant": "Другой продавец", "note": "Поздняя правка партнёра"})
	r = a.undo(r, decision)
	f.balance(account, "owned", "4500")
	if r.Merchant == nil || *r.Merchant != "Другой продавец" || r.ActorId != string(f.p.UserID()) {
		t.Fatal("later change lost", r)
	}
	history := decode[generated.TransactionHistoryPage](t, a.call("GET", "/transactions/"+r.Id+"/history?limit=2", "", nil, 200))
	if len(history.Items) != 2 || history.NextCursor == nil || history.Items[0].Before == nil {
		t.Fatal("history missing")
	}
	boundary, _ := contract.NewBoundary()
	data, _ := json.Marshal(history)
	var validated generated.TransactionHistoryPage
	if err := boundary.Decode("TransactionHistoryPage", data, &validated); err != nil {
		t.Fatal(err)
	}
	a.call("GET", "/transactions/"+r.Id+"/history?cursor="+url.QueryEscape(*history.NextCursor), "", nil, 200)
	b.call("GET", "/transactions/"+r.Id+"/history?cursor="+url.QueryEscape(*history.NextCursor), "", nil, 400)
	old := decode[generated.Transaction](t, a.call("GET", "/transactions/"+r.Id+"/revisions/1", "", nil, 200))
	if old.Postings[0].Money.Amount != "-500" {
		t.Fatal("history mutated")
	}
}
func TestNativeCorrectionRoundTripAndExclusion(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	for _, asset := range []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH} {
		account := f.create(asset, "5000")
		r := c.expense(account, asset, "500")
		principal := append([]generated.Posting(nil), r.Postings...)
		principal[0].Money.Amount = "-700.00000000000000000123"
		r = c.correct(r, map[string]any{"principal": principal})
		if r.Postings[0].Money.Amount != "-700.00000000000000000123" {
			t.Fatal("lost precision")
		}
		out := decode[generated.CommandSucceeded](t, c.call("POST", "/transactions/"+r.Id+"/exclude", uuid.NewString(), map[string]any{"expectedRevision": r.Revision, "reason": "Ошибочная запись"}, 202))
		if out.Status != "succeeded" {
			t.Fatal(out)
		}
		r = c.transaction(r.Id)
		if r.State != "posted" || r.AccountingState != "excluded" || len(r.BalanceEffects) != 0 || len(r.EconomicComponents) != 0 {
			t.Fatal("exclusion changed source or retained effect")
		}
		f.balance(account, "owned", "5000")
		r = c.undo(r, *r.DecisionId)
		f.balance(account, "owned", "4299.99999999999999999877")
	}
}
func TestUndoRejectsOverlappingABAAndNoOp(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	r := c.expense(f.create(money.RUB, "5000"), money.RUB, "500")
	r = c.correct(r, map[string]any{"merchant": "A"})
	first := *r.DecisionId
	r = c.correct(r, map[string]any{"merchant": "B"})
	r = c.correct(r, map[string]any{"merchant": "A"})
	out := decode[generated.CommandFailed](t, c.call("POST", "/transactions/"+r.Id+"/undo", uuid.NewString(), map[string]any{"decisionId": first, "reason": "undo earlier", "expectedRevisions": []any{map[string]any{"transactionId": r.Id, "expectedRevision": r.Revision}}}, 202))
	if out.Status != "failed" {
		t.Fatal("ABA undo allowed")
	}
	before := f.count("operation_revisions")
	events := f.count("outbox")
	c.call("POST", "/transactions/"+r.Id+"/corrections", uuid.NewString(), map[string]any{"expectedRevision": r.Revision, "reason": "same", "merchant": "A"}, 202)
	if f.count("operation_revisions") != before || f.count("outbox") != events {
		t.Fatal("no-op changed state")
	}
}
func TestUndoRestoresPreviousProtectionProvenance(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	r := c.expense(f.create(money.RUB, "5000"), money.RUB, "500")
	r = c.correct(r, map[string]any{"merchant": "First override"})
	first := *r.DecisionId
	r = c.correct(r, map[string]any{"merchant": "Second override"})
	r = c.undo(r, *r.DecisionId)
	for _, p := range r.ProtectedFields {
		if p.Field == "merchant" {
			if p.DecisionId == nil || *p.DecisionId != first || p.Revision != 2 {
				t.Fatal("override origin replaced by undo revision", p)
			}
			return
		}
	}
	t.Fatal("previous override disappeared")
}
func TestReviewIsVersionBoundAndDoesNotLoop(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	r := c.expense(f.create(money.RUB, "5000"), money.RUB, "500")
	in := journal.ReviewInput{OperationID: r.Id, Revision: 1, State: "reviewed", Rationale: "Synthetic review"}
	review := func(in journal.ReviewInput) error {
		return f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.ledgerService().CompleteReview(ctx, f.p, in) })
	}
	before, events := f.count("operation_revisions"), f.count("outbox")
	if err := review(in); err != nil {
		t.Fatal(err)
	}
	if err := review(in); err != nil {
		t.Fatal(err)
	}
	if f.count("operation_revisions") != before || f.count("outbox") != events || c.transaction(r.Id).AiState != "reviewed" {
		t.Fatal("review loop")
	}
	if c.transaction(r.Id).Review == nil || c.transaction(r.Id).Review.Rationale != in.Rationale {
		t.Fatal("safe review outcome unavailable")
	}
	r = c.correct(r, map[string]any{"merchant": "Protected"})
	name := "Model replacement"
	in.Revision = uint64(r.Revision)
	in.Correction = &ledger.Correction{Merchant: &name}
	if err := review(in); err == nil {
		t.Fatal("model overwrote human")
	}
	in.Revision = 1
	in.Rationale = "Different result"
	if err := review(in); err == nil {
		t.Fatal("replay payload changed")
	}
}
