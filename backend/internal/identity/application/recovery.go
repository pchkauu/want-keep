package application

import (
	"context"

	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

type RecoveryAttempt struct {
	Attempt identity.Attempt
	Token   identity.Token
}

func (s *Service) Recover(ctx context.Context, r RequestContext, code identity.Token) (RecoveryAttempt, error) {
	var result RecoveryAttempt
	if err := s.Limit(ctx, r); err != nil {
		return result, err
	}
	if !code.Valid(16) {
		return result, identity.ErrUnauthorized
	}
	err := s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		user, err := s.repo.ConsumeRecoveryCode(ctx, code.Hash())
		if err != nil {
			return err
		}
		p, err := s.repo.IdentityProfile(ctx, user)
		if err != nil {
			return err
		}
		if _, err = s.repo.IdentityMembership(ctx, p); err != nil {
			return err
		}
		if err = s.repo.IdentityRateLimit(ctx, "user:"+string(user), 30, s.now()); err != nil {
			return err
		}
		a, err := s.attempt(identity.RecoveryGrant, r)
		if err != nil {
			return err
		}
		token, err := identity.NewToken(32)
		if err != nil {
			return err
		}
		a.TokenHash, a.UserID, a.Generation = token.Hash(), p.UserID, p.Generation
		a.ExpiresAt = a.CreatedAt.Add(identity.RecoveryLifetime)
		if err = s.repo.SaveIdentityAttempt(ctx, a); err != nil {
			return err
		}
		result = RecoveryAttempt{Attempt: a, Token: token}
		return s.repo.IdentityAudit(ctx, user, "recovery_started", a.ID, s.now())
	})
	return result, err
}
func (s *Service) replaceCodes(ctx context.Context, p identity.Profile) ([]string, error) {
	codes, hashes := make([]string, 10), make([]string, 10)
	for i := range codes {
		token, err := identity.NewToken(16)
		if err != nil {
			return nil, err
		}
		codes[i], hashes[i] = string(token), token.Hash()
	}
	if err := s.repo.ReplaceRecoveryCodes(ctx, p.UserID, hashes); err != nil {
		return nil, err
	}
	return codes, nil
}
func (s *Service) ReplaceCodes(ctx context.Context, token identity.Token) ([]string, error) {
	var codes []string
	err := s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		access, err := s.access(ctx, token)
		if err != nil {
			return err
		}
		if err = access.Session.RequireFresh(s.now()); err != nil {
			return err
		}
		codes, err = s.replaceCodes(ctx, access.Profile)
		if err != nil {
			return err
		}
		return s.repo.IdentityAudit(ctx, access.Profile.UserID, "recovery_codes_replaced", access.Session.ID, s.now())
	})
	return codes, err
}
func (s *Service) revokeAll(ctx context.Context, p identity.Profile) error {
	keys, err := s.repo.IdentityCredentials(ctx, p.UserID)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if !key.Revoked {
			key.Revoked = true
			if err = s.repo.SaveIdentityCredential(ctx, key); err != nil {
				return err
			}
		}
	}
	sessions, err := s.repo.IdentitySessions(ctx, p.UserID)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if !session.Revoked {
			if err = s.revokeSession(ctx, session); err != nil {
				return err
			}
		}
	}
	return s.repo.RevokeIdentitySubscriptions(ctx, p.UserID, "")
}
