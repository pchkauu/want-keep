package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
)

func (s *Store) MatchingGroup(ctx context.Context, p household.Principal, id string) (matching.Group, error) {
	return s.matchingGroupRevision(ctx, p, id, 0)
}

func (s *Store) matchingGroupRevision(ctx context.Context, p household.Principal, id string, revision uint64) (matching.Group, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return matching.Group{}, err
	}
	g := matching.Group{ID: id, Members: []matching.Member{}, Candidates: []matching.Candidate{}}
	var at time.Time
	var ns int16
	err = q.QueryRow(ctx, `SELECT r.revision,r.primary_id,r.kind,r.state,r.actor_id,r.at,r.at_ns,r.reason,COALESCE(r.decision_id::text,''),r.candidates_complete FROM want_keep.matching_cases c JOIN want_keep.matching_revisions r ON (r.household_id,r.id)=(c.household_id,c.id) AND r.revision=CASE WHEN $3::bigint=0 THEN c.revision ELSE $3 END WHERE c.household_id=$1 AND c.id=$2`, p.HouseholdID(), id, revision).Scan(&g.Revision, &g.PrimaryID, &g.Kind, &g.State, &g.ActorID, &at, &ns, &g.Reason, &g.DecisionID, &g.CandidatesComplete)
	if errors.Is(err, pgx.ErrNoRows) {
		return g, matching.ErrNotFound
	}
	if err != nil {
		return g, err
	}
	g.At, err = restoreInstant(at, ns)
	if err != nil {
		return g, err
	}
	rows, err := q.Query(ctx, `SELECT operation_id,operation_revision FROM want_keep.matching_members WHERE household_id=$1 AND group_id=$2 AND group_revision=$3 ORDER BY operation_id`, p.HouseholdID(), id, g.Revision)
	if err != nil {
		return g, err
	}
	defer rows.Close()
	for rows.Next() {
		var m matching.Member
		if err = rows.Scan(&m.OperationID, &m.Revision); err != nil {
			return g, err
		}
		g.Members = append(g.Members, m)
	}
	if err = rows.Err(); err != nil {
		return g, err
	}
	rows.Close()
	rows, err = q.Query(ctx, `SELECT operation_id,operation_revision,reason FROM want_keep.matching_candidates WHERE household_id=$1 AND group_id=$2 AND group_revision=$3 ORDER BY operation_id`, p.HouseholdID(), id, g.Revision)
	if err != nil {
		return g, err
	}
	defer rows.Close()
	for rows.Next() {
		var c matching.Candidate
		if err = rows.Scan(&c.OperationID, &c.Revision, &c.Reason); err != nil {
			return g, err
		}
		g.Candidates = append(g.Candidates, c)
	}
	if err = rows.Err(); err != nil {
		return g, err
	}
	return g, g.Validate()
}

func (s *Store) MatchingForOperation(ctx context.Context, p household.Principal, id string) (matching.Group, bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return matching.Group{}, false, err
	}
	var group string
	err = q.QueryRow(ctx, `SELECT group_id FROM want_keep.matching_active_members WHERE household_id=$1 AND operation_id=$2`, p.HouseholdID(), id).Scan(&group)
	if errors.Is(err, pgx.ErrNoRows) {
		return matching.Group{}, false, nil
	}
	if err != nil {
		return matching.Group{}, false, err
	}
	g, err := s.MatchingGroup(ctx, p, group)
	return g, err == nil, err
}

func (s *Store) SaveMatchingGroup(ctx context.Context, g matching.Group, expected uint64) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if err = g.Validate(); err != nil {
		return err
	}
	if g.ActorID != scope.principal.UserID() {
		return household.ErrForbidden
	}
	if expected >= command.MaxRevision || g.Revision != expected+1 {
		return command.ErrVersionConflict
	}
	hh := scope.principal.HouseholdID()
	if expected == 0 {
		tag, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.matching_cases(household_id,id,revision) VALUES($1,$2,1) ON CONFLICT DO NOTHING`, hh, g.ID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return command.ErrVersionConflict
		}
	} else {
		tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.matching_cases SET revision=$3 WHERE household_id=$1 AND id=$2 AND revision=$4`, hh, g.ID, g.Revision, expected)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return command.ErrVersionConflict
		}
	}
	at, ns := splitInstant(g.At)
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.matching_revisions(household_id,id,revision,primary_id,kind,state,actor_id,at,at_ns,reason,decision_id,candidates_complete) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,'')::uuid,$12)`, hh, g.ID, g.Revision, g.PrimaryID, g.Kind, g.State, g.ActorID, at, ns, g.Reason, g.DecisionID, g.CandidatesComplete)
	if err != nil {
		return err
	}
	if _, err = scope.tx.Exec(ctx, `DELETE FROM want_keep.matching_active_members WHERE household_id=$1 AND group_id=$2`, hh, g.ID); err != nil {
		return err
	}
	for _, m := range g.Members {
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.matching_members(household_id,group_id,group_revision,operation_id,operation_revision) VALUES($1,$2,$3,$4,$5)`, hh, g.ID, g.Revision, m.OperationID, m.Revision)
		if err != nil {
			return err
		}
		if g.State != matching.Separate && g.State != matching.Unlinked {
			_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.matching_active_members(household_id,operation_id,group_id) VALUES($1,$2,$3)`, hh, m.OperationID, g.ID)
			if err != nil {
				return err
			}
		}
	}
	for _, c := range g.Candidates {
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.matching_candidates(household_id,group_id,group_revision,operation_id,operation_revision,reason) VALUES($1,$2,$3,$4,$5,$6)`, hh, g.ID, g.Revision, c.OperationID, c.Revision, c.Reason)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ReleaseMatchingCarriers(ctx context.Context, id string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `DELETE FROM want_keep.matching_carriers WHERE household_id=$1 AND group_id=$2`, scope.principal.HouseholdID(), id)
	return err
}

func (s *Store) RestoreMatchingCarriers(ctx context.Context, id string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.matching_carriers(household_id,group_id,operation_id,position,account_id,asset,role,component_id)
 SELECT o.household_id,a.group_id,o.id,c.position,p.account_id,p.asset,c.role,c.component_id
 FROM want_keep.matching_active_members a JOIN want_keep.operations o ON(o.household_id,o.id)=(a.household_id,a.operation_id)
 JOIN want_keep.ledger_participations lp ON(lp.household_id,lp.operation_id,lp.revision,lp.group_id)=(o.household_id,o.id,o.revision,a.group_id)
 JOIN want_keep.ledger_contributions c ON(c.household_id,c.operation_id,c.revision)=(o.household_id,o.id,o.revision)
 JOIN want_keep.postings p ON(p.household_id,p.operation_id,p.revision,p.position)=(c.household_id,c.operation_id,c.revision,c.position)
 WHERE a.household_id=$1 AND a.group_id=$2 AND c.carrier_id=o.id
 AND NOT EXISTS(SELECT 1 FROM want_keep.matching_carriers old WHERE(old.household_id,old.operation_id,old.position)=(o.household_id,o.id,c.position))`, scope.principal.HouseholdID(), id)
	return err
}
