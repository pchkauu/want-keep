//go:build integration

package reconciliation_test

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

func TestMigrationPreservesExistingDataAndProtectsHistory(t *testing.T) {
	files := fstest.MapFS{}
	names, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if name >= "011_" {
			continue
		}
		data, readErr := fs.ReadFile(migrations.Files, name)
		if readErr != nil {
			t.Fatal(readErr)
		}
		files[name] = &fstest.MapFile{Data: data}
	}
	f := newFixtureUsingMigrations(t, files)
	accountID := uuid.NewString()
	ownership, _ := household.NewOwnership(f.family.ID, household.Shared, "")
	date, _ := calendar.ParseDate(f.now.Time().AddDate(0, 0, -7).Format("2006-01-02"))
	if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		return f.store.CreateAccount(ctx, account.Account{ID: accountID, Name: "Existing account", Ownership: ownership, Asset: money.RUB, Product: "cash", Revision: 1, OpeningDate: date})
	}); err != nil {
		t.Fatal(err)
	}
	if err = storage.Migrate(testContext, f.admin, migrations.Files); err != nil {
		t.Fatal(err)
	}
	if err = storage.Migrate(testContext, f.admin, migrations.Files); err != nil {
		t.Fatal("repeat migration", err)
	}
	if _, err = f.store.Account(testContext, f.p, accountID); err != nil {
		t.Fatal("existing account was lost", err)
	}
	reconciledAccount := f.importAccount(money.RUB, "current", exactAmounts(money.RUB, "100", "100", "0", "0"), completeCoverage(), reporting.Fresh)
	current := f.active(reconciledAccount)
	for _, table := range []string{"reconciliation_revisions", "reconciliation_components", "reconciliation_explanations", "reconciliation_operations", "reconciliation_resolutions"} {
		var allowed bool
		if err = f.admin.QueryRow(testContext, `SELECT has_table_privilege('want_keep_app',$1,'UPDATE') OR has_table_privilege('want_keep_app',$1,'DELETE')`, "want_keep."+table).Scan(&allowed); err != nil || allowed {
			t.Fatal("mutable reconciliation history", table, err)
		}
	}
	if _, err = f.admin.Exec(testContext, `UPDATE want_keep.reconciliation_revisions SET result='balanced' WHERE household_id=$1 AND reconciliation_id=$2`, f.family.ID, current.ID); err == nil {
		t.Fatal("reconciliation history accepted an update")
	}
}
