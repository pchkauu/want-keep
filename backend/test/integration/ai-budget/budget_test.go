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
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
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

func TestGatewayWaitingJobResumesWhenGatewayBecomesAvailable(t *testing.T) {
	f := newFixture(t)
	job := f.newReviewJobs(1)[0]
	if err := f.store.SetJobOutcome(testContext, f.p, job, jobs.Waiting, jobs.GatewayUnavailable, 0); err != nil {
		t.Fatal(err)
	}
	if err := aiapp.NewGatewayQueue(f.store, time.Minute).Step(testContext); err != nil {
		t.Fatal(err)
	}
	claimed, err := f.store.ClaimJobs(testContext, string(jobs.AI), 1, time.Minute)
	if err != nil || len(claimed) != 1 || claimed[0].ID != job.ID {
		t.Fatalf("gateway-waiting job was not resumed: %#v, %v", claimed, err)
	}
}

func TestGatewayQueueEventuallyResumesEveryBoundedBatch(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(101)
	if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET state='waiting',reason='gateway_unavailable' WHERE kind='ai'`); err != nil {
		t.Fatal(err)
	}
	queue := aiapp.NewGatewayQueue(f.store, time.Minute)
	if err := queue.Step(testContext); err != nil {
		t.Fatal(err)
	}
	var waiting int
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.jobs WHERE kind='ai' AND state='waiting' AND reason='gateway_unavailable'`).Scan(&waiting); err != nil || waiting != 1 {
		t.Fatalf("first bounded resume left %d jobs: %v", waiting, err)
	}
	if err := queue.Step(testContext); err != nil {
		t.Fatal(err)
	}
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.jobs WHERE kind='ai' AND state='waiting' AND reason='gateway_unavailable'`).Scan(&waiting); err != nil || waiting != 0 {
		t.Fatalf("second bounded resume left %d jobs: %v", waiting, err)
	}
}

func TestReservationRequiresPositiveCountWithoutChangingAttempt(t *testing.T) {
	f := newFixture(t)
	job := f.newReviewJobs(1)[0]
	request := f.request(job)
	if err := f.store.StartAIAttempt(testContext, f.p, job, request, strings.Repeat("d", 64), f.now.Time()); err != nil {
		t.Fatal(err)
	}
	reservation, err := ai.TerraPricing().Reservation(1, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.store.ReserveAIAttempt(testContext, f.p, job, request.ID, 0, reservation, f.now.Time()); !errors.Is(err, ai.ErrInvalidAttempt) {
		t.Fatalf("zero count accepted: %v", err)
	}
	if err = f.store.ReserveAIAttempt(testContext, f.p, job, request.ID, 1, reservation, f.now.Time()); err != nil {
		t.Fatalf("rejected reservation changed the attempt: %v", err)
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
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		return f.store.MarkAIUnknown(ctx, f.p, claimed[0], firstRequest.ID, "provider_timeout", aiapp.ProviderObservation{}, f.now.Time())
	}); err != nil {
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
	if err := f.admin.QueryRow(testContext, `SELECT state,code FROM want_keep.ai_attempt_states WHERE household_id=$1 AND attempt_id=$2 ORDER BY revision DESC LIMIT 1`, f.family.ID, third.ID).Scan(&state, &code); err != nil || state != "known_rejection" || code != "budget_exhausted" {
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

func TestReservationRechecksUnknownOutcomeAfterCounting(t *testing.T) {
	f := newFixture(t)
	claimed := f.newReviewJobs(2)
	counting := f.request(claimed[0])
	if err := f.store.StartAIAttempt(testContext, f.p, claimed[0], counting, strings.Repeat("d", 64), f.now.Time()); err != nil {
		t.Fatal(err)
	}

	unknown := f.request(claimed[1])
	reservation, _ := ai.TerraPricing().Reservation(100, 2048)
	if err := f.store.StartAIAttempt(testContext, f.p, claimed[1], unknown, strings.Repeat("e", 64), f.now.Time()); err != nil {
		t.Fatal(err)
	}
	if err := f.store.ReserveAIAttempt(testContext, f.p, claimed[1], unknown.ID, 100, reservation, f.now.Time()); err != nil {
		t.Fatal(err)
	}
	if err := f.store.BeginAIGeneration(testContext, f.p, claimed[1], unknown.ID, f.now.Time()); err != nil {
		t.Fatal(err)
	}
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		if err := f.store.MarkAIUnknown(ctx, f.p, claimed[1], unknown.ID, "provider_timeout", aiapp.ProviderObservation{}, f.now.Time()); err != nil {
			return err
		}
		return f.store.SetJobOutcome(ctx, f.p, claimed[1], jobs.Unresolved, jobs.ExternalUnknown, 0)
	}); err != nil {
		t.Fatal(err)
	}

	if err := f.store.ReserveAIAttempt(testContext, f.p, claimed[0], counting.ID, 100, reservation, f.now.Time()); !errors.Is(err, aiapp.ErrBudgetBlocked) {
		t.Fatalf("counting attempt crossed a new global block: %v", err)
	}
	var state, code string
	if err := f.admin.QueryRow(testContext, `SELECT state,code FROM want_keep.ai_attempt_states WHERE household_id=$1 AND attempt_id=$2 ORDER BY revision DESC LIMIT 1`, f.family.ID, counting.ID).Scan(&state, &code); err != nil || state != "known_rejection" || code != "budget_blocked" {
		t.Fatalf("blocked reservation state: %s/%s %v", state, code, err)
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
	var external bool
	if err := f.admin.QueryRow(testContext, `SELECT count(*),count(*) FILTER(WHERE state='completed'),count(*) FILTER(WHERE validation_state='pending_validation') FROM want_keep.ai_attempt_states`).Scan(&attempts, &completed, &pendingValidation); err != nil {
		t.Fatal(err)
	}
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.job_receipts r JOIN want_keep.jobs j ON (j.household_id,j.id)=(r.household_id,r.job_id) WHERE j.kind='ai'`).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if err := f.admin.QueryRow(testContext, `SELECT external_started FROM want_keep.jobs WHERE kind='ai'`).Scan(&external); err != nil {
		t.Fatal(err)
	}
	if attempts != 4 || completed != 1 || pendingValidation != 1 || receipts != 1 || external {
		t.Fatalf("atomic completion mismatch: states=%d completed=%d validation=%d receipts=%d external=%t", attempts, completed, pendingValidation, receipts, external)
	}
}

