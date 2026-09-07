//go:build integration

package ledger_test

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
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	ledgerdelivery "github.com/pchkauu/want-keep/backend/internal/delivery/ledger"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	application "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
)

type client struct {
	f        *fixture
	handler  http.Handler
	token    identity.Token
	p        household.Principal
	sessions *application.Service
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
	lh, err := ledgerdelivery.New(f.ledgerService(), journal.NewQueries(f.store), f.executor, queries, sessions, f.store, security.Config{Environment: "test", Origin: "http://localhost"}, func() calendar.Instant { return f.now })
	if err != nil {
		f.t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/v1/transactions", lh)
	mux.Handle("/api/v1/transactions/", lh)
	mux.Handle("/api/v1/transfers", lh)
	mux.Handle("/api/v1/commands/", h)
	mux.Handle("/api/v1/accounts", h)
	mux.Handle("/api/v1/accounts/", h)
	return &client{f, mux, token, p, sessions}
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
