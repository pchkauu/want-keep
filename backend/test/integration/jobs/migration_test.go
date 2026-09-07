//go:build integration

package storage_test

import (
	"io/fs"
	"net/url"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

func TestUpgradePreservesUnresolvedJobsAndHistory(t *testing.T) {
	admin, dsn := newDatabase(t)
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
