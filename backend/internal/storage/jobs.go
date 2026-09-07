package storage

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

const jobColumns = `household_id,id,actor_id,kind,COALESCE(connection_id::text,''),COALESCE(connection_generation,0),binding,COALESCE(admission_revision,0),state,attempt,max_attempts,COALESCE(lease_token::text,''),lease_until,deadline,cancel_requested,cursor,coverage,gaps,secret_purpose,COALESCE(replay_request_id::text,''),range_from,range_from_ns,range_to,range_to_ns`

func scanJob(row pgx.Row) (jobs.Job, error) {
	var j jobs.Job
	var binding []byte
	var until *time.Time
	var rangeFrom, rangeTo *time.Time
	var rangeFromNS, rangeToNS *int16
	err := row.Scan(&j.HouseholdID, &j.ID, &j.ActorID, &j.Kind, &j.ConnectionID, &j.ConnectionGeneration, &binding, &j.AdmissionRevision, &j.State, &j.Attempt, &j.MaxAttempts, &j.LeaseToken, &until, &j.Deadline, &j.CancelRequested, &j.Cursor, &j.Coverage, &j.Gaps, &j.SecretPurpose, &j.ReplayRequestID, &rangeFrom, &rangeFromNS, &rangeTo, &rangeToNS)
	if err != nil {
		return j, err
	}
	if until != nil {
		j.LeaseUntil = *until
	}
	if rangeFrom != nil && rangeFromNS != nil && rangeTo != nil && rangeToNS != nil {
		j.RangeFrom = rangeFrom.UTC().Add(time.Duration(*rangeFromNS))
		j.RangeTo = rangeTo.UTC().Add(time.Duration(*rangeToNS))
	}
	if len(binding) > 0 {
		j.Binding, err = decodeBinding(binding)
		if err != nil {
			return j, err
		}
		if err = j.Binding.Validate(); err != nil {
			return j, err
		}
	}
	return j, nil
}
func (s *Store) EmitEvent(ctx context.Context, resourceType, id string, revision uint64, event string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if id == "" || resourceType == "" || event == "" || revision < 1 {
		return jobs.ErrInvalidJob
	}
	var exists bool
	err = scope.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.outbox WHERE household_id=$1 AND resource_type=$2 AND resource_id=$3 AND revision=$4 AND event_type=$5)`, scope.principal.HouseholdID(), resourceType, id, revision, event).Scan(&exists)
	if err != nil || exists {
		return err
	}
	jobID := newID()
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.jobs(household_id,id,actor_id,kind,state,max_attempts,available_at,deadline) VALUES($1,$2,$3,'outbox','ready',5,clock_timestamp(),clock_timestamp()+INTERVAL '24 hours')`, scope.principal.HouseholdID(), jobID, scope.principal.UserID())
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.outbox(household_id,id,actor_id,resource_type,resource_id,revision,event_type) VALUES($1,$2,$3,$4,$5,$6,$7)`, scope.principal.HouseholdID(), jobID, scope.principal.UserID(), resourceType, id, revision, event)
	return err
}
func (s *Store) ClaimJobs(ctx context.Context, kind string, limit int, lease time.Duration) ([]jobs.Job, error) {
	if (kind != "sync" && kind != "outbox") || limit < 1 || limit > 100 || lease < time.Second || lease > 5*time.Minute {
		return nil, jobs.ErrInvalidJob
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	// Terminalize exhausted work instead of leaving it permanently eligible but unclaimable.
	_, err = tx.Exec(ctx, `WITH exhausted AS (SELECT household_id,id FROM want_keep.jobs WHERE kind=$1 AND ((state='ready' AND (deadline<=clock_timestamp() OR attempt>=max_attempts)) OR (state='running' AND lease_until<=clock_timestamp() AND (deadline<=clock_timestamp() OR attempt>=max_attempts))) ORDER BY available_at,id LIMIT 100 FOR UPDATE SKIP LOCKED) UPDATE want_keep.jobs j SET state='failed' FROM exhausted e WHERE (j.household_id,j.id)=(e.household_id,e.id)`, kind)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT `+jobColumns+` FROM want_keep.jobs WHERE kind=$1 AND NOT cancel_requested AND deadline>clock_timestamp() AND attempt<max_attempts AND ((state='ready' AND available_at<=clock_timestamp()) OR (state='running' AND lease_until<=clock_timestamp())) ORDER BY available_at,id LIMIT $2 FOR UPDATE SKIP LOCKED`, kind, limit)
	if err != nil {
		return nil, err
	}
	result := []jobs.Job{}
	for rows.Next() {
		j, e := scanJob(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		result = append(result, j)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range result {
		j := &result[i]
		j.LeaseToken = newID()
		err = tx.QueryRow(ctx, `UPDATE want_keep.jobs SET state='running',attempt=attempt+1,lease_token=$3,lease_until=LEAST(deadline,clock_timestamp()+$4::bigint*INTERVAL '1 microsecond') WHERE household_id=$1 AND id=$2 RETURNING attempt,lease_until`, j.HouseholdID, j.ID, j.LeaseToken, lease.Microseconds()).Scan(&j.Attempt, &j.LeaseUntil)
		if err != nil {
			return nil, err
		}
		j.State = "running"
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *Store) Job(ctx context.Context, p household.Principal, id string) (jobs.Job, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return jobs.Job{}, err
	}
	return scanJob(q.QueryRow(ctx, `SELECT `+jobColumns+` FROM want_keep.jobs WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id))
}
func (s *Store) Heartbeat(ctx context.Context, p household.Principal, j jobs.Job, lease time.Duration) error {
	if lease < time.Second || lease > 5*time.Minute {
		return jobs.ErrInvalidJob
	}
	return s.WithinHousehold(ctx, p, func(ctx context.Context) error {
		scope, err := s.familyScope(ctx)
		if err != nil {
			return err
		}
		if j.HouseholdID != p.HouseholdID() {
			return household.ErrForbidden
		}
		tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.jobs SET lease_until=LEAST(deadline,clock_timestamp()+$5::bigint*INTERVAL '1 microsecond') WHERE household_id=$1 AND id=$2 AND lease_token=$3 AND attempt=$4 AND state='running' AND NOT cancel_requested AND lease_until>clock_timestamp() AND deadline>clock_timestamp()`, p.HouseholdID(), j.ID, j.LeaseToken, j.Attempt, lease.Microseconds())
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return jobs.ErrStaleAttempt
		}
		return nil
	})
}
func (s *Store) FinishJob(ctx context.Context, p household.Principal, j jobs.Job) error {
	return s.transitionJob(ctx, p, j, "succeeded", 0)
}
func (s *Store) RetryJob(ctx context.Context, p household.Principal, j jobs.Job, delay time.Duration, ambiguous bool) error {
	if delay < 0 || delay > time.Hour {
		return jobs.ErrInvalidJob
	}
	state := "ready"
	if ambiguous {
		state = "unresolved"
	}
	return s.transitionJob(ctx, p, j, state, delay)
}
func (s *Store) transitionJob(ctx context.Context, p household.Principal, j jobs.Job, state string, delay time.Duration) error {
	return s.WithinHousehold(ctx, p, func(ctx context.Context) error {
		scope, err := s.familyScope(ctx)
		if err != nil {
			return err
		}
		if j.HouseholdID != p.HouseholdID() {
			return household.ErrForbidden
		}
		tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.jobs SET state=CASE WHEN $5='ready' AND attempt>=max_attempts THEN 'failed' ELSE $5 END,available_at=clock_timestamp()+$6::bigint*INTERVAL '1 microsecond' WHERE household_id=$1 AND id=$2 AND lease_token=$3 AND attempt=$4 AND state='running' AND NOT cancel_requested AND lease_until>clock_timestamp() AND deadline>clock_timestamp()`, p.HouseholdID(), j.ID, j.LeaseToken, j.Attempt, state, delay.Microseconds())
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return jobs.ErrStaleAttempt
		}
		return nil
	})
}
func (s *Store) DatabaseTime(ctx context.Context) (time.Time, error) {
	scope, err := s.scope(ctx)
	if err != nil {
		return time.Time{}, err
	}
	var now time.Time
	err = scope.tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now)
	return now, err
}
