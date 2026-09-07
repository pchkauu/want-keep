//go:build integration

package storage_test

import (
	"io/fs"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

func TestMigrationsAreAtomicVerifiedAndConcurrent(t *testing.T) {
	db, _ := newDatabase(t)
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, 3)
	for range 3 {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; errs <- storage.Migrate(testContext, db, migrations.Files) }()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := storage.Migrate(testContext, db, migrations.Files); err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	names, _ := fs.Glob(migrations.Files, "*.sql")
	for _, name := range names {
		data, _ := fs.ReadFile(migrations.Files, name)
		files[name] = &fstest.MapFile{Data: data}
	}
	files[names[0]] = &fstest.MapFile{Data: []byte("SELECT 1;")}
	if err := storage.Migrate(testContext, db, files); err == nil {
		t.Fatal("modified migration accepted")
	}
	if _, err := db.Exec(testContext, "INSERT INTO public.want_keep_schema_migrations VALUES(99,'099_unknown.sql','unknown')"); err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(testContext, db, migrations.Files); err == nil {
		t.Fatal("unknown version accepted")
	}
	broken, _ := newDatabase(t)
	bad := fstest.MapFS{"001_broken.sql": {Data: []byte("CREATE TABLE should_rollback(id integer); SELECT definitely_missing();")}}
	if err := storage.Migrate(testContext, broken, bad); err == nil {
		t.Fatal("bad migration passed")
	}
	var relation *string
	if err := broken.QueryRow(testContext, "SELECT to_regclass('should_rollback')::text").Scan(&relation); err != nil {
		t.Fatal(err)
	}
	if relation != nil {
		t.Fatal("partial DDL survived")
	}
	var count int
	if err := broken.QueryRow(testContext, "SELECT count(*) FROM public.want_keep_schema_migrations").Scan(&count); err != nil || count != 0 {
		t.Fatal("failed migration marked applied")
	}
}
