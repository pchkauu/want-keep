//go:build integration

package storage_test

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/url"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/uuid"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

func TestUpgradePreservesUnresolvedJobsAndHistory(t *testing.T) {
	admin, dsn := newDatabase(t)
	old := legacyMigrations(t)
	var err error
	if err = storage.Migrate(testContext, admin, old); err != nil {
		t.Fatal(err)
	}
	h, u, m := uuid.NewString(), uuid.NewString(), uuid.NewString()
	a, b := uuid.NewString(), uuid.NewString()
	for _, query := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO want_keep.households(id,name,timezone,max_members) VALUES($1,'Synthetic','UTC',2)`, []any{h}},
		{`INSERT INTO want_keep.users(id,name) VALUES($1,'Synthetic')`, []any{u}},
		{`INSERT INTO want_keep.memberships(household_id,id,user_id,active) VALUES($1,$2,$3,true)`, []any{h, m, u}},
		{`INSERT INTO want_keep.jobs(household_id,id,actor_id,kind,state,max_attempts,available_at,deadline) VALUES($1,$2,$3,'outbox','unresolved',5,clock_timestamp(),clock_timestamp()+INTERVAL '1 hour')`, []any{h, a, u}},
		{`INSERT INTO want_keep.outbox(household_id,id,actor_id,resource_type,resource_id,revision,event_type) VALUES($1,$2,$3,'synthetic',$4,1,'synthetic.old')`, []any{h, a, u, b}},
	} {
		if _, err = admin.Exec(testContext, query.sql, query.args...); err != nil {
			t.Fatal(err)
		}
	}
	if err = storage.Migrate(testContext, admin, migrations.Files); err != nil {
		t.Fatal(err)
	}
	if err = storage.Migrate(testContext, admin, migrations.Files); err != nil {
		t.Fatal(err)
	}
	var state string
	var n int
	if err = admin.QueryRow(testContext, `SELECT state FROM want_keep.jobs WHERE id=$1`, a).Scan(&state); err != nil || state != "unresolved" {
		t.Fatal("unknown state lost", err)
	}
	if err = admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.outbox`).Scan(&n); err != nil || n != 1 {
		t.Fatal("history lost", err)
	}
	parsed, _ := url.Parse(dsn)
	parsed.User = url.UserPassword("want_keep_app", "synthetic-app")
	store, err := storage.Open(testContext, storage.Config{DSN: parsed.String(), Environment: "test", MaxConnections: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
}

func legacyMigrations(t *testing.T) fs.FS {
	t.Helper()
	names, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	old := fstest.MapFS{}
	for _, name := range names {
		if name >= "010_" {
			continue
		}
		data, e := fs.ReadFile(migrations.Files, name)
		if e != nil {
			t.Fatal(e)
		}
		old[name] = &fstest.MapFile{Data: data}
	}
	return old
}

func TestUpgradeSelectsSafeSourceProgress(t *testing.T) {
	for _, test := range []struct{ name, state, cursor, gap string }{
		{"active", "ready", "page2", "history_limited"},
		{"ambiguous", "failed", "", "legacy_checkpoint_ambiguous"},
		{"unresolved", "unresolved", "", "legacy_checkpoint_ambiguous"},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newFixtureWithMigrations(t, legacyMigrations(t))
			connection := uuid.NewString()
			b := binding()
			encoded, err := json.Marshal(map[string]string{"provider": b.Provider, "environment": b.Environment, "adapter_build_digest": b.AdapterBuildDigest, "collector_image_digest": b.CollectorImageDigest, "contract_version": b.ContractVersion, "allowlist_revision": b.AllowlistRevision, "non_secret_config_revision": b.NonSecretConfigRevision, "operator_permission_revision": b.OperatorPermissionRevision})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.connections(household_id,id,provider,external_owner_id,generation,authorized) VALUES($1,$2,$3,$4,2,true)`, f.family.ID, connection, b.Provider, f.p.UserID()); err != nil {
				t.Fatal(err)
			}
			current := uuid.NewString()
			for _, row := range []struct {
				id, state, cursor, deadline string
				generation                  int
			}{{uuid.NewString(), "succeeded", "old-generation", "24 hours", 1}, {uuid.NewString(), "succeeded", "old-page", "23 hours", 2}, {current, test.state, "page2", "1 hour", 2}} {
				if _, err = f.admin.Exec(testContext, `INSERT INTO want_keep.jobs(household_id,id,actor_id,kind,connection_id,connection_generation,binding,admission_revision,state,max_attempts,available_at,deadline,cursor,coverage,gaps,secret_purpose) VALUES($1,$2,$3,'sync',$4,$5,$6,2,$7,5,clock_timestamp(),clock_timestamp()+$8::interval,$9,'partial',ARRAY['history_limited'],'api_credentials')`, f.family.ID, row.id, f.p.UserID(), connection, row.generation, encoded, row.state, row.deadline, row.cursor); err != nil {
					t.Fatal(err)
				}
			}
			if err = storage.Migrate(testContext, f.admin, migrations.Files); err != nil {
				t.Fatal(err)
			}
			progress, err := f.store.SyncProgress(testContext, f.p, connection)
			if err != nil || progress.Cursor != test.cursor || progress.Completed || len(progress.Gaps) != 1 || progress.Gaps[0] != test.gap {
				t.Fatal("active legacy checkpoint lost", err)
			}
			if test.state == "unresolved" {
				if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.store.Disconnect(ctx, connection) }); err != nil {
					t.Fatal(err)
				}
				j, err := f.store.Job(testContext, f.p, current)
				if err != nil || j.State != jobs.Unresolved || j.ExternalStarted {
					t.Fatal("legacy unresolved lost", err)
				}
				return
			}
			// No additional page is saved between upgrade and the current job's terminal failure.
			if _, err = f.admin.Exec(testContext, `UPDATE want_keep.jobs SET state='failed' WHERE id=$1`, current); err != nil {
				t.Fatal(err)
			}
			gate := f.admit(b)
			next := f.issued(gate, connection, b)
			if next.Cursor != test.cursor || next.ConnectionGeneration != 2 || len(next.Gaps) != 1 || next.Gaps[0] != test.gap {
				t.Fatal("replacement lost legacy progress")
			}
			if err = f.store.RetryJob(testContext, f.p, next, 0, true); err != nil {
				t.Fatal(err)
			}
			if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.store.Disconnect(ctx, connection) }); err != nil {
				t.Fatal(err)
			}
			currentJob, err := f.store.Job(testContext, f.p, next.ID)
			if err != nil || currentJob.State != jobs.Unresolved {
				t.Fatal("upgraded uncertainty lost", err)
			}
			if rows, err := f.store.ClaimJobs(testContext, "sync", 1, time.Minute); err != nil || len(rows) != 0 {
				t.Fatal("upgraded uncertainty repeated", err)
			}
		})
	}
}
