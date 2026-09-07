//go:build integration

package identity_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	domain "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func TestIdentityPrivilegesAndMaintenance(t *testing.T) {
	f := newFixture(t)
	if err := f.store.IssueIdentityBootstrap(testContext, f.bootstrap, f.clock()); err == nil {
		t.Fatal("app issued operator token")
	}
	b := f.browser()
	_, _, _ = b.setup()
	// Use a dedicated connection so SET ROLE cannot escape into pooled fixture operations.
	connection, err := f.admin.Acquire(testContext)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Release()
	if _, err = connection.Exec(testContext, "SET ROLE want_keep_app"); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"UPDATE want_keep.identity_audit SET event='login_completed'", "DELETE FROM want_keep.identity_audit", "UPDATE want_keep.identity_bootstrap SET token_hash='bad'", "CREATE TABLE want_keep.forbidden_identity(id int)"} {
		if _, err = connection.Exec(testContext, query); err == nil {
			t.Fatal("privileged operation succeeded")
		}
	}
	if _, err = connection.Exec(testContext, "RESET ROLE"); err != nil {
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
	f.advance(time.Hour)
	if err = maintenance.CleanupIdentity(testContext, f.clock(), 1000); err != nil {
		t.Fatal("maintenance", err)
	}
	var count int
	if err = f.admin.QueryRow(testContext, "SELECT count(*) FROM want_keep.identity_attempts").Scan(&count); err != nil || count != 0 {
		t.Fatal("expired attempts remain", err)
	}
	if err = f.admin.QueryRow(testContext, "SELECT count(*) FROM want_keep.identity_audit").Scan(&count); err != nil || count == 0 {
		t.Fatal("audit disappeared", err)
	}
	if err = maintenance.WithinIdentity(context.Background(), func(ctx context.Context) error {
		return maintenance.SaveIdentityBootstrap(ctx, domain.BootstrapState{Initialized: false})
	}); err == nil {
		t.Fatal("maintenance modified access state")
	}
}
