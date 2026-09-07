//go:build integration

package identity_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	application "github.com/pchkauu/want-keep/backend/internal/identity/application"
	domain "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
)

func (b *browser) invite() generated.InvitationCreated {
	b.f.t.Helper()
	state := decode[generated.InvitationState](b.f.t, b.call("GET", "/household/invitations", nil, 200))
	return decode[generated.InvitationCreated](b.f.t, b.call("POST", "/household/invitations", map[string]any{"expectedRevision": state.Revision}, 200))
}
func (b *browser) beginInvitation(token string) generated.EnrollmentOptions {
	b.f.t.Helper()
	return decode[generated.EnrollmentOptions](b.f.t, b.call("POST", "/auth/enrollment/options", map[string]any{"purpose": "invitation", "authorizationToken": token, "name": "Synthetic B", "locale": "en", "reportingAsset": "USD"}, 200))
}
func (b *browser) acceptInvitation(token string, opts generated.EnrollmentOptions, key *authenticator, expected int) generated.InitialEnrollmentResult {
	b.f.t.Helper()
	rr := b.call("POST", "/invitations/accept", map[string]any{"invitationToken": token, "enrollmentAttemptId": opts.AttemptId, "credentialName": "Partner passkey", "credential": key.registration(b.f.t, opts.Challenge, "http://localhost", "localhost", 5)}, expected)
	if expected != 200 {
		return generated.InitialEnrollmentResult{}
	}
	result := decode[generated.InitialEnrollmentResult](b.f.t, rr)
	b.csrf = result.Me.CsrfToken
	return result
}
func (a *authenticator) registrationInput(t *testing.T, opts generated.EnrollmentOptions) application.Registration {
	t.Helper()
	data, err := json.Marshal(a.registration(t, opts.Challenge, "http://localhost", "localhost", 5))
	if err != nil {
		t.Fatal(err)
	}
	var c generated.RegistrationCredential
	if err = json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	return application.Registration{ID: c.Id, RawID: c.RawId, ClientDataJSON: c.ClientDataJSON, AttestationObject: c.AttestationObject, Transports: []string{"internal"}}
}

func TestInvitationCreatesSeparateAccessAndPrivateRecovery(t *testing.T) {
	f := newFixture(t)
	a := f.browser()
	keyA, codesA, meA := a.setup()
	invite := a.invite()
	if len(invite.InvitationToken) != 43 || invite.State.Revision != 2 {
		t.Fatal("invitation entropy or revision")
	}
	b := f.browser()
	preview := decode[generated.InvitationPreview](t, b.call("POST", "/invitations/preview", map[string]any{"invitationToken": invite.InvitationToken}, 200))
	if preview.HouseholdName != "Synthetic family" || preview.InviterName != "Synthetic A" {
		t.Fatal("missing invitation context")
	}
	b.call("GET", "/household", nil, 401)
	opts := b.beginInvitation(invite.InvitationToken)
	keyB := newAuthenticator(t, opts.UserId, true)
	result := b.acceptInvitation(invite.InvitationToken, opts, keyB, 200)
	if result.Purpose != "invitation" || len(result.RecoveryCodes.Codes) != 10 || result.Me.User.Id == meA.User.Id || result.Me.Membership.HouseholdId != meA.Membership.HouseholdId || result.Me.User.Name != "Synthetic B" {
		t.Fatal("separate identity not created")
	}
	if result.Me.Preferences.Locale != "en" || result.Me.Preferences.ReportingAsset != "USD" {
		t.Fatal("member preferences lost")
	}
	for _, code := range codesA {
		for _, second := range result.RecoveryCodes.Codes {
			if code == second {
				t.Fatal("shared recovery code")
			}
		}
	}
	boundary, _ := contract.NewBoundary()
	for path, schema := range map[string]string{"/household": "Household", "/household/invitations": "InvitationState"} {
		rr := b.call("GET", path, nil, 200)
		var target json.RawMessage
		if err := boundary.Decode(schema, rr.Body.Bytes(), &target); err != nil {
			t.Fatal(path, err)
		}
		if strings.Contains(rr.Body.String(), invite.InvitationToken) || strings.Contains(rr.Body.String(), "tokenHash") || strings.Contains(rr.Body.String(), "recoveryCodes") {
			t.Fatal("private fields leaked")
		}
	}
	d := decode[generated.Household](t, b.call("GET", "/household", nil, 200))
	if len(d.Members) != 2 || d.MaxActiveMembers != 2 {
		t.Fatal("household view")
	}
	a.login(keyA, "reauthentication")
	f.browser().login(keyB, "login")
	f.browser().call("POST", "/invitations/preview", map[string]any{"invitationToken": invite.InvitationToken}, 409)
	a.call("POST", "/household/invitations", map[string]any{"expectedRevision": 3}, 409)
	var events int
	if err := f.admin.QueryRow(testContext, "SELECT count(*) FROM want_keep.outbox WHERE event_type='household.member_joined' AND actor_id=$1", result.Me.User.Id).Scan(&events); err != nil || events != 1 {
		t.Fatal("join outbox", err)
	}
	// Recovery through the real invited identity preserves the first member's session.
	r := f.browser()
	grant := decode[generated.RecoveryAttempt](t, r.call("POST", "/auth/recovery", map[string]any{"recoveryCode": result.RecoveryCodes.Codes[0]}, 200))
	recoveryOpts := decode[generated.EnrollmentOptions](t, r.call("POST", "/auth/enrollment/options", map[string]any{"purpose": "recovery", "authorizationToken": grant.EnrollmentToken}, 200))
	r.enroll(recoveryOpts, newAuthenticator(t, recoveryOpts.UserId, false), "recovery")
	a.call("GET", "/me", nil, 200)
	b.call("GET", "/me", nil, 401)
}

