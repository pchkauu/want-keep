package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	domain "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func (s *Store) ReviewResult(ctx context.Context, p household.Principal, id string, revision uint64) (ledger.ReviewResult, bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return ledger.ReviewResult{}, false, err
	}
	r := ledger.ReviewResult{OperationID: id, Revision: revision}
	var at time.Time
	var ns int16
	err = q.QueryRow(ctx, `SELECT actor_id,state,rationale,payload_hash,recorded_at,recorded_ns FROM want_keep.ledger_review_results WHERE household_id=$1 AND operation_id=$2 AND revision=$3`, p.HouseholdID(), id, revision).Scan(&r.ActorID, &r.State, &r.Rationale, &r.PayloadHash, &at, &ns)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, false, nil
	}
	if err != nil {
		return r, false, err
	}
	r.At, err = restoreInstant(at, ns)
	if err != nil {
		return r, false, err
	}
	rows, err := q.Query(ctx, `SELECT kind,evidence_id,evidence_revision FROM want_keep.ledger_review_evidence WHERE household_id=$1 AND operation_id=$2 AND revision=$3 ORDER BY kind,evidence_id,evidence_revision`, p.HouseholdID(), id, revision)
	if err != nil {
		return r, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var e domain.Evidence
		if err = rows.Scan(&e.Kind, &e.ID, &e.Revision); err != nil {
			return r, false, err
		}
		r.Evidence = append(r.Evidence, e)
	}
	return r, rows.Err() == nil, rows.Err()
}
func (s *Store) SaveReviewResult(ctx context.Context, r ledger.ReviewResult) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.principal.UserID() != r.ActorID {
		return household.ErrForbidden
	}
	at, ns := splitInstant(r.At)
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_review_results(household_id,operation_id,revision,actor_id,state,rationale,payload_hash,recorded_at,recorded_ns) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, scope.principal.HouseholdID(), r.OperationID, r.Revision, r.ActorID, r.State, r.Rationale, r.PayloadHash, at, ns)
	if err != nil {
		return err
	}
	for _, e := range r.Evidence {
		if err = s.requireDecisionEvidence(ctx, scope, e); err != nil {
			return err
		}
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_review_evidence(household_id,operation_id,revision,kind,evidence_id,evidence_revision) VALUES($1,$2,$3,$4,$5,$6)`, scope.principal.HouseholdID(), r.OperationID, r.Revision, e.Kind, e.ID, e.Revision); err != nil {
			return err
		}
	}
	return nil
}
