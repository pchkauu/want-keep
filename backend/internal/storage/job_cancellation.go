package storage

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// The caller selects and locks the affected jobs in its existing transaction.
func (s *Store) cancelJobs(ctx context.Context, tx pgx.Tx, rows pgx.Rows) error {
	batch := &pgx.Batch{}
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			rows.Close()
			return err
		}
		outcome := job.Cancel()
		batch.Queue(`UPDATE want_keep.jobs SET state=$3,reason=$4,cancel_requested=true WHERE household_id=$1 AND id=$2`, job.HouseholdID, job.ID, outcome.State, outcome.Reason)
	}
	err := rows.Err()
	rows.Close()
	if err != nil || batch.Len() == 0 {
		return err
	}
	return tx.SendBatch(ctx, batch).Close()
}
