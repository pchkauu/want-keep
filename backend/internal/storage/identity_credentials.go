package storage

import (
	"context"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

const identityCredentialColumns = `id,user_id,name,rp_id,raw_id,public_key,aaguid,attestation_object,attestation_client_data,attestation_client_hash,attestation_format,attestation_type,attestation_algorithm,transports,sign_count,backup_eligible,backup_state,user_verified,created_at,revoked`

type identityCredentialRow struct{ credential identity.Credential }

func (r *identityCredentialRow) scan(row interface{ Scan(...any) error }) error {
	c := &r.credential
	return row.Scan(&c.ID, &c.UserID, &c.Name, &c.RPID, &c.RawID, &c.PublicKey, &c.AAGUID, &c.AttestationObject, &c.AttestationClientData, &c.AttestationClientHash, &c.AttestationFormat, &c.AttestationType, &c.AttestationAlgorithm, &c.Transports, &c.SignCount, &c.BackupEligible, &c.BackupState, &c.UserVerified, &c.CreatedAt, &c.Revoked)
}
func (s *Store) IdentityCredential(ctx context.Context, rawID []byte) (identity.Credential, error) {
	var r identityCredentialRow
	scope, err := s.scope(ctx)
	if err != nil {
		return r.credential, err
	}
	err = r.scan(scope.tx.QueryRow(ctx, "SELECT "+identityCredentialColumns+" FROM want_keep.identity_credentials WHERE raw_id=$1", rawID))
	return r.credential, s.identityError(err)
}
func (s *Store) IdentityCredentials(ctx context.Context, user household.UserID) ([]identity.Credential, error) {
	scope, err := s.scope(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := scope.tx.Query(ctx, "SELECT "+identityCredentialColumns+" FROM want_keep.identity_credentials WHERE user_id=$1 ORDER BY id", user)
	if err != nil {
		return nil, s.identityError(err)
	}
	defer rows.Close()
	result := []identity.Credential{}
	for rows.Next() {
		var r identityCredentialRow
		if err = r.scan(rows); err != nil {
			return nil, s.identityError(err)
		}
		result = append(result, r.credential)
	}
	return result, s.identityError(rows.Err())
}
func (s *Store) SaveIdentityCredential(ctx context.Context, c identity.Credential) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.identity_credentials(`+identityCredentialColumns+`) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
 ON CONFLICT(id) DO UPDATE SET sign_count=EXCLUDED.sign_count,backup_state=EXCLUDED.backup_state,user_verified=EXCLUDED.user_verified,revoked=EXCLUDED.revoked
 WHERE identity_credentials.user_id=EXCLUDED.user_id`, c.ID, c.UserID, c.Name, c.RPID, c.RawID, c.PublicKey, c.AAGUID, c.AttestationObject, c.AttestationClientData, c.AttestationClientHash, c.AttestationFormat, c.AttestationType, c.AttestationAlgorithm, c.Transports, c.SignCount, c.BackupEligible, c.BackupState, c.UserVerified, c.CreatedAt, c.Revoked)
	return s.identityError(err)
}
