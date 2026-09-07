package application

import (
	"context"
	"time"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

type Queue interface {
	ClaimJobs(context.Context, string, int, time.Duration) ([]jobs.Job, error)
	Heartbeat(context.Context, household.Principal, jobs.Job, time.Duration) error
	FinishJob(context.Context, household.Principal, jobs.Job) error
	RetryJob(context.Context, household.Principal, jobs.Job, time.Duration, bool) error
}
type RetentionRepository interface {
	CleanupCommandDetails(context.Context, calendar.Instant, int) (int64, error)
	CleanupCommandTombstones(context.Context, calendar.Instant, int) (int64, error)
}
type Retention struct{ Repository RetentionRepository }

func (r Retention) RunDetails(ctx context.Context, now calendar.Instant, batch int) (int64, error) {
	return r.Repository.CleanupCommandDetails(ctx, now, batch)
}
func (r Retention) RunTombstones(ctx context.Context, now calendar.Instant, batch int) (int64, error) {
	return r.Repository.CleanupCommandTombstones(ctx, now, batch)
}