func TestProviderOutcomeAndJobTransitionRollBackTogether(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	if _, err := f.admin.Exec(testContext, `CREATE FUNCTION want_keep.reject_ai_failed_transition() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.kind='ai' AND NEW.state='failed' THEN RAISE EXCEPTION 'synthetic terminal failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_ai_failed_transition BEFORE UPDATE ON want_keep.jobs FOR EACH ROW EXECUTE FUNCTION want_keep.reject_ai_failed_transition()`); err != nil {
		t.Fatal(err)
	}
	usage := completedResult().Usage
	gateway := &fakeGateway{result: ai.Result{ProviderID: "resp_synthetic_refusal", ProviderModel: "gpt-5.6-terra", State: ai.Refused, Code: "policy_refusal", Usage: usage}}
	handler := aiapp.NewHandler(f.store, gateway, func() time.Time { return f.now.Time() }, uuid.NewString)
	worker := jobapp.Worker{Repository: f.store, Handler: handler, Config: jobapp.DefaultWorkerConfig(jobs.AI)}
	if err := worker.Step(testContext); err == nil {
		t.Fatal("terminal job transition failure was hidden")
	}
	var jobState, attemptState string
	var jobExternal, attemptExternal bool
	if err := f.admin.QueryRow(testContext, `SELECT state,external_started FROM want_keep.jobs WHERE kind='ai'`).Scan(&jobState, &jobExternal); err != nil {
		t.Fatal(err)
	}
	if err := f.admin.QueryRow(testContext, `SELECT state,external_started FROM want_keep.ai_attempt_states ORDER BY revision DESC LIMIT 1`).Scan(&attemptState, &attemptExternal); err != nil {
		t.Fatal(err)
	}
	if jobState != "running" || !jobExternal || attemptState != "reserved" || !attemptExternal {
		t.Fatalf("partial terminal outcome escaped rollback: job=%s/%t attempt=%s/%t", jobState, jobExternal, attemptState, attemptExternal)
	}
}

