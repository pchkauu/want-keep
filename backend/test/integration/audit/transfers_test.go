//go:build integration

package audit_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestTransferExchangeAndThirdAssetFeeCorrections(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	a, b, crypto, fee := f.create(money.RUB, "20000"), f.create(money.RUB, "0"), f.create(money.USDT, "0"), f.create(money.BTC, "1")
	in := map[string]any{"fromAccountId": a, "toAccountId": b, "occurredAt": "2026-09-07T10:00:00Z", "sent": map[string]any{"amount": "1000", "asset": "RUB"}, "received": map[string]any{"amount": "1000", "asset": "RUB"}, "fees": []any{}, "existingTransactions": []any{}}
	result := decode[generated.CommandSucceeded](t, c.call("POST", "/transfers", uuid.NewString(), in, 202))
	r := c.transaction(result.Result.Id)
	principal := append([]generated.Posting(nil), r.Postings...)
	principal[0].Money.Amount = "-2000"
	principal[1].Money.Amount = "2000"
	fees := []any{map[string]any{"accountId": a, "role": "fee", "money": map[string]any{"asset": "RUB", "amount": "-10"}}, map[string]any{"accountId": fee, "role": "fee", "money": map[string]any{"asset": "BTC", "amount": "-0.00001"}}}
	r = c.correct(r, map[string]any{"principal": principal, "fees": fees})
	f.balance(a, "owned", "17990")
	f.balance(b, "owned", "2000")
	f.balance(fee, "owned", "0.99999")
	bad := append([]generated.Posting(nil), principal...)
	bad[0].Money.Amount = "-2001"
	out := decode[generated.CommandFailed](t, c.call("POST", "/transactions/"+r.Id+"/corrections", uuid.NewString(), map[string]any{"expectedRevision": r.Revision, "reason": "unbalanced", "principal": bad}, 202))
	if out.Status != "failed" {
		t.Fatal("unbalanced transfer allowed")
	}
	f.balance(a, "owned", "17990")
	r = c.undo(r, *r.DecisionId)
	f.balance(a, "owned", "19000")
	f.balance(b, "owned", "1000")
	f.balance(fee, "owned", "1")
	in["toAccountId"] = crypto
	in["sent"] = map[string]any{"amount": "9000", "asset": "RUB"}
	in["received"] = map[string]any{"amount": "100", "asset": "USDT"}
	result = decode[generated.CommandSucceeded](t, c.call("POST", "/transfers", uuid.NewString(), in, 202))
	r = c.transaction(result.Result.Id)
	principal = append([]generated.Posting(nil), r.Postings...)
	principal[0].Money.Amount = "-9100"
	principal[1].Money.Amount = "100.123456789"
	r = c.correct(r, map[string]any{"principal": principal})
	f.balance(a, "owned", "9900")
	f.balance(crypto, "owned", "100.123456789")
	if r.Exchange == nil || r.Exchange.Sent.Amount != "9100" || r.Exchange.Received.Amount != "100.123456789" {
		t.Fatal("exchange amounts", r.Exchange)
	}
}
