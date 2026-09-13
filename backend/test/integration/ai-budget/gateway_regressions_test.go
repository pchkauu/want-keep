//go:build integration

package aibudget_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	aiapp "github.com/pchkauu/want-keep/backend/internal/ai/application"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobapp "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func TestGenerationRechecksSameMonthReconciliationBarrier(t *testing.T) {
	f := newFixture(t)
	claimed := f.newReviewJobs(2)
	reservation, err := ai.TerraPricing().Reservation(100, 2048)
	if err != nil {
		t.Fatal(err)
	}
	requests := []ai.Request{f.request(claimed[0]), f.request(claimed[1])}
	for index, request := range requests {
		if err = f.store.StartAIAttempt(testContext, f.p, claimed[index], request, strings.Repeat("d", 64), f.now.Time()); err != nil {
			t.Fatal(err)
		}
		if err = f.store.ReserveAIAttempt(testContext, f.p, claimed[index], request.ID, 100, reservation, f.now.Time()); err != nil {
			t.Fatal(err)
		}
	}
	if err = f.store.BeginAIGeneration(testContext, f.p, claimed[0], requests[0].ID, f.now.Time()); err != nil {
		t.Fatal(err)
	}
	if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		if err := f.store.MarkAIUnknown(ctx, f.p, claimed[0], requests[0].ID, "provider_timeout", aiapp.ProviderObservation{}, f.now.Time()); err != nil {
			return err
		}
		return f.store.SetJobOutcome(ctx, f.p, claimed[0], jobs.Unresolved, jobs.ExternalUnknown, 0)
	}); err != nil {
		t.Fatal(err)
	}
	if err = f.store.BeginAIGeneration(testContext, f.p, claimed[1], requests[1].ID, f.now.Time()); !errors.Is(err, aiapp.ErrBudgetBlocked) {
		t.Fatalf("generation crossed reconciliation barrier: %v", err)
	}
	var state, code string
	var external bool
	if err = f.admin.QueryRow(testContext, `SELECT state,code,external_started FROM want_keep.ai_attempt_states WHERE household_id=$1 AND attempt_id=$2 ORDER BY revision DESC LIMIT 1`, f.family.ID, requests[1].ID).Scan(&state, &code, &external); err != nil {
		t.Fatal(err)
	}
	if state != "known_rejection" || code != "budget_blocked" || external {
		t.Fatalf("unsafe generation state: %s/%s external=%t", state, code, external)
	}
}

func TestUnknownModelKeepsReservationUntilReconciliation(t *testing.T) {
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
	if err = f.store.BeginAIGeneration(testContext, f.p, job, request.ID, f.now.Time()); err != nil {
		t.Fatal(err)
	}
	input, cached, output, reasoning, total := int64(100), int64(20), int64(50), int64(12), int64(150)
	observation := aiapp.ProviderObservation{ID: "resp_synthetic", Model: "unexpected-model", Usage: &aiapp.ObservedUsage{
		InputTokens: &input, CachedTokens: &cached, OutputTokens: &output, ReasoningTokens: &reasoning, TotalTokens: &total,
	}}
	if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		return f.store.MarkAIUnknown(ctx, f.p, job, request.ID, "model_mismatch", observation, f.now.Time())
	}); err != nil {
		t.Fatal(err)
	}
	var storedReservation, reconciliation string
	var actual *string
	if err = f.admin.QueryRow(testContext, `SELECT reservation_usd::text,actual_usd::text,reconciliation_state FROM want_keep.ai_attempt_states WHERE household_id=$1 AND attempt_id=$2 ORDER BY revision DESC LIMIT 1`, f.family.ID, request.ID).Scan(&storedReservation, &actual, &reconciliation); err != nil {
		t.Fatal(err)
	}
	if storedReservation != reservation.String() || actual != nil || reconciliation != "pending" {
		t.Fatalf("unknown model changed accounting: reservation=%s actual=%v reconciliation=%s", storedReservation, actual, reconciliation)
	}
}

func TestRecoveryTerminatesUnstartedAIJobForRevokedMember(t *testing.T) {
	f := newFixture(t)
	job := f.newReviewJobs(1)[0]
	if _, err := f.admin.Exec(testContext, `UPDATE want_keep.memberships SET active=false WHERE household_id=$1 AND user_id=$2`, f.family.ID, f.p.UserID()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET lease_until=clock_timestamp()-INTERVAL '1 second' WHERE household_id=$1 AND id=$2`, f.family.ID, job.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.store.RecoverJobs(testContext, jobs.AI); err != nil {
		t.Fatal(err)
	}
	var state, reason string
	var attempt int
	if err := f.admin.QueryRow(testContext, `SELECT state,reason,attempt FROM want_keep.jobs WHERE household_id=$1 AND id=$2`, f.family.ID, job.ID).Scan(&state, &reason, &attempt); err != nil {
		t.Fatal(err)
	}
	if state != "failed" || reason != "membership_revoked" || attempt != job.Attempt {
		t.Fatalf("revoked job recovered: %s/%s attempt=%d", state, reason, attempt)
	}
	claimed, err := f.store.ClaimJobs(testContext, string(jobs.AI), 1, time.Minute)
	if err != nil || len(claimed) != 0 {
		t.Fatalf("revoked job reclaimed: %d/%v", len(claimed), err)
	}
}

func TestPreGenerationRepositoryFailuresDoNotExhaustJob(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	repository := &failingReviewRepository{Repository: f.store, remaining: 5}
	gateway := &fakeGateway{result: completedResult()}
	handler := aiapp.NewHandler(repository, gateway, func() time.Time { return f.now.Time() }, uuid.NewString)
	worker := jobapp.Worker{Repository: f.store, Handler: handler, Config: jobapp.DefaultWorkerConfig(jobs.AI)}
	queue := aiapp.NewGatewayQueue(f.store, time.Minute)
	for range 5 {
		if err := worker.Step(testContext); err == nil {
			t.Fatal("repository failure was not returned")
		}
		var state, reason string
		var attempt int
		if err := f.admin.QueryRow(testContext, `SELECT state,reason,attempt FROM want_keep.jobs WHERE kind='ai'`).Scan(&state, &reason, &attempt); err != nil || state != "waiting" || reason != "gateway_unavailable" || attempt != 0 {
			t.Fatalf("pre-generation failure consumed attempt: %s/%s/%d %v", state, reason, attempt, err)
		}
		if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET available_at=clock_timestamp() WHERE kind='ai'`); err != nil {
			t.Fatal(err)
		}
		if err := queue.Step(testContext); err != nil {
			t.Fatal(err)
		}
	}
	if err := worker.Step(testContext); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := f.admin.QueryRow(testContext, `SELECT state FROM want_keep.jobs WHERE kind='ai'`).Scan(&state); err != nil || state != "succeeded" {
		t.Fatalf("review did not recover: %s %v", state, err)
	}
	if gateway.countCalls.Load() != 1 || gateway.generateCalls.Load() != 1 {
		t.Fatalf("provider calls after recovery: count=%d generation=%d", gateway.countCalls.Load(), gateway.generateCalls.Load())
	}
}

type failingReviewRepository struct {
	aiapp.Repository
	remaining int
}

func (r *failingReviewRepository) ReviewInput(ctx context.Context, principal household.Principal, job jobs.Job) (json.RawMessage, error) {
	if r.remaining > 0 {
		r.remaining--
		return nil, errors.New("synthetic repository failure")
	}
	return r.Repository.ReviewInput(ctx, principal, job)
}
