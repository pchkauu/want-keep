package storage

import (
	"context"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

const identitySessionColumns = `id,user_id,token_hash,credential_id,name,created_at,authenticated_at,last_activity_at,revoked`

type identitySessionRow struct{ session identity.Session }

func (r *identitySessionRow) scan(row interface{ Scan(...any) error }) error {
	x := &r.session
	return row.Scan(&x.ID, &x.UserID, &x.TokenHash, &x.CredentialID, &x.Name, &x.CreatedAt, &x.AuthenticatedAt, &x.LastActivityAt, &x.Revoked)
}
func (s *Store) IdentitySession(ctx context.Context, hash string) (identity.Session, error) {
	var r identitySessionRow
	scope, err := s.scope(ctx)
	if err != nil {
		return r.session, err
	}
	err = r.scan(scope.tx.QueryRow(ctx, "SELECT "+identitySessionColumns+" FROM want_keep.identity_sessions WHERE token_hash=$1", hash))
	return r.session, s.identityError(err)
}
func (s *Store) IdentitySessions(ctx context.Context, user household.UserID) ([]identity.Session, error) {
	scope, err := s.scope(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := scope.tx.Query(ctx, "SELECT "+identitySessionColumns+" FROM want_keep.identity_sessions WHERE user_id=$1 ORDER BY id", user)
	if err != nil {
		return nil, s.identityError(err)
	}
	defer rows.Close()
	result := []identity.Session{}
	for rows.Next() {
		var r identitySessionRow
		if err = r.scan(rows); err != nil {
			return nil, s.identityError(err)
		}
		result = append(result, r.session)
	}
	return result, s.identityError(rows.Err())
}
func (s *Store) SaveIdentitySession(ctx context.Context, x identity.Session) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.identity_sessions(`+identitySessionColumns+`) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
 ON CONFLICT(id) DO UPDATE SET authenticated_at=EXCLUDED.authenticated_at,last_activity_at=EXCLUDED.last_activity_at,revoked=EXCLUDED.revoked
 WHERE identity_sessions.user_id=EXCLUDED.user_id`, x.ID, x.UserID, x.TokenHash, x.CredentialID, x.Name, x.CreatedAt, x.AuthenticatedAt, x.LastActivityAt, x.Revoked)
	return s.identityError(err)
}
func (s *Store) RevokeIdentitySubscriptions(ctx context.Context, user household.UserID, session string) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, "UPDATE want_keep.identity_subscription_bindings SET revoked=true WHERE user_id=$1 AND ($2='' OR session_id::text=$2)", user, session)
	return s.identityError(err)
}

// BindIdentitySubscription is the storage handoff to the future notification service; revoked sessions cannot be rebound.
func (s *Store) BindIdentitySubscription(ctx context.Context, user household.UserID, sessionID, id string) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	tag, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.identity_subscription_bindings(id,user_id,session_id)
 SELECT $1,$2,$3 WHERE EXISTS(SELECT 1 FROM want_keep.identity_sessions WHERE id=$3 AND user_id=$2 AND revoked=false)`, id, user, sessionID)
	if err != nil {
		return s.identityError(err)
	}
	if tag.RowsAffected() != 1 {
		return identity.ErrUnauthorized
	}
	return nil
}