func TestReconciliationRejectsLiveLeaseAndTransitionsExpiredCall(t *testing.T) {
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
	service := aiapp.NewReconciliationService(f.maintenanceStore())
	if err := service.Reconcile(testContext, request.ID, aiapp.Charged, ai.MustCost("0.25"), "provider-dashboard:synthetic-live"); !errors.Is(err, aiapp.ErrInvalidReconciliation) {
		t.Fatalf("live provider lease was reconciled: %v", err)
	}
	if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET lease_until=clock_timestamp()-INTERVAL '1 second' WHERE household_id=$1 AND id=$2`, f.family.ID, job.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.maintenancePool().Exec(testContext, `SELECT want_keep.reconcile_ai_attempt($1::uuid,'charged',0::numeric,$2)`, request.ID, "provider-dashboard:synthetic-zero"); err == nil {
		t.Fatal("database reconciliation accepted a zero charged cost")
	}
	if _, err := f.maintenancePool().Exec(testContext, `SELECT want_keep.reconcile_ai_attempt($1::uuid,'charged',repeat('9',129)::numeric,$2)`, request.ID, "provider-dashboard:synthetic-oversized"); err == nil {
		t.Fatal("database reconciliation accepted an oversized charged cost")
	}
	if err := service.Reconcile(testContext, request.ID, aiapp.Charged, ai.MustCost("0.25"), "provider-dashboard:synthetic-expired"); err != nil {
		t.Fatal(err)
	}
	var jobState, attemptState, reconciliation, actual string
	var external bool
	if err := f.admin.QueryRow(testContext, `SELECT j.state,s.state,s.reconciliation_state,s.actual_usd::text,j.external_started FROM want_keep.jobs j JOIN LATERAL(SELECT * FROM want_keep.ai_attempt_states s WHERE (s.household_id,s.attempt_id)=($1,$2) ORDER BY revision DESC LIMIT 1) s ON true WHERE (j.household_id,j.id)=($1,$3)`, f.family.ID, request.ID, job.ID).Scan(&jobState, &attemptState, &reconciliation, &actual, &external); err != nil {
		t.Fatal(err)
	}
	if jobState != "failed" || attemptState != "unknown" || reconciliation != "resolved" || actual != "0.25" || external {
		t.Fatalf("expired reconciliation transition: %s/%s/%s/%s/%t", jobState, attemptState, reconciliation, actual, external)
	}
}

func TestReconciliationResolvesTerminalOverReservation(t *testing.T) {
	for _, test := range []struct {
		name, attemptState, jobState, validation string
		result                                   ai.Result
	}{
		{name: "completed", attemptState: "completed", jobState: "succeeded", validation: "pending_validation", result: completedResult()},
		{name: "refused", attemptState: "refused", jobState: "failed", result: ai.Result{ProviderID: "resp_synthetic_refusal", ProviderModel: "gpt-5.6-terra", State: ai.Refused, Code: "model_refusal"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			test.result.Usage = completedResult().Usage
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
			if err = f.store.BeginAIGeneration(testContext, f.p, job, request.ID, f.now.Time()); err != nil {
				t.Fatal(err)
			}
			actual := ai.MustCost("0.5")
			settlement := aiapp.Settlement{Result: test.result, Reservation: reservation, Actual: &actual, NeedsReconciliation: true}
			state, reason := jobs.Failed, jobs.PermanentFailure
			apply := func(ctx context.Context, _ household.Principal) error {
				return f.store.SaveAIOutcome(ctx, f.p, job, request.ID, settlement, f.now.Time())
			}
			if test.result.State == ai.Completed {
				state, reason = jobs.Succeeded, ""
				apply = func(ctx context.Context, _ household.Principal) error {
					return f.store.SaveAICompletion(ctx, f.p, job, request.ID, settlement, f.now.Time())
				}
			}
			execution := jobapp.Execution{Job: job, Principal: f.p}
			if err = (jobapp.Executor{Repository: f.store}).CommitOutcome(testContext, execution, jobapp.Result{State: state, Reason: reason, Apply: apply}, 0); err != nil {
				t.Fatal(err)
			}

			var attemptID, attemptState, jobState, reconciliation, validation string
			if err = f.admin.QueryRow(testContext, `SELECT a.id,s.state,j.state,s.reconciliation_state,s.validation_state FROM want_keep.ai_attempts a JOIN want_keep.jobs j ON (j.household_id,j.id)=(a.household_id,a.job_id) JOIN LATERAL(SELECT * FROM want_keep.ai_attempt_states current WHERE (current.household_id,current.attempt_id)=(a.household_id,a.id) ORDER BY revision DESC LIMIT 1) s ON true`).Scan(&attemptID, &attemptState, &jobState, &reconciliation, &validation); err != nil {
				t.Fatal(err)
			}
			if attemptState != test.attemptState || jobState != test.jobState || reconciliation != "pending" || validation != test.validation {
				t.Fatalf("terminal over-reservation before reconciliation: %s/%s/%s/%s", attemptState, jobState, reconciliation, validation)
			}

			blockedJob := f.newReviewJobs(1)[0]
			blockedRequest := f.request(blockedJob)
			if err = f.store.StartAIAttempt(testContext, f.p, blockedJob, blockedRequest, strings.Repeat("e", 64), f.now.Time()); !errors.Is(err, aiapp.ErrBudgetBlocked) {
				t.Fatalf("pending terminal reconciliation did not block: %v", err)
			}
			service := aiapp.NewReconciliationService(f.maintenanceStore())
			if err = service.Reconcile(testContext, attemptID, aiapp.Charged, actual, "provider-dashboard:terminal-result"); err != nil {
				t.Fatal(err)
			}
			var reconciledActual string
			if err = f.admin.QueryRow(testContext, `SELECT s.state,j.state,s.reconciliation_state,s.validation_state,s.actual_usd::text FROM want_keep.ai_attempts a JOIN want_keep.jobs j ON (j.household_id,j.id)=(a.household_id,a.job_id) JOIN LATERAL(SELECT * FROM want_keep.ai_attempt_states current WHERE (current.household_id,current.attempt_id)=(a.household_id,a.id) ORDER BY revision DESC LIMIT 1) s ON true WHERE a.id=$1`, attemptID).Scan(&attemptState, &jobState, &reconciliation, &validation, &reconciledActual); err != nil {
				t.Fatal(err)
			}
			if attemptState != test.attemptState || jobState != test.jobState || reconciliation != "resolved" || validation != test.validation || reconciledActual != actual.String() {
				t.Fatalf("terminal reconciliation changed outcome: %s/%s/%s/%s/%s", attemptState, jobState, reconciliation, validation, reconciledActual)
			}
			if err = f.store.StartAIAttempt(testContext, f.p, blockedJob, blockedRequest, strings.Repeat("e", 64), f.now.Time()); err != nil {
				t.Fatalf("resolved terminal result kept budget gate closed: %v", err)
			}
		})
	}
}

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

func TestUnconfirmedRetryableGenerationFailureKeepsReservation(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	gateway := &fakeGateway{generateErr: aiapp.GatewayFailure{Code: "provider_5xx", Retryable: true}}
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
}

func TestKnownRetryableFailureAllowsOnlyOneNewAttempt(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	gateway := &fakeGateway{generateErr: aiapp.GatewayFailure{Code: "provider_429", Retryable: true, ConfirmedNoCharge: true}}
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

func TestPrecountOutageWaitsAndDoesNotConsumeProviderRetry(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	gateway := &fakeGateway{
		result:   completedResult(),
		countErr: aiapp.GatewayFailure{Code: "provider_unavailable", Retryable: true},
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
	gateway.generateErr = aiapp.GatewayFailure{Code: "provider_429", Retryable: true, ConfirmedNoCharge: true}
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
	gateway := &fakeGateway{generateErr: aiapp.GatewayFailure{Code: "provider_429", Retryable: true, ConfirmedNoCharge: true}}
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
