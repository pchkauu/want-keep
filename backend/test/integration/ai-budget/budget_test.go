//go:build integration

package aibudget_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	aiapp "github.com/pchkauu/want-keep/backend/internal/ai/application"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	jobapp "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

var testContract = ai.RuntimeContract{
	Model: ai.Terra, Qualification: ai.TerraXHigh,
	PromptFingerprint: strings.Repeat("a", 64), SchemaFingerprint: strings.Repeat("b", 64), ConfigFingerprint: strings.Repeat("c", 64),
}

func TestFamilyConcurrencyIsAtomic(t *testing.T) {
	f := newFixture(t)
	claimed := f.newReviewJobs(3)
	start := make(chan struct{})
	errs := make(chan error, len(claimed))
	var wait sync.WaitGroup
	for _, job := range claimed {
		job := job
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			request := f.request(job)
			errs <- f.store.StartAIAttempt(testContext, f.p, job, request, strings.Repeat("d", 64), f.now.Time())
		}()
	}
	close(start)
	wait.Wait()
	close(errs)
	succeeded, refused := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, aiapp.ErrConcurrency):
			refused++
		default:
			t.Fatal(err)
		}
	}
	if succeeded != 2 || refused != 1 {
		t.Fatalf("concurrency result: succeeded=%d refused=%d", succeeded, refused)
	}
	var active int
	if err := f.admin.QueryRow(testContext, `WITH latest AS (SELECT DISTINCT ON(attempt_id) state FROM want_keep.ai_attempt_states WHERE household_id=$1 ORDER BY attempt_id,revision DESC) SELECT count(*) FROM latest WHERE state IN ('counting','reserved')`, f.family.ID).Scan(&active); err != nil || active != 2 {
		t.Fatalf("persisted active calls: %d, %v", active, err)
	}
	other := f.anotherHousehold()
	otherJob := other.newReviewJobs(1)[0]
	otherRequest := other.request(otherJob)
	if err := other.store.StartAIAttempt(testContext, other.p, otherJob, otherRequest, strings.Repeat("e", 64), other.now.Time()); err != nil {
		t.Fatalf("other household shared concurrency slots: %v", err)
	}
}

func TestTwoActiveProviderCallsAreAllowed(t *testing.T) {
	f := newFixture(t)
	claimed := f.newReviewJobs(3)
	reservation, err := ai.TerraPricing().Reservation(1, 2048)
	if err != nil {
		t.Fatal(err)
	}
	for _, job := range claimed[:2] {
		request := f.request(job)
		if err = f.store.StartAIAttempt(testContext, f.p, job, request, strings.Repeat("d", 64), f.now.Time()); err != nil {
			t.Fatalf("start allowed call: %v", err)
		}
		if err = f.store.ReserveAIAttempt(testContext, f.p, job, request.ID, 1, reservation, f.now.Time()); err != nil {
			t.Fatalf("reserve allowed call: %v", err)
		}
		if err = f.store.BeginAIGeneration(testContext, f.p, job, request.ID, f.now.Time()); err != nil {
			t.Fatalf("begin allowed call: %v", err)
		}
	}
	third := f.request(claimed[2])
	if err = f.store.StartAIAttempt(testContext, f.p, claimed[2], third, strings.Repeat("e", 64), f.now.Time()); !errors.Is(err, aiapp.ErrConcurrency) {
		t.Fatalf("third provider call was not bounded: %v", err)
	}
}

