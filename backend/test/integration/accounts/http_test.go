//go:build integration

package accounts_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	delivery "github.com/pchkauu/want-keep/backend/internal/delivery/accounts"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	application "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
)

type client struct {
	f       *fixture
	handler http.Handler
	token   identity.Token
	p       household.Principal
}

func (f *fixture) client(p household.Principal) *client {
	f.t.Helper()
	token, err := identity.NewToken(32)
	if err != nil {
		f.t.Fatal(err)
	}
	var member household.Membership
	for _, m := range f.members {
		if m.UserID == p.UserID() {
			member = m
		}
	}
	handle := make([]byte, 32)
	if _, err = rand.Read(handle); err != nil {
		f.t.Fatal(err)
	}
	credentialID := uuid.NewString()
	err = f.store.WithinIdentity(testContext, func(ctx context.Context) error {
		profile := identity.Profile{UserID: p.UserID(), HouseholdID: p.HouseholdID(), MembershipID: member.ID, Name: "Synthetic member", Locale: "en", ReportingAsset: "RUB", Handle: handle, Generation: 1}
		if err := f.store.CreateIdentityProfile(ctx, profile); err != nil {
			return err
		}
		c := identity.Credential{ID: credentialID, UserID: p.UserID(), Name: "Synthetic credential", RPID: "localhost", RawID: handle, PublicKey: []byte{}, AAGUID: []byte{}, AttestationObject: []byte{}, AttestationClientData: []byte{}, AttestationClientHash: []byte{}, Transports: []string{}, UserVerified: true, CreatedAt: f.now.Time()}
		if err := f.store.SaveIdentityCredential(ctx, c); err != nil {
			return err
		}
		return f.store.SaveIdentitySession(ctx, identity.Session{ID: uuid.NewString(), TokenHash: token.Hash(), CredentialID: credentialID, Name: "Synthetic browser", UserID: p.UserID(), CreatedAt: f.now.Time(), AuthenticatedAt: f.now.Time(), LastActivityAt: f.now.Time()})
	})
	if err != nil {
		f.t.Fatal(err)
	}
	verifier, err := webauthn.New("localhost", "http://localhost")
	if err != nil {
		f.t.Fatal(err)
	}
	sessions, err := application.NewService(f.store, f.store, verifier, "localhost", "http://localhost", 2, func() time.Time { return f.now.Time() })
	if err != nil {
		f.t.Fatal(err)
	}
	queries := commands.NewQueries(f.store, f.store.AuthorizeCommandResult)
	h, err := delivery.New(f.service(), f.executor, queries, sessions, f.store, security.Config{Environment: "test", Origin: "http://localhost"}, func() calendar.Instant { return f.now })
	if err != nil {
		f.t.Fatal(err)
	}
	return &client{f, h, token, p}
}
func (c *client) call(method, path, key string, input any, expected int) *httptest.ResponseRecorder {
	c.f.t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		c.f.t.Fatal(err)
	}
	req := httptest.NewRequest(method, "http://localhost/api/v1"+path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost")
	req.Header.Set("X-CSRF-Token", c.token.CSRF())
	req.Header.Set("Idempotency-Key", key)
	req.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(c.token)})
	rr := httptest.NewRecorder()
	c.handler.ServeHTTP(rr, req)
	if rr.Code != expected {
		c.f.t.Fatalf("%s %s => %d want %d: %s", method, path, rr.Code, expected, rr.Body.String())
	}
	if rr.Header().Get("Cache-Control") != "no-store" {
		c.f.t.Fatal("missing no-store")
	}
	return rr
}
func decode[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rr.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func (c *client) input(asset, amount string) map[string]any {
	return map[string]any{"name": "Cash", "product": "cash", "asset": asset, "ownership": map[string]any{"scope": "personal", "personalOwnerId": string(c.p.UserID())}, "openingDate": "2026-08-01", "openingBalance": map[string]any{"amount": amount, "asset": asset}}
}
func TestHTTPAccountPrecisionPaginationAndCommandRecovery(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	b := f.client(f.q)
	ids := []string{}
	for _, asset := range []string{"RUB", "USD", "USDT", "USDC", "BTC", "ETH"} {
		key := uuid.NewString()
		input := c.input(asset, "12.00000000000000000123")
		out := decode[generated.CommandSucceeded](t, c.call("POST", "/accounts", key, input, 202))
		ids = append(ids, out.Result.Id)
		dto := decode[generated.Account](t, b.call("GET", "/accounts/"+out.Result.Id, "", nil, 200))
		value, _ := dto.Balance.Owned.AsKnownAmount()
		if value.Value.Amount != "12.00000000000000000123" || dto.Opening == nil || !dto.Opening.Confirmed {
			t.Fatal("round trip failed")
		}
		boundary, _ := contract.NewBoundary()
		if err := boundary.Decode("Account", b.call("GET", "/accounts/"+out.Result.Id, "", nil, 200).Body.Bytes(), &dto); err != nil {
			t.Fatal(err)
		}
		c.call("POST", "/accounts", key, input, 202)
		c.call("GET", "/commands/"+key, "", nil, 200)
		b.call("GET", "/commands/"+key, "", nil, 404)
		input["name"] = "changed"
		c.call("POST", "/accounts", key, input, 409)
	}
	if f.count("accounts") != 6 || f.count("account_events") != 6 {
		t.Fatal("duplicate creation")
	}
	bank := f.importAccount(f.admit(), f.input(f.connection(f.p)))
	if bank.Account == nil {
		t.Fatal(bank.Reason)
	}
	c.call("GET", "/accounts/"+bank.Account.ID, "", nil, 200)
	if f.count("accounts") != 7 {
		t.Fatal("AC-002 account set incomplete")
	}
	if err := f.store.WithinFinancialRead(testContext, f.p, func(ctx context.Context) error {
		totals, err := f.service().NativeTotals(ctx, f.p)
		if err == nil && (len(totals) != 6 || totals[2].Owned.KnownSubtotal.Amount() != "112.00000000000000000123") {
			t.Fatal("native family total duplicated or converted accounts", totals)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	page := decode[generated.AccountPage](t, c.call("GET", "/accounts?limit=2", "", nil, 200))
	if len(page.Items) != 2 || page.NextCursor == nil {
		t.Fatal("missing pagination")
	}
	c.call("GET", "/accounts?limit=2&cursor="+*page.NextCursor, "", nil, 200)
	b.call("GET", "/accounts?cursor="+*page.NextCursor, "", nil, 400)
	c.call("GET", "/commands/recent?cursor="+*page.NextCursor, "", nil, 400)
	c.call("GET", "/accounts?actor=forged", "", nil, 400)
	c.call("GET", "/accounts?limit=101", "", nil, 400)
	forged := c.input("RUB", "1")
	forged["actorId"] = string(f.q.UserID())
	c.call("POST", "/accounts", uuid.NewString(), forged, 400)
	bad := c.input("RUB", "1")
	bad["openingBalance"] = map[string]any{"amount": 1, "asset": "RUB"}
	c.call("POST", "/accounts", uuid.NewString(), bad, 400)
	if err := f.store.WithinIdentity(testContext, func(ctx context.Context) error {
		x, e := f.store.IdentitySession(ctx, c.token.Hash())
		if e != nil {
			return e
		}
		x.Revoked = true
		return f.store.SaveIdentitySession(ctx, x)
	}); err != nil {
		t.Fatal(err)
	}
	c.call("POST", "/accounts", uuid.NewString(), c.input("RUB", "1"), 401)
}
func TestHTTPCSRFAndForeignResources(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	foreign := f.otherFamily()
	foreignID := foreign.create("RUB", "100")
	c.call("GET", "/accounts/"+foreignID, "", nil, 404)
	input := c.input("RUB", "100")
	raw, _ := json.Marshal(input)
	for _, origin := range []string{"http://localhost", "https://untrusted.invalid"} {
		req := httptest.NewRequest("POST", "http://localhost/api/v1/accounts", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		req.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(c.token)})
		rr := httptest.NewRecorder()
		c.handler.ServeHTTP(rr, req)
		if rr.Code < 400 {
			t.Fatal("CSRF bypass")
		}
	}
	if f.count("accounts") != 1 {
		t.Fatal("unauthorized effect")
	}
}

func TestHTTPCommandRetentionAndPartialOpening(t *testing.T) {
	f := newFixture(t)
	present := f.now
	f.now = instant(present.Time().Add(-90 * 24 * time.Hour).Format(time.RFC3339Nano))
	id := f.create("RUB", "5000")
	var key string
	if err := f.admin.QueryRow(testContext, `SELECT id FROM want_keep.command_tombstones WHERE household_id=$1`, f.family.ID).Scan(&key); err != nil {
		t.Fatal(err)
	}
	pending := request()
	if _, err := f.executor.Register(testContext, f.p, pending); err != nil {
		t.Fatal(err)
	}
	f.now = present
	c := f.client(f.p)
	partner := f.client(f.q)
	expired := decode[generated.ExpiredCommand](t, c.call("GET", "/commands/"+key, "", nil, 410))
	if expired.Code != "command_expired" {
		t.Fatal("expired result misreported")
	}
	partner.call("GET", "/commands/"+key, "", nil, 404)
	page := decode[generated.CommandStatusPage](t, c.call("GET", "/commands/recent", "", nil, 200))
	if len(page.Items) != 1 {
		t.Fatal("old terminal in recent or unresolved lost")
	}
	state, _ := page.Items[0].AsCommandPending()
	if state.Id != pending.ID {
		t.Fatal("wrong pending command")
	}
	dto := decode[generated.Account](t, c.call("GET", "/accounts/"+id, "", nil, 200))
	amount := func(v string) map[string]any {
		return map[string]any{"knowledge": "known", "value": map[string]any{"asset": "RUB", "amount": v}}
	}
	correction := map[string]any{"expectedRevision": dto.Revision, "date": "2026-08-02", "reason": "Counted cash", "balances": map[string]any{"owned": amount("6000"), "available": amount("6000"), "locked": amount("0"), "debt": amount("0")}}
	r := decode[generated.CommandSucceeded](t, partner.call("POST", "/accounts/"+id+"/opening-corrections", uuid.NewString(), correction, 202))
	if r.Status != "succeeded" {
		t.Fatal("partner could not correct fact")
	}
	failed := decode[generated.CommandFailed](t, c.call("POST", "/accounts/"+id+"/opening-corrections", uuid.NewString(), correction, 202))
	if failed.Error.Code != "version_conflict" {
		t.Fatal("stale opening overwritten")
	}
	foreign := f.otherFamily()
	foreignID := foreign.create("RUB", "10")
	c.call("GET", "/accounts/"+foreignID, "", nil, 404)
	ownPage := decode[generated.AccountPage](t, c.call("GET", "/accounts", "", nil, 200))
	if len(ownPage.Items) != 1 {
		t.Fatal("foreign household exposed")
	}
}
