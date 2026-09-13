package application

import (
	"context"
	"time"
)

type BudgetQueue struct {
	repository BudgetQueueRepository
	now        func() time.Time
	poll       time.Duration
}

func NewBudgetQueue(repository BudgetQueueRepository, now func() time.Time, poll time.Duration) *BudgetQueue {
	if now == nil {
		now = time.Now
	}
	return &BudgetQueue{repository: repository, now: now, poll: poll}
}

func (q *BudgetQueue) Step(ctx context.Context) (int64, error) {
	if q == nil || q.repository == nil {
		return 0, ErrInvalidBudgetQueue
	}
	return q.repository.ResumeAIBudgetWaiting(ctx, q.now().UTC())
}

func (q *BudgetQueue) Run(ctx context.Context) error {
	if q == nil || q.repository == nil || q.poll < time.Second || q.poll > time.Hour {
		return ErrInvalidBudgetQueue
	}
	ticker := time.NewTicker(q.poll)
	defer ticker.Stop()
	for {
		if _, err := q.Step(ctx); err != nil && ctx.Err() == nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
