//go:build integration

package ledger_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (c *client) input(id, asset, amount string) map[string]any {
	return map[string]any{"type": "expense", "accountId": id, "amount": map[string]any{"amount": amount, "asset": asset}, "occurredAt": "2026-08-31T20:59:59.123456789Z", "payer": map[string]any{"state": "known", "memberId": string(c.f.members[1].ID)}, "allocation": map[string]any{"mode": "unresolved", "reason": "Need category and allocation"}, "merchant": "Synthetic market", "note": "Shared facts await allocation"}
}
func TestHTTPNativeRoundTripFamilyFactAndCommands(t *testing.T) {
	f := newFixture(t)
	a := f.client(f.p)
	b := f.client(f.q)
	boundary, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	for _, asset := range []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH} {
		id := f.create(asset, "1000")
		input := a.input(id, string(asset), "0.0000000000000000012345")
		key := uuid.NewString()
		c := decode[generated.CommandSucceeded](t, a.call("POST", "/transactions", key, input, 202))
		if c.Status != "succeeded" {
			t.Fatal("manual command failed", c)
		}
		raw := b.call("GET", "/transactions/"+c.Result.Id, "", nil, 200)
		var transaction generated.Transaction
		if err = boundary.Decode("Transaction", raw.Body.Bytes(), &transaction); err != nil {
			t.Fatal(err, raw.Body.String())
		}
		if transaction.Postings[0].Money.Amount != "-0.0000000000000000012345" || transaction.ExpenseMonth == nil || *transaction.ExpenseMonth != "2026-08" || transaction.ActorId != string(f.p.UserID()) || transaction.State != "posted" {
			t.Fatal("round trip or family fact", transaction)
		}
		payer, _ := transaction.Payer.AsKnownPayer()
		if payer.MemberId != string(f.members[1].ID) {
			t.Fatal("payer conflated with actor")
		}
		allocation, _ := transaction.Allocation.AsUnresolvedAllocation()
		if allocation.Mode != "unresolved" {
			t.Fatal("invented distribution")
		}
		if transaction.AiState != "waiting" || len(transaction.EconomicComponents) != 1 {
			t.Fatal("unreviewed fact missing")
		}
		a.call("POST", "/transactions", key, input, 202)
		a.call("GET", "/commands/"+key, "", nil, 200)
		b.call("GET", "/commands/"+key, "", nil, 404)
		input["note"] = "Different payload"
		a.call("POST", "/transactions", key, input, 409)
	}
	page := decode[generated.TransactionPage](t, a.call("GET", "/transactions?type=expense&limit=2", "", nil, 200))
	seen := map[string]bool{}
	for {
		for _, v := range page.Items {
			if seen[v.Id] {
				t.Fatal("duplicate page entry")
			}
			seen[v.Id] = true
		}
		if page.NextCursor == nil {
			break
		}
		cursor := url.QueryEscape(*page.NextCursor)
		a.call("GET", "/transactions?type=income&limit=2&cursor="+cursor, "", nil, 400)
		b.call("GET", "/transactions?type=expense&limit=2&cursor="+cursor, "", nil, 400)
		page = decode[generated.TransactionPage](t, a.call("GET", "/transactions?type=expense&limit=2&cursor="+cursor, "", nil, 200))
	}
	if len(seen) != 6 {
		t.Fatal("pagination lost facts", len(seen))
	}
	search := decode[generated.TransactionPage](t, a.call("GET", "/transactions?search=market&from=2026-08-31&to=2026-08-31", "", nil, 200))
	if len(search.Items) != 6 {
		t.Fatal("filters", len(search.Items))
	}
	empty := decode[generated.TransactionPage](t, a.call("GET", "/transactions?search=unmatched", "", nil, 200))
	if len(empty.Items) != 0 {
		t.Fatal("empty search")
	}
}

