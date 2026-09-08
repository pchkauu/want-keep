package storage

import (
	"context"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/application"
)

func (s *Store) OutboxEvent(ctx context.Context, p household.Principal, id string) (jobs.Event, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return jobs.Event{}, err
	}
	var event jobs.Event
	err = q.QueryRow(ctx, `SELECT resource_type,resource_id,revision,event_type FROM want_keep.outbox WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id).Scan(&event.ResourceType, &event.ResourceID, &event.Revision, &event.Type)
	return event, err
}
func (s *Store) EnqueueReview(ctx context.Context, p household.Principal, event jobs.Event) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.principal != p {
		return household.ErrForbidden
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.jobs(household_id,id,actor_id,kind,state,max_attempts,available_at,deadline,resource_id,resource_revision) VALUES($1,$2,$3,'ai','ready',5,clock_timestamp(),clock_timestamp()+INTERVAL '24 hours',$4,$5) ON CONFLICT(household_id,resource_id,resource_revision) WHERE kind='ai' DO NOTHING`, p.HouseholdID(), newID(), p.UserID(), event.ResourceID, event.Revision)
	return err
}
