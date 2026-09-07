//go:build integration

package identity_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	domain "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func TestRecoveryRenewsExistingBrowserBinding(t *testing.T) {
	f := newFixture(t)
	b := f.browser()
	_, codes, _ := b.setup()
	binding := b.cookies["want_keep_ceremony"].Value
	f.advance(29 * time.Minute)
	grant := decode[generated.RecoveryAttempt](t, b.call("POST", "/auth/recovery", map[string]any{"recoveryCode": codes[0]}, 200))
	if b.cookies["want_keep_ceremony"].Value != binding {
		t.Fatal("begin replaced a live browser binding")
	}
	f.advance(2 * time.Minute)
	opts := decode[generated.EnrollmentOptions](t, b.call("POST", "/auth/enrollment/options", map[string]any{"purpose": "recovery", "authorizationToken": grant.EnrollmentToken}, 200))
	key := newAuthenticator(t, opts.UserId, false)
	b.enroll(opts, key, "recovery")
}
func TestUnsolicitedAuthenticatorExtensionsAreRejected(t *testing.T) {
	f := newFixture(t)
	b := f.browser()
	opts := decode[generated.EnrollmentOptions](t, b.call("POST", "/household/bootstrap", map[string]any{"name": "A", "householdName": "Family", "timezone": "UTC", "locale": "en", "reportingAsset": "USD", "bootstrapToken": string(f.bootstrap)}, 200))
	key := newAuthenticator(t, opts.UserId, false)
	key.extensions = map[string]any{"unrequested-test-extension": true}
	b.call("POST", "/auth/enrollment/verify", map[string]any{"attemptId": opts.AttemptId, "name": "Key", "credential": key.registration(t, opts.Challenge, "http://localhost", "localhost", 5)}, 400)
	var families int
	if err := f.admin.QueryRow(testContext, "SELECT count(*) FROM want_keep.households").Scan(&families); err != nil || families != 0 {
		t.Fatal("invalid enrollment committed", err)
	}
	key, _, _ = b.setup()
	key.extensions = map[string]any{"unrequested-test-extension": true}
	other := f.browser()
	login := decode[generated.LoginOptions](t, other.call("POST", "/auth/login/options", map[string]any{}, 200))
	other.call("POST", "/auth/login/verify", map[string]any{"attemptId": login.AttemptId, "credential": key.assertion(t, login.Challenge, "http://localhost", "localhost", 5)}, 400)
	other.call("GET", "/me", nil, 401)
	b.call("GET", "/me", nil, 200)
}
func TestMaintenancePreservesConcurrentlyRenewedRateWindow(t *testing.T) {
	f := newFixture(t)
	scope := "synthetic-cleanup-scope"
	if err := f.store.WithinIdentity(testContext, func(ctx context.Context) error {
		return f.store.IdentityRateLimit(ctx, scope, 10, f.clock().Add(-time.Hour))
	}); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(f.admin.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("want_keep_maintenance", "synthetic-maintenance")
	maintenance, err := storage.Open(testContext, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer maintenance.Close()
	ctx, cancel := context.WithTimeout(testContext, 5*time.Second)
	defer cancel()
	updated, release := make(chan struct{}), make(chan struct{})
	updateDone, cleanupDone := make(chan error, 1), make(chan error, 1)
	go func() {
		updateDone <- f.store.WithinIdentity(ctx, func(ctx context.Context) error {
			if err := f.store.IdentityRateLimit(ctx, scope, 10, f.clock()); err != nil {
				return err
			}
			close(updated)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	select {
	case <-updated:
	case err := <-updateDone:
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	go func() { cleanupDone <- maintenance.CleanupIdentity(ctx, f.clock(), 1000) }()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		var waiting bool
		err = f.admin.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND usename='want_keep_maintenance' AND wait_event_type='Lock' AND query LIKE '%identity_rate_limits%')`).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatal("cleanup did not reach row-lock barrier")
		}
	}
	close(release)
	if err = <-updateDone; err != nil {
		t.Fatal(err)
	}
	if err = <-cleanupDone; err != nil {
		t.Fatal(err)
	}
	var attempts int
	if err = f.admin.QueryRow(testContext, "SELECT attempts FROM want_keep.identity_rate_limits WHERE scope_hash=$1", domain.Token(scope).Hash()).Scan(&attempts); err != nil || attempts != 1 {
		t.Fatal("renewed rate window deleted", err)
	}
}