func TestExpiredExternalCallBlocksUntilReconciliation(t *testing.T) {
	f := newFixture(t)
	claimed := f.newReviewJobs(2)
	request := f.request(claimed[0])
	reservation, err := ai.TerraPricing().Reservation(1, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.store.StartAIAttempt(testContext, f.p, claimed[0], request, strings.Repeat("d", 64), f.now.Time()); err != nil {
		t.Fatal(err)
	}
	if err = f.store.ReserveAIAttempt(testContext, f.p, claimed[0], request.ID, 1, reservation, f.now.Time()); err != nil {
		t.Fatal(err)
	}
	if err = f.store.BeginAIGeneration(testContext, f.p, claimed[0], request.ID, f.now.Time()); err != nil {
		t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, `UPDATE want_keep.jobs SET lease_until=clock_timestamp()-INTERVAL '1 second' WHERE household_id=$1 AND id=$2`, f.family.ID, claimed[0].ID); err != nil {
		t.Fatal(err)
	}
	next := f.request(claimed[1])
	if err = f.store.StartAIAttempt(testContext, f.p, claimed[1], next, strings.Repeat("e", 64), f.now.Time()); !errors.Is(err, aiapp.ErrBudgetBlocked) {
		t.Fatalf("expired external call did not block: %v", err)
	}
}

func TestExactMonthlyLimitAndGlobalUnknownBlock(t *testing.T) {
	f := newFixture(t)
	claimed := f.newReviewJobs(4)
	firstRequest := f.request(claimed[0])
	if err := f.store.StartAIAttempt(testContext, f.p, claimed[0], firstRequest, strings.Repeat("d", 64), f.now.Time()); err != nil {
		t.Fatal(err)
	}
	reservation, _ := ai.TerraPricing().Reservation(1, 2048)
	if err := f.store.ReserveAIAttempt(testContext, f.p, claimed[0], firstRequest.ID, 1, reservation, f.now.Time()); err != nil {
		t.Fatal(err)
	}
	if err := f.store.BeginAIGeneration(testContext, f.p, claimed[0], firstRequest.ID, f.now.Time()); err != nil {
		t.Fatal(err)
	}
	if err := f.store.MarkAIUnknown(testContext, f.p, claimed[0], firstRequest.ID, "provider_timeout", f.now.Time()); err != nil {
		t.Fatal(err)
	}
	if err := f.store.SetJobOutcome(testContext, f.p, claimed[0], jobs.Unresolved, jobs.ExternalUnknown, 0); err != nil {
		t.Fatal(err)
	}

	rolled := f.request(claimed[1])
	if err := f.store.StartAIAttempt(testContext, f.p, claimed[1], rolled, strings.Repeat("e", 64), f.now.Time().Add(2*time.Minute)); !errors.Is(err, aiapp.ErrBudgetBlocked) {
		t.Fatalf("unknown outcome did not block next UTC month: %v", err)
	}
	service := aiapp.NewReconciliationService(f.maintenanceStore())
	if err := service.Reconcile(testContext, firstRequest.ID, aiapp.Charged, ai.MustCost("49.9"), "provider-dashboard:synthetic-1"); err != nil {
		t.Fatal(err)
	}

	second := f.request(claimed[1])
	if err := f.store.StartAIAttempt(testContext, f.p, claimed[1], second, strings.Repeat("f", 64), f.now.Time()); err != nil {
		t.Fatal(err)
	}
	under, _ := ai.TerraPricing().Reservation(646, 8192)
	if under.String() != "0.099999" {
		t.Fatalf("unexpected exact reservation: %s", under.String())
	}
	if err := f.store.ReserveAIAttempt(testContext, f.p, claimed[1], second.ID, 646, under, f.now.Time()); err != nil {
		t.Fatal(err)
	}
	third := f.request(claimed[2])
	if err := f.store.StartAIAttempt(testContext, f.p, claimed[2], third, strings.Repeat("1", 64), f.now.Time()); err != nil {
		t.Fatal(err)
	}
	over, _ := ai.TerraPricing().Reservation(647, 8192)
	if err := f.store.ReserveAIAttempt(testContext, f.p, claimed[2], third.ID, 647, over, f.now.Time()); !errors.Is(err, aiapp.ErrBudgetExhausted) {
		t.Fatalf("reservation over $50 accepted: %v", err)
	}
	var state, code string
	if err := f.admin.QueryRow(testContext, `SELECT state,code FROM want_keep.ai_attempt_states WHERE household_id=$1 AND attempt_id=$2 ORDER BY revision DESC LIMIT 1`, f.family.ID, third.ID).Scan(&state, &code); err != nil || state != "refused" || code != "budget_exhausted" {
		t.Fatalf("budget refusal not durable: %s/%s %v", state, code, err)
	}
	if err := f.store.SetJobOutcome(testContext, f.p, claimed[2], jobs.Waiting, jobs.BudgetWait, 0); err != nil {
		t.Fatal(err)
	}
	queue := aiapp.NewBudgetQueue(f.store, func() time.Time { return f.now.Time() }, time.Minute)
	if resumed, err := queue.Step(testContext); err != nil || resumed != 0 {
		t.Fatalf("exhausted job resumed in the same UTC month: %d, %v", resumed, err)
	}
	queue = aiapp.NewBudgetQueue(f.store, func() time.Time { return f.now.Time().Add(2 * time.Minute) }, time.Minute)
	if resumed, err := queue.Step(testContext); err != nil || resumed != 1 {
		t.Fatalf("eligible job did not resume after UTC rollover: %d, %v", resumed, err)
	}
	if err := f.admin.QueryRow(testContext, `SELECT state FROM want_keep.jobs WHERE household_id=$1 AND id=$2`, f.family.ID, claimed[2].ID).Scan(&state); err != nil || state != "ready" {
		t.Fatalf("resumed job state: %s, %v", state, err)
	}
}

func TestWorkerPersistsCompletionAndNeverCallsProviderTwice(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	gateway := &fakeGateway{result: completedResult()}
	handler := aiapp.NewHandler(f.store, gateway, func() time.Time { return f.now.Time() }, uuid.NewString)
	worker := jobapp.Worker{Repository: f.store, Handler: handler, Config: jobapp.DefaultWorkerConfig(jobs.AI)}
	if err := worker.Step(testContext); err != nil {
		t.Fatal(err)
	}
	if err := worker.Step(testContext); err != nil {
		t.Fatal(err)
	}
	if gateway.countCalls.Load() != 1 || gateway.generateCalls.Load() != 1 {
		t.Fatalf("provider called more than once: count=%d generation=%d", gateway.countCalls.Load(), gateway.generateCalls.Load())
	}
	var attempts, completed, pendingValidation, receipts int
	if err := f.admin.QueryRow(testContext, `SELECT count(*),count(*) FILTER(WHERE state='completed'),count(*) FILTER(WHERE validation_state='pending_validation') FROM want_keep.ai_attempt_states`).Scan(&attempts, &completed, &pendingValidation); err != nil {
		t.Fatal(err)
	}
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.job_receipts r JOIN want_keep.jobs j ON (j.household_id,j.id)=(r.household_id,r.job_id) WHERE j.kind='ai'`).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if attempts != 4 || completed != 1 || pendingValidation != 1 || receipts != 1 {
		t.Fatalf("atomic completion mismatch: states=%d completed=%d validation=%d receipts=%d", attempts, completed, pendingValidation, receipts)
	}
}

func TestUnknownProviderOutcomeIsNotRetried(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	gateway := &fakeGateway{generateErr: aiapp.GatewayFailure{Code: "provider_timeout", OutcomeUnknown: true}}
	handler := aiapp.NewHandler(f.store, gateway, func() time.Time { return f.now.Time() }, uuid.NewString)
	worker := jobapp.Worker{Repository: f.store, Handler: handler, Config: jobapp.DefaultWorkerConfig(jobs.AI)}
	if err := worker.Step(testContext); err != nil {
		t.Fatal(err)
	}
	if err := worker.Step(testContext); err != nil {
		t.Fatal(err)
	}
	if gateway.generateCalls.Load() != 1 {
		t.Fatalf("unknown provider outcome retried: %d", gateway.generateCalls.Load())
	}
	var jobState, attemptState, reconciliation string
	if err := f.admin.QueryRow(testContext, `SELECT state FROM want_keep.jobs WHERE kind='ai'`).Scan(&jobState); err != nil {
		t.Fatal(err)
	}
	if err := f.admin.QueryRow(testContext, `SELECT state,reconciliation_state FROM want_keep.ai_attempt_states ORDER BY revision DESC LIMIT 1`).Scan(&attemptState, &reconciliation); err != nil {
		t.Fatal(err)
	}
	if jobState != "unresolved" || attemptState != "unknown" || reconciliation != "pending" {
		t.Fatalf("unknown outcome mismatch: %s/%s/%s", jobState, attemptState, reconciliation)
	}
}

func TestKnownRetryableFailureAllowsOnlyOneNewAttempt(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	gateway := &fakeGateway{generateErr: aiapp.GatewayFailure{Code: "provider_5xx", Retryable: true}}
	handler := aiapp.NewHandler(f.store, gateway, func() time.Time { return f.now.Time() }, uuid.NewString)
	worker := jobapp.Worker{Repository: f.store, Handler: handler, Config: jobapp.DefaultWorkerConfig(jobs.AI)}
	for step := range 3 {
		if err := worker.Step(testContext); err != nil {
			t.Fatal(err)
		}
		if step == 0 {
			if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET available_at=clock_timestamp() WHERE kind='ai' AND state='ready'`); err != nil {
				t.Fatal(err)
			}
		}
	}
	if gateway.generateCalls.Load() != 2 {
		t.Fatalf("retry count = %d, want 2 total provider calls", gateway.generateCalls.Load())
	}
	var state string
	if err := f.admin.QueryRow(testContext, `SELECT state FROM want_keep.jobs WHERE kind='ai'`).Scan(&state); err != nil || state != "failed" {
		t.Fatalf("job did not stop after one retry: %s %v", state, err)
	}
}

