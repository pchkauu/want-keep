//go:build integration

package familyallocation_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

func TestMigrationBackfillsUnknownWithoutInventingMembersAndProtectsHistory(t *testing.T) {
	files := fstest.MapFS{}
	names, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if name >= "016_" {
			continue
		}
		data, readErr := fs.ReadFile(migrations.Files, name)
		if readErr != nil {
			t.Fatal(readErr)
		}
		files[name] = &fstest.MapFile{Data: data}
	}
	fixture := newFixtureUsingMigrations(t, files)
	accountID := fixture.account(money.RUB, "1000")
	operationID, itemID := uuid.NewString(), uuid.NewString()
	excludedOperationID := uuid.NewString()
	linkedOperationID, groupID := uuid.NewString(), uuid.NewString()
	statements := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO want_keep.operations(household_id,id,revision) VALUES($1,$2,1)`, []any{fixture.family.ID, operationID}},
		{`INSERT INTO want_keep.operation_revisions(household_id,operation_id,revision,actor_id,reason,economic_type,state,occurred_at,occurred_ns,cash_date,payer_state,human_override) VALUES($1,$2,1,$3,'legacy expense','expense','posted','2026-09-08T10:00:00Z',0,'2026-09-08','unknown',false)`, []any{fixture.family.ID, operationID, fixture.p.UserID()}},
		{`INSERT INTO want_keep.ledger_revision_audit(household_id,operation_id,revision,accounting_state,recorded_at,recorded_ns) VALUES($1,$2,1,'included','2026-09-08T10:00:00Z',0)`, []any{fixture.family.ID, operationID}},
		{`INSERT INTO want_keep.postings(household_id,operation_id,revision,position,account_id,amount,asset,role,funding,treatment) VALUES($1,$2,1,0,$3,-1000,'RUB','principal','own','movement')`, []any{fixture.family.ID, operationID, accountID}},
		{`INSERT INTO want_keep.receipt_items(household_id,operation_id,revision,position,id,name,quantity,gross,discount,asset) VALUES($1,$2,1,0,$3,'Shared item',1,1000,0,'RUB')`, []any{fixture.family.ID, operationID, itemID}},
		{`INSERT INTO want_keep.operations(household_id,id,revision) VALUES($1,$2,1)`, []any{fixture.family.ID, excludedOperationID}},
		{`INSERT INTO want_keep.operation_revisions(household_id,operation_id,revision,actor_id,reason,economic_type,state,occurred_at,occurred_ns,cash_date,payer_state,human_override) VALUES($1,$2,1,$3,'excluded legacy expense','expense','posted','2026-09-08T10:00:00Z',0,'2026-09-08','unknown',false)`, []any{fixture.family.ID, excludedOperationID, fixture.p.UserID()}},
		{`INSERT INTO want_keep.ledger_revision_audit(household_id,operation_id,revision,accounting_state,recorded_at,recorded_ns) VALUES($1,$2,1,'excluded','2026-09-08T10:00:00Z',0)`, []any{fixture.family.ID, excludedOperationID}},
		{`INSERT INTO want_keep.postings(household_id,operation_id,revision,position,account_id,amount,asset,role,funding,treatment) VALUES($1,$2,1,0,$3,-100,'RUB','principal','own','movement')`, []any{fixture.family.ID, excludedOperationID, accountID}},
		{`INSERT INTO want_keep.operations(household_id,id,revision) VALUES($1,$2,1)`, []any{fixture.family.ID, linkedOperationID}},
		{`INSERT INTO want_keep.operation_revisions(household_id,operation_id,revision,actor_id,reason,economic_type,state,occurred_at,occurred_ns,cash_date,payer_state,human_override) VALUES($1,$2,1,$3,'linked transfer evidence','expense','posted','2026-09-08T10:00:00Z',0,'2026-09-08','unknown',false)`, []any{fixture.family.ID, linkedOperationID, fixture.p.UserID()}},
		{`INSERT INTO want_keep.ledger_revision_audit(household_id,operation_id,revision,accounting_state,recorded_at,recorded_ns) VALUES($1,$2,1,'included','2026-09-08T10:00:00Z',0)`, []any{fixture.family.ID, linkedOperationID}},
		{`INSERT INTO want_keep.postings(household_id,operation_id,revision,position,account_id,amount,asset,role,funding,treatment) VALUES($1,$2,1,0,$3,-1000,'RUB','principal','own','movement')`, []any{fixture.family.ID, linkedOperationID, accountID}},
		{`INSERT INTO want_keep.matching_cases(household_id,id,revision) VALUES($1,$2,1)`, []any{fixture.family.ID, groupID}},
		{`INSERT INTO want_keep.ledger_participations(household_id,operation_id,revision,group_id,kind,state) VALUES($1,$2,1,$3,'transfer','linked')`, []any{fixture.family.ID, linkedOperationID, groupID}},
		{`INSERT INTO want_keep.ledger_contributions(household_id,operation_id,revision,position,carrier_id,carrier_position,effect_state,effect_at,effect_ns,component_id,role) VALUES($1,$2::uuid,1,0,$2::uuid,0,'posted','2026-09-08T10:00:00Z',0,$2::text||':0','outgoing')`, []any{fixture.family.ID, linkedOperationID}},
	}
	for _, statement := range statements {
		if _, err = fixture.admin.Exec(testContext, statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	if err = storage.Migrate(testContext, fixture.admin, migrations.Files); err != nil {
		t.Fatal(err)
	}
	var aggregateState, itemState, unallocated string
	var members int
	if err = fixture.admin.QueryRow(testContext, `SELECT state FROM want_keep.ledger_allocation_snapshots WHERE household_id=$1 AND operation_id=$2 AND position=0`, fixture.family.ID, operationID).Scan(&aggregateState); err != nil {
		t.Fatal(err)
	}
	if err = fixture.admin.QueryRow(testContext, `SELECT state FROM want_keep.ledger_allocation_snapshots WHERE household_id=$1 AND operation_id=$2 AND position=1`, fixture.family.ID, operationID).Scan(&itemState); err != nil {
		t.Fatal(err)
	}
	if err = fixture.admin.QueryRow(testContext, `SELECT amount::text FROM want_keep.ledger_allocation_unallocated WHERE household_id=$1 AND operation_id=$2 AND snapshot_position=0`, fixture.family.ID, operationID).Scan(&unallocated); err != nil {
		t.Fatal(err)
	}
	if err = fixture.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.ledger_allocation_member_amounts WHERE household_id=$1 AND operation_id=$2`, fixture.family.ID, operationID).Scan(&members); err != nil {
		t.Fatal(err)
	}
	if aggregateState != "unresolved" || itemState != "unresolved" || unallocated != "1000" || members != 0 {
		t.Fatalf("backfill invented allocation: aggregate=%s item=%s unallocated=%s members=%d", aggregateState, itemState, unallocated, members)
	}
	excluded, found, loadErr := fixture.store.CurrentLedgerRevision(testContext, fixture.p, excludedOperationID)
	if loadErr != nil || !found || excluded.Accounting() != "excluded" || excluded.Allocation.State != "unresolved" || excluded.Validate() != nil {
		t.Fatalf("excluded expense backfill = %+v, found=%v, err=%v", excluded.Allocation, found, loadErr)
	}
	var linkedState string
	if err = fixture.admin.QueryRow(testContext, `SELECT state FROM want_keep.ledger_allocation_snapshots WHERE household_id=$1 AND operation_id=$2 AND position=0`, fixture.family.ID, linkedOperationID).Scan(&linkedState); err != nil || linkedState != "not_applicable" {
		t.Fatalf("linked transfer backfill = %s, err=%v", linkedState, err)
	}
	for _, table := range []string{"allocation_rule_revisions", "allocation_rule_shares", "ledger_allocation_snapshots", "ledger_allocation_inputs", "ledger_allocation_fallback_inputs", "ledger_allocation_member_amounts", "ledger_allocation_unallocated", "ledger_allocation_rule_refs", "ledger_allocation_fallback_rule_refs"} {
		var mutable bool
		if err = fixture.admin.QueryRow(testContext, `SELECT has_table_privilege('want_keep_app',$1,'UPDATE') OR has_table_privilege('want_keep_app',$1,'DELETE')`, "want_keep."+table).Scan(&mutable); err != nil || mutable {
			t.Fatalf("mutable allocation history %s: %v", table, err)
		}
	}
	if _, err = fixture.admin.Exec(testContext, `UPDATE want_keep.ledger_allocation_snapshots SET reason='invented' WHERE household_id=$1 AND operation_id=$2`, fixture.family.ID, operationID); err == nil {
		t.Fatal("immutable allocation history accepted an update")
	}
}
