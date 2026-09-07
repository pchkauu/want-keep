//go:build integration

package matching_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestMatchingHTTPNativeRoundTripAndCursorScope(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	for _, asset := range []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH} {
		account := f.create(asset, "10")
		input := map[string]any{"type": "expense", "accountId": account, "amount": map[string]string{"asset": string(asset), "amount": "0.000000000000000123456789"}, "occurredAt": f.now.String(), "payer": map[string]string{"state": "unknown"}, "allocation": map[string]string{"mode": "unresolved", "reason": "Unresolved"}}
		c.result("/transactions", input)
		c.result("/transactions", input)
		f.balance(account, "owned", "9.999999999999999876543211")
	}
	page := decode[generated.MatchingPage](t, c.call("GET", "/matching?limit=2", "", nil, 200))
	if len(page.Items) != 2 || page.NextCursor == nil {
		t.Fatal(page)
	}
	next := decode[generated.MatchingPage](t, c.call("GET", "/matching?limit=2&cursor="+*page.NextCursor, "", nil, 200))
	if len(next.Items) != 2 || next.Items[0].Id == page.Items[0].Id {
		t.Fatal("pagination repeats")
	}
	c.call("GET", "/matching?limit=2&state=clarification&cursor="+*page.NextCursor, "", nil, 400)
	partner := f.client(f.q)
	partner.call("GET", "/matching?cursor="+*page.NextCursor, "", nil, 400)
	c.call("GET", "/matching?limit=101", "", nil, 400)
	raw := []byte(`{"expectedRevision":1,"decision":"separate","expectedRevisions":[],"reason":"invalid"}`)
	for _, headers := range []struct{ origin, csrf string }{{"http://evil.invalid", c.token.CSRF()}, {"http://localhost", "incorrect"}} {
		req := httptest.NewRequest("POST", "http://localhost/api/v1/matching/"+page.Items[0].Id+"/resolve", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", headers.origin)
		req.Header.Set("X-CSRF-Token", headers.csrf)
		req.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(c.token)})
		rr := httptest.NewRecorder()
		c.handler.ServeHTTP(rr, req)
		if rr.Code != 401 {
			t.Fatal("CSRF/origin guard", rr.Code)
		}
	}
}
func TestLinkedSourceConflictKeepsLastGoodEffectsAndEvidence(t *testing.T) {
	f := newFixture(t)
	gate := f.admit()
	connection := f.connection(f.p)
	from, to := f.create(money.RUB, "5000"), f.create(money.RUB, "0")
	proof := ledger.Correspondence{Kind: "transfer", Namespace: "synthetic:transfer", Reference: "change", FromAccountID: from, ToAccountID: to}
	a, b := f.revision(uuid.NewString(), from, "-1000", money.RUB, 1), f.revision(uuid.NewString(), to, "1000", money.RUB, 1)
	a.Type = ledger.Transfer
	b.Type = ledger.Transfer
	a.Correspondence = &proof
	b.Correspondence = &proof
	for _, r := range []*ledger.Revision{&a, &b} {
		if _, err := f.importSource(gate, connection, f.sourceInput(r, r.OperationID)); err != nil {
			t.Fatal(err)
		}
	}
	before := f.current(a.OperationID)
	a.Postings[0].Money = cash("-1100", money.RUB)
	in := f.sourceInput(&a, a.OperationID)
	in.Classification = "correction"
	in.ExpectedRevision = 1
	in.PayloadHash = strings.Repeat("b", 64)
	if _, err := f.importSource(gate, connection, in); err != nil {
		t.Fatal(err)
	}
	if f.current(a.OperationID).Revision != before.Revision {
		t.Fatal("incompatible financial revision created")
	}
	f.balance(from, "owned", "4000")
	f.balance(to, "owned", "1000")
	g, found, err := f.store.MatchingForOperation(testContext, f.p, a.OperationID)
	if err != nil || !found || g.State != matching.Conflict {
		t.Fatal(g, err)
	}
	c := f.client(f.p)
	v := c.transaction(a.OperationID)
	if !v.SourceConflict || len(v.SourceFacts) != 1 || v.SourceFacts[0].ConflictAtImport != "matching_conflict" {
		t.Fatal("source evidence conflict unavailable", v.SourceFacts)
	}
}
func TestCompoundExclusionAndUndoPreserveOnePaymentEffect(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "5000")
	a, b := uuid.NewString(), uuid.NewString()
	c := f.client(f.p)
	for _, id := range []string{a, b} {
		if _, err := f.write(f.revision(id, account, "-300", money.RUB, 1), request()); err != nil {
			t.Fatal(err)
		}
	}
	f.link(matching.Payment, a, a, b)
	c.result("/transactions/"+a+"/exclude", map[string]any{"expectedRevision": f.current(a).Revision, "relatedRevisions": f.versions(b), "reason": "Erroneous payment"})
	f.balance(account, "owned", "5000")
	v := c.transaction(a)
	decision := *v.DecisionId
	c.result("/transactions/"+a+"/undo", map[string]any{"decisionId": decision, "expectedRevisions": f.versions(a, b), "reason": "Restore accounting"})
	f.balance(account, "owned", "4700")
}
