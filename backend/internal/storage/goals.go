package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	goals "github.com/pchkauu/want-keep/backend/internal/goals/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Store) CreateGoal(ctx context.Context, g goals.Goal) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if err = g.Validate(); err != nil {
		return err
	}
	if err = g.Ownership.RequireCreate(scope.principal); err != nil {
		return err
	}
	if g.Revision != 1 {
		return command.ErrVersionConflict
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.goals(household_id,id,scope,owner_id,asset,revision) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,1)`, g.Ownership.HouseholdID(), g.ID, g.Ownership.Scope(), string(g.Ownership.PersonalOwnerID()), g.Asset)
	return err
}
func (s *Store) Goal(ctx context.Context, p household.Principal, id string) (goals.Goal, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return goals.Goal{}, err
	}
	g := goals.Goal{ID: id}
	var scope, owner, asset string
	err = q.QueryRow(ctx, `SELECT scope,COALESCE(owner_id::text,''),asset,revision FROM want_keep.goals WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id).Scan(&scope, &owner, &asset, &g.Revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return g, ErrNotFound
	}
	if err != nil {
		return g, err
	}
	g.Asset, err = money.ParseAsset(asset)
	if err != nil {
		return g, err
	}
	g.Ownership, err = household.NewOwnership(p.HouseholdID(), household.Scope(scope), household.UserID(owner))
	if err != nil {
		return g, err
	}
	return g, g.Validate()
}
func (s *Store) Funding(ctx context.Context, p household.Principal, goalID string, ids []string) (map[string]goals.Funding, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return nil, err
	}
	if scope.principal != p {
		return nil, household.ErrForbidden
	}
	result := map[string]goals.Funding{}
	for _, id := range ids {
		a, err := s.Account(ctx, p, id)
		if err != nil {
			return nil, err
		}
		available, err := s.AccountFunding(ctx, p, id)
		if err != nil {
			return nil, err
		}
		f := goals.Funding{AccountID: id, Asset: a.Asset, Available: available}
		rows, err := scope.tx.Query(ctx, `SELECT r.mode,r.amount::text,r.asset FROM want_keep.reservations r JOIN want_keep.goals g ON (g.household_id,g.id,g.revision)=(r.household_id,r.goal_id,r.revision) WHERE r.household_id=$1 AND r.account_id=$2 AND r.goal_id!=$3`, p.HouseholdID(), id, goalID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			r := goals.Reservation{AccountID: id}
			var amount *string
			var asset string
			if err = rows.Scan(&r.Mode, &amount, &asset); err != nil {
				rows.Close()
				return nil, err
			}
			if amount != nil {
				r.Amount, err = money.NewMoney(*amount, money.Asset(asset))
				if err != nil {
					rows.Close()
					return nil, err
				}
			}
			f.Other = append(f.Other, r)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		result[id] = f
	}
	return result, nil
}
func (s *Store) WriteReservations(ctx context.Context, g goals.Goal, reservations []goals.Reservation, reason string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if err = g.Ownership.RequireOwnerEdit(scope.principal); err != nil {
		return err
	}
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.goals SET revision=$3 WHERE household_id=$1 AND id=$2 AND revision=$4`, g.Ownership.HouseholdID(), g.ID, g.Revision, g.Revision-1)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return command.ErrVersionConflict
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.reservation_revisions(household_id,goal_id,revision,actor_id,command_id,reason) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,$6)`, g.Ownership.HouseholdID(), g.ID, g.Revision, scope.principal.UserID(), commands.CurrentCommandID(ctx), reason)
	if err != nil {
		return err
	}
	for _, r := range reservations {
		var amount any
		if r.Mode == "virtual" {
			amount = r.Amount.Amount()
		}
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.reservations(household_id,goal_id,revision,account_id,mode,amount,asset) VALUES($1,$2,$3,$4,$5,$6::numeric,$7)`, g.Ownership.HouseholdID(), g.ID, g.Revision, r.AccountID, r.Mode, amount, g.Asset); err != nil {
			return err
		}
	}
	return nil
}
