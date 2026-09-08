package storage

import (
	"context"

	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	domain "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func (s *Store) DueSources(ctx context.Context, b connections.Binding, limit int) ([]jobs.ScheduledSource, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 100 {
		return nil, connections.ErrInvalidAdmission
	}
	rows, err := s.pool.Query(ctx, `SELECT c.id,c.household_id,c.external_owner_id,clock_timestamp()+INTERVAL '24 hours' FROM want_keep.connections c JOIN want_keep.sync_schedules s ON (s.household_id,s.connection_id)=(c.household_id,c.id) WHERE c.provider=$1 AND c.authorized AND s.next_due<=clock_timestamp() ORDER BY s.next_due,c.household_id,c.id LIMIT $2`, b.Provider, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []jobs.ScheduledSource{}
	for rows.Next() {
		var source jobs.ScheduledSource
		if err = rows.Scan(&source.ConnectionID, &source.HouseholdID, &source.OwnerID, &source.Deadline); err != nil {
			return nil, err
		}
		result = append(result, source)
	}
	return result, rows.Err()
}
func (s *Store) SyncDue(ctx context.Context, id string) (bool, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return false, err
	}
	var due bool
	err = scope.tx.QueryRow(ctx, `SELECT next_due<=clock_timestamp() FROM want_keep.sync_schedules WHERE household_id=$1 AND connection_id=$2 FOR UPDATE`, scope.principal.HouseholdID(), id).Scan(&due)
	return due, err
}
func (s *Store) AdvanceSyncSchedule(ctx context.Context, id string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `UPDATE want_keep.sync_schedules SET next_due=clock_timestamp()+INTERVAL '1 hour' WHERE household_id=$1 AND connection_id=$2`, scope.principal.HouseholdID(), id)
	return err
}
func (s *Store) SyncProgress(ctx context.Context, p household.Principal, id string) (domain.SyncProgress, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return domain.SyncProgress{}, err
	}
	var progress domain.SyncProgress
	err = q.QueryRow(ctx, `SELECT cursor,coverage,gaps,last_success_at,completed FROM want_keep.sync_progress WHERE household_id=$1 AND connection_id=$2`, p.HouseholdID(), id).Scan(&progress.Cursor, &progress.Coverage, &progress.Gaps, &progress.LastSuccessAt, &progress.Completed)
	return progress, err
}
