package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	application "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func (s *Store) RequireLedgerPayer(ctx context.Context, p household.Principal, id household.MembershipID) error {
	q, err := s.reader(ctx, p)
	if err != nil {
		return err
	}
	var active bool
	err = q.QueryRow(ctx, `SELECT active FROM want_keep.memberships WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && !active {
		return household.ErrForbidden
	}
	return err
}

func (s *Store) TransactionReferences(ctx context.Context, p household.Principal, f application.Filter, c application.Cursor, limit int) ([]ledger.Revision, *application.Cursor, error) {
	if err := f.Validate(); err != nil {
		return nil, nil, err
	}
	if limit < 1 || limit > 100 {
		return nil, nil, ledger.ErrInvalidRevision
	}
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, nil, err
	}
	var at, ns any
	if c.ID != "" {
		if c.At.String() == "" {
			return nil, nil, ledger.ErrInvalidRevision
		}
		at, ns = splitInstant(c.At)
	}
	rows, err := q.Query(ctx, `SELECT r.operation_id,r.revision FROM want_keep.operations o JOIN want_keep.operation_revisions r ON (r.household_id,r.operation_id,r.revision)=(o.household_id,o.id,o.revision) LEFT JOIN want_keep.transaction_details d ON (d.household_id,d.operation_id,d.revision)=(r.household_id,r.operation_id,r.revision)
 WHERE o.household_id=$1 AND ($2='' OR EXISTS(SELECT 1 FROM want_keep.postings p WHERE (p.household_id,p.operation_id,p.revision)=(r.household_id,r.operation_id,r.revision) AND p.account_id=NULLIF($2,'')::uuid))
 AND ($3='' OR r.cash_date>=NULLIF($3,'')::date) AND ($4='' OR r.cash_date<=NULLIF($4,'')::date)
 AND ($5='' OR r.economic_type=$5) AND ($6='' OR r.state=$6)
 AND ($7='' OR strpos(lower(COALESCE(d.merchant,'') || ' ' || COALESCE(d.note,'')),lower($7))>0)
 AND ($8='' OR (r.occurred_at,r.occurred_ns,r.operation_id)<($9::timestamptz,$10::smallint,NULLIF($8,'')::uuid))
 ORDER BY r.occurred_at DESC,r.occurred_ns DESC,r.operation_id DESC LIMIT $11`, p.HouseholdID(), f.AccountID, f.From.String(), f.To.String(), f.Type, f.State, f.Search, c.ID, at, ns, limit+1)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	type ref struct {
		id       string
		revision uint64
	}
	refs := []ref{}
	for rows.Next() {
		var v ref
		if err = rows.Scan(&v.id, &v.revision); err != nil {
			return nil, nil, err
		}
		refs = append(refs, v)
	}
	if err = rows.Err(); err != nil {
		return nil, nil, err
	}
	rows.Close()
	more := len(refs) > limit
	if more {
		refs = refs[:limit]
	}
	result := make([]ledger.Revision, 0, len(refs))
	for _, ref := range refs {
		r, err := s.LedgerRevision(ctx, p, ref.id, ref.revision)
		if err != nil {
			return nil, nil, err
		}
		result = append(result, r)
	}
	var next *application.Cursor
	if more {
		last := result[len(result)-1]
		next = &application.Cursor{At: last.OccurredAt, ID: last.OperationID}
	}
	return result, next, nil
}

func (s *Store) TransactionSources(ctx context.Context, p household.Principal, id string, revision uint64) ([]application.SourceReference, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT DISTINCT s.provider,a.stable_id,s.product,s.log,s.provider_record_id,link.source_revision,pr.connection_id FROM want_keep.ledger_revision_sources link JOIN want_keep.source_records s ON (s.household_id,s.id)=(link.household_id,link.source_id) JOIN want_keep.external_accounts a ON (a.household_id,a.id)=(s.household_id,s.external_account_id) JOIN want_keep.source_provenance pr ON (pr.household_id,pr.source_id,pr.revision)=(link.household_id,link.source_id,link.source_revision) WHERE link.household_id=$1 AND link.operation_id=$2 AND link.revision=$3 ORDER BY s.provider,a.stable_id,s.product,s.log,s.provider_record_id,link.source_revision,pr.connection_id`, p.HouseholdID(), id, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []application.SourceReference{}
	for rows.Next() {
		v := application.SourceReference{Key: ledger.SourceKey{HouseholdID: p.HouseholdID()}}
		if err = rows.Scan(&v.Key.Provider, &v.Key.ExternalAccountID, &v.Key.Product, &v.Key.Log, &v.Key.RecordID, &v.Revision, &v.ConnectionID); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (s *Store) TransactionCoverage(ctx context.Context, p household.Principal) (reporting.Coverage, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return reporting.Coverage{}, err
	}
	var sources bool
	err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.connections WHERE household_id=$1)`, p.HouseholdID()).Scan(&sources)
	if err != nil {
		return reporting.Coverage{}, err
	}
	if sources {
		return reporting.NewCoverage(reporting.Partial, []string{"source_history_not_reconciled"})
	}
	return reporting.NewCoverage(reporting.Complete, nil)
}