func TestInvitationReissueRevokeFreshnessAndUnknownIssuance(t *testing.T) {
	f := newFixture(t)
	a := f.browser()
	key, _, _ := a.setup()
	first := a.invite()
	b := f.browser()
	opts := b.beginInvitation(first.InvitationToken)
	keyB := newAuthenticator(t, opts.UserId, false)
	// A lost issuance response is recoverable only as metadata; stale retry cannot rotate it.
	a.call("POST", "/household/invitations", map[string]any{"expectedRevision": 1}, 409)
	second := a.invite()
	if second.InvitationToken == first.InvitationToken || second.State.Current.Id == first.State.Current.Id {
		t.Fatal("token reused")
	}
	b.acceptInvitation(first.InvitationToken, opts, keyB, 409)
	a.call("POST", "/household/invitations/"+first.State.Current.Id+"/revoke", map[string]any{"expectedRevision": second.State.Revision}, 400)
	opts = b.beginInvitation(second.InvitationToken)
	keyB = newAuthenticator(t, opts.UserId, false)
	f.advance(5 * time.Minute)
	a.call("POST", "/household/invitations", map[string]any{"expectedRevision": second.State.Revision}, 403)
	// Revocation still works without a fresh passkey confirmation.
	a.call("POST", "/household/invitations/"+second.State.Current.Id+"/revoke", map[string]any{"expectedRevision": second.State.Revision}, 200)
	b.call("POST", "/invitations/preview", map[string]any{"invitationToken": second.InvitationToken}, 409)
	a.login(key, "reauthentication")
	third := a.invite()
	if third.State.Revision != 5 {
		t.Fatal("revision not monotonic", third.State.Revision)
	}
	// No-op requests must not reset lifetime or revision.
	a.call("POST", "/household/invitations/"+third.State.Current.Id+"/revoke", map[string]any{"expectedRevision": 4}, 409)
}

