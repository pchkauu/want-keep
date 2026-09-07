package storage

import (
	"context"

	"github.com/jackc/pgx/v5"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func (s *Store) RecoverJobs(ctx context.Context, kind jobs.Kind) error {
	if !kind.Valid() {
		return jobs.ErrInvalidJob
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if err = s.recoverJobs(ctx, tx, string(kind)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Store) recoverJobs(ctx context.Context, tx pgx.Tx, kind string) error {
	var err error
	// Recover uncertainty before considering retries, including the last permitted attempt.
	_, err = tx.Exec(ctx, `WITH expired AS (SELECT household_id,id FROM want_keep.jobs WHERE kind=$1 AND state='running' AND (lease_until<=clock_timestamp() OR COALESCE(run_deadline,deadline)<=clock_timestamp() OR cancel_requested) ORDER BY available_at,id LIMIT 100 FOR UPDATE SKIP LOCKED) UPDATE want_keep.jobs j SET state=CASE WHEN external_started THEN 'unresolved' WHEN cancel_requested THEN 'canceled' WHEN COALESCE(run_deadline,deadline)<=clock_timestamp() OR attempt>=max_attempts THEN 'failed' ELSE 'ready' END,reason=CASE WHEN external_started THEN 'external_unknown' WHEN cancel_requested THEN 'canceled' WHEN COALESCE(run_deadline,deadline)<=clock_timestamp() THEN 'deadline_exceeded' WHEN attempt>=max_attempts THEN 'attempts_exhausted' ELSE 'temporary_failure' END FROM expired e WHERE (j.household_id,j.id)=(e.household_id,e.id)`, kind)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `WITH expired AS (SELECT j.household_id,j.id FROM want_keep.jobs j WHERE kind=$1 AND state='ready' AND (cancel_requested OR COALESCE(run_deadline,deadline)<=clock_timestamp() OR attempt>=max_attempts OR NOT EXISTS(SELECT 1 FROM want_keep.memberships m WHERE m.household_id=j.household_id AND m.user_id=j.actor_id AND m.active)) ORDER BY available_at,id LIMIT 100 FOR UPDATE OF j SKIP LOCKED) UPDATE want_keep.jobs j SET state=CASE WHEN cancel_requested THEN 'canceled' ELSE 'failed' END,reason=CASE WHEN cancel_requested THEN 'canceled' WHEN COALESCE(run_deadline,deadline)<=clock_timestamp() THEN 'deadline_exceeded' WHEN attempt>=max_attempts THEN 'attempts_exhausted' ELSE 'membership_revoked' END FROM expired e WHERE (j.household_id,j.id)=(e.household_id,e.id)`, kind)
	if err != nil {
		return err
	}
	return nil
}
