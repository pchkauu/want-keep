package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobapp "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

type Handler struct {
	repository Repository
	gateway    Gateway
	now        func() time.Time
	newID      func() string
}

func NewHandler(repository Repository, gateway Gateway, now func() time.Time, newID func() string) *Handler {
	if now == nil {
		now = time.Now
	}
	h := &Handler{repository: repository, gateway: gateway, now: now, newID: newID}
	return h
}

func (h *Handler) Prepare(ctx context.Context, execution jobapp.Execution) (jobapp.Result, error) {
	job := execution.Job
	if h.repository == nil || h.gateway == nil || h.newID == nil || job.Kind != jobs.AI || job.ResourceID == "" || job.ResourceRevision < 1 {
		return jobapp.Result{}, jobs.ErrInvalidJob
	}
	input, err := h.repository.ReviewInput(ctx, execution.Principal, job)
	if err != nil {
		return jobapp.Result{}, err
	}
	contract := h.gateway.Contract()
	request := ai.Request{
		ID: h.newID(), JobID: job.ID, ResourceID: job.ResourceID,
		HouseholdID: job.HouseholdID, ActorID: job.ActorID, ResourceRevision: job.ResourceRevision,
		Purpose: ai.TransactionReview, Model: contract.Model, Qualification: contract.Qualification,
		PromptFingerprint: contract.PromptFingerprint, SchemaFingerprint: contract.SchemaFingerprint,
		ConfigFingerprint: contract.ConfigFingerprint, Input: input, MaximumOutputTokens: 2048,
	}
	if err = request.Validate(); err != nil {
		return jobapp.Result{}, err
	}
	now := h.now().UTC()
	if err = h.repository.StartAIAttempt(ctx, execution.Principal, job, request, requestFingerprint(request), now); err != nil {
		return h.waitOrFail(err)
	}
	counted, err := h.gateway.Count(ctx, request)
	if err != nil {
		return h.handleGatewayFailure(ctx, execution, request.ID, ai.MustCost("0"), false, err)
	}
	reservation, err := ai.TerraPricing().Reservation(counted, request.MaximumOutputTokens)
	if err != nil {
		return h.recordKnownFailure(ctx, execution, request.ID, ai.MustCost("0"), "reservation_invalid", false)
	}
	if err = h.repository.ReserveAIAttempt(ctx, execution.Principal, job, request.ID, counted, reservation, now); err != nil {
		return h.waitOrFail(err)
	}
	if err = h.repository.BeginAIGeneration(ctx, execution.Principal, job, request.ID, h.now().UTC()); err != nil {
		return jobapp.Result{}, err
	}
	providerResult, err := h.gateway.Generate(ctx, request)
	if err != nil {
		return h.handleGatewayFailure(ctx, execution, request.ID, reservation, true, err)
	}
	actual, conservative, err := ai.TerraPricing().Actual(providerResult.Usage)
	if err != nil {
		return h.markUnknown(ctx, execution, request.ID, "usage_invalid")
	}
	comparison, err := actual.Compare(reservation)
	if err != nil {
		return h.markUnknown(ctx, execution, request.ID, "cost_invalid")
	}
	settlement := Settlement{Result: providerResult, Reservation: reservation, Actual: &actual, Conservative: conservative, NeedsReconciliation: comparison > 0}
	if providerResult.State == ai.Completed {
		return jobapp.Result{State: jobs.Succeeded, Apply: func(ctx context.Context, _ household.Principal) error {
			return h.repository.SaveAICompletion(ctx, execution.Principal, job, request.ID, settlement, h.now().UTC())
		}}, nil
	}
	if _, err = h.repository.RecordAIOutcome(ctx, execution.Principal, job, request.ID, settlement, h.now().UTC()); err != nil {
		return jobapp.Result{}, err
	}
	return jobapp.Result{State: jobs.Failed, Reason: jobs.PermanentFailure}, nil
}

func (h *Handler) handleGatewayFailure(ctx context.Context, execution jobapp.Execution, attemptID string, reservation ai.Cost, generation bool, err error) (jobapp.Result, error) {
	var failure GatewayFailure
	if !errors.As(err, &failure) {
		if generation {
			return h.markUnknown(ctx, execution, attemptID, "gateway_unknown")
		}
		failure = GatewayFailure{Code: "gateway_unavailable", Retryable: true}
	}
	if failure.OutcomeUnknown {
		return h.markUnknown(ctx, execution, attemptID, failure.Code)
	}
	result := ai.Result{State: ai.KnownRejection, Code: failure.Code}
	return h.recordKnownFailure(ctx, execution, attemptID, reservation, result.Code, failure.Retryable)
}

func (h *Handler) recordKnownFailure(ctx context.Context, execution jobapp.Execution, attemptID string, reservation ai.Cost, code string, retryable bool) (jobapp.Result, error) {
	settlement := Settlement{Result: ai.Result{State: ai.KnownRejection, Code: code}, Reservation: reservation}
	canRetry, recordErr := h.repository.RecordAIOutcome(ctx, execution.Principal, execution.Job, attemptID, settlement, h.now().UTC())
	if recordErr != nil {
		return jobapp.Result{}, recordErr
	}
	if retryable && canRetry {
		return jobapp.Result{State: jobs.Ready, Reason: jobs.TemporaryFailure}, nil
	}
	return jobapp.Result{State: jobs.Failed, Reason: jobs.PermanentFailure}, nil
}

func (h *Handler) markUnknown(ctx context.Context, execution jobapp.Execution, attemptID, code string) (jobapp.Result, error) {
	if err := h.repository.MarkAIUnknown(ctx, execution.Principal, execution.Job, attemptID, code, h.now().UTC()); err != nil {
		return jobapp.Result{}, err
	}
	return jobapp.Result{State: jobs.Unresolved, Reason: jobs.ExternalUnknown}, nil
}

func (h *Handler) waitOrFail(err error) (jobapp.Result, error) {
	switch {
	case errors.Is(err, ErrBudgetExhausted), errors.Is(err, ErrBudgetBlocked), errors.Is(err, ErrConcurrency):
		return jobapp.Result{State: jobs.Waiting, Reason: jobs.BudgetWait}, nil
	case errors.Is(err, ErrRetryExhausted), errors.Is(err, ai.ErrInvalidAttempt):
		return jobapp.Result{State: jobs.Failed, Reason: jobs.PermanentFailure}, nil
	default:
		return jobapp.Result{}, err
	}
}

func requestFingerprint(request ai.Request) string {
	encoded, _ := json.Marshal(struct {
		Purpose, Model, Qualification, Prompt, Schema, Config string
		ResourceID                                            string
		ResourceRevision                                      uint64
		Input                                                 json.RawMessage
	}{string(request.Purpose), string(request.Model), string(request.Qualification), request.PromptFingerprint, request.SchemaFingerprint, request.ConfigFingerprint, request.ResourceID, request.ResourceRevision, request.Input})
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
