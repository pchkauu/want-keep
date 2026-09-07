package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func (s *Store) RecordObservation(ctx context.Context, p household.Principal, o account.Observation) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.principal != p || scope.syncJobID == "" || scope.syncJobID != o.JobID || scope.syncConnectionID != o.ConnectionID {
		return ErrTransactionRequired
	}
	a, err := s.Account(ctx, p, o.AccountID)
	if err != nil {
		return err
	}
	if err = o.Validate(a.Asset); err != nil {
		return err
	}
	c, err := s.Connection(ctx, p, o.ConnectionID)
	if err != nil {
		return err
	}
	var provider, owner string
	if err = scope.tx.QueryRow(ctx, `SELECT provider,external_owner_id FROM want_keep.external_accounts WHERE household_id=$1 AND id=$2`, p.HouseholdID(), a.ExternalAccountID).Scan(&provider, &owner); err != nil {
		return err
	}
	if provider != c.Provider || owner != string(c.Owner) {
		return household.ErrForbidden
	}
	existing, found, err := s.observation(ctx, p, o.ID)
	if err != nil {
		return err
	}
	if found {
		if existing.AccountID != o.AccountID || existing.AsOf.String() != o.AsOf.String() || !existing.Equivalent(o) {
			return ledger.ErrSourceAmbiguous
		}
		return nil
	}
	if a.Revision >= command.MaxRevision {
		return command.ErrVersionConflict
	}
	asof, asns := splitInstant(o.AsOf)
	fetched, fetchns := splitInstant(o.FetchedAt)
	reasons := o.Coverage.Reasons()
	if reasons == nil {
		reasons = []string{}
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.account_observations(household_id,id,account_id,connection_id,job_id,evidence_ref,as_of,as_of_ns,fetched_at,fetched_ns,own_available,coverage,coverage_reasons,freshness) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, p.HouseholdID(), o.ID, o.AccountID, o.ConnectionID, o.JobID, o.EvidenceRef, asof, asns, fetched, fetchns, o.OwnAvailable, o.Coverage.State(), reasons, o.Freshness)
	if err != nil {
		return err
	}
	values := append(o.Amounts.Fields(), o.CreditLimit)
	for i, field := range []string{"owned", "available", "locked", "debt", "credit_limit"} {
		v := values[i]
		var amount any
		if m, known := v.Value(); known {
			amount = m.Amount()
		}
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.observation_amounts(household_id,observation_id,account_id,field,knowledge,amount,asset,reason) VALUES($1,$2,$3,$4,$5,$6::numeric,$7,$8)`, p.HouseholdID(), o.ID, o.AccountID, field, v.Knowledge(), amount, a.Asset, v.Reason())
		if err != nil {
			return err
		}
	}
	_, err = scope.tx.Exec(ctx, `UPDATE want_keep.accounts SET revision=revision+1 WHERE household_id=$1 AND id=$2`, p.HouseholdID(), o.AccountID)
	return err
}
func (s *Store) observation(ctx context.Context, p household.Principal, id string) (account.Observation, bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return account.Observation{}, false, err
	}
	o := account.Observation{ID: id}
	var asof, fetched time.Time
	var asns, fetchns int16
	var coverage, fresh string
	var reasons []string
	err = q.QueryRow(ctx, `SELECT account_id,connection_id,job_id,evidence_ref,as_of,as_of_ns,fetched_at,fetched_ns,own_available,coverage,coverage_reasons,freshness FROM want_keep.account_observations WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id).Scan(&o.AccountID, &o.ConnectionID, &o.JobID, &o.EvidenceRef, &asof, &asns, &fetched, &fetchns, &o.OwnAvailable, &coverage, &reasons, &fresh)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, false, nil
	}
	if err != nil {
		return o, false, err
	}
	o.AsOf, err = restoreInstant(asof, asns)
	if err != nil {
		return o, false, err
	}
	o.FetchedAt, err = restoreInstant(fetched, fetchns)
	if err != nil {
		return o, false, err
	}
	o.Coverage, err = reporting.NewCoverage(reporting.CoverageState(coverage), reasons)
	if err != nil {
		return o, false, err
	}
	o.Freshness, err = reporting.ParseFreshness(fresh)
	if err != nil {
		return o, false, err
	}
	rows, err := q.Query(ctx, `SELECT field,knowledge,amount::text,asset,reason FROM want_keep.observation_amounts WHERE household_id=$1 AND observation_id=$2`, p.HouseholdID(), id)
	if err != nil {
		return o, false, err
	}
	defer rows.Close()
	values := map[string]reporting.Amount{}
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
	}
	if err = rows.Err(); err != nil {
		return o, false, err
	}
	if len(values) != 5 {
		return o, false, account.ErrInvalidAccount
	}
	o.Amounts = account.Amounts{Owned: values["owned"], Available: values["available"], Locked: values["locked"], Debt: values["debt"]}
	o.CreditLimit = values["credit_limit"]
	return o, true, nil
}
func (s *Store) LatestObservation(ctx context.Context, p household.Principal, id string) (account.Observation, bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return account.Observation{}, false, err
	}
	var chosen string
	err = q.QueryRow(ctx, `SELECT id FROM want_keep.account_observations WHERE household_id=$1 AND account_id=$2 ORDER BY as_of DESC,as_of_ns DESC,fetched_at DESC,fetched_ns DESC,id LIMIT 1`, p.HouseholdID(), id).Scan(&chosen)
	if errors.Is(err, pgx.ErrNoRows) {
		return account.Observation{}, false, nil
	}
	if err != nil {
		return account.Observation{}, false, err
	}
	o, found, err := s.observation(ctx, p, chosen)
	if err != nil {
		return o, false, err
	}
	at, ns := splitInstant(o.AsOf)
	rows, err := q.Query(ctx, `SELECT id FROM want_keep.account_observations WHERE household_id=$1 AND account_id=$2 AND as_of=$3 AND as_of_ns=$4 AND id!=$5`, p.HouseholdID(), id, at, ns, chosen)
	if err != nil {
		return o, false, err
	}
	var ids []string
	for rows.Next() {
		var x string
		if err = rows.Scan(&x); err != nil {
			rows.Close()
			return o, false, err
		}
		ids = append(ids, x)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return o, false, err
	}
	for _, x := range ids {
		other, _, err := s.observation(ctx, p, x)
		if err != nil {
			return o, false, err
		}
		if !o.Equivalent(other) {
			o.Amounts = account.UnknownAmounts("source_ambiguous")
			o.OwnAvailable = false
			o.CreditLimit = account.UnknownAmounts("source_ambiguous").Owned
			o.Coverage, _ = reporting.NewCoverage(reporting.Partial, []string{"source_ambiguous"})
			break
		}
	}
	return o, found, nil
}
