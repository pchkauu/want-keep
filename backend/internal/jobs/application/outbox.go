package application

import (
	"context"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

type Event struct {
	ResourceType, ResourceID, Type string
	Revision                       uint64
}
type OutboxRepository interface {
	OutboxEvent(context.Context, household.Principal, string) (Event, error)
	EnqueueReview(context.Context, household.Principal, Event) error
}
type OutboxHandler struct{ Repository OutboxRepository }

func (h OutboxHandler) Prepare(ctx context.Context, x Execution) (Result, error) {
	event, err := h.Repository.OutboxEvent(ctx, x.Principal, x.Job.ID)
	if err != nil {
		return Result{}, err
	}
	if event.Type != "transaction.changed" || event.ResourceType != "transaction" {
		return Result{State: jobs.Waiting, Reason: jobs.ConsumerUnavailable}, nil
	}
	return Result{State: jobs.Succeeded, Apply: func(ctx context.Context, p household.Principal) error {
		return h.Repository.EnqueueReview(ctx, p, event)
	}}, nil
}
