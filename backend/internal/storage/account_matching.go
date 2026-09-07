package storage

import (
	"context"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func (s *Store) AccountUnresolvedMatching(ctx context.Context, p household.Principal, id string) (bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return false, err
	}
	var unresolved bool
	err = q.QueryRow(ctx, `SELECT EXISTS(
 SELECT 1 FROM want_keep.matching_active_members m
 JOIN want_keep.matching_cases c ON(c.household_id,c.id)=(m.household_id,m.group_id)
 JOIN want_keep.matching_revisions g ON(g.household_id,g.id,g.revision)=(c.household_id,c.id,c.revision)
 JOIN want_keep.operations o ON(o.household_id,o.id)=(m.household_id,m.operation_id)
 JOIN want_keep.postings p ON(p.household_id,p.operation_id,p.revision)=(o.household_id,o.id,o.revision)
 WHERE m.household_id=$1 AND p.account_id=$2 AND g.state IN ('clarification','waiting_side','conflict'))`, p.HouseholdID(), id).Scan(&unresolved)
	return unresolved, err
}
