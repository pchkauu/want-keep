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
	rows, err := scope.tx.Query(ctx, `SELECT DISTINCT reason FROM want_keep.quarantine WHERE household_id=$1 AND job_id=$2 AND reason IN ('source_ambiguous','unsupported_asset','transaction_unresolved')
 UNION SELECT 'matching_unresolved' FROM want_keep.source_provenance pr
 JOIN want_keep.source_revisions sr ON(sr.household_id,sr.source_id,sr.revision)=(pr.household_id,pr.source_id,pr.revision)
 JOIN want_keep.source_records source ON(source.household_id,source.id)=(pr.household_id,pr.source_id)
 JOIN want_keep.matching_active_members m ON(m.household_id,m.operation_id)=(source.household_id,source.operation_id)
 JOIN want_keep.matching_cases c ON(c.household_id,c.id)=(m.household_id,m.group_id)
 JOIN want_keep.matching_revisions r ON(r.household_id,r.id,r.revision)=(c.household_id,c.id,c.revision)
 WHERE pr.household_id=$1 AND pr.job_id=$2 AND r.state IN ('clarification','waiting_side','conflict') ORDER BY reason`, p.HouseholdID(), jobID)
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
