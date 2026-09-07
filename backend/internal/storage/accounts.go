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

func (s *Store) CreateAccount(ctx context.Context, a account.Account) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if err = a.Validate(); err != nil {
		return err
	}
	if err = a.Ownership.RequireRead(scope.principal); err != nil {
		return err
	}
	if a.Revision != 1 {
		return command.ErrVersionConflict
	}
	if a.Ownership.Scope() == household.Personal && a.Ownership.PersonalOwnerID() != scope.principal.UserID() {
		return household.ErrForbidden
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.accounts(household_id,id,name,asset,scope,owner_id,product,revision,opening_date,external_account_id) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,$7,$8,$9,NULLIF($10,'')::uuid)`, a.Ownership.HouseholdID(), a.ID, a.Name, a.Asset, a.Ownership.Scope(), string(a.Ownership.PersonalOwnerID()), a.Product, a.Revision, a.OpeningDate.String(), a.ExternalAccountID)
	return err
}
func (s *Store) Account(ctx context.Context, p household.Principal, id string) (account.Account, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return account.Account{}, err
	}
	var a account.Account
	var scope, owner, asset string
	var date time.Time
	err = q.QueryRow(ctx, `SELECT id,name,asset,scope,COALESCE(owner_id::text,''),product,revision,opening_date,COALESCE(external_account_id::text,'') FROM want_keep.accounts WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id).Scan(&a.ID, &a.Name, &asset, &scope, &owner, &a.Product, &a.Revision, &date, &a.ExternalAccountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	if err != nil {
		return a, err
	}
	a.Asset, err = money.ParseAsset(asset)
	if err != nil {
		return a, err
	}
	a.Ownership, err = household.NewOwnership(p.HouseholdID(), household.Scope(scope), household.UserID(owner))
	if err != nil {
		return a, err
	}
	a.OpeningDate, err = calendar.ParseDate(date.Format(time.DateOnly))
	if err != nil {
		return a, err
	}
	return a, a.Validate()
}
func (s *Store) RecordBalance(ctx context.Context, b account.Balance) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if err = b.Validate(); err != nil {
		return err
	}
	a, err := s.Account(ctx, scope.principal, b.AccountID)
	if err != nil {
		return err
	}
	if a.Revision >= command.MaxRevision {
		return command.ErrVersionConflict
	}
	var value any
	if m, known := b.Amount.Value(); known {
		if m.Asset() != a.Asset {
			return money.ErrAssetMismatch
		}
		value = m.Amount()
	}
	at, ns := splitInstant(b.ObservedAt)
	reasons := b.Coverage.Reasons()
	if reasons == nil {
		reasons = []string{}
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.balance_snapshots(household_id,account_id,revision,field,knowledge,amount,asset,reason,coverage,coverage_reasons,freshness,observed_at,observed_ns) VALUES($1,$2,$3,$4,$5,$6::numeric,$7,$8,$9,$10,$11,$12,$13)`, scope.principal.HouseholdID(), b.AccountID, a.Revision+1, b.Field, b.Amount.Knowledge(), value, a.Asset, b.Amount.Reason(), b.Coverage.State(), reasons, b.Freshness, at, ns)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, "UPDATE want_keep.accounts SET revision=revision+1 WHERE household_id=$1 AND id=$2", scope.principal.HouseholdID(), b.AccountID)
	return err
}
func (s *Store) Balance(ctx context.Context, p household.Principal, id, field string) (account.Balance, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return account.Balance{}, err
	}
	b := account.Balance{AccountID: id, Field: field}
	var knowledge, reason, coverage, freshness, asset string
	var amount *string
	var reasons []string
	var at time.Time
	var ns int16
	err = q.QueryRow(ctx, `SELECT knowledge,amount::text,asset,reason,coverage,coverage_reasons,freshness,observed_at,observed_ns FROM want_keep.balance_snapshots WHERE household_id=$1 AND account_id=$2 AND field=$3 ORDER BY revision DESC LIMIT 1`, p.HouseholdID(), id, field).Scan(&knowledge, &amount, &asset, &reason, &coverage, &reasons, &freshness, &at, &ns)
	if errors.Is(err, pgx.ErrNoRows) {
		return b, ErrNotFound
	}
	if err != nil {
		return b, err
	}
	if amount != nil {
		m, e := money.NewMoney(*amount, money.Asset(asset))
		if e != nil {
			return b, e
		}
		b.Amount, err = reporting.KnownAmount(m)
	} else {
		b.Amount, err = reporting.MissingAmount(reporting.Knowledge(knowledge), reason)
	}
	if err != nil {
		return b, err
	}
	b.Coverage, err = reporting.NewCoverage(reporting.CoverageState(coverage), reasons)
	if err != nil {
		return b, err
	}
	b.Freshness, err = reporting.ParseFreshness(freshness)
	if err != nil {
		return b, err
	}
	b.ObservedAt, err = restoreInstant(at, ns)
	if err != nil {
		return b, err
	}
	return b, b.Validate()
}
