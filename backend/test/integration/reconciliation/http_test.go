//go:build integration

package reconciliation_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	delivery "github.com/pchkauu/want-keep/backend/internal/delivery/reconciliation"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identityapp "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type client struct {
	f       *fixture
	handler http.Handler
	token   identity.Token
}

func (f *fixture) client(principal household.Principal) *client {
	f.t.Helper()
	token, err := identity.NewToken(32)
	if err != nil {
		f.t.Fatal(err)
	}
	var membership household.Membership
	for _, value := range f.members {
		if value.UserID == principal.UserID() {
			membership = value
		}
	}
	handle := make([]byte, 32)
	if _, err = rand.Read(handle); err != nil {
		f.t.Fatal(err)
	}
	credentialID := uuid.NewString()
	err = f.store.WithinIdentity(testContext, func(ctx context.Context) error {
		profile := identity.Profile{UserID: principal.UserID(), HouseholdID: principal.HouseholdID(), MembershipID: membership.ID, Name: "Synthetic member", Locale: "en", ReportingAsset: string(money.RUB), Handle: handle, Generation: 1}
		if err := f.store.CreateIdentityProfile(ctx, profile); err != nil {
			return fmt.Errorf("create identity profile: %w", err)
		}
		credential := identity.Credential{ID: credentialID, UserID: principal.UserID(), Name: "Synthetic credential", RPID: "localhost", RawID: handle, PublicKey: []byte{}, AAGUID: []byte{}, AttestationObject: []byte{}, AttestationClientData: []byte{}, AttestationClientHash: []byte{}, Transports: []string{}, UserVerified: true, CreatedAt: f.now.Time()}
		if err := f.store.SaveIdentityCredential(ctx, credential); err != nil {
			return fmt.Errorf("save identity credential: %w", err)
		}
		if err := f.store.SaveIdentitySession(ctx, identity.Session{ID: uuid.NewString(), TokenHash: token.Hash(), CredentialID: credentialID, Name: "Synthetic browser", UserID: principal.UserID(), CreatedAt: f.now.Time(), AuthenticatedAt: f.now.Time(), LastActivityAt: f.now.Time()}); err != nil {
			return fmt.Errorf("save identity session: %w", err)
		}
		return nil
	})
	if err != nil {
		f.t.Fatal(err)
	}
	verifier, _ := webauthn.New("localhost", "http://localhost")
	sessions, err := identityapp.NewService(f.store, f.store, verifier, "localhost", "http://localhost", 2, func() time.Time { return f.now.Time() })
	if err != nil {
		f.t.Fatal(err)
	}
	queries := commands.NewQueries(f.store, f.store.AuthorizeCommandResult)
	handler, err := delivery.New(f.reconciler, f.executor, queries, sessions, f.store, security.Config{Environment: "test", Origin: "http://localhost"}, func() calendar.Instant { return f.now })
	if err != nil {
		f.t.Fatal(err)
	}
	return &client{f: f, handler: handler, token: token}
}

func (c *client) call(method, path, key string, input any, expected int) *httptest.ResponseRecorder {
	c.f.t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		c.f.t.Fatal(err)
	}
	request := httptest.NewRequest(method, "http://localhost/api/v1"+path, bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost")
	request.Header.Set("X-CSRF-Token", c.token.CSRF())
	if key != "" {
		request.Header.Set("Idempotency-Key", key)
	}
	request.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(c.token)})
	response := httptest.NewRecorder()
	c.handler.ServeHTTP(response, request)
	if response.Code != expected {
		c.f.t.Fatalf("%s %s => %d want %d: %s", method, path, response.Code, expected, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		c.f.t.Fatal("missing no-store")
	}
	return response
}

func decode[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var result T
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestHTTPListReadResolveRecoveryAndCSRF(t *testing.T) {
	f := newFixture(t)
	id := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "1000", "900", "0", "0"), completeCoverage(), reporting.Fresh)
	f.correctOpening(id, exactAmounts(money.RUB, "900", "900", "0", "0"))
	current := f.completeReplay(id)
	client := f.client(f.p)
	partner := f.client(f.q)

	page := decode[generated.ReconciliationPage](t, client.call("GET", "/reconciliations?result=discrepant&limit=1", "", nil, http.StatusOK))
	if len(page.Items) != 1 || page.Items[0].Id != current.ID {
		t.Fatalf("missing reconciliation: %#v", page)
	}
	item := decode[generated.Reconciliation](t, partner.call("GET", "/reconciliations/"+current.ID, "", nil, http.StatusOK))
	if item.Revision != int64(current.Revision) || item.Replay.Status != generated.ReconciliationReplayStatusCompleted {
		t.Fatalf("wrong read model: %#v", item)
	}
	boundary, _ := contract.NewBoundary()
	rawItem := client.call("GET", "/reconciliations/"+current.ID, "", nil, http.StatusOK).Body.Bytes()
	if err := boundary.Decode("Reconciliation", rawItem, &item); err != nil {
		t.Fatalf("contract validation failed: %v: %s", err, rawItem)
	}
	client.call("GET", "/reconciliations/"+uuid.NewString(), "", nil, http.StatusNotFound)

	bad := map[string]any{"expectedRevision": current.Revision, "reason": "Direct availability edit", "components": []string{"available"}}
	client.call("POST", "/reconciliations/"+current.ID+"/resolve", uuid.NewString(), bad, http.StatusBadRequest)

	key := uuid.NewString()
	input := map[string]any{"expectedRevision": current.Revision, "reason": "Confirmed source difference", "components": []string{"owned"}}
	succeeded := decode[generated.CommandSucceeded](t, client.call("POST", "/reconciliations/"+current.ID+"/resolve", key, input, http.StatusAccepted))
	if succeeded.Result.Id != current.ID {
		t.Fatalf("wrong command result: %#v", succeeded)
	}
	client.call("POST", "/reconciliations/"+current.ID+"/resolve", key, input, http.StatusAccepted)
	input["reason"] = "Changed payload"
	client.call("POST", "/reconciliations/"+current.ID+"/resolve", key, input, http.StatusConflict)

	raw, _ := json.Marshal(input)
	request := httptest.NewRequest("POST", "http://localhost/api/v1/reconciliations/"+current.ID+"/resolve", bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost")
	request.Header.Set("Idempotency-Key", uuid.NewString())
	request.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(client.token)})
	response := httptest.NewRecorder()
	client.handler.ServeHTTP(response, request)
	if response.Code < 400 {
		t.Fatal("missing CSRF token was accepted")
	}
}

func TestHTTPPaginationCursorIsSessionBound(t *testing.T) {
	f := newFixture(t)
	for _, asset := range []money.Asset{money.RUB, money.USD} {
		id := f.importAccount(asset, "current", exactAmounts(asset, "100", "100", "0", "0"), completeCoverage(), reporting.Fresh)
		f.correctOpening(id, exactAmounts(asset, "100", "100", "0", "0"))
	}
	client := f.client(f.p)
	partner := f.client(f.q)
	page := decode[generated.ReconciliationPage](t, client.call("GET", "/reconciliations?limit=1", "", nil, http.StatusOK))
	if len(page.Items) != 1 || page.NextCursor == nil {
		t.Fatalf("missing cursor: %#v", page)
	}
	client.call("GET", "/reconciliations?limit=1&cursor="+*page.NextCursor, "", nil, http.StatusOK)
	partner.call("GET", "/reconciliations?limit=1&cursor="+*page.NextCursor, "", nil, http.StatusBadRequest)
}
