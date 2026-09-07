package storage

import (
	"context"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func (s *Store) ImportOmissions(ctx context.Context, p household.Principal, jobID string) ([]string, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return nil, err
	}
	if scope.principal != p || scope.syncJobID != jobID {
		return nil, ErrTransactionRequired
	}
	rows, err := scope.tx.Query(ctx, `SELECT DISTINCT reason FROM want_keep.quarantine WHERE household_id=$1 AND job_id=$2 AND reason IN ('source_ambiguous','unsupported_asset') ORDER BY reason`, p.HouseholdID(), jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var reason string
		if err = rows.Scan(&reason); err != nil {
			return nil, err
		}
		result = append(result, reason)
	}
	return result, rows.Err()
}