func TestInvitationExpiredAttemptAndRestart(t *testing.T) {
	f := newFixture(t)
	a := f.browser()
	a.setup()
	invite := a.invite()
	b := f.browser()
	opts := b.beginInvitation(invite.InvitationToken)
	key := newAuthenticator(t, opts.UserId, false)
	f.advance(5 * time.Minute)
	f.restart()
	b.acceptInvitation(invite.InvitationToken, opts, key, 400)
	opts = b.beginInvitation(invite.InvitationToken)
	key = newAuthenticator(t, opts.UserId, false)
	f.restart()
	b.acceptInvitation(invite.InvitationToken, opts, key, 200)
	f2 := newFixture(t)
	a2 := f2.browser()
	a2.setup()
	other := a2.invite()
	f2.advance(24 * time.Hour)
	f2.restart()
	f2.browser().call("POST", "/invitations/preview", map[string]any{"invitationToken": other.InvitationToken}, 409)
}

func TestInvitationRollbackAndLostCompletion(t *testing.T) {
	f := newFixture(t)
	a := f.browser()
	a.setup()
	invite := a.invite()
	b := f.browser()
	opts := b.beginInvitation(invite.InvitationToken)
	key := newAuthenticator(t, opts.UserId, false)
	input := key.registrationInput(t, opts)
	v, _ := webauthn.New("localhost", "http://localhost")
	faulty, err := application.NewService(failSessionRepository{f.store}, f.store, v, "localhost", "http://localhost", 2, f.clock)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = faulty.CompleteInvitation(testContext, b.context(), domain.Token(invite.InvitationToken), opts.AttemptId, "Key", input); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatal(err)
	}
	for _, table := range []string{"users", "memberships", "identity_profiles", "identity_credentials", "identity_sessions"} {
		var n int
		if err = f.admin.QueryRow(testContext, "SELECT count(*) FROM want_keep."+table).Scan(&n); err != nil || n != 1 {
			t.Fatal("partial registration", table, n, err)
		}
	}
	var n int
	if err = f.admin.QueryRow(testContext, "SELECT count(*) FROM want_keep.outbox").Scan(&n); err != nil || n != 0 {
		t.Fatal("partial outbox", err)
	}
	f.restart()
	if _, err = f.service.CompleteInvitation(testContext, b.context(), domain.Token(invite.InvitationToken), opts.AttemptId, "Key", input); err != nil {
		t.Fatal(err)
	}
	// Discard the successful response and restart. Neither codes nor a second membership are replayed.
	f.restart()
	if _, err = f.service.CompleteInvitation(testContext, b.context(), domain.Token(invite.InvitationToken), opts.AttemptId, "Key", input); !errors.Is(err, domain.ErrAttempt) {
		t.Fatal("completion replay", err)
	}
	loggedIn := f.browser()
	loggedIn.login(key, "login")
	loggedIn.call("POST", "/security/recovery-codes", map[string]any{}, 200)
}

