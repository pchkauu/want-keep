package application

import (
	"context"
	"time"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/application"
)

type ExecutionRepository interface {
	Queue
	RecoverJobs(context.Context, jobs.Kind) error
	WithinHousehold(context.Context, household.Principal, func(context.Context) error) error
	JobPrincipal(context.Context, jobs.Job) (household.Principal, error)
	FenceJob(context.Context, household.Principal, jobs.Job) (jobs.Job, error)
	JobReceipt(context.Context, household.Principal, jobs.Job) (bool, error)
	RecordJobReceipt(context.Context, household.Principal, jobs.Job) error
	BeginExternal(context.Context, household.Principal, jobs.Job) error
	SetJobOutcome(context.Context, household.Principal, jobs.Job, jobs.State, jobs.Reason, time.Duration) error
	PauseReady(context.Context, jobs.Kind, jobs.Reason) error
	ResumeWaiting(context.Context, jobs.Kind, jobs.Reason) error
}

// An effect is trusted application code and may perform only transactional database work.
type Effect func(context.Context, household.Principal) error
type Execution struct {
	Job        jobs.Job
	Principal  household.Principal
	repository ExecutionRepository
}

func (e Execution) BeginExternal(ctx context.Context) error {
	return e.repository.BeginExternal(ctx, e.Principal, e.Job)
}

type Result struct {
	State  jobs.State
	Reason jobs.Reason
	Apply  Effect
}
type Handler interface {
	Prepare(context.Context, Execution) (Result, error)
}

type Executor struct{ Repository ExecutionRepository }

func (e Executor) Complete(ctx context.Context, x Execution, apply Effect) error {
	return e.Repository.WithinHousehold(ctx, x.Principal, func(ctx context.Context) error {
		done, err := e.Repository.JobReceipt(ctx, x.Principal, x.Job)
		if err != nil || done {
			return err
		}
		// Source handlers commit pages through admission, which creates the receipt atomically.
		if x.Job.Kind == jobs.Sync {
			return jobs.ErrInvalidJob
		}
		if _, err = e.Repository.FenceJob(ctx, x.Principal, x.Job); err != nil {
			return err
		}
		if apply != nil {
			if err = apply(ctx, x.Principal); err != nil {
				return err
			}
		}
		return e.Repository.FinishJob(ctx, x.Principal, x.Job)
	})
}

// CompleteReview binds the ledger result to the target in the persisted job, never model identity.
func (e Executor) CompleteReview(ctx context.Context, x Execution, service *ledger.Service, in ledger.ReviewInput) error {
	if x.Job.Kind != jobs.AI || x.Job.ResourceID != in.OperationID || x.Job.ResourceRevision != in.Revision {
		return jobs.ErrInvalidJob
	}
	return e.Complete(ctx, x, func(ctx context.Context, p household.Principal) error { return service.CompleteReview(ctx, p, in) })
}
