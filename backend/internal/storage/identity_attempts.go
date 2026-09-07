package storage

import (
	"context"
	"encoding/json"
	"time"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

type identitySetupRow struct {
	Name, HouseholdName, Timezone, Locale, ReportingAsset, UserID, HouseholdID, MembershipID string
	Handle                                                                                   []byte
}

func (r identitySetupRow) domain() *identity.Setup {
	return &identity.Setup{Name: r.Name, HouseholdName: r.HouseholdName, Timezone: r.Timezone, Locale: r.Locale, ReportingAsset: r.ReportingAsset, UserID: household.UserID(r.UserID), HouseholdID: household.HouseholdID(r.HouseholdID), MembershipID: household.MembershipID(r.MembershipID), Handle: r.Handle}
}

const identityAttemptColumns = `id,browser_hash,token_hash,session_hash,challenge,rp_id,origin,grant_id,purpose,COALESCE(user_id::text,''),generation,created_at,expires_at,consumed,setup,COALESCE(invitation_revision,0)`

type identityAttemptRow struct{ attempt identity.Attempt }

func (r *identityAttemptRow) scan(row interface{ Scan(...any) error }) error {
	a := &r.attempt
	var setup []byte
	err := row.Scan(&a.ID, &a.BrowserHash, &a.TokenHash, &a.SessionHash, &a.Challenge, &a.RPID, &a.Origin, &a.GrantID, &a.Purpose, &a.UserID, &a.Generation, &a.CreatedAt, &a.ExpiresAt, &a.Consumed, &setup, &a.InvitationRevision)
	if err != nil {
		return err
	}
	if len(setup) > 0 {
		var x identitySetupRow
		if err = json.Unmarshal(setup, &x); err != nil {
			return err
		}
		a.Setup = x.domain()
	}
	return nil
}
func (s *Store) IdentityAttempt(ctx context.Context, id string) (identity.Attempt, error) {
	var r identityAttemptRow
	scope, err := s.scope(ctx)
	if err != nil {
		return r.attempt, err
	}
	err = r.scan(scope.tx.QueryRow(ctx, "SELECT "+identityAttemptColumns+" FROM want_keep.identity_attempts WHERE id=$1", id))
	return r.attempt, s.identityError(err)
}
func (s *Store) IdentityGrant(ctx context.Context, hash string) (identity.Attempt, error) {
	var r identityAttemptRow
	scope, err := s.scope(ctx)
	if err != nil {
		return r.attempt, err
	}
	err = r.scan(scope.tx.QueryRow(ctx, "SELECT "+identityAttemptColumns+" FROM want_keep.identity_attempts WHERE token_hash=$1 AND purpose='recovery_grant'", hash))
	return r.attempt, s.identityError(err)
}
func (s *Store) SaveIdentityAttempt(ctx context.Context, a identity.Attempt) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	var setup []byte
	if x := a.Setup; x != nil {
		setup, err = json.Marshal(identitySetupRow{Name: x.Name, HouseholdName: x.HouseholdName, Timezone: x.Timezone, Locale: x.Locale, ReportingAsset: x.ReportingAsset, UserID: string(x.UserID), HouseholdID: string(x.HouseholdID), MembershipID: string(x.MembershipID), Handle: x.Handle})
		if err != nil {
			return s.identityError(err)
		}
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.identity_attempts(id,browser_hash,token_hash,session_hash,challenge,rp_id,origin,grant_id,purpose,user_id,generation,created_at,expires_at,consumed,setup,invitation_revision)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,'')::uuid,$11,$12,$13,$14,$15,NULLIF($16,0)) ON CONFLICT(id) DO UPDATE SET consumed=EXCLUDED.consumed`, a.ID, a.BrowserHash, a.TokenHash, a.SessionHash, a.Challenge, a.RPID, a.Origin, a.GrantID, a.Purpose, string(a.UserID), a.Generation, a.CreatedAt, a.ExpiresAt, a.Consumed, setup, a.InvitationRevision)
	return s.identityError(err)
}
func (s *Store) ConsumeRecoveryCode(ctx context.Context, hash string) (household.UserID, error) {
	scope, err := s.scope(ctx)
	if err != nil {
		return "", err
	}
	var user household.UserID
	err = scope.tx.QueryRow(ctx, "UPDATE want_keep.identity_recovery_codes SET consumed=true WHERE code_hash=$1 AND consumed=false RETURNING user_id", hash).Scan(&user)
	return user, s.identityError(err)
}
func (s *Store) ReplaceRecoveryCodes(ctx context.Context, user household.UserID, hashes []string) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if _, err = scope.tx.Exec(ctx, "UPDATE want_keep.identity_recovery_codes SET consumed=true WHERE user_id=$1", user); err != nil {
		return s.identityError(err)
	}
	for _, hash := range hashes {
		if _, err = scope.tx.Exec(ctx, "INSERT INTO want_keep.identity_recovery_codes(code_hash,user_id) VALUES($1,$2)", hash, user); err != nil {
			return s.identityError(err)
		}
	}
	return nil
}
func (s *Store) CleanupIdentity(ctx context.Context, now time.Time, limit int) error {
	if limit < 1 || limit > 1000 {
		return identity.ErrAttempt
	}
	// Maintenance runs independently and only removes expired transient records, never access/audit history.
	_, err := s.pool.Exec(ctx, `DELETE FROM want_keep.identity_attempts WHERE id IN
 (SELECT id FROM want_keep.identity_attempts WHERE expires_at<=$1 ORDER BY expires_at,id LIMIT $2)`, now, limit)
	if err != nil {
		return s.identityError(err)
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM want_keep.identity_rate_limits WHERE window_start<$1 AND scope_hash IN
 (SELECT scope_hash FROM want_keep.identity_rate_limits WHERE window_start<$1 ORDER BY window_start,scope_hash LIMIT $2)`, now.Add(-15*time.Minute), limit)
	return s.identityError(err)
}
