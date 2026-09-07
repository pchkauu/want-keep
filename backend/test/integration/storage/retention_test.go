//go:build integration

package storage_test

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func TestRetentionBoundariesRecoveryAndIndependentAudit(t *testing.T) {
	f := newFixture(t)
	id := f.account(money.ETH, "1")
	key := request()
	op := f.revision(uuid.NewString(), id, "-0.000000000000000001", money.ETH, 1)
	if _, err := f.write(op, key); err != nil {
		t.Fatal(err)
	}
	pendingKey := request()
	if _, err := f.executor.Register(testContext, f.p, pendingKey); err != nil {
		t.Fatal(err)
	}
	at := func(days int, delta time.Duration) time.Time {
		return f.now.Time().Add(time.Duration(days)*24*time.Hour + delta)
	}
	when := func(days int, delta time.Duration) string { return at(days, delta).Format(time.RFC3339Nano) }
	authorize := func(ctx context.Context, p household.Principal, r command.Result) error {
		_, err := f.store.LedgerRevision(ctx, p, r.ResourceID, r.Revision)
		return err
	}
	for _, row := range []struct {
		days    int
		delta   time.Duration
		expired bool
	}{{90, -1, false}, {90, 0, true}, {90, 1, true}} {
		_, err := commands.NewQueries(f.store, authorize).Read(testContext, f.p, key.ID, instant(when(row.days, row.delta)))
		if row.expired != errors.Is(err, command.ErrCommandExpired) {
			t.Fatalf("detail boundary: %v", err)
		}
	}
	for _, delta := range []time.Duration{-1, 0, 1} {
		entries, _, err := commands.NewQueries(f.store, authorize).Recent(testContext, f.p, instant(when(30, delta)), "", 100)
		if err != nil {
			t.Fatal(err)
		}
		expected := 1
		if delta < 0 {
			expected = 2
		}
		if len(entries) != expected {
			t.Fatalf("recent boundary %s: %d", delta, len(entries))
		}
	}
	u, _ := url.Parse(f.dsn)
	u.User = url.UserPassword("want_keep_maintenance", "synthetic-maintenance")
	maintenance, err := storage.Open(testContext, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer maintenance.Close()
	if n, err := maintenance.CleanupCommandDetails(testContext, instant(when(90, -1)), 100); err != nil || n != 0 {
		t.Fatalf("early detail cleanup: %d %v", n, err)
	}
	if n, err := maintenance.CleanupCommandDetails(testContext, instant(when(90, 0)), 100); err != nil || n != 1 {
		t.Fatalf("detail cleanup: %d %v", n, err)
	}
	if f.count("command_details") != 1 || f.count("command_tombstones") != 2 {
		t.Fatal("pending or tombstone removed")
	}
	c, err := commands.NewQueries(f.store, authorize).Read(testContext, f.p, key.ID, instant(when(100, 0)))
	if !errors.Is(err, command.ErrCommandExpired) {
		t.Fatal(err)
	}
	outcome, err := c.Recover(f.p, key.Kind, key.PayloadHash, instant(when(100, 0)))
	if err != nil || outcome.Result.ResourceID != op.OperationID {
		t.Fatal("compact outcome missing")
	}
	if _, err = commands.NewQueries(f.store, authorize).Read(testContext, f.q, key.ID, instant(when(100, 0))); !errors.Is(err, command.ErrCommandNotFound) {
		t.Fatal("partner read expired command")
	}
	if _, err = commands.NewQueries(f.store, func(context.Context, household.Principal, command.Result) error { return household.ErrForbidden }).Read(testContext, f.p, key.ID, instant(when(100, 0))); !errors.Is(err, household.ErrForbidden) {
		t.Fatal("result authorization bypass")
	}
	if n, err := maintenance.CleanupCommandTombstones(testContext, instant(when(400, -1)), 100); err != nil || n != 0 {
		t.Fatalf("early tombstone cleanup: %d %v", n, err)
	}
	if n, err := maintenance.CleanupCommandTombstones(testContext, instant(when(400, 0)), 100); err != nil || n != 1 {
		t.Fatalf("tombstone cleanup: %d %v", n, err)
	}
	if f.count("postings") != 1 || f.count("operation_revisions") != 1 || f.count("command_tombstones") != 1 {
		t.Fatal("cleanup changed financial history or unresolved command")
	}
	if _, err = f.executor.Register(testContext, f.p, key); !errors.Is(err, command.ErrCommandExpired) {
		t.Fatalf("old financial command registered again: %v", err)
	}
	entries, _, err := commands.NewQueries(f.store, authorize).Recent(testContext, f.p, instant(when(500, 0)), "", 100)
	if err != nil || len(entries) != 1 || entries[0].ID() != pendingKey.ID {
		t.Fatal("old unresolved command lost")
	}
}
