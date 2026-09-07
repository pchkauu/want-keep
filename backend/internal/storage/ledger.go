package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Store) AppendRevision(ctx context.Context, r ledger.Revision, expected uint64) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if err = r.Validate(); err != nil {
		return err
	}
	if r.ActorID != scope.principal.UserID() {
		return household.ErrForbidden
	}
	if expected >= command.MaxRevision || r.Revision != expected+1 {
		return command.ErrVersionConflict
	}
	family := scope.principal.HouseholdID()
	if expected == 0 {
		tag, e := scope.tx.Exec(ctx, `INSERT INTO want_keep.operations(household_id,id,revision) VALUES($1,$2,1) ON CONFLICT DO NOTHING`, family, r.OperationID)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return command.ErrVersionConflict
		}
	} else {
		tag, e := scope.tx.Exec(ctx, `UPDATE want_keep.operations SET revision=$3 WHERE household_id=$1 AND id=$2 AND revision=$4`, family, r.OperationID, r.Revision, expected)
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return command.ErrVersionConflict
		}
	}
	at, ns := splitInstant(r.OccurredAt)
	var previous any
	if expected > 0 {
		previous = expected
	}
	var month any
	if r.ExpenseMonth.String() != "" {
		month = r.ExpenseMonth.String() + "-01"
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.operation_revisions(household_id,operation_id,revision,previous_revision,actor_id,command_id,reason,economic_type,state,occurred_at,occurred_ns,cash_date,expense_month,human_override,payer_state,payer_member_id) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,$7,$8,$9,$10,$11,$12,$13,$14,$15,NULLIF($16,'')::uuid)`, family, r.OperationID, r.Revision, previous, r.ActorID, commands.CurrentCommandID(ctx), r.Reason, r.Type, r.State, at, ns, r.CashDate.String(), month, r.HumanOverride, r.PayerState, string(r.PayerMemberID))
	if err != nil {
		return err
	}
	for i, p := range r.Postings {
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.postings(household_id,operation_id,revision,position,account_id,amount,asset,role,funding,treatment) VALUES($1,$2,$3,$4,$5,$6::numeric,$7,$8,$9,$10)`, family, r.OperationID, r.Revision, i, p.AccountID, p.Money.Amount(), p.Money.Asset(), p.Role, p.Funding, p.Treatment); err != nil {
			return err
		}
	}
	if err = s.saveTransactionDetails(ctx, r); err != nil {
		return err
	}
	return s.saveLedgerAudit(ctx, r)
}
func (s *Store) CurrentLedgerRevision(ctx context.Context, p household.Principal, id string) (ledger.Revision, bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return ledger.Revision{}, false, err
	}
	var revision uint64
	err = q.QueryRow(ctx, "SELECT revision FROM want_keep.operations WHERE household_id=$1 AND id=$2", p.HouseholdID(), id).Scan(&revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return ledger.Revision{}, false, nil
	}
	if err != nil {
		return ledger.Revision{}, false, err
	}
	r, err := s.LedgerRevision(ctx, p, id, revision)
	return r, err == nil, err
}
func (s *Store) LedgerRevision(ctx context.Context, p household.Principal, id string, revision uint64) (ledger.Revision, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return ledger.Revision{}, err
	}
	r := ledger.Revision{OperationID: id, Revision: revision}
	var at, date time.Time
	var month *time.Time
	var ns int16
	err = q.QueryRow(ctx, `SELECT actor_id,reason,economic_type,state,occurred_at,occurred_ns,cash_date,expense_month,human_override,payer_state,COALESCE(payer_member_id::text,'') FROM want_keep.operation_revisions WHERE household_id=$1 AND operation_id=$2 AND revision=$3`, p.HouseholdID(), id, revision).Scan(&r.ActorID, &r.Reason, &r.Type, &r.State, &at, &ns, &date, &month, &r.HumanOverride, &r.PayerState, &r.PayerMemberID)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, errors.Join(ErrNotFound, ledger.ErrNotFound)
	}
	if err != nil {
		return r, err
	}
	r.OccurredAt, err = restoreInstant(at, ns)
	if err != nil {
		return r, err
	}
	r.CashDate, err = calendar.ParseDate(date.Format(time.DateOnly))
	if err != nil {
		return r, err
	}
	if month != nil {
		r.ExpenseMonth, err = calendar.ParseMonth(month.Format("2006-01"))
		if err != nil {
			return r, err
		}
	}
	rows, err := q.Query(ctx, `SELECT p.account_id,p.amount::text,p.asset,p.role,CASE WHEN p.funding='' AND a.product='credit_card' THEN 'unknown' ELSE p.funding END,p.treatment FROM want_keep.postings p JOIN want_keep.accounts a ON (a.household_id,a.id)=(p.household_id,p.account_id) WHERE p.household_id=$1 AND p.operation_id=$2 AND p.revision=$3 ORDER BY p.position`, p.HouseholdID(), id, revision)
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var entry ledger.Posting
		var amount, asset string
		if err = rows.Scan(&entry.AccountID, &amount, &asset, &entry.Role, &entry.Funding, &entry.Treatment); err != nil {
			return r, err
		}
		entry.Money, err = money.NewMoney(amount, money.Asset(asset))
		if err != nil {
			return r, err
		}
		r.Postings = append(r.Postings, entry)
	}
	if err = rows.Err(); err != nil {
		return r, err
	}
	rows.Close()
	if err = s.loadTransactionDetails(ctx, q, p, &r); err != nil {
		return r, err
	}
	if err = s.loadLedgerAudit(ctx, q, p, &r); err != nil {
		return r, err
	}
	return r, r.Validate()
}
