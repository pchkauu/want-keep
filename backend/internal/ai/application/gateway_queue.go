package application

import (
	"context"
	"time"

	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

type GatewayQueue struct {
	repository GatewayQueueRepository
	poll       time.Duration
}

func NewGatewayQueue(repository GatewayQueueRepository, poll time.Duration) *GatewayQueue {
	return &GatewayQueue{repository: repository, poll: poll}
}

func (q *GatewayQueue) Step(ctx context.Context) error {
	if q == nil || q.repository == nil {
		return ErrInvalidGatewayQueue
	}
	return q.repository.ResumeWaiting(ctx, jobs.AI, jobs.GatewayUnavailable)
}

func (q *GatewayQueue) Run(ctx context.Context) error {
	if q == nil || q.repository == nil || q.poll < time.Second || q.poll > time.Hour {
		return ErrInvalidGatewayQueue
	}
	ticker := time.NewTicker(q.poll)
	defer ticker.Stop()
	for {
		if err := q.Step(ctx); err != nil && ctx.Err() == nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
