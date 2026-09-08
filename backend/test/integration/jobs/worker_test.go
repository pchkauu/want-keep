//go:build integration

package storage_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	app "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type handlerFunc func(context.Context, app.Execution) (app.Result, error)

func (h handlerFunc) Prepare(ctx context.Context, x app.Execution) (app.Result, error) {
	return h(ctx, x)
}
func (f *fixture) event() {
	f.t.Helper()
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		return f.store.EmitEvent(ctx, "synthetic", uuid.NewString(), 1, "synthetic.work")
	}); err != nil {
		f.t.Fatal(err)
	}
}
func (f *fixture) worker(h app.Handler) app.Worker {
	return app.Worker{Repository: f.store, Handler: h, Config: app.DefaultWorkerConfig(jobs.Outbox)}
}
func TestWorkerConcurrentOutboxAndReviewQueue(t *testing.T) {
	f := newFixture(t)
	id := f.account(money.RUB, "100")
	r := f.revision(uuid.NewString(), id, "-10", money.RUB, 1)
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	second := f.restart()
	workers := []app.Worker{f.worker(app.OutboxHandler{Repository: f.store}), {Repository: second, Handler: app.OutboxHandler{Repository: second}, Config: app.DefaultWorkerConfig(jobs.Outbox)}}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, worker := range workers {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- worker.Step(testContext) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.jobs WHERE kind='ai'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("AI enqueue %d: %v", count, err)
	}
	if f.count("job_receipts") != 1 || f.available(id) != "90" {
		t.Fatal("non-atomic outbox effect")
	}
	worker := app.Worker{Repository: f.store, Config: app.DefaultWorkerConfig(jobs.AI)}
	if err := worker.Step(testContext); err != nil {
		t.Fatal(err)
	}
	var state, reason string
	var attempt int
	if err := f.admin.QueryRow(testContext, `SELECT state,reason,attempt FROM want_keep.jobs WHERE kind='ai'`).Scan(&state, &reason, &attempt); err != nil {
		t.Fatal(err)
	}
	if state != "waiting" || reason != "handler_unavailable" || attempt != 0 {
		t.Fatal("missing handler consumed work")
	}
}
func TestWorkerBudgetWaitAndUnknownRecovery(t *testing.T) {
	f := newFixture(t)
	f.event()
	var issued jobs.Job
	worker := f.worker(handlerFunc(func(ctx context.Context, x app.Execution) (app.Result, error) {
		issued = x.Job
		return app.Result{State: jobs.Waiting, Reason: jobs.BudgetWait}, nil
	}))
	if err := worker.Step(testContext); err != nil {
		t.Fatal(err)
	}
	current, err := f.store.Job(testContext, f.p, issued.ID)
	if err != nil || current.Attempt != 0 || current.State != jobs.Waiting {
		t.Fatalf("wait: %+v %v", current, err)
	}
	if err = f.store.ResumeWaiting(testContext, jobs.Outbox, jobs.BudgetWait); err != nil {
		t.Fatal(err)
	}
	worker.Handler = handlerFunc(func(ctx context.Context, x app.Execution) (app.Result, error) {
		issued = x.Job
		if err := x.BeginExternal(ctx); err != nil {
			return app.Result{}, err
		}
		return app.Result{}, errors.New("synthetic lost response")
	})
	if err = worker.Step(testContext); err == nil {
		t.Fatal("provider error hidden")
	}
	current, err = f.store.Job(testContext, f.p, issued.ID)
	if err != nil || current.State != jobs.Unresolved {
		t.Fatal("unknown outcome not persisted", err)
	}
	if rows, err := f.restart().ClaimJobs(testContext, "outbox", 1, time.Minute); err != nil || len(rows) != 0 {
		t.Fatal("ambiguous request retried")
	}
	calls := 0
	r := app.Reconciliation{Job: issued, EvidenceRef: "synthetic:response", Outcome: "confirmed"}
	effect := func(context.Context, household.Principal) error { calls++; return nil }
	for range 2 {
		if err = f.store.ReconcileJob(testContext, f.p, r, effect); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 || f.count("job_reconciliations") != 1 {
		t.Fatal("duplicate reconciled effect")
	}
	r.Outcome = "absent"
	if err = f.store.ReconcileJob(testContext, f.p, r, nil); !errors.Is(err, jobs.ErrStaleAttempt) {
		t.Fatal("conflicting reconciliation accepted", err)
	}
}

func TestMissingHandlerRetainsAgedUnstartedReview(t *testing.T) {
	f := newFixture(t)
	account := f.account(money.RUB, "100")
	r := f.revision(uuid.NewString(), account, "-10", money.RUB, 1)
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	if err := f.worker(app.OutboxHandler{Repository: f.store}).Step(testContext); err != nil {
		t.Fatal(err)
	}
	if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET deadline=clock_timestamp()-INTERVAL '1 day' WHERE kind='ai'`); err != nil {
		t.Fatal(err)
	}
	worker := app.Worker{Repository: f.restart(), Config: app.DefaultWorkerConfig(jobs.AI)}
	if err := worker.Step(testContext); err != nil {
		t.Fatal(err)
	}
	var state, reason string
	var attempt int
	if err := f.admin.QueryRow(testContext, `SELECT state,reason,attempt FROM want_keep.jobs WHERE kind='ai'`).Scan(&state, &reason, &attempt); err != nil {
		t.Fatal(err)
	}
	if state != "waiting" || reason != "handler_unavailable" || attempt != 0 {
		t.Fatalf("unstarted review lost: %s/%s attempt %d", state, reason, attempt)
	}
	calls := 0
	worker.Handler = handlerFunc(func(_ context.Context, x app.Execution) (app.Result, error) {
		calls++
		if x.Job.ResourceID != r.OperationID || x.Job.ResourceRevision != 1 {
			t.Fatal("review identity changed")
		}
		return app.Result{State: jobs.Succeeded}, nil
	})
	for range 2 {
		if err := worker.Step(testContext); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 || f.count("job_receipts") != 2 || f.available(account) != "90" {
		t.Fatal("resumed review duplicated or changed accounting")
	}
}
func TestExternalCrashAndExplicitAbsent(t *testing.T) {
	f := newFixture(t)
	f.event()
	rows, err := f.store.ClaimJobs(testContext, "outbox", 1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	issued := rows[0]
	if err = f.store.BeginExternal(testContext, f.p, issued); err != nil {
		t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, `UPDATE want_keep.jobs SET lease_until=clock_timestamp()-INTERVAL '1 second' WHERE id=$1`, issued.ID); err != nil {
		t.Fatal(err)
	}
	rows, err = f.restart().ClaimJobs(testContext, "outbox", 1, time.Minute)
	if err != nil || len(rows) != 0 {
		t.Fatal("uncertain crash retried")
	}
	if err = f.store.ReconcileJob(testContext, f.p, app.Reconciliation{Job: issued, EvidenceRef: "synthetic:absent", Outcome: "absent"}, nil); err != nil {
		t.Fatal(err)
	}
	rows, err = f.store.ClaimJobs(testContext, "outbox", 1, time.Minute)
	if err != nil || len(rows) != 1 || rows[0].LeaseToken == issued.LeaseToken {
		t.Fatal("proven absent not resumed")
	}
}
func TestWorkerLateEffectAndRevokedMembership(t *testing.T) {
	f := newFixture(t)
	f.event()
	var calls atomic.Int32
	worker := f.worker(handlerFunc(func(ctx context.Context, x app.Execution) (app.Result, error) {
		if _, err := f.admin.Exec(ctx, `UPDATE want_keep.jobs SET lease_until=clock_timestamp()-INTERVAL '1 second' WHERE id=$1`, x.Job.ID); err != nil {
			return app.Result{}, err
		}
		return app.Result{State: jobs.Succeeded, Apply: func(context.Context, household.Principal) error { calls.Add(1); return nil }}, nil
	}))
	if err := worker.Step(testContext); !errors.Is(err, jobs.ErrStaleAttempt) {
		t.Fatal("late effect accepted", err)
	}
	if calls.Load() != 0 || f.count("job_receipts") != 0 {
		t.Fatal("late effect applied")
	}
	if _, err := f.admin.Exec(testContext, `UPDATE want_keep.memberships SET active=false WHERE household_id=$1 AND user_id=$2`, f.p.HouseholdID(), f.p.UserID()); err != nil {
		t.Fatal(err)
	}
	rows, err := f.store.ClaimJobs(testContext, "outbox", 1, time.Minute)
	if err != nil || len(rows) != 0 {
		t.Fatal("revoked membership claimed", err)
	}
}
func TestWorkerShutdownAndHeartbeatFailure(t *testing.T) {
	f := newFixture(t)
	f.event()
	entered := make(chan app.Execution, 1)
	worker := f.worker(handlerFunc(func(ctx context.Context, x app.Execution) (app.Result, error) {
		entered <- x
		<-ctx.Done()
		return app.Result{}, ctx.Err()
	}))
	worker.Config.Lease = time.Second
	worker.Config.Heartbeat = 50 * time.Millisecond
	ctx, cancel := context.WithCancel(testContext)
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("worker not entered")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown stuck")
	}
	if f.count("job_receipts") != 0 {
		t.Fatal("canceled handler committed")
	}
}

func TestHeartbeatLossCancelsHandlerBeforeEffect(t *testing.T) {
	f := newFixture(t)
	f.event()
	var applied atomic.Bool
	worker := f.worker(handlerFunc(func(ctx context.Context, x app.Execution) (app.Result, error) {
		if _, err := f.admin.Exec(ctx, `UPDATE want_keep.jobs SET lease_until=clock_timestamp()-INTERVAL '1 second' WHERE id=$1`, x.Job.ID); err != nil {
			return app.Result{}, err
		}
		select {
		case <-ctx.Done():
		case <-time.After(3 * time.Second):
			return app.Result{}, errors.New("heartbeat did not cancel")
		}
		return app.Result{State: jobs.Succeeded, Apply: func(context.Context, household.Principal) error { applied.Store(true); return nil }}, nil
	}))
	worker.Config.Heartbeat = 20 * time.Millisecond
	if err := worker.Step(testContext); !errors.Is(err, jobs.ErrStaleAttempt) {
		t.Fatal("heartbeat loss ignored", err)
	}
	if applied.Load() {
		t.Fatal("lost lease effect applied")
	}
}
func TestWaitingDeadlineAndCrossHouseholdFence(t *testing.T) {
	f := newFixture(t)
	f.event()
	if err := f.store.PauseReady(testContext, jobs.Outbox, jobs.BudgetWait); err != nil {
		t.Fatal(err)
	}
	if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET deadline=clock_timestamp()-INTERVAL '1 day'`); err != nil {
		t.Fatal(err)
	}
	if err := f.store.RecoverJobs(testContext, jobs.Outbox); err != nil {
		t.Fatal(err)
	}
	if err := f.store.ResumeWaiting(testContext, jobs.Outbox, jobs.BudgetWait); err != nil {
		t.Fatal(err)
	}
	rows, err := f.store.ClaimJobs(testContext, "outbox", 1, time.Minute)
	if err != nil || len(rows) != 1 {
		t.Fatal("waiting deadline consumed work", err)
	}
	impersonated := rows[0]
	impersonated.ActorID = f.q.UserID()
	if err = f.store.Heartbeat(testContext, f.q, impersonated, time.Minute); !errors.Is(err, jobs.ErrStaleAttempt) {
		t.Fatal("forged actor accepted", err)
	}
	foreign := f.p
	forged := rows[0]
	forged.HouseholdID = household.HouseholdID(uuid.NewString())
	if err = f.store.Heartbeat(testContext, foreign, forged, time.Minute); !errors.Is(err, household.ErrForbidden) {
		t.Fatal("forged family accepted", err)
	}
	if err = f.store.BeginExternal(testContext, foreign, forged); !errors.Is(err, household.ErrForbidden) {
		t.Fatal("foreign external marker accepted", err)
	}
}