func TestConcurrentInvitationAcceptanceHasOneEffect(t *testing.T) {
	f := newFixture(t)
	a := f.browser()
	a.setup()
	invite := a.invite()
	type attempt struct {
		r       application.RequestContext
		options generated.EnrollmentOptions
		input   application.Registration
	}
	attempts := []attempt{}
	for range 2 {
		b := f.browser()
		opts := b.beginInvitation(invite.InvitationToken)
		key := newAuthenticator(t, opts.UserId, false)
		attempts = append(attempts, attempt{b.context(), opts, key.registrationInput(t, opts)})
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, x := range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := f.service.CompleteInvitation(testContext, x.r, domain.Token(invite.InvitationToken), x.options.AttemptId, "Key", x.input)
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
		} else if errors.Is(err, household.ErrInvitationUsed) {
			denied++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || denied != 1 {
		t.Fatal(success, denied)
	}
	var n int
	if err := f.admin.QueryRow(testContext, "SELECT count(*) FROM want_keep.memberships").Scan(&n); err != nil || n != 2 {
		t.Fatal("member limit", n, err)
	}
	if err := f.admin.QueryRow(testContext, "SELECT count(*) FROM want_keep.outbox").Scan(&n); err != nil || n != 1 {
		t.Fatal("outbox duplicate", n, err)
	}
}

func TestInvitationAcceptanceRacesWithRevocationAndReplacement(t *testing.T) {
	for _, action := range []string{"revoke", "replace"} {
		t.Run(action, func(t *testing.T) {
			f := newFixture(t)
			a := f.browser()
			a.setup()
			invite := a.invite()
			b := f.browser()
			opts := b.beginInvitation(invite.InvitationToken)
			key := newAuthenticator(t, opts.UserId, false)
			input := key.registrationInput(t, opts)
			start := make(chan struct{})
			accepted := make(chan error, 1)
			changed := make(chan error, 1)
			go func() {
				<-start
				_, err := f.service.CompleteInvitation(testContext, b.context(), domain.Token(invite.InvitationToken), opts.AttemptId, "Key", input)
				accepted <- err
			}()
			go func() {
				<-start
				var err error
				if action == "revoke" {
					_, err = f.service.RevokeInvitation(testContext, a.context(), invite.State.Current.Id, uint64(invite.State.Revision))
				} else {
					_, err = f.service.IssueInvitation(testContext, a.context(), uint64(invite.State.Revision))
				}
				changed <- err
			}()
			close(start)
			joinErr, changeErr := <-accepted, <-changed
			if joinErr == nil {
				if action == "revoke" && !errors.Is(changeErr, household.ErrInvitationRevision) {
					t.Fatal("accepted invitation revoked", changeErr)
				}
				if action == "replace" && !errors.Is(changeErr, household.ErrMemberLimit) {
					t.Fatal("invited after full membership", changeErr)
				}
			} else if !errors.Is(joinErr, household.ErrInvitationRevoked) || changeErr != nil {
				t.Fatal("invalid race outcome", joinErr, changeErr)
			}
			var members, events int
			if err := f.admin.QueryRow(testContext, "SELECT (SELECT count(*) FROM want_keep.memberships),(SELECT count(*) FROM want_keep.outbox)").Scan(&members, &events); err != nil {
				t.Fatal(err)
			}
			if joinErr == nil && (members != 2 || events != 1) || joinErr != nil && (members != 1 || events != 0) {
				t.Fatal("partial race effect", members, events)
			}
		})
	}
}

func TestInvitationDatabasePermissionsAndImmutableBinding(t *testing.T) {
	f := newFixture(t)
	a := f.browser()
	_, _, me := a.setup()
	invite := a.invite()
	b := f.browser()
	opts := b.beginInvitation(invite.InvitationToken)
	if err := f.store.WithinIdentity(testContext, func(ctx context.Context) error {
		attempt, err := f.store.IdentityAttempt(ctx, opts.AttemptId)
		if err != nil {
			return err
		}
		if attempt.InvitationRevision != 2 || attempt.GrantID != invite.State.Current.Id || string(attempt.Setup.HouseholdID) != me.Membership.HouseholdId {
			t.Fatal("attempt binding lost")
		}
		attempt.InvitationRevision = 999
		if err = f.store.SaveIdentityAttempt(ctx, attempt); err != nil {
			return err
		}
		restored, err := f.store.IdentityAttempt(ctx, opts.AttemptId)
		if err == nil && restored.InvitationRevision != 2 {
			t.Fatal("attempt binding mutated")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var appCanChangeAudit, appCanChangeHash, appCanRemoveInvites bool
	if err := f.admin.QueryRow(testContext, `SELECT has_table_privilege('want_keep_app','want_keep.household_invitation_audit','UPDATE'),
 has_column_privilege('want_keep_app','want_keep.household_invitations','token_hash','UPDATE'),
 has_table_privilege('want_keep_app','want_keep.household_invitations','DELETE')`).Scan(&appCanChangeAudit, &appCanChangeHash, &appCanRemoveInvites); err != nil {
		t.Fatal(err)
	}
	if appCanChangeAudit || appCanChangeHash || appCanRemoveInvites {
		t.Fatal("excessive invitation privileges")
	}
	if _, err := f.admin.Exec(testContext, "UPDATE want_keep.household_invitation_audit SET event=event"); err == nil {
		t.Fatal("audit is mutable")
	}
}

func TestInvitationBindingsAndPermissionRechecks(t *testing.T) {
	for _, scenario := range []string{"signed-in", "other-browser", "wrong-purpose", "revoked", "issuer-inactive", "full"} {
		t.Run(scenario, func(t *testing.T) {
			f := newFixture(t)
			a := f.browser()
			keyA, _, me := a.setup()
			invite := a.invite()
			b := f.browser()
			opts := b.beginInvitation(invite.InvitationToken)
			key := newAuthenticator(t, opts.UserId, false)
			expected := 409
			switch scenario {
			case "signed-in":
				b.login(keyA, "login")
			case "other-browser":
				b = f.browser()
				b.call("POST", "/invitations/preview", map[string]any{"invitationToken": invite.InvitationToken}, 200)
				expected = 400
			case "wrong-purpose":
				b.call("POST", "/auth/enrollment/verify", map[string]any{"attemptId": opts.AttemptId, "name": "Key", "credential": key.registration(t, opts.Challenge, "http://localhost", "localhost", 5)}, 400)
				expected = 400
			case "revoked":
				a.call("POST", "/household/invitations/"+invite.State.Current.Id+"/revoke", map[string]any{"expectedRevision": invite.State.Revision}, 200)
			case "issuer-inactive":
				if _, err := f.admin.Exec(testContext, "UPDATE want_keep.memberships SET active=false WHERE user_id=$1", me.User.Id); err != nil {
					t.Fatal(err)
				}
				expected = 400
			case "full":
				if _, err := f.admin.Exec(testContext, "UPDATE want_keep.households SET max_members=1 WHERE id=$1", me.Membership.HouseholdId); err != nil {
					t.Fatal(err)
				}
			}
			b.acceptInvitation(invite.InvitationToken, opts, key, expected)
			var n int
			if err := f.admin.QueryRow(testContext, "SELECT count(*) FROM want_keep.users").Scan(&n); err != nil || n != 1 {
				t.Fatal("unauthorized member", err)
			}
		})
	}
}

func TestInvitationHTTPRejectsSpoofingAndLeakedSecrets(t *testing.T) {
	f := newFixture(t)
	a := f.browser()
	_, _, me := a.setup()
	csrf := a.csrf
	a.csrf = "forged"
	a.call("POST", "/household/invitations", map[string]any{"expectedRevision": 1}, 401)
	a.csrf = csrf
	for _, name := range []string{"actor", "householdId", "ownerId"} {
		a.call("POST", "/household/invitations", map[string]any{"expectedRevision": 1, name: me.User.Id}, 400)
	}
	for _, revision := range []any{0, -1, 9007199254740992, "1"} {
		a.call("POST", "/household/invitations", map[string]any{"expectedRevision": revision}, 400)
	}
	i := a.invite()
	b := f.browser()
	b.call("POST", "/invitations/preview", map[string]any{"invitationToken": "not-a-token"}, 400)
	var stored string
	if err := f.admin.QueryRow(testContext, "SELECT row_to_json(i)::text FROM want_keep.household_invitations i").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, i.InvitationToken) {
		t.Fatal("plaintext invitation persisted")
	}
	// Auth-derived principal remains authoritative even after a member filter is supplied.
	rr := a.call("GET", "/household?householdId=foreign&memberId=foreign", nil, 200)
	if !strings.Contains(rr.Body.String(), me.Membership.HouseholdId) {
		t.Fatal("query replaced principal")
	}
	r := b.context()
	for range 65 {
		_, err := f.service.PreviewInvitation(context.Background(), r, domain.Token("invalid"))
		if errors.Is(err, domain.ErrRateLimited) {
			return
		}
		if !errors.Is(err, household.ErrInvitation) {
			t.Fatal(err)
		}
	}
	t.Fatal("invitation endpoint is not rate limited")
}
