//go:build integration

package identity_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	application "github.com/pchkauu/want-keep/backend/internal/identity/application"
	domain "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
)

type failSessionRepository struct{ application.Repository }

func (f failSessionRepository) SaveIdentitySession(context.Context, domain.Session) error {
	return domain.ErrUnavailable
}
func (b *browser) context() application.RequestContext {
	r := application.RequestContext{Source: "192.0.2.1"}
	if x := b.cookies["want_keep_ceremony"]; x != nil {
		r.Browser = domain.Token(x.Value)
	}
	if x := b.cookies["want_keep_session"]; x != nil {
		r.Session = domain.Token(x.Value)
	}
	return r
}
func TestBootstrapRollbackAndUnknownCompletion(t *testing.T) {
	f := newFixture(t)
	b := f.browser()
	opts := decode[generated.EnrollmentOptions](t, b.call("POST", "/household/bootstrap", map[string]any{"name": "A", "householdName": "Family", "timezone": "UTC", "locale": "en", "reportingAsset": "USD", "bootstrapToken": string(f.bootstrap)}, 200))
	key := newAuthenticator(t, opts.UserId, false)
	encoded, _ := json.Marshal(key.registration(t, opts.Challenge, "http://localhost", "localhost", 5))
	var input application.Registration
	var dto generated.RegistrationCredential
	if err := json.Unmarshal(encoded, &dto); err != nil {
		t.Fatal(err)
	}
	input = application.Registration{ID: dto.Id, RawID: dto.RawId, ClientDataJSON: dto.ClientDataJSON, AttestationObject: dto.AttestationObject, Transports: []string{"internal"}}
	v, _ := webauthn.New("localhost", "http://localhost")
	faulty, err := application.NewService(failSessionRepository{f.store}, f.store, v, "localhost", "http://localhost", 2, f.clock)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = faulty.CompleteEnrollment(testContext, b.context(), opts.AttemptId, "Key", input); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatal(err)
	}
	var count int
	if err = f.admin.QueryRow(testContext, "SELECT count(*) FROM want_keep.users").Scan(&count); err != nil || count != 0 {
		t.Fatal("partial bootstrap committed", err)
	}
	if _, err = f.service.CompleteEnrollment(testContext, b.context(), opts.AttemptId, "Key", input); err != nil {
		t.Fatal(err)
	} // Drop successful response, including codes and session.
	if _, err = f.service.CompleteEnrollment(testContext, b.context(), opts.AttemptId, "Key", input); !errors.Is(err, domain.ErrAttempt) {
		t.Fatal("completion replay", err)
	}
	f.restart()
	login := f.browser()
	login.login(key, "login")
	login.call("POST", "/security/recovery-codes", map[string]any{}, 200)
	if err = f.admin.QueryRow(testContext, "SELECT count(*) FROM want_keep.users").Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate bootstrap", err)
	}
}
func TestAddPasskeyPreservesCodesAndSessionRevocation(t *testing.T) {
	f := newFixture(t)
	b := f.browser()
	key, codes, _ := b.setup()
	opts := decode[generated.EnrollmentOptions](t, b.call("POST", "/auth/enrollment/options", map[string]any{"purpose": "add_passkey"}, 200))
	extra := newAuthenticator(t, opts.UserId, false)
	rr := b.call("POST", "/auth/enrollment/verify", map[string]any{"attemptId": opts.AttemptId, "name": "Backup", "credential": extra.registration(t, opts.Challenge, "http://localhost", "localhost", 5)}, 200)
	if strings.Contains(rr.Body.String(), "recoveryCodes") {
		t.Fatal("add passkey unexpectedly rotates recovery codes")
	}
	backup := f.browser()
	backup.login(extra, "login")
	keys := decode[generated.PasskeyPage](t, b.call("GET", "/security/passkeys?limit=1", nil, 200))
	if keys.NextCursor == nil {
		t.Fatal("missing page cursor")
	}
	next := decode[generated.PasskeyPage](t, b.call("GET", "/security/passkeys?cursor="+*keys.NextCursor, nil, 200))
	if len(next.Items) != 1 || next.Items[0].Id == keys.Items[0].Id {
		t.Fatal("pagination")
	}
	b.call("GET", "/security/sessions?cursor="+*keys.NextCursor, nil, 400)
	all := decode[generated.PasskeyPage](t, b.call("GET", "/security/passkeys", nil, 200))
	var id string
	for _, c := range all.Items {
		if c.Name == "Backup" {
			id = c.Id
		}
	}
	b.call("DELETE", "/security/passkeys/"+id, nil, 204)
	backup.call("GET", "/me", nil, 401)
	me := b.login(key, "reauthentication")
	if me.User.Id == "" {
		t.Fatal("lost original key")
	}
	f.browser().call("POST", "/auth/recovery", map[string]any{"recoveryCode": codes[0]}, 200)
}
func TestBoundariesRateLimitsAndSafeResponses(t *testing.T) {
	f := newFixture(t)
	b := f.browser()
	_, codes, _ := b.setup()
	old := b.csrf
	b.csrf = "wrong"
	b.call("POST", "/auth/logout", nil, 401)
	b.csrf = old
	b.call("POST", "/auth/recovery", map[string]any{}, 400)
	b.call("POST", "/auth/enrollment/options", map[string]any{"purpose": "invitation", "authorizationToken": "forged"}, 400)
	b.call("POST", "/auth/enrollment/options", map[string]any{"purpose": "add_passkey", "actor": "another"}, 400)
	boundary, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	for path, schema := range map[string]string{"/me": "Me", "/security/passkeys": "PasskeyPage", "/security/sessions": "SessionPage"} {
		rr := b.call("GET", path, nil, 200)
		var target json.RawMessage
		if err = boundary.Decode(schema, rr.Body.Bytes(), &target); err != nil {
			t.Fatalf("%s: %#v", path, err)
		}
	}
	var raw string
	if err = f.admin.QueryRow(testContext, "SELECT string_agg(code_hash,',') FROM want_keep.identity_recovery_codes").Scan(&raw); err != nil {
		t.Fatal(err)
	}
	for _, code := range codes {
		if strings.Contains(raw, code) {
			t.Fatal("plaintext code persisted")
		}
	}
	r := b.context()
	for range 60 {
		err = f.service.Limit(testContext, r)
		if errors.Is(err, domain.ErrRateLimited) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Fatal("limit missing")
	}
	f.restart()
	if err = f.service.Limit(testContext, r); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatal("restart reset limit")
	}
	f.advance(15 * time.Minute)
	if err = f.service.Limit(testContext, r); err != nil {
		t.Fatal(err)
	}
	if err = f.store.IssueIdentityBootstrap(testContext, f.bootstrap, f.clock()); !errors.Is(err, domain.ErrBootstrap) {
		t.Fatal("initialized bootstrap reopened")
	}
}
