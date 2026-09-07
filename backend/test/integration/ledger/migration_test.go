//go:build integration

package ledger_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

func TestMigrationPreservesLegacyFactsAndImmutableHistory(t *testing.T) {
	files := fstest.MapFS{}
	names, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if name >= "008_" {
			continue
		}
		data, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			t.Fatal(err)
		}
		files[name] = &fstest.MapFile{Data: data}
	}
	f := newFixtureUsingMigrations(t, files)
	accountID := f.account(money.BTC, "1")
	id := uuid.NewString()
	if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.operations(household_id,id,revision) VALUES($1,$2,1)`, f.family.ID, id); err != nil {
		t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.operation_revisions(household_id,operation_id,revision,actor_id,reason,economic_type,state,occurred_at,occurred_ns,cash_date,payer_state,human_override) VALUES($1,$2,1,$3,'synthetic legacy','expense','posted','2026-08-31T20:59:59.123456Z',789,'2026-08-31','unknown',false)`, f.family.ID, id, f.p.UserID()); err != nil {
		t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.postings(household_id,operation_id,revision,position,account_id,amount,asset,role) VALUES($1,$2,1,0,$3,-0.0000000000000000123,'BTC','principal')`, f.family.ID, id, accountID); err != nil {
		t.Fatal(err)
	}
	if err = storage.Migrate(testContext, f.admin, migrations.Files); err != nil {
		t.Fatal(err)
	}
	if err = storage.Migrate(testContext, f.admin, migrations.Files); err != nil {
		t.Fatal("repeat", err)
	}
	r, exists, err := f.store.CurrentLedgerRevision(testContext, f.p, id)
	if err != nil || !exists || r.Postings[0].Money.Amount() != "-0.0000000000000000123" || r.OccurredAt.String() != "2026-08-31T20:59:59.123456789Z" || r.Origin != "" || r.PostedAt.String() != "" || r.ExpenseMonth.String() != "" {
		t.Fatal("migration invented historical facts", err)
	}
	for _, table := range []string{"operation_revisions", "postings", "transaction_details"} {
		var allowed bool
		if err = f.admin.QueryRow(testContext, `SELECT has_table_privilege('want_keep_app',$1,'UPDATE') OR has_table_privilege('want_keep_app',$1,'DELETE')`, "want_keep."+table).Scan(&allowed); err != nil || allowed {
			t.Fatal("mutable financial history", table, err)
		}
	}
	r.Revision = 2
	r.Postings[0].Money = cash("-0.0000000000000000456", money.BTC)
	if _, err = f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, `UPDATE want_keep.transaction_details SET note='tampered' WHERE household_id=$1`, f.family.ID); err == nil {
		t.Fatal("history trigger permitted rewrite")
	}
	old, err := f.store.LedgerRevision(testContext, f.p, id, 1)
	if err != nil || old.Postings[0].Money.Amount() != "-0.0000000000000000123" {
		t.Fatal("legacy revision changed", err)
	}
}
