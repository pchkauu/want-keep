package application

import (
	"context"
	"time"

	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

type ScheduledSource struct {
	ConnectionID string
	HouseholdID  household.HouseholdID
	OwnerID      household.UserID
	Deadline     time.Time
}
type ScheduleRepository interface {
	DueSources(context.Context, connections.Binding, int) ([]ScheduledSource, error)
	Membership(context.Context, household.HouseholdID, household.UserID) (household.Membership, error)
}
type SyncAdmission interface {
	ScheduleSync(context.Context, household.Principal, string, connections.Binding, time.Time) (jobs.Job, error)
}
type Scheduler struct {
	Repository ScheduleRepository
	Admission  SyncAdmission
	Bindings   []connections.Binding
	Report     func(string)
}

func (s Scheduler) Tick(ctx context.Context) error {
	for _, b := range s.Bindings {
		if err := b.Validate(); err != nil {
			return err
		}
		sources, err := s.Repository.DueSources(ctx, b, 100)
		if err != nil {
			return err
		}
		for _, source := range sources {
			member, err := s.Repository.Membership(ctx, source.HouseholdID, source.OwnerID)
			if err == nil {
				p, e := member.Principal()
				err = e
				if err == nil {
					_, err = s.Admission.ScheduleSync(ctx, p, source.ConnectionID, b, source.Deadline)
				}
			}
			if err != nil && s.Report != nil {
				s.Report("source_schedule_blocked")
			}
		}
	}
	return nil
}
func (s Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if err := s.Tick(ctx); err != nil && ctx.Err() == nil && s.Report != nil {
			s.Report("scheduler_unavailable")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
