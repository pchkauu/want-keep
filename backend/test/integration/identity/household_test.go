//go:build integration

package identity_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	application "github.com/pchkauu/want-keep/backend/internal/identity/application"
	domain "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
)

// The invitation application belongs to task-1.6. This fixture provisions its resulting second membership.
func (f *fixture) partner(me generated.Me) *authenticator {
	f.t.Helper()
	user, member := uuid.NewString(), uuid.NewString()
	handle, err := domain.NewToken(32)
	if err != nil {
		f.t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, "INSERT INTO want_keep.users(id,name) VALUES($1,'Synthetic B')", user); err != nil {
		f.t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, "INSERT INTO want_keep.memberships(household_id,id,user_id,active) VALUES($1,$2,$3,true)", me.Membership.HouseholdId, member, user); err != nil {
		f.t.Fatal(err)
	}
	decoded, _ := base64.RawURLEncoding.DecodeString(string(handle))
	p := domain.Profile{UserID: household.UserID(user), HouseholdID: household.HouseholdID(me.Membership.HouseholdId), MembershipID: household.MembershipID(member), Handle: decoded, Name: "Synthetic B", Locale: "en", ReportingAsset: "USD", Generation: 1}
	key := newAuthenticator(f.t, string(handle), false)
	challenge, _ := domain.NewToken(32)
	data, _ := json.Marshal(key.registration(f.t, string(challenge), "http://localhost", "localhost", 5))
	var dto generated.RegistrationCredential
	if err = json.Unmarshal(data, &dto); err != nil {
		f.t.Fatal(err)
	}
	v, err := webauthn.New("localhost", "http://localhost")
	if err != nil {
		f.t.Fatal(err)
	}
	c, err := v.Register(p, domain.Attempt{Challenge: string(challenge), RPID: "localhost", Origin: "http://localhost"}, application.Registration{ID: dto.Id, RawID: dto.RawId, ClientDataJSON: dto.ClientDataJSON, AttestationObject: dto.AttestationObject, Transports: []string{"internal"}})
	if err != nil {
		f.t.Fatal(err)
	}
	c.ID, c.UserID, c.Name, c.CreatedAt = uuid.NewString(), p.UserID, "Synthetic B passkey", f.clock()
	if err = f.store.WithinIdentity(testContext, func(ctx context.Context) error {
		if err := f.store.CreateIdentityProfile(ctx, p); err != nil {
			return err
		}
		return f.store.SaveIdentityCredential(ctx, c)
	}); err != nil {
		f.t.Fatal(err)
	}
	return key
}
func TestRecoveryPreservesPartnerAndRevokesOnlyOwnSubscriptions(t *testing.T) {
	f := newFixture(t)
	a := f.browser()
	key, codes, me := a.setup()
	partner := f.partner(me)
	b := f.browser()
	bMe := b.login(partner, "login")
	for _, m := range []generated.Me{me, bMe} {
		if err := f.store.WithinIdentity(testContext, func(ctx context.Context) error {
			return f.store.BindIdentitySubscription(ctx, household.UserID(m.User.Id), m.Session.Id, uuid.NewString())
		}); err != nil {
			t.Fatal(err)
		}
	}
	b.call("DELETE", "/security/sessions/"+me.Session.Id, nil, 401)
	page := decode[generated.PasskeyPage](t, a.call("GET", "/security/passkeys", nil, 200))
	b.call("DELETE", "/security/passkeys/"+page.Items[0].Id, nil, 401)
	foreign := decode[generated.LoginOptions](t, b.call("POST", "/auth/login/options", map[string]any{"purpose": "reauthentication"}, 200))
	b.call("POST", "/auth/login/verify", map[string]any{"attemptId": foreign.AttemptId, "credential": key.assertion(t, foreign.Challenge, "http://localhost", "localhost", 5)}, 401)
	recovering := f.browser()
	grant := decode[generated.RecoveryAttempt](t, recovering.call("POST", "/auth/recovery", map[string]any{"recoveryCode": codes[0]}, 200))
	opts := decode[generated.EnrollmentOptions](t, recovering.call("POST", "/auth/enrollment/options", map[string]any{"purpose": "recovery", "authorizationToken": grant.EnrollmentToken}, 200))
	recovering.enroll(opts, newAuthenticator(t, opts.UserId, false), "recovery")
	b.call("GET", "/me", nil, 200)
	var aRevoked, bRevoked bool
	if err := f.admin.QueryRow(testContext, "SELECT revoked FROM want_keep.identity_subscription_bindings WHERE user_id=$1", me.User.Id).Scan(&aRevoked); err != nil {
		t.Fatal(err)
	}
	if err := f.admin.QueryRow(testContext, "SELECT revoked FROM want_keep.identity_subscription_bindings WHERE user_id=$1", bMe.User.Id).Scan(&bRevoked); err != nil {
		t.Fatal(err)
	}
	if !aRevoked || bRevoked {
		t.Fatal("subscription revocation crossed user boundary")
	}
}
func TestConcurrentRecoveryCodeAndExpiredGrant(t *testing.T) {
	f := newFixture(t)
	a := f.browser()
	_, codes, _ := a.setup()
	token, _ := domain.NewToken(32)
	r := application.RequestContext{Browser: token, Source: "192.0.2.1"}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := f.service.Recover(testContext, r, domain.Token(codes[0]))
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	success, denied := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, domain.ErrUnauthorized) {
			denied++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || denied != 1 {
		t.Fatal("code consumed more than once")
	}
	grant, err := f.service.Recover(testContext, r, domain.Token(codes[1]))
	if err != nil {
		t.Fatal(err)
	}
	f.advance(10 * time.Minute)
	f.restart()
	if _, err = f.service.BeginEnrollment(testContext, r, domain.Recovery, grant.Token); !errors.Is(err, domain.ErrAttempt) {
		t.Fatal("expired grant accepted", err)
	}
	a.call("GET", "/me", nil, 200)
}
func TestPendingLoginAndEnrollmentCannotSurviveRecovery(t *testing.T) {
	f := newFixture(t)
	a := f.browser()
	key, codes, _ := a.setup()
	stale := f.browser()
	login := decode[generated.LoginOptions](t, stale.call("POST", "/auth/login/options", map[string]any{}, 200))
	enrollment := decode[generated.EnrollmentOptions](t, a.call("POST", "/auth/enrollment/options", map[string]any{"purpose": "add_passkey"}, 200))
	newOldKey := newAuthenticator(t, enrollment.UserId, false)
	r := f.browser()
	grant := decode[generated.RecoveryAttempt](t, r.call("POST", "/auth/recovery", map[string]any{"recoveryCode": codes[0]}, 200))
	opts := decode[generated.EnrollmentOptions](t, r.call("POST", "/auth/enrollment/options", map[string]any{"purpose": "recovery", "authorizationToken": grant.EnrollmentToken}, 200))
	r.enroll(opts, newAuthenticator(t, opts.UserId, false), "recovery")
	stale.call("POST", "/auth/login/verify", map[string]any{"attemptId": login.AttemptId, "credential": key.assertion(t, login.Challenge, "http://localhost", "localhost", 5)}, 401)
	a.call("POST", "/auth/enrollment/verify", map[string]any{"attemptId": enrollment.AttemptId, "name": "Stale key", "credential": newOldKey.registration(t, enrollment.Challenge, "http://localhost", "localhost", 5)}, 400)
}
