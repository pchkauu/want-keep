package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type accountAmountRow struct {
	field, knowledge, asset, reason string
	amount                          *string
}

func (r accountAmountRow) value() (reporting.Amount, error) {
	if r.knowledge == "known" {
		if r.amount == nil || r.reason != "" {
			return reporting.Amount{}, account.ErrInvalidAccount
		}
		m, err := money.NewMoney(*r.amount, money.Asset(r.asset))
		if err != nil {
			return reporting.Amount{}, err
		}
		return reporting.KnownAmount(m)
	}
	if r.amount != nil {
		return reporting.Amount{}, account.ErrInvalidAccount
	}
	return reporting.MissingAmount(reporting.Knowledge(r.knowledge), r.reason)
}
func (s *Store) Opening(ctx context.Context, p household.Principal, id string) (account.Opening, bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return account.Opening{}, false, err
	}
	o := account.Opening{AccountID: id}
	var date, at time.Time
	var ns int16
	var zone string
	err = q.QueryRow(ctx, `SELECT revision,COALESCE(operation_id::text,''),date,timezone,confirmed,actor_id,reason,at,at_ns FROM want_keep.account_openings WHERE household_id=$1 AND account_id=$2 ORDER BY revision DESC LIMIT 1`, p.HouseholdID(), id).Scan(&o.Revision, &o.OperationID, &date, &zone, &o.Confirmed, &o.ActorID, &o.Reason, &at, &ns)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, false, nil
	}
	if err != nil {
		return o, false, err
	}
	o.Date, err = calendar.ParseDate(date.Format(time.DateOnly))
	if err != nil {
		return o, false, err
	}
	o.Timezone, err = calendar.ParseTimezone(zone)
	if err != nil {
		return o, false, err
	}
	o.At, err = restoreInstant(at, ns)
	if err != nil {
		return o, false, err
	}
	rows, err := q.Query(ctx, `SELECT field,knowledge,amount::text,asset,reason FROM want_keep.opening_amounts WHERE household_id=$1 AND account_id=$2 AND revision=$3`, p.HouseholdID(), id, o.Revision)
	if err != nil {
		return o, false, err
	}
	defer rows.Close()
	values := map[string]reporting.Amount{}
	var asset money.Asset
	for rows.Next() {
		var r accountAmountRow
		if err = rows.Scan(&r.field, &r.knowledge, &r.amount, &r.asset, &r.reason); err != nil {
			return o, false, err
		}
		v, e := r.value()
		if e != nil {
			return o, false, e
		}
		values[r.field] = v
		asset = money.Asset(r.asset)
	}
	if err = rows.Err(); err != nil {
		return o, false, err
	}
	if len(values) != 4 {
		return o, false, account.ErrInvalidAccount
	}
	o.Amounts = account.Amounts{Owned: values["owned"], Available: values["available"], Locked: values["locked"], Debt: values["debt"]}
	return o, true, o.Validate(asset)
}
func (s *Store) SaveOpening(ctx context.Context, o account.Opening) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	a, err := s.Account(ctx, scope.principal, o.AccountID)
	if err != nil {
		return err
	}
	if err = o.Validate(a.Asset); err != nil {
		return err
	}
	if o.ActorID != scope.principal.UserID() {
		return household.ErrForbidden
	}
	current, exists, err := s.Opening(ctx, scope.principal, o.AccountID)
	if err != nil {
		return err
	}
	if (!exists && o.Revision != 1) || (exists && (current.Revision >= command.MaxRevision || o.Revision != current.Revision+1 || o.OperationID != current.OperationID)) {
		return command.ErrVersionConflict
	}
	if a.Revision >= command.MaxRevision {
		return command.ErrVersionConflict
	}
	at, ns := splitInstant(o.At)
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.account_openings(household_id,account_id,revision,operation_id,date,timezone,confirmed,actor_id,reason,at,at_ns) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,$8,$9,$10,$11)`, scope.principal.HouseholdID(), o.AccountID, o.Revision, o.OperationID, o.Date.String(), o.Timezone.String(), o.Confirmed, o.ActorID, o.Reason, at, ns)
	if err != nil {
		return err
	}
	for i, field := range []string{"owned", "available", "locked", "debt"} {
		v := o.Amounts.Fields()[i]
		var amount any
		if m, known := v.Value(); known {
			amount = m.Amount()
		}
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.opening_amounts(household_id,account_id,revision,field,knowledge,amount,asset,reason) VALUES($1,$2,$3,$4,$5,$6::numeric,$7,$8)`, scope.principal.HouseholdID(), o.AccountID, o.Revision, field, v.Knowledge(), amount, a.Asset, v.Reason())
		if err != nil {
			return err
		}
	}
	_, err = scope.tx.Exec(ctx, `UPDATE want_keep.accounts SET opening_date=$3,revision=revision+1 WHERE household_id=$1 AND id=$2`, scope.principal.HouseholdID(), o.AccountID, o.Date.String())
	return err
}
func (s *Store) AccountEffects(ctx context.Context, p household.Principal, id string) ([]account.Effect, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT r.operation_id,r.revision,r.occurred_at,r.occurred_ns,p.amount::text,p.asset FROM want_keep.operations o JOIN want_keep.operation_revisions r ON (r.household_id,r.operation_id,r.revision)=(o.household_id,o.id,o.revision) JOIN want_keep.postings p ON (p.household_id,p.operation_id,p.revision)=(r.household_id,r.operation_id,r.revision) WHERE o.household_id=$1 AND p.account_id=$2 AND r.state='posted' AND r.economic_type!='opening' ORDER BY r.operation_id,p.position`, p.HouseholdID(), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []account.Effect
	for rows.Next() {
		var e account.Effect
		var at time.Time
		var ns int16
		var amount, asset string
		if err = rows.Scan(&e.OperationID, &e.Revision, &at, &ns, &amount, &asset); err != nil {
			return nil, err
		}
		e.At, err = restoreInstant(at, ns)
		if err != nil {
			return nil, err
		}
		e.Amount, err = money.NewMoney(amount, money.Asset(asset))
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