func TestHTTPTransfersExchangeAndUnsupportedFeatures(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	a, b, crypto, fee := f.create(money.RUB, "20000"), f.create(money.RUB, "0"), f.create(money.USDT, "0"), f.create(money.BTC, "1")
	in := map[string]any{"fromAccountId": a, "toAccountId": b, "occurredAt": "2026-09-07T10:00:00Z", "sent": map[string]any{"amount": "1000", "asset": "RUB"}, "received": map[string]any{"amount": "1000", "asset": "RUB"}, "fees": []any{map[string]any{"accountId": a, "amount": map[string]any{"amount": "10", "asset": "RUB"}}}, "existingTransactions": []any{}}
	key := uuid.NewString()
	out := decode[generated.CommandSucceeded](t, c.call("POST", "/transfers", key, in, 202))
	if out.Status != "succeeded" {
		t.Fatal(out)
	}
	f.balance(a, "owned", "18990")
	f.balance(b, "owned", "1000")
	r := decode[generated.Transaction](t, c.call("GET", "/transactions/"+out.Result.Id, "", nil, 200))
	if len(r.EconomicComponents) != 1 || r.EconomicComponents[0].Kind != "fee" {
		t.Fatal("transfer became expense")
	}
	in["existingTransactions"] = []any{map[string]any{"transactionId": out.Result.Id, "expectedRevision": 1}}
	failed := decode[generated.CommandFailed](t, c.call("POST", "/transfers", uuid.NewString(), in, 202))
	if failed.Status != "failed" || failed.Error.Code != "invalid_transaction" {
		t.Fatal("incomplete existing transfer was accepted", failed)
	}
	f.balance(a, "owned", "18990")
	f.balance(b, "owned", "1000")
	in["existingTransactions"] = []any{}
	in["toAccountId"] = crypto
	in["sent"] = map[string]any{"amount": "9000", "asset": "RUB"}
	in["received"] = map[string]any{"amount": "100", "asset": "USDT"}
	in["fees"] = []any{map[string]any{"accountId": a, "amount": map[string]any{"amount": "50", "asset": "RUB"}}, map[string]any{"accountId": fee, "amount": map[string]any{"amount": "0.00001", "asset": "BTC"}}}
	out = decode[generated.CommandSucceeded](t, c.call("POST", "/transfers", uuid.NewString(), in, 202))
	if out.Status != "succeeded" {
		t.Fatal(out)
	}
	r = decode[generated.Transaction](t, c.call("GET", "/transactions/"+out.Result.Id, "", nil, 200))
	if r.Exchange == nil || r.Exchange.Sent.Amount != "9000" || r.Exchange.Received.Amount != "100" || len(r.EconomicComponents) != 2 {
		t.Fatal(r)
	}
	f.balance(a, "owned", "9940")
	f.balance(crypto, "owned", "100")
	f.balance(fee, "owned", "0.99999")
	manual := c.input(a, "RUB", "12000")
	manual["categoryId"] = uuid.NewString()
	c.call("POST", "/transactions", uuid.NewString(), manual, 422)
	delete(manual, "categoryId")
	manual["allocation"] = map[string]any{"mode": "shares", "purpose": "shared", "members": []any{map[string]any{"memberId": string(f.members[0].ID), "share": "50"}, map[string]any{"memberId": string(f.members[1].ID), "share": "50"}}}
	c.call("POST", "/transactions", uuid.NewString(), manual, 422)
	manual = c.input(a, "RUB", "12000")
	out = decode[generated.CommandSucceeded](t, c.call("POST", "/transactions", uuid.NewString(), manual, 202))
	if out.Status != "succeeded" {
		t.Fatal("confirmed expense blocked by funds")
	}
	r = decode[generated.Transaction](t, c.call("GET", "/transactions/"+out.Result.Id, "", nil, 200))
	if r.EconomicComponents[0].Money.Amount != "-12000" || *r.ExpenseMonth != "2026-08" {
		t.Fatal("annual subscription split")
	}
}

func TestHTTPIsolationValidationAndCSRF(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	id := f.create(money.RUB, "1000")
	other := f.otherFamily()
	foreign := other.client(other.p)
	out := decode[generated.CommandSucceeded](t, c.call("POST", "/transactions", uuid.NewString(), c.input(id, "RUB", "1"), 202))
	foreign.call("GET", "/transactions/"+out.Result.Id, "", nil, 404)
	page := decode[generated.TransactionPage](t, foreign.call("GET", "/transactions?accountId="+id, "", nil, 200))
	if len(page.Items) != 0 {
		t.Fatal("cross family list leak")
	}
	for _, field := range []string{"actorId", "householdId", "state", "aiState"} {
		in := c.input(id, "RUB", "1")
		in[field] = "forged"
		c.call("POST", "/transactions", uuid.NewString(), in, 400)
	}
	for _, amount := range []any{1.2, "NaN", "1e3", " 1", "-1", "0", strings.Repeat("1", 257)} {
		in := c.input(id, "RUB", "1")
		in["amount"] = map[string]any{"amount": amount, "asset": "RUB"}
		c.call("POST", "/transactions", uuid.NewString(), in, 400)
	}
	for _, query := range []string{"limit=0", "limit=101", "limit=bad", "limit=2&limit=3", "cursor=bad", "unknown=x"} {
		c.call("GET", "/transactions?"+query, "", nil, 400)
	}
	input := c.input(id, "RUB", "1")
	raw, _ := json.Marshal(input)
	for _, bad := range []string{"csrf", "origin", "session"} {
		req := httptest.NewRequest("POST", "http://localhost/api/v1/transactions", strings.NewReader(string(raw)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.NewString())
		req.Header.Set("Origin", "http://localhost")
		req.Header.Set("X-CSRF-Token", c.token.CSRF())
		req.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(c.token)})
		switch bad {
		case "csrf":
			req.Header.Set("X-CSRF-Token", "wrong")
		case "origin":
			req.Header.Set("Origin", "http://attacker.test")
		case "session":
			req.Header.Del("Cookie")
		}
		rr := httptest.NewRecorder()
		c.handler.ServeHTTP(rr, req)
		if rr.Code < 400 {
			t.Fatal("accepted", bad)
		}
	}
}

func TestHTTPTextBoundsCountUnicodeCharacters(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	id := f.create(money.RUB, "1000")
	input := c.input(id, "RUB", "1")
	input["merchant"] = strings.Repeat("я", 2000)
	input["note"] = strings.Repeat("ю", 2000)
	input["allocation"] = map[string]any{"mode": "unresolved", "reason": strings.Repeat("э", 2000)}
	out := decode[generated.CommandSucceeded](t, c.call("POST", "/transactions", uuid.NewString(), input, 202))
	if out.Status != "succeeded" {
		t.Fatal(out)
	}
	page := decode[generated.TransactionPage](t, c.call("GET", "/transactions?search="+url.QueryEscape(strings.Repeat("я", 200)), "", nil, 200))
	if len(page.Items) != 1 || *page.Items[0].Note != input["note"] {
		t.Fatal("Unicode text changed or filter rejected")
	}
	input["note"] = strings.Repeat("я", 2001)
	c.call("POST", "/transactions", uuid.NewString(), input, 400)
}
