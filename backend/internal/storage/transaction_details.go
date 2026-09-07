package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func (s *Store) saveTransactionDetails(ctx context.Context, r ledger.Revision) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	var at, ns any
	if r.PostedAt.String() != "" {
		at, ns = splitInstant(r.PostedAt)
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.transaction_details(household_id,operation_id,revision,posted_at,posted_ns,timezone,origin,fee_knowledge,pnl_basis,merchant,note,attachment_id,allocation_reason) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NULLIF($12,'')::uuid,$13)`, scope.principal.HouseholdID(), r.OperationID, r.Revision, at, ns, r.Timezone.String(), r.Origin, r.FeeKnowledge, r.PnLBasis, r.Merchant, r.Note, r.AttachmentID, r.AllocationReason)
	return err
}

func (s *Store) loadTransactionDetails(ctx context.Context, q reader, p household.Principal, r *ledger.Revision) error {
	var at *time.Time
	var ns *int16
	var timezone string
	err := q.QueryRow(ctx, `SELECT posted_at,posted_ns,timezone,origin,fee_knowledge,pnl_basis,merchant,note,COALESCE(attachment_id::text,''),allocation_reason FROM want_keep.transaction_details WHERE household_id=$1 AND operation_id=$2 AND revision=$3`, p.HouseholdID(), r.OperationID, r.Revision).Scan(&at, &ns, &timezone, &r.Origin, &r.FeeKnowledge, &r.PnLBasis, &r.Merchant, &r.Note, &r.AttachmentID, &r.AllocationReason)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if at != nil {
		if ns == nil {
			return ErrStorage
		}
		r.PostedAt, err = restoreInstant(*at, *ns)
		if err != nil {
			return err
		}
	}
	if timezone != "" {
		r.Timezone, err = calendar.ParseTimezone(timezone)
	}
	return err
}
