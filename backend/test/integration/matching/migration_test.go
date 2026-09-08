//go:build integration

package matching_test

import (
	"io/fs"
	"net/url"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

func TestMigrationDoesNotInventMatchingAndProtectsHistory(t *testing.T) {
	files := fstest.MapFS{}
	names, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if name >= "013_" {
			continue
		}
		data, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			t.Fatal(err)
		}
		files[name] = &fstest.MapFile{Data: data}
	}
	f := newFixtureUsingMigrations(t, files)
	account := f.account(money.ETH, "1")
	id := uuid.NewString()
	if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.operations(household_id,id,revision) VALUES($1,$2,1)`, f.family.ID, id); err != nil {
		t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.operation_revisions(household_id,operation_id,revision,actor_id,reason,economic_type,state,occurred_at,occurred_ns,cash_date,payer_state,human_override) VALUES($1,$2,1,$3,'Existing expense','expense','posted','2026-09-07T10:00:00.123456Z',789,'2026-09-07','unknown',false)`, f.family.ID, id, f.p.UserID()); err != nil {
		t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.postings(household_id,operation_id,revision,position,account_id,amount,asset,role) VALUES($1,$2,1,0,$3,-0.0000000000000000123,'ETH','principal')`, f.family.ID, id, account); err != nil {
		t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.ledger_revision_audit(household_id,operation_id,revision,accounting_state) VALUES($1,$2,1,'included')`, f.family.ID, id); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"category", "merchant_identity", "receipt_items"} {
		if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.ledger_field_origins(household_id,operation_id,revision,field,changed_revision,protected,protection_revision) VALUES($1,$2,1,$3,1,true,1)`, f.family.ID, id, field); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if err = storage.Migrate(testContext, f.admin, migrations.Files); err != nil {
			t.Fatal(err)
		}
	}
	r, found, err := f.store.CurrentLedgerRevision(testContext, f.p, id)
	if err != nil || !found || r.Participation.GroupID != "" || r.Correspondence != nil || r.Postings[0].Money.Amount() != "-0.0000000000000000123" || r.OccurredAt.String() != "2026-09-07T10:00:00.123456789Z" {
		t.Fatal("legacy facts changed", r, err)
	}
	if f.count("matching_cases") != 0 {
		t.Fatal("migration invented matching")
	}
	if len(r.Protections) != 3 {
		t.Fatal("migration lost classification protection")
	}
	for _, table := range []string{"matching_decision_groups", "matching_revisions", "matching_members", "matching_candidates", "ledger_participations", "ledger_contributions", "ledger_correspondences", "ledger_fee_ids"} {
		var allowed bool
		if err = f.admin.QueryRow(testContext, `SELECT has_table_privilege('want_keep_app',$1,'UPDATE') OR has_table_privilege('want_keep_app',$1,'DELETE')`, "want_keep."+table).Scan(&allowed); err != nil || allowed {
			t.Fatal("mutable history", table, err)
		}
	}
}

func TestMatchingHistorySurvivesCommandRetention(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "5000")
	a, b := uuid.NewString(), uuid.NewString()
	for _, id := range []string{a, b} {
		if _, err := f.write(f.revision(id, account, "-300", money.RUB, 1), request()); err != nil {
			t.Fatal(err)
		}
	}
	f.link(matching.Payment, a, a, b)
	before := f.count("matching_revisions")
	basis := f.count("matching_decision_groups")
	if basis == 0 {
		t.Fatal("missing immutable decision basis")
	}
	u, err := url.Parse(f.dsn)
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("want_keep_maintenance", "synthetic-maintenance")
	m, err := storage.Open(testContext, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	future := instant("2028-01-01T00:00:00Z")
	if _, err = m.CleanupCommandDetails(testContext, future, 1000); err != nil {
		t.Fatal(err)
	}
	if _, err = m.CleanupCommandTombstones(testContext, future, 1000); err != nil {
		t.Fatal(err)
	}
	if f.count("matching_revisions") != before || f.count("matching_decision_groups") != basis || f.count("command_details") != 0 {
		t.Fatal("cleanup removed matching history")
	}
	g, found, err := f.store.MatchingForOperation(testContext, f.p, a)
	if err != nil || !found || len(g.Members) != 2 {
		t.Fatal("lost matching after cleanup", g, err)
	}
	if _, err = f.admin.Exec(testContext, `UPDATE want_keep.matching_revisions SET reason='rewrite' WHERE household_id=$1`, f.family.ID); err == nil {
		t.Fatal("immutable matching history rewritten")
	}
	f.balance(account, "owned", "4700")
}

func TestMatchingRetainedPreFundingExpense(t *testing.T) {
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
	account := f.account(money.RUB, "5000")
	legacy := uuid.NewString()
	if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.operations(household_id,id,revision) VALUES($1,$2,1)`, f.family.ID, legacy); err != nil {
		t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.operation_revisions(household_id,operation_id,revision,actor_id,reason,economic_type,state,occurred_at,occurred_ns,cash_date,payer_state,human_override) VALUES($1,$2,1,$3,'Retained payment','expense','posted','2026-09-07T12:00:00Z',0,'2026-09-07','unknown',false)`, f.family.ID, legacy, f.p.UserID()); err != nil {
		t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.postings(household_id,operation_id,revision,position,account_id,amount,asset,role) VALUES($1,$2,1,0,$3,-300,'RUB','principal')`, f.family.ID, legacy, account); err != nil {
		t.Fatal(err)
	}
	if err = storage.Migrate(testContext, f.admin, migrations.Files); err != nil {
		t.Fatal(err)
	}
	incoming := f.revision(uuid.NewString(), account, "-300", money.RUB, 1)
	if _, err = f.write(incoming, request()); err != nil {
		t.Fatal(err)
	}
	old, newer := f.current(legacy), f.current(incoming.OperationID)
	t.Logf("retained cash funding=%q; incoming cash funding=%q", old.Postings[0].Funding, newer.Postings[0].Funding)
	c := f.client(f.p)
	c.result("/transactions/"+legacy+"/links", map[string]any{"kind": "receipt_match", "expectedRevisions": f.versions(legacy, incoming.OperationID), "reason": "Same retained payment"})
}
