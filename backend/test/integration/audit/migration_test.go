//go:build integration

package audit_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

func TestMigrationKeepsLegacyProtectionAndImmutableAudit(t *testing.T) {
	files := fstest.MapFS{}
	names, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if name >= "009_" {
			continue
		}
		data, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			t.Fatal(err)
		}
		files[name] = &fstest.MapFile{Data: data}
	}
	f := newFixtureUsingMigrations(t, files)
	account := f.account(money.BTC, "1")
	id := uuid.NewString()
	if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.operations(household_id,id,revision) VALUES($1,$2,1)`, f.family.ID, id); err != nil {
		t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.operation_revisions(household_id,operation_id,revision,actor_id,reason,economic_type,state,occurred_at,occurred_ns,cash_date,payer_state,human_override) VALUES($1,$2,1,$3,'Legacy correction','expense','posted','2026-09-07T10:00:00.123456Z',789,'2026-09-07','unknown',true)`, f.family.ID, id, f.p.UserID()); err != nil {
		t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.postings(household_id,operation_id,revision,position,account_id,amount,asset,role) VALUES($1,$2,1,0,$3,-0.0000000000000000123,'BTC','principal')`, f.family.ID, id, account); err != nil {
		t.Fatal(err)
	}
	if err = storage.Migrate(testContext, f.admin, migrations.Files); err != nil {
		t.Fatal(err)
	}
	if err = storage.Migrate(testContext, f.admin, migrations.Files); err != nil {
		t.Fatal(err)
	}
	r, ok, err := f.store.CurrentLedgerRevision(testContext, f.p, id)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if _, protected := r.Protections[ledger.LegacyField]; !protected {
		t.Fatal("lost legacy protection")
	}
	if r.RecordedAt.String() != "" || r.DecisionID != "" || r.PostedAt.String() != "" || r.OccurredAt.String() != "2026-09-07T10:00:00.123456789Z" {
		t.Fatal("invented audit facts")
	}
	for _, table := range []string{"ledger_decisions", "ledger_revision_audit", "ledger_field_origins", "ledger_source_facts", "ledger_review_results"} {
		var allowed bool
		if err = f.admin.QueryRow(testContext, `SELECT has_table_privilege('want_keep_app',$1,'UPDATE') OR has_table_privilege('want_keep_app',$1,'DELETE')`, "want_keep."+table).Scan(&allowed); err != nil || allowed {
			t.Fatal("mutable audit", table, err)
		}
	}
	if _, err = f.admin.Exec(testContext, `UPDATE want_keep.ledger_revision_audit SET accounting_state='excluded' WHERE household_id=$1`, f.family.ID); err == nil {
		t.Fatal("history was rewritten")
	}
}
