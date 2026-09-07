package storage

import (
	"context"
	"crypto/sha256"
	"errors"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func (s *Store) ResolveExternalAccount(ctx context.Context, provider, stableID string, owner household.UserID) (string, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return "", err
	}
	if len(stableID) < 1 || len(stableID) > 2000 {
		return "", ledger.ErrInvalidSource
	}
	switch provider {
	case "alfa", "raiffeisen", "ozon", "bybit", "aifory", "emcd":
	default:
		return "", ledger.ErrInvalidSource
	}
	digest := sha256.Sum256([]byte(stableID))
	var id, existing string
	var existingOwner household.UserID
	err = scope.tx.QueryRow(ctx, `SELECT id,stable_id,external_owner_id FROM want_keep.external_accounts WHERE household_id=$1 AND provider=$2 AND identity_digest=$3`, scope.principal.HouseholdID(), provider, digest[:]).Scan(&id, &existing, &existingOwner)
	if err == nil {
		if existing != stableID || existingOwner != owner {
			return "", ledger.ErrSourceAmbiguous
		}
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	id = newID()
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.external_accounts(household_id,id,provider,stable_id,identity_digest,external_owner_id) VALUES($1,$2,$3,$4,$5,$6)`, scope.principal.HouseholdID(), id, provider, stableID, digest[:], owner)
	return id, err
}
func (s *Store) Source(ctx context.Context, p household.Principal, key ledger.SourceKey) (ledger.SourceRecord, bool, error) {
	if err := key.Validate(); err != nil {
		return ledger.SourceRecord{}, false, err
	}
	if err := p.RequireHousehold(key.HouseholdID); err != nil {
		return ledger.SourceRecord{}, false, err
	}
	q, err := s.reader(ctx, p)
	if err != nil {
		return ledger.SourceRecord{}, false, err
	}
	digest := key.Digest()
	r := ledger.SourceRecord{Key: ledger.SourceKey{HouseholdID: p.HouseholdID()}}
	err = q.QueryRow(ctx, `SELECT s.id,s.provider,a.stable_id,s.product,s.log,s.provider_record_id,s.revision,s.ambiguous,COALESCE(s.operation_id::text,''),v.payload_hash FROM want_keep.source_records s JOIN want_keep.external_accounts a ON (a.household_id,a.id)=(s.household_id,s.external_account_id) JOIN want_keep.source_revisions v ON (v.household_id,v.source_id,v.revision)=(s.household_id,s.id,s.revision) WHERE s.household_id=$1 AND s.identity_digest=$2`, p.HouseholdID(), digest[:]).Scan(&r.ID, &r.Key.Provider, &r.Key.ExternalAccountID, &r.Key.Product, &r.Key.Log, &r.Key.RecordID, &r.Revision, &r.Ambiguous, &r.OperationID, &r.PayloadHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, false, nil
	}
	if err != nil {
		return r, false, err
	}
	if r.Key != key {
		return r, true, ledger.ErrSourceAmbiguous
	}
	return r, true, nil
}
func (s *Store) SaveSource(ctx context.Context, r ledger.SourceRecord, input ledger.SourceInput) (ledger.SourceRecord, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return r, err
	}
	if err = scope.principal.RequireHousehold(r.Key.HouseholdID); err != nil {
		return r, err
	}
	if scope.syncJobID != input.JobID || scope.syncConnectionID != input.ConnectionID {
		return r, ErrTransactionRequired
	}
	if err = input.Validate(); err != nil {
		return r, err
	}
	c, err := s.Connection(ctx, scope.principal, input.ConnectionID)
	if err != nil {
		return r, err
	}
	if c.Provider != r.Key.Provider {
		return r, ledger.ErrInvalidSource
	}
	external, err := s.ResolveExternalAccount(ctx, c.Provider, r.Key.ExternalAccountID, c.Owner)
	if err != nil {
		return r, err
	}
	digest := r.Key.Digest()
	if r.ID == "" {
		r.ID = newID()
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.source_records(household_id,id,provider,external_account_id,product,log,provider_record_id,identity_digest,revision,ambiguous,operation_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,'')::uuid)`, r.Key.HouseholdID, r.ID, r.Key.Provider, external, r.Key.Product, r.Key.Log, r.Key.RecordID, digest[:], r.Revision, r.Ambiguous, r.OperationID)
	} else {
		var tag int64
		result, e := scope.tx.Exec(ctx, `UPDATE want_keep.source_records SET revision=$3,ambiguous=$4,operation_id=COALESCE(operation_id,NULLIF($6,'')::uuid) WHERE household_id=$1 AND id=$2 AND revision=$5`, r.Key.HouseholdID, r.ID, r.Revision, r.Ambiguous, r.Revision-1, r.OperationID)
		err = e
		tag = result.RowsAffected()
		if err == nil && tag != 1 {
			err = ledger.ErrInvalidSource
		}
	}
	if err != nil {
		return r, err
	}
	at, ns := splitInstant(input.FetchedAt)
	classification := input.Classification
	if r.Ambiguous {
		classification = "ambiguous"
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.source_revisions(household_id,source_id,revision,payload_hash,evidence_ref,classification,fetched_at,fetched_ns) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, r.Key.HouseholdID, r.ID, r.Revision, input.PayloadHash, input.EvidenceRef, classification, at, ns)
	return r, err
}
func (s *Store) RecordProvenance(ctx context.Context, r ledger.SourceRecord, input ledger.SourceInput) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if err = scope.principal.RequireHousehold(r.Key.HouseholdID); err != nil {
		return err
	}
	if scope.syncJobID != input.JobID || scope.syncConnectionID != input.ConnectionID {
		return ErrTransactionRequired
	}
	c, err := s.Connection(ctx, scope.principal, input.ConnectionID)
	if err != nil {
		return err
	}
	if c.Provider != r.Key.Provider {
		return ledger.ErrInvalidSource
	}
	if _, err = s.ResolveExternalAccount(ctx, c.Provider, r.Key.ExternalAccountID, c.Owner); err != nil {
		return err
	}
	j, err := s.Job(ctx, scope.principal, input.JobID)
	if err != nil {
		return err
	}
	if j.ConnectionID != input.ConnectionID {
		return ledger.ErrInvalidSource
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.source_provenance(household_id,id,source_id,revision,connection_id,job_id,evidence_ref) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`, r.Key.HouseholdID, newID(), r.ID, r.Revision, input.ConnectionID, input.JobID, input.EvidenceRef)
	return err
}

func (s *Store) RecordSourceAmbiguity(ctx context.Context, input ledger.SourceInput) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.syncJobID != input.JobID || scope.syncConnectionID != input.ConnectionID {
		return ErrTransactionRequired
	}
	j, err := s.Job(ctx, scope.principal, input.JobID)
	if err != nil {
		return err
	}
	if err = s.Quarantine(ctx, j, input.EvidenceRef, "source_ambiguous"); err != nil {
		return err
	}
	return s.EmitEvent(ctx, "job", j.ID, 1, "source.ambiguous")
}

func (s *Store) RecordUnresolvedTransaction(ctx context.Context, input ledger.SourceInput) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.syncJobID != input.JobID || scope.syncConnectionID != input.ConnectionID {
		return ErrTransactionRequired
	}
	j, err := s.Job(ctx, scope.principal, input.JobID)
	if err != nil {
		return err
	}
	return s.Quarantine(ctx, j, input.EvidenceRef, "transaction_unresolved")
}