func (f *fixture) request(job jobs.Job) ai.Request {
	f.t.Helper()
	input, err := f.store.ReviewInput(testContext, f.p, job)
	if err != nil {
		f.t.Fatal(err)
	}
	return ai.Request{
		ID: uuid.NewString(), JobID: job.ID, ResourceID: job.ResourceID,
		HouseholdID: f.p.HouseholdID(), ActorID: f.p.UserID(), ResourceRevision: job.ResourceRevision,
		Purpose: ai.TransactionReview, Model: testContract.Model, Qualification: testContract.Qualification,
		PromptFingerprint: testContract.PromptFingerprint, SchemaFingerprint: testContract.SchemaFingerprint,
		ConfigFingerprint: testContract.ConfigFingerprint, Input: input, MaximumOutputTokens: 2048,
	}
}

type fakeGateway struct {
	countCalls, generateCalls atomic.Int32
	result                    ai.Result
	generateErr               error
}

func (g *fakeGateway) Contract() ai.RuntimeContract { return testContract }
func (g *fakeGateway) Count(context.Context, ai.Request) (int64, error) {
	g.countCalls.Add(1)
	return 100, nil
}
func (g *fakeGateway) Generate(context.Context, ai.Request) (ai.Result, error) {
	g.generateCalls.Add(1)
	return g.result, g.generateErr
}

func completedResult() ai.Result {
	writes := int64(10)
	return ai.Result{
		ProviderID: "resp_synthetic", State: ai.Completed,
		Output: json.RawMessage(`{"results":[{"id":"1","action":"skip","kind":null,"amount":null,"fee":null,"asset":null,"target":null,"month":null,"shares":[],"items":[],"evidence":[],"explanation":"ok"}]}`),
		Usage:  ai.Usage{InputTokens: 100, CachedTokens: 20, CacheWriteTokens: &writes, OutputTokens: 50, ReasoningTokens: 12},
	}
}
