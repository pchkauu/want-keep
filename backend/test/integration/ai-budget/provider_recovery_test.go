//go:build integration

package aibudget_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	aiapp "github.com/pchkauu/want-keep/backend/internal/ai/application"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	jobapp "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func TestMaintenanceRoleCannotReadOrMutateGatewayTables(t *testing.T) {
	f := newFixture(t)
	pool := f.maintenancePool()
	for name, statement := range map[string]string{
		"read attempts": `SELECT count(*) FROM want_keep.ai_attempts`,
		"mutate jobs":   `UPDATE want_keep.jobs SET reason='' WHERE false`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := pool.Exec(testContext, statement); err == nil {
				t.Fatalf("maintenance role executed %q", statement)
			}
		})
	}
}

func TestRecoveryReleasesOnlyAttemptsThatNeverReachedProvider(t *testing.T) {
	for _, stage := range []string{"counting", "reserved"} {
		t.Run(stage, func(t *testing.T) {
			f := newFixture(t)
			job := f.newReviewJobs(1)[0]
			request := f.request(job)
			if err := f.store.StartAIAttempt(testContext, f.p, job, request, strings.Repeat("d", 64), f.now.Time()); err != nil {
				t.Fatal(err)
			}
			if stage == "reserved" {
				reservation, _ := ai.TerraPricing().Reservation(100, 2048)
				if err := f.store.ReserveAIAttempt(testContext, f.p, job, request.ID, 100, reservation, f.now.Time()); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET cancel_requested=true WHERE household_id=$1 AND id=$2`, f.family.ID, job.ID); err != nil {
				t.Fatal(err)
			}
			if err := f.store.RecoverJobs(testContext, jobs.AI); err != nil {
				t.Fatal(err)
			}
			var jobState, attemptState, code string
			if err := f.admin.QueryRow(testContext, `SELECT j.state,s.state,s.code FROM want_keep.jobs j JOIN LATERAL(SELECT state,code FROM want_keep.ai_attempt_states s WHERE (s.household_id,s.attempt_id)=($1,$2) ORDER BY revision DESC LIMIT 1) s ON true WHERE (j.household_id,j.id)=($1,$3)`, f.family.ID, request.ID, job.ID).Scan(&jobState, &attemptState, &code); err != nil {
				t.Fatal(err)
			}
			if jobState != "canceled" || attemptState != "known_rejection" || code != "job_terminated_before_send" {
				t.Fatalf("orphaned attempt not released: %s/%s/%s", jobState, attemptState, code)
			}
		})
	}
}

func TestRepeatedPreGenerationRecoveryDoesNotExhaustReview(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	for cycle := range 5 {
		claimed, err := f.store.ClaimJobs(testContext, string(jobs.AI), 1, time.Minute)
		if err != nil || len(claimed) != 1 {
			t.Fatalf("claim cycle %d: %#v, %v", cycle, claimed, err)
		}
		request := f.request(claimed[0])
		if err = f.store.StartAIAttempt(testContext, f.p, claimed[0], request, strings.Repeat("d", 64), f.now.Time()); err != nil {
			t.Fatal(err)
		}
		if _, err = f.admin.Exec(testContext, `UPDATE want_keep.jobs SET lease_until=clock_timestamp()-INTERVAL '1 second' WHERE household_id=$1 AND id=$2`, f.family.ID, claimed[0].ID); err != nil {
			t.Fatal(err)
		}
		if err = f.store.RecoverJobs(testContext, jobs.AI); err != nil {
			t.Fatal(err)
		}
		var state string
		var attempt int
		if err = f.admin.QueryRow(testContext, `SELECT state,attempt FROM want_keep.jobs WHERE household_id=$1 AND id=$2`, f.family.ID, claimed[0].ID).Scan(&state, &attempt); err != nil || state != "ready" || attempt != 0 {
			t.Fatalf("recovery cycle %d: %s/%d, %v", cycle, state, attempt, err)
		}
	}
	gateway := &fakeGateway{result: completedResult()}
	handler := aiapp.NewHandler(f.store, gateway, func() time.Time { return f.now.Time() }, uuid.NewString)
	worker := jobapp.Worker{Repository: f.store, Handler: handler, Config: jobapp.DefaultWorkerConfig(jobs.AI)}
	if err := worker.Step(testContext); err != nil {
		t.Fatal(err)
	}
	if gateway.generateCalls.Load() != 1 {
		t.Fatalf("review did not run after safe recoveries: %d", gateway.generateCalls.Load())
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

func TestUnconfirmedGenerationFailureKeepsReservation(t *testing.T) {
	for _, failure := range []aiapp.GatewayFailure{
		{Code: "provider_400"}, {Code: "provider_401"}, {Code: "provider_403"},
		{Code: "provider_5xx", Retryable: true},
	} {
		t.Run(failure.Code, func(t *testing.T) {
			f := newFixture(t)
			f.enqueueReviewJobs(1)
			gateway := &fakeGateway{generateErr: failure}
			handler := aiapp.NewHandler(f.store, gateway, func() time.Time { return f.now.Time() }, uuid.NewString)
			worker := jobapp.Worker{Repository: f.store, Handler: handler, Config: jobapp.DefaultWorkerConfig(jobs.AI)}
			if err := worker.Step(testContext); err != nil {
				t.Fatal(err)
			}
			var jobState, attemptState, code, reconciliation string
			if err := f.admin.QueryRow(testContext, `SELECT j.state,s.state,s.code,s.reconciliation_state FROM want_keep.jobs j JOIN want_keep.ai_attempts a ON (a.household_id,a.job_id)=(j.household_id,j.id) JOIN LATERAL(SELECT state,code,reconciliation_state FROM want_keep.ai_attempt_states current WHERE (current.household_id,current.attempt_id)=(a.household_id,a.id) ORDER BY revision DESC LIMIT 1) s ON true WHERE j.kind='ai'`).Scan(&jobState, &attemptState, &code, &reconciliation); err != nil {
				t.Fatal(err)
			}
			if jobState != "unresolved" || attemptState != "unknown" || code != "provider_charge_unknown" || reconciliation != "pending" || gateway.generateCalls.Load() != 1 {
				t.Fatalf("unconfirmed provider failure released reservation: %s/%s/%s/%s calls=%d", jobState, attemptState, code, reconciliation, gateway.generateCalls.Load())
			}
		})
	}
}

func TestUnknownPersistsContradictoryProviderUsageEvidence(t *testing.T) {
	f := newFixture(t)
	job := f.newReviewJobs(1)[0]
	request := f.request(job)
	reservation, _ := ai.TerraPricing().Reservation(100, 2048)
	if err := f.store.StartAIAttempt(testContext, f.p, job, request, strings.Repeat("d", 64), f.now.Time()); err != nil {
		t.Fatal(err)
	}
	if err := f.store.ReserveAIAttempt(testContext, f.p, job, request.ID, 100, reservation, f.now.Time()); err != nil {
		t.Fatal(err)
	}
	if err := f.store.BeginAIGeneration(testContext, f.p, job, request.ID, f.now.Time()); err != nil {
		t.Fatal(err)
	}
	input, cached, output, reasoning, total := int64(100), int64(20), int64(50), int64(12), int64(149)
	observation := aiapp.ProviderObservation{ID: "resp_synthetic", Model: "gpt-5.6-terra", Usage: &aiapp.ObservedUsage{
		InputTokens: &input, CachedTokens: &cached, OutputTokens: &output, ReasoningTokens: &reasoning, TotalTokens: &total,
	}}
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		return f.store.MarkAIUnknown(ctx, f.p, job, request.ID, "usage_invalid", observation, f.now.Time())
	}); err != nil {
		t.Fatal(err)
	}
	var raw string
	var actual *string
	if err := f.admin.QueryRow(testContext, `SELECT observed_usage::text,actual_usd::text FROM want_keep.ai_attempt_states WHERE household_id=$1 AND attempt_id=$2 ORDER BY revision DESC LIMIT 1`, f.family.ID, request.ID).Scan(&raw, &actual); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{`"input_tokens": 100`, `"output_tokens": 50`, `"total_tokens": 149`} {
		if !strings.Contains(raw, fragment) {
			t.Fatalf("provider evidence %q missing from %s", fragment, raw)
		}
	}
	if actual != nil {
		t.Fatalf("contradictory usage affected accounting: %s", *actual)
	}
}

func TestConservativeCacheWriteCostCanBeReconciledWithoutBlocking(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(2)
	result := completedResult()
	result.Usage.CacheWriteTokens = nil
	gateway := &fakeGateway{result: result}
	handler := aiapp.NewHandler(f.store, gateway, func() time.Time { return f.now.Time() }, uuid.NewString)
	worker := jobapp.Worker{Repository: f.store, Handler: handler, Config: jobapp.DefaultWorkerConfig(jobs.AI)}
	if err := worker.Step(testContext); err != nil {
		t.Fatal(err)
	}
	var attemptID, state, reconciliation, actual string
	var conservative bool
	if err := f.admin.QueryRow(testContext, `SELECT a.id,s.state,s.reconciliation_state,s.actual_usd::text,s.conservative_cost FROM want_keep.ai_attempts a JOIN LATERAL(SELECT * FROM want_keep.ai_attempt_states current WHERE (current.household_id,current.attempt_id)=(a.household_id,a.id) ORDER BY revision DESC LIMIT 1) s ON true ORDER BY a.created_at LIMIT 1`).Scan(&attemptID, &state, &reconciliation, &actual, &conservative); err != nil {
		t.Fatal(err)
	}
	if state != "completed" || reconciliation != "conservative" || actual != "0.000804" || !conservative {
		t.Fatalf("conservative settlement mismatch: %s/%s/%s/%t", state, reconciliation, actual, conservative)
	}
	jobsReady := f.newReviewJobs(1)
	next := f.request(jobsReady[0])
	if err := f.store.StartAIAttempt(testContext, f.p, jobsReady[0], next, strings.Repeat("e", 64), f.now.Time()); err != nil {
		t.Fatalf("conservative estimate blocked later work: %v", err)
	}
	service := aiapp.NewReconciliationService(f.maintenanceStore())
	if err := service.Reconcile(testContext, attemptID, aiapp.Charged, ai.MustCost("0.000764"), "provider-dashboard:cache-write-exact"); err != nil {
		t.Fatal(err)
	}
	if err := f.admin.QueryRow(testContext, `SELECT reconciliation_state,actual_usd::text,conservative_cost FROM want_keep.ai_attempt_states WHERE attempt_id=$1 ORDER BY revision DESC LIMIT 1`, attemptID).Scan(&reconciliation, &actual, &conservative); err != nil {
		t.Fatal(err)
	}
	if reconciliation != "resolved" || actual != "0.000764" || conservative {
		t.Fatalf("exact reconciliation mismatch: %s/%s/%t", reconciliation, actual, conservative)
	}
}

func TestKnownRetryableFailureAllowsOnlyOneNewAttempt(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	gateway := &fakeGateway{generateErr: aiapp.GatewayFailure{Code: "provider_rate_limited", Retryable: true, ConfirmedNoCharge: true, RetryAfter: 2 * time.Minute}}
	handler := aiapp.NewHandler(f.store, gateway, func() time.Time { return f.now.Time() }, uuid.NewString)
	worker := jobapp.Worker{Repository: f.store, Handler: handler, Config: jobapp.DefaultWorkerConfig(jobs.AI)}
	for step := range 3 {
		if err := worker.Step(testContext); err != nil {
			t.Fatal(err)
		}
		if step == 0 {
			var remaining float64
			if err := f.admin.QueryRow(testContext, `SELECT EXTRACT(EPOCH FROM available_at-clock_timestamp()) FROM want_keep.jobs WHERE kind='ai'`).Scan(&remaining); err != nil || remaining < 119 {
				t.Fatalf("provider retry delay was not preserved: %.3f, %v", remaining, err)
			}
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

func TestPrecountConfigurationFailureWaitsAndRecovers(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	gateway := &fakeGateway{
		result:   completedResult(),
		countErr: aiapp.GatewayFailure{Code: "provider_configuration_invalid", Retryable: true},
	}
	handler := aiapp.NewHandler(f.store, gateway, func() time.Time { return f.now.Time() }, uuid.NewString)
	worker := jobapp.Worker{Repository: f.store, Handler: handler, Config: jobapp.DefaultWorkerConfig(jobs.AI)}
	if err := worker.Step(testContext); err != nil {
		t.Fatal(err)
	}
	var state, reason string
	if err := f.admin.QueryRow(testContext, `SELECT state,reason FROM want_keep.jobs WHERE kind='ai'`).Scan(&state, &reason); err != nil || state != "waiting" || reason != "gateway_unavailable" {
		t.Fatalf("precount outage did not wait: %s/%s %v", state, reason, err)
	}
	gateway.countErr = nil
	gateway.generateErr = aiapp.GatewayFailure{Code: "provider_rate_limited", Retryable: true, ConfirmedNoCharge: true}
	if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET available_at=clock_timestamp() WHERE kind='ai'`); err != nil {
		t.Fatal(err)
	}
	if err := aiapp.NewGatewayQueue(f.store, time.Minute).Step(testContext); err != nil {
		t.Fatal(err)
	}
	for step := range 2 {
		if err := worker.Step(testContext); err != nil {
			t.Fatal(err)
		}
		if step == 0 {
			if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET available_at=clock_timestamp() WHERE kind='ai' AND state='ready'`); err != nil {
				t.Fatal(err)
			}
		}
	}
	if gateway.countCalls.Load() != 3 || gateway.generateCalls.Load() != 2 {
		t.Fatalf("provider attempts after precount recovery: count=%d generation=%d", gateway.countCalls.Load(), gateway.generateCalls.Load())
	}
	if err := f.admin.QueryRow(testContext, `SELECT state FROM want_keep.jobs WHERE kind='ai'`).Scan(&state); err != nil || state != "failed" {
		t.Fatalf("provider retry allowance mismatch: %s %v", state, err)
	}
}

func TestPrecountRateLimitHonorsDelayWithoutConsumingGenerationRetry(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	gateway := &fakeGateway{
		result:   completedResult(),
		countErr: aiapp.GatewayFailure{Code: "provider_rate_limited", Retryable: true, RetryAfter: 2 * time.Minute},
	}
	handler := aiapp.NewHandler(f.store, gateway, func() time.Time { return f.now.Time() }, uuid.NewString)
	worker := jobapp.Worker{Repository: f.store, Handler: handler, Config: jobapp.DefaultWorkerConfig(jobs.AI)}
	queue := aiapp.NewGatewayQueue(f.store, time.Minute)
	for range 4 {
		if err := worker.Step(testContext); err != nil {
			t.Fatal(err)
		}
		var state string
		var attempt int
		var remaining float64
		if err := f.admin.QueryRow(testContext, `SELECT state,attempt,EXTRACT(EPOCH FROM available_at-clock_timestamp()) FROM want_keep.jobs WHERE kind='ai'`).Scan(&state, &attempt, &remaining); err != nil || state != "waiting" || attempt != 0 || remaining < 119 {
			t.Fatalf("pre-count rate limit was not delayed without consuming an attempt: %s/%d %.3f %v", state, attempt, remaining, err)
		}
		if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET available_at=clock_timestamp() WHERE kind='ai'`); err != nil {
			t.Fatal(err)
		}
		if err := queue.Step(testContext); err != nil {
			t.Fatal(err)
		}
	}
	gateway.countErr = nil
	gateway.generateErr = aiapp.GatewayFailure{Code: "provider_rate_limited", Retryable: true, ConfirmedNoCharge: true}
	for step := range 2 {
		if err := worker.Step(testContext); err != nil {
			t.Fatal(err)
		}
		if step == 0 {
			if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET available_at=clock_timestamp() WHERE kind='ai' AND state='ready'`); err != nil {
				t.Fatal(err)
			}
		}
	}
	var state string
	if err := f.admin.QueryRow(testContext, `SELECT state FROM want_keep.jobs WHERE kind='ai'`).Scan(&state); err != nil || state != "failed" {
		t.Fatalf("generation retry allowance changed after pre-count rate limits: %s %v", state, err)
	}
	if gateway.countCalls.Load() != 6 || gateway.generateCalls.Load() != 2 {
		t.Fatalf("provider attempts: count=%d generation=%d", gateway.countCalls.Load(), gateway.generateCalls.Load())
	}
}

func TestGatewayWaitingPreservesDeadlineAndBoundsAttemptHistory(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	gateway := &fakeGateway{countErr: aiapp.GatewayFailure{Code: "provider_unavailable", Retryable: true}}
	handler := aiapp.NewHandler(f.store, gateway, func() time.Time { return f.now.Time() }, uuid.NewString)
	worker := jobapp.Worker{Repository: f.store, Handler: handler, Config: jobapp.DefaultWorkerConfig(jobs.AI)}
	queue := aiapp.NewGatewayQueue(f.store, time.Minute)

	var deadline time.Time
	if err := f.admin.QueryRow(testContext, `SELECT deadline FROM want_keep.jobs WHERE kind='ai'`).Scan(&deadline); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := worker.Step(testContext); err != nil {
			t.Fatal(err)
		}
		var state string
		var attempt int
		var currentDeadline time.Time
		var retryDelay float64
		if err := f.admin.QueryRow(testContext, `SELECT state,attempt,COALESCE(run_deadline,deadline),EXTRACT(EPOCH FROM available_at-clock_timestamp()) FROM want_keep.jobs WHERE kind='ai'`).Scan(&state, &attempt, &currentDeadline, &retryDelay); err != nil || state != "waiting" || attempt != 0 || !currentDeadline.Equal(deadline) || retryDelay < 299 {
			t.Fatalf("gateway wait lost its bound: %s/%d/%s/%.3f %v", state, attempt, currentDeadline, retryDelay, err)
		}
		if err := queue.Step(testContext); err != nil {
			t.Fatal(err)
		}
		if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET available_at=clock_timestamp() WHERE kind='ai'`); err != nil {
			t.Fatal(err)
		}
		if err := queue.Step(testContext); err != nil {
			t.Fatal(err)
		}
	}
	var attempts int
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.ai_attempts`).Scan(&attempts); err != nil || attempts != 2 {
		t.Fatalf("unexpected pre-count history: %d, %v", attempts, err)
	}
	if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET state='waiting',reason='gateway_unavailable',run_deadline=clock_timestamp()-INTERVAL '1 second' WHERE kind='ai'`); err != nil {
		t.Fatal(err)
	}
	if err := queue.Step(testContext); err != nil {
		t.Fatal(err)
	}
	var state, reason string
	if err := f.admin.QueryRow(testContext, `SELECT state,reason FROM want_keep.jobs WHERE kind='ai'`).Scan(&state, &reason); err != nil || state != "failed" || reason != "deadline_exceeded" {
		t.Fatalf("expired gateway wait survived: %s/%s %v", state, reason, err)
	}
}

func TestRecoveredPreSendReservationDoesNotConsumeGenerationLimit(t *testing.T) {
	f := newFixture(t)
	job := f.newReviewJobs(1)[0]
	request := f.request(job)
	reservation, err := ai.TerraPricing().Reservation(100, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.store.StartAIAttempt(testContext, f.p, job, request, strings.Repeat("d", 64), f.now.Time()); err != nil {
		t.Fatal(err)
	}
	if err = f.store.ReserveAIAttempt(testContext, f.p, job, request.ID, 100, reservation, f.now.Time()); err != nil {
		t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, `UPDATE want_keep.jobs SET lease_until=clock_timestamp()-INTERVAL '1 second' WHERE household_id=$1 AND id=$2`, f.family.ID, job.ID); err != nil {
		t.Fatal(err)
	}
	if err = f.store.RecoverJobs(testContext, jobs.AI); err != nil {
		t.Fatal(err)
	}
	gateway := &fakeGateway{generateErr: aiapp.GatewayFailure{Code: "provider_rate_limited", Retryable: true, ConfirmedNoCharge: true}}
	handler := aiapp.NewHandler(f.store, gateway, func() time.Time { return f.now.Time() }, uuid.NewString)
	worker := jobapp.Worker{Repository: f.store, Handler: handler, Config: jobapp.DefaultWorkerConfig(jobs.AI)}
	for step := range 2 {
		if err = worker.Step(testContext); err != nil {
			t.Fatal(err)
		}
		if step == 0 {
			if _, err = f.admin.Exec(testContext, `UPDATE want_keep.jobs SET available_at=clock_timestamp() WHERE kind='ai' AND state='ready'`); err != nil {
				t.Fatal(err)
			}
		}
	}
	if gateway.generateCalls.Load() != 2 {
		t.Fatalf("pre-send reservation consumed generation limit: %d", gateway.generateCalls.Load())
	}
}

func TestGenerationUsageMustFitCountReservationEnvelope(t *testing.T) {
	for _, test := range []struct {
		name          string
		input, output int64
		code          string
	}{
		{name: "underreported input", input: 99, output: 50, code: "input_count_mismatch"},
		{name: "input above margin", input: 133, output: 50, code: "input_count_mismatch"},
		{name: "output above route cap", input: 100, output: 2049, code: "output_count_mismatch"},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			f.enqueueReviewJobs(1)
			result := completedResult()
			result.Usage.InputTokens, result.Usage.OutputTokens = test.input, test.output
			gateway := &fakeGateway{result: result}
			handler := aiapp.NewHandler(f.store, gateway, func() time.Time { return f.now.Time() }, uuid.NewString)
			worker := jobapp.Worker{Repository: f.store, Handler: handler, Config: jobapp.DefaultWorkerConfig(jobs.AI)}
			if err := worker.Step(testContext); err != nil {
				t.Fatal(err)
			}
			var jobState, attemptState, code string
			if err := f.admin.QueryRow(testContext, `SELECT j.state,s.state,s.code FROM want_keep.jobs j JOIN want_keep.ai_attempts a ON (a.household_id,a.job_id)=(j.household_id,j.id) JOIN LATERAL(SELECT state,code FROM want_keep.ai_attempt_states current WHERE (current.household_id,current.attempt_id)=(a.household_id,a.id) ORDER BY revision DESC LIMIT 1) s ON true WHERE j.kind='ai'`).Scan(&jobState, &attemptState, &code); err != nil {
				t.Fatal(err)
			}
			if jobState != "unresolved" || attemptState != "unknown" || code != test.code {
				t.Fatalf("usage mismatch was accepted: %s/%s/%s", jobState, attemptState, code)
			}
		})
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
	countErr                  error
	generateErr               error
}

func (g *fakeGateway) Contract() ai.RuntimeContract { return testContract }
func (g *fakeGateway) Count(context.Context, ai.Request) (int64, error) {
	g.countCalls.Add(1)
	return 100, g.countErr
}
func (g *fakeGateway) Generate(context.Context, ai.Request) (ai.Result, error) {
	g.generateCalls.Add(1)
	return g.result, g.generateErr
}

func completedResult() ai.Result {
	writes := int64(10)
	return ai.Result{
		ProviderID: "resp_synthetic", ProviderModel: "gpt-5.6-terra", State: ai.Completed,
		Output: json.RawMessage(`{"results":[{"id":"1","action":"skip","kind":null,"amount":null,"fee":null,"asset":null,"target":null,"month":null,"shares":[],"items":[],"evidence":[],"explanation":"ok"}]}`),
		Usage:  ai.Usage{InputTokens: 100, CachedTokens: 20, CacheWriteTokens: &writes, OutputTokens: 50, ReasoningTokens: 12},
	}
}

func waitForDatabaseLock(t *testing.T, f *fixture, user, queryFragment string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var waiting bool
		err := f.admin.QueryRow(testContext, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND usename=$1 AND wait_event_type='Lock' AND query LIKE '%' || $2 || '%')`, user, queryFragment).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("%s did not reach the %s lock", user, queryFragment)
}
