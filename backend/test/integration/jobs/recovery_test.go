//go:build integration

package storage_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestOutboxLeasesRestartUnknownAndBoundedRetry(t *testing.T) {
	f := newFixture(t)
	id := f.account(money.USDT, "100")
	r := f.revision(uuid.NewString(), id, "-10", money.USDT, 1)
	key := request()
	if _, err := f.write(r, key); err != nil {
		t.Fatal(err)
	}
	if _, err := f.write(r, key); err != nil {
		t.Fatal(err)
	}
	if f.count("outbox") != 1 {
		t.Fatal("duplicate outbox")
	}
	start := make(chan struct{})
	claimed := make(chan []jobs.Job, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			rows, err := f.store.ClaimJobs(testContext, "outbox", 1, time.Minute)
			claimed <- rows
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(claimed)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	all := []jobs.Job{}
	for rows := range claimed {
		all = append(all, rows...)
	}
	if len(all) != 1 {
		t.Fatal("job leased twice")
	}
	old := all[0]
	if err := f.store.Heartbeat(testContext, f.p, old, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := f.admin.Exec(testContext, "UPDATE want_keep.jobs SET lease_until=clock_timestamp()-INTERVAL '1 second' WHERE id=$1", old.ID); err != nil {
		t.Fatal(err)
	}
	restarted := f.restart()
	rows, err := restarted.ClaimJobs(testContext, "outbox", 1, time.Minute)
	if err != nil || len(rows) != 1 {
		t.Fatalf("lease recovery: %v", err)
	}
	current := rows[0]
	if current.LeaseToken == old.LeaseToken || current.Attempt != old.Attempt+1 {
		t.Fatal("lease token reused")
	}
	if err = restarted.FinishJob(testContext, f.p, old); !errors.Is(err, jobs.ErrStaleAttempt) {
		t.Fatal("stale acknowledgement accepted")
	}
	if err = restarted.RetryJob(testContext, f.p, current, 0, true); err != nil {
		t.Fatal(err)
	}
	rows, err = restarted.ClaimJobs(testContext, "outbox", 1, time.Minute)
	if err != nil || len(rows) != 0 {
		t.Fatal("unknown external effect retried")
	}
	if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		return f.store.EmitEvent(ctx, "transaction", r.OperationID, 2, "synthetic.retry")
	}); err != nil {
		t.Fatal(err)
	}
	for attempt := 1; attempt <= 5; attempt++ {
		rows, err = f.store.ClaimJobs(testContext, "outbox", 1, time.Minute)
		if err != nil || len(rows) != 1 {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
		if err = f.store.RetryJob(testContext, f.p, rows[0], 0, false); err != nil {
			t.Fatal(err)
		}
	}
	rows, err = f.store.ClaimJobs(testContext, "outbox", 1, time.Minute)
	if err != nil || len(rows) != 0 {
		t.Fatal("retry budget ignored")
	}
}
func TestDatabaseDisconnectRollsBackBeforeRetry(t *testing.T) {
	f := newFixture(t)
	id := f.account(money.RUB, "100")
	key := request()
	r := f.revision(uuid.NewString(), id, "-10", money.RUB, 1)
	written := make(chan struct{})
	finish := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := f.executor.Execute(testContext, f.p, key, func(ctx context.Context) (result command.Result, err error) {
			if err = f.writer.Append(ctx, f.p, r, 0); err != nil {
				return
			}
			close(written)
			<-finish
			return command.Result{ResourceType: "transaction", ResourceID: r.OperationID, Revision: 1}, nil
		})
		done <- err
	}()
	select {
	case <-written:
	case err := <-done:
		t.Fatalf("write failed before barrier: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("write barrier timed out")
	}
	var pid int
	if err := f.admin.QueryRow(testContext, "SELECT pid FROM pg_stat_activity WHERE datname=current_database() AND usename='want_keep_app' AND state='idle in transaction'").Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if _, err := f.admin.Exec(testContext, "SELECT pg_terminate_backend($1)", pid); err != nil {
		t.Fatal(err)
	}
	close(finish)
	if err := <-done; err == nil {
		t.Fatal("lost connection committed")
	}
	if f.count("operations") != 0 || f.count("outbox") != 0 {
		t.Fatal("terminated transaction partially saved")
	}
	c, err := f.write(r, key)
	if err != nil || c.Status() != command.Succeeded || f.count("postings") != 1 || f.available(id) != "90" {
		t.Fatalf("retry after crash: %v", err)
	}
}
