package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

// Authentication is serialized for the private two-member deployment. Financial transactions do not acquire this lock.
func (s *Store) WithinIdentity(ctx context.Context, fn func(context.Context) error) error {
	return s.transact(ctx, func(ctx context.Context, scope *transactionScope) error {
		if scope.householdLocked || scope.admissionKey != "" || scope.invitationHouseholdID != "" {
			return ErrTransactionRequired
		}
		if _, err := scope.tx.Exec(ctx, "SELECT singleton FROM want_keep.identity_bootstrap WHERE singleton=true FOR UPDATE"); err != nil {
			return identity.ErrUnavailable
		}
		scope.identityLocked = true
		return fn(ctx)
	})
}
func (s *Store) IdentityBootstrap(ctx context.Context) (identity.BootstrapState, error) {
	var b identity.BootstrapState
	scope, err := s.scope(ctx)
	if err != nil {
		return b, err
	}
	// NULL avoids mapping PostgreSQL infinity into time.Time before a token has been issued.
	var expires *time.Time
	err = scope.tx.QueryRow(ctx, "SELECT token_hash,CASE WHEN isfinite(expires_at) THEN expires_at END,initialized FROM want_keep.identity_bootstrap WHERE singleton=true").Scan(&b.TokenHash, &expires, &b.Initialized)
	if expires != nil {
		b.ExpiresAt = *expires
	}
	return b, s.identityError(err)
}
func (s *Store) SaveIdentityBootstrap(ctx context.Context, b identity.BootstrapState) error {
	if !b.Initialized {
		return identity.ErrBootstrap
	}
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, "UPDATE want_keep.identity_bootstrap SET initialized=$1 WHERE singleton=true", b.Initialized)
	return s.identityError(err)
}
func (s *Store) IssueIdentityBootstrap(ctx context.Context, token identity.Token, now time.Time) error {
	if !token.Valid(32) {
		return identity.ErrBootstrap
	}
	return s.WithinIdentity(ctx, func(ctx context.Context) error {
		b, err := s.IdentityBootstrap(ctx)
		if err != nil {
			return err
		}
		if b.Initialized {
			return identity.ErrBootstrap
		}
		scope, _ := s.scope(ctx)
		var exists bool
		if err = scope.tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM want_keep.households)").Scan(&exists); err != nil {
			return s.identityError(err)
		}
		if exists {
			return identity.ErrBootstrap
		}
		_, err = scope.tx.Exec(ctx, "UPDATE want_keep.identity_bootstrap SET token_hash=$1,expires_at=$2 WHERE singleton=true", token.Hash(), now.Add(identity.BootstrapLifetime))
		return s.identityError(err)
	})
}
func (s *Store) IdentityProfile(ctx context.Context, id household.UserID) (identity.Profile, error) {
	p := identity.Profile{UserID: id}
	scope, err := s.scope(ctx)
	if err != nil {
		return p, err
	}
	err = scope.tx.QueryRow(ctx, `SELECT p.household_id,p.membership_id,p.handle,p.generation,p.locale,p.reporting_asset,u.name
 FROM want_keep.identity_profiles p JOIN want_keep.users u ON u.id=p.user_id WHERE p.user_id=$1 FOR UPDATE OF p`, id).
		Scan(&p.HouseholdID, &p.MembershipID, &p.Handle, &p.Generation, &p.Locale, &p.ReportingAsset, &p.Name)
	return p, s.identityError(err)
}
func (s *Store) CreateIdentityProfile(ctx context.Context, p identity.Profile) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	tag, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.identity_profiles(user_id,household_id,membership_id,handle,generation,locale,reporting_asset)
 SELECT $1,$2,$3,$4,$5,$6,$7 WHERE EXISTS(SELECT 1 FROM want_keep.memberships WHERE household_id=$2 AND id=$3 AND user_id=$1 AND active=true)`, p.UserID, p.HouseholdID, p.MembershipID, p.Handle, p.Generation, p.Locale, p.ReportingAsset)
	if err != nil {
		return s.identityError(err)
	}
	if tag.RowsAffected() != 1 {
		return identity.ErrUnauthorized
	}
	return nil
}
func (s *Store) SaveIdentityGeneration(ctx context.Context, p identity.Profile) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, "UPDATE want_keep.identity_profiles SET generation=$2 WHERE user_id=$1", p.UserID, p.Generation)
	return s.identityError(err)
}
func (s *Store) IdentityMembership(ctx context.Context, p identity.Profile) (household.Principal, error) {
	scope, err := s.scope(ctx)
	if err != nil {
		return household.Principal{}, err
	}
	m := household.Membership{ID: p.MembershipID, UserID: p.UserID, HouseholdID: p.HouseholdID}
	err = scope.tx.QueryRow(ctx, "SELECT active FROM want_keep.memberships WHERE household_id=$1 AND id=$2 AND user_id=$3 FOR SHARE", m.HouseholdID, m.ID, m.UserID).Scan(&m.Active)
	if err != nil {
		return household.Principal{}, s.identityError(err)
	}
	principal, err := m.Principal()
	if err != nil {
		return principal, identity.ErrUnauthorized
	}
	return principal, nil
}
func (s *Store) IdentityAudit(ctx context.Context, user household.UserID, event, id string, now time.Time) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, "INSERT INTO want_keep.identity_audit(user_id,event,resource_id,occurred_at) VALUES(NULLIF($1,'')::uuid,$2,NULLIF($3,'')::uuid,$4)", string(user), event, id, now)
	return s.identityError(err)
}
func (s *Store) IdentityRateLimit(ctx context.Context, key string, maximum int, now time.Time) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	var count int
	err = scope.tx.QueryRow(ctx, `INSERT INTO want_keep.identity_rate_limits(scope_hash,window_start,attempts) VALUES($1,$2,1)
 ON CONFLICT(scope_hash) DO UPDATE SET window_start=EXCLUDED.window_start,
 attempts=CASE WHEN identity_rate_limits.window_start=EXCLUDED.window_start THEN identity_rate_limits.attempts+1 ELSE 1 END RETURNING attempts`, identity.Token(key).Hash(), now.Truncate(15*time.Minute)).Scan(&count)
	if err != nil {
		return s.identityError(err)
	}
	if count > maximum {
		return identity.ErrRateLimited
	}
	return nil
}
func (s *Store) identityError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.ErrUnauthorized
	}
	return identity.ErrUnavailable
}
