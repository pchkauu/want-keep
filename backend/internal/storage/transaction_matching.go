package storage

import (
	"context"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func (s *Store) TransactionMatchingConflict(ctx context.Context, p household.Principal, id string, revision uint64) (bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return false, err
	}
	var conflict bool
	err = q.QueryRow(ctx, `SELECT EXISTS(
 SELECT 1 FROM want_keep.matching_active_members a
 JOIN want_keep.matching_cases c ON(c.household_id,c.id)=(a.household_id,a.group_id)
 JOIN want_keep.matching_revisions g ON(g.household_id,g.id,g.revision)=(c.household_id,c.id,c.revision)
 JOIN want_keep.matching_members m ON(m.household_id,m.group_id,m.group_revision,m.operation_id)=(c.household_id,c.id,c.revision,a.operation_id)
 WHERE a.household_id=$1 AND a.operation_id=$2 AND m.operation_revision=$3 AND g.state='conflict')`, p.HouseholdID(), id, revision).Scan(&conflict)
	return conflict, err
}
