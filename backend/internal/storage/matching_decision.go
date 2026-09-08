package storage

import (
	"context"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
)

func (s *Store) SaveMatchingDecisionGroups(ctx context.Context, decisionID string, groups []matching.Group) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	for _, g := range groups {
		if _, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.matching_decision_groups(household_id,decision_id,group_id,group_revision) VALUES($1,$2,$3,$4)`, scope.principal.HouseholdID(), decisionID, g.ID, g.Revision); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) MatchingDecisionGroups(ctx context.Context, p household.Principal, decisionID string) ([]matching.Group, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT group_id,group_revision FROM want_keep.matching_decision_groups WHERE household_id=$1 AND decision_id=$2 ORDER BY group_id`, p.HouseholdID(), decisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	refs := []matching.Group{}
	for rows.Next() {
		var g matching.Group
		if err := rows.Scan(&g.ID, &g.Revision); err != nil {
			return nil, err
		}
		refs = append(refs, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for i, g := range refs {
		refs[i], err = s.matchingGroupRevision(ctx, p, g.ID, g.Revision)
		if err != nil {
			return nil, err
		}
	}
	return refs, nil
}
