package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func (s *Store) saveLedgerParticipation(ctx context.Context, r ledger.Revision) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	hh := scope.principal.HouseholdID()
	if _, err = scope.tx.Exec(ctx, `DELETE FROM want_keep.matching_carriers WHERE household_id=$1 AND operation_id=$2`, hh, r.OperationID); err != nil {
		return err
	}
	p := r.Participation
	if p.GroupID != "" {
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_participations(household_id,operation_id,revision,group_id,kind,state) VALUES($1,$2,$3,$4,$5,$6)`, hh, r.OperationID, r.Revision, p.GroupID, p.Kind, p.State)
		if err != nil {
			return err
		}
		for _, part := range p.Parts {
			at, ns := splitInstant(part.At)
			_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_contributions(household_id,operation_id,revision,position,carrier_id,carrier_position,role,effect_state,effect_at,effect_ns,component_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, hh, r.OperationID, r.Revision, part.Position, part.CarrierID, part.CarrierPosition, part.Role, part.State, at, ns, part.ComponentID)
			if err != nil {
				return err
			}
			if part.CarrierID == r.OperationID {
				posting := r.Postings[part.Position]
				_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.matching_carriers(household_id,group_id,operation_id,position,account_id,asset,role,component_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, hh, p.GroupID, r.OperationID, part.Position, posting.AccountID, posting.Money.Asset(), part.Role, part.ComponentID)
				if err != nil {
					return err
				}
			}
		}
	}
	for i, p := range r.Postings {
		if p.FeeID != "" {
			if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_fee_ids(household_id,operation_id,revision,position,fee_id) VALUES($1,$2,$3,$4,$5)`, hh, r.OperationID, r.Revision, i, p.FeeID); err != nil {
				return err
			}
		}
	}
	if r.Correspondence != nil {
		c := r.Correspondence
		digest := c.Digest()
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_correspondences(household_id,operation_id,revision,digest,kind,namespace,reference,network,movement,from_account_id,to_account_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,'')::uuid,NULLIF($11,'')::uuid)`, hh, r.OperationID, r.Revision, digest[:], c.Kind, c.Namespace, c.Reference, c.Network, c.Movement, c.FromAccountID, c.ToAccountID)
	}
	return err
}

func (s *Store) loadLedgerParticipation(ctx context.Context, q reader, p household.Principal, r *ledger.Revision) error {
	rows, e := q.Query(ctx, `SELECT position,fee_id FROM want_keep.ledger_fee_ids WHERE household_id=$1 AND operation_id=$2 AND revision=$3`, p.HouseholdID(), r.OperationID, r.Revision)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var position int
		var id string
		if e = rows.Scan(&position, &id); e != nil {
			return e
		}
		if position < 0 || position >= len(r.Postings) {
			return ledger.ErrInvalidRevision
		}
		r.Postings[position].FeeID = id
	}
	if e = rows.Err(); e != nil {
		return e
	}
	rows.Close()
	err := q.QueryRow(ctx, `SELECT group_id,kind,state FROM want_keep.ledger_participations WHERE household_id=$1 AND operation_id=$2 AND revision=$3`, p.HouseholdID(), r.OperationID, r.Revision).Scan(&r.Participation.GroupID, &r.Participation.Kind, &r.Participation.State)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err == nil {
		rows, err := q.Query(ctx, `SELECT position,carrier_id,carrier_position,role,effect_state,effect_at,effect_ns,component_id FROM want_keep.ledger_contributions WHERE household_id=$1 AND operation_id=$2 AND revision=$3 ORDER BY position`, p.HouseholdID(), r.OperationID, r.Revision)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var part ledger.Contribution
			var at time.Time
			var ns int16
			if err = rows.Scan(&part.Position, &part.CarrierID, &part.CarrierPosition, &part.Role, &part.State, &at, &ns, &part.ComponentID); err != nil {
				return err
			}
			part.At, err = restoreInstant(at, ns)
			if err != nil {
				return err
			}
			r.Participation.Parts = append(r.Participation.Parts, part)
		}
		if err = rows.Err(); err != nil {
			return err
		}
		rows.Close()
	}
	c := ledger.Correspondence{}
	err = q.QueryRow(ctx, `SELECT kind,namespace,reference,network,movement,COALESCE(from_account_id::text,''),COALESCE(to_account_id::text,'') FROM want_keep.ledger_correspondences WHERE household_id=$1 AND operation_id=$2 AND revision=$3`, p.HouseholdID(), r.OperationID, r.Revision).Scan(&c.Kind, &c.Namespace, &c.Reference, &c.Network, &c.Movement, &c.FromAccountID, &c.ToAccountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err = c.Validate(); err != nil {
		return ErrStorage
	}
	r.Correspondence = &c
	return nil
}
