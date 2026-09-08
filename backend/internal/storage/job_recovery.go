package storage

import (
	"context"
	"time"

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
	if err = s.recoverJobs(ctx, tx, string(kind), ""); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Store) recoverJobs(ctx context.Context, tx pgx.Tx, kind string, dependency jobs.Reason) error {
	rows, err := tx.Query(ctx, `SELECT `+jobColumns+`,EXISTS(SELECT 1 FROM want_keep.memberships m WHERE m.household_id=j.household_id AND m.user_id=j.actor_id AND m.active),clock_timestamp() FROM want_keep.jobs j WHERE kind=$1 AND ((state='running' AND (lease_until<=clock_timestamp() OR COALESCE(run_deadline,deadline)<=clock_timestamp() OR cancel_requested)) OR (state='ready' AND ($2::text!='' OR cancel_requested OR COALESCE(run_deadline,deadline)<=clock_timestamp() OR attempt>=max_attempts OR NOT EXISTS(SELECT 1 FROM want_keep.memberships m WHERE m.household_id=j.household_id AND m.user_id=j.actor_id AND m.active)))) ORDER BY available_at,id LIMIT 100 FOR UPDATE OF j SKIP LOCKED`, kind, dependency)
	if err != nil {
		return err
	}
	batch := &pgx.Batch{}
	for rows.Next() {
		var active bool
		var now time.Time
		job, err := scanJob(rows, &active, &now)
		if err != nil {
			rows.Close()
			return err
		}
		outcome, changed := job.Recover(now, active, dependency)
		if changed {
			batch.Queue(`UPDATE want_keep.jobs SET state=$3,reason=$4 WHERE household_id=$1 AND id=$2`, job.HouseholdID, job.ID, outcome.State, outcome.Reason)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil || batch.Len() == 0 {
		return err
	}
	return tx.SendBatch(ctx, batch).Close()
}
