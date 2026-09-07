//go:build integration

package identity_test

import (
	"testing"
	"time"

	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
)

func TestBootstrapLoginRecoveryAndRevocation(t *testing.T) {
	f := newFixture(t)
	a := f.browser()
	key, codes, me := a.setup()
	if len(codes) != 10 || me.Membership.Status != "active" {
		t.Fatal("bootstrap result")
	}
	cookie := a.cookies["want_keep_session"]
	if !cookie.Secure || !cookie.HttpOnly || cookie.Domain != "" || cookie.Path != "/" || cookie.MaxAge != 43200 {
		t.Fatal("unsafe session cookie")
	}
	a.call("POST", "/household/bootstrap", map[string]any{"name": "Synthetic X", "householdName": "Another", "timezone": "UTC", "locale": "en", "reportingAsset": "USD", "bootstrapToken": string(f.bootstrap)}, 409)
	other := f.browser()
	other.login(key, "login")
	f.restart()
	other.call("GET", "/me", nil, 200)
	recovering := f.browser()
	grant := decode[generated.RecoveryAttempt](t, recovering.call("POST", "/auth/recovery", map[string]any{"recoveryCode": codes[0]}, 200))
	recovering.call("POST", "/auth/recovery", map[string]any{"recoveryCode": codes[0]}, 401)
	a.call("GET", "/me", nil, 200)
	opts := decode[generated.EnrollmentOptions](t, recovering.call("POST", "/auth/enrollment/options", map[string]any{"purpose": "recovery", "authorizationToken": grant.EnrollmentToken}, 200))
	newKey := newAuthenticator(t, opts.UserId, true)
	result := recovering.enroll(opts, newKey, "recovery")
	if result.Me.User.Id != me.User.Id || len(result.RecoveryCodes.Codes) != 10 {
		t.Fatal("recovery identity changed")
	}
	a.call("GET", "/me", nil, 401)
	other.call("GET", "/me", nil, 401)
	attempt := decode[generated.LoginOptions](t, other.call("POST", "/auth/login/options", map[string]any{}, 200))
	other.call("POST", "/auth/login/verify", map[string]any{"attemptId": attempt.AttemptId, "credential": key.assertion(t, attempt.Challenge, "http://localhost", "localhost", 5)}, 401)
	recovering.call("POST", "/auth/recovery", map[string]any{"recoveryCode": codes[1]}, 401)
	fresh := f.browser()
	fresh.login(newKey, "login")
	page := decode[generated.PasskeyPage](t, fresh.call("GET", "/security/passkeys", nil, 200))
	if len(page.Items) != 1 {
		t.Fatal("old keys remain")
	}
	fresh.call("DELETE", "/security/passkeys/"+page.Items[0].Id, nil, 409)
	fresh.call("POST", "/auth/logout", nil, 204)
	fresh.call("GET", "/me", nil, 401)
}
func TestCeremonyRejectionsAndSingleUse(t *testing.T) {
	f := newFixture(t)
	key, _, _ := f.browser().setup()
	for _, bad := range []string{"origin", "rp", "uv", "signature", "handle", "raw_id", "challenge", "counter"} {
		t.Run(bad, func(t *testing.T) {
			b := f.browser()
			o := decode[generated.LoginOptions](t, b.call("POST", "/auth/login/options", map[string]any{}, 200))
			origin, rp, challenge, flags := "http://localhost", "localhost", o.Challenge, byte(5)
			switch bad {
			case "origin":
				origin = "https://attacker.invalid"
			case "rp":
				rp = "attacker.invalid"
			case "uv":
				flags = 1
			case "challenge":
				challenge = "wrong"
			case "counter":
				key.count = 0
			}
			credential := key.assertion(t, challenge, origin, rp, flags)
			switch bad {
			case "signature":
				credential["signature"] = "AAAA"
			case "handle":
				credential["userHandle"] = "AAAA"
			case "raw_id":
				credential["id"] = "AAAA"
			}
			if bad == "counter" { // Establish a larger persisted counter before submitting this stale signed assertion.
				key.count = 10
				f.browser().login(key, "login")
			}
			b.call("POST", "/auth/login/verify", map[string]any{"attemptId": o.AttemptId, "credential": credential}, 400)
			b.call("POST", "/auth/login/verify", map[string]any{"attemptId": o.AttemptId, "credential": credential}, 400)
		})
	}
}
func TestSessionAndFreshnessBoundaries(t *testing.T) {
	f := newFixture(t)
	b := f.browser()
	key, _, _ := b.setup()
	f.advance(5 * time.Minute)
	b.call("POST", "/security/recovery-codes", map[string]any{}, 403)
	b.login(key, "reauthentication")
	b.call("POST", "/security/recovery-codes", map[string]any{}, 200)
	f.advance(29 * time.Minute)
	b.call("GET", "/me", nil, 200)
	f.advance(time.Minute)
	b.call("GET", "/me", nil, 401)
	b.call("POST", "/auth/session/activity", map[string]any{}, 401)
	b.login(key, "login")
	for range 35 {
		f.advance(20 * time.Minute)
		b.call("POST", "/auth/session/activity", map[string]any{}, 204)
	}
	f.advance(20 * time.Minute)
	b.call("POST", "/auth/session/activity", map[string]any{}, 401)
}
