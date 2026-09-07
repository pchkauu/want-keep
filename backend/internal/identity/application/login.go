package application

import (
	"context"
	"encoding/base64"

	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

func (s *Service) BeginLogin(ctx context.Context, r RequestContext, purpose identity.Purpose) (identity.Attempt, error) {
	var result identity.Attempt
	if purpose != identity.Login && purpose != identity.Reauthentication {
		return result, identity.ErrAttempt
	}
	if err := s.Limit(ctx, r); err != nil {
		return result, err
	}
	err := s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		a, err := s.attempt(purpose, r)
		if err != nil {
			return err
		}
		if purpose == identity.Reauthentication {
			access, err := s.access(ctx, r.Session)
			if err != nil {
				return err
			}
			a.UserID, a.Generation, a.SessionHash = access.Profile.UserID, access.Profile.Generation, r.Session.Hash()
		}
		result = a
		return s.repo.SaveIdentityAttempt(ctx, a)
	})
	return result, err
}
func (s *Service) CompleteLogin(ctx context.Context, r RequestContext, id string, input Assertion) (Access, error) {
	var result Access
	if err := s.Limit(ctx, r); err != nil {
		return result, err
	}
	if err := s.limitCredential(ctx, input.RawID); err != nil {
		return result, err
	}
	err := s.finish(ctx, r, id, func(ctx context.Context, a identity.Attempt) error {
		if a.Purpose != identity.Login && a.Purpose != identity.Reauthentication {
			return identity.ErrAttempt
		}
		raw, err := base64.RawURLEncoding.DecodeString(input.RawID)
		if err != nil {
			return identity.ErrAttempt
		}
		c, err := s.repo.IdentityCredential(ctx, raw)
		if err != nil {
			return err
		}
		if c.Revoked || c.RPID != s.rpID {
			return identity.ErrUnauthorized
		}
		p, err := s.repo.IdentityProfile(ctx, c.UserID)
		if err != nil {
			return err
		}
		if _, err = s.repo.IdentityMembership(ctx, p); err != nil {
			return err
		}
		if a.Purpose == identity.Reauthentication {
			access, err := s.access(ctx, r.Session)
			if err != nil {
				return err
			}
			if a.SessionHash != r.Session.Hash() || a.UserID != p.UserID || a.Generation != p.Generation || access.Profile.UserID != p.UserID {
				return identity.ErrUnauthorized
			}
		}
		updated, err := s.verifier.Authenticate(p, a, c, input)
		if err != nil {
			return err
		}
		if err = s.repo.SaveIdentityCredential(ctx, updated); err != nil {
			return err
		}
		if r.Session.Valid(32) {
			old, err := s.repo.IdentitySession(ctx, r.Session.Hash())
			if err == nil {
				if err = s.revokeSession(ctx, old); err != nil {
					return err
				}
			} else if err != identity.ErrUnauthorized {
				return err
			}
		}
		result, err = s.createSession(ctx, p, c.ID)
		if err != nil {
			return err
		}
		event := "login_completed"
		if a.Purpose == identity.Reauthentication {
			event = "reauthentication_completed"
		}
		return s.repo.IdentityAudit(ctx, p.UserID, event, result.Session.ID, s.now())
	})
	return result, err
}
func (s *Service) createSession(ctx context.Context, p identity.Profile, credentialID string) (Access, error) {
	token, err := identity.NewToken(32)
	if err != nil {
		return Access{}, err
	}
	id, err := s.newID()
	if err != nil {
		return Access{}, err
	}
	now := s.now()
	x := identity.Session{ID: id, TokenHash: token.Hash(), UserID: p.UserID, CredentialID: credentialID, Name: "Browser session", CreatedAt: now, AuthenticatedAt: now, LastActivityAt: now}
	if err = s.repo.SaveIdentitySession(ctx, x); err != nil {
		return Access{}, err
	}
	principal, err := s.repo.IdentityMembership(ctx, p)
	if err != nil {
		return Access{}, err
	}
	return Access{Profile: p, Principal: principal, Session: x, Token: token}, nil
}
func (s *Service) Activity(ctx context.Context, token identity.Token) error {
	return s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		access, err := s.access(ctx, token)
		if err != nil {
			return err
		}
		next, err := access.Session.Activity(s.now())
		if err != nil {
			return err
		}
		return s.repo.SaveIdentitySession(ctx, next)
	})
}
func (s *Service) Logout(ctx context.Context, token identity.Token) error {
	return s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		access, err := s.access(ctx, token)
		if err != nil {
			return err
		}
		return s.revokeSession(ctx, access.Session)
	})
}
func (s *Service) revokeSession(ctx context.Context, x identity.Session) error {
	x.Revoked = true
	if err := s.repo.SaveIdentitySession(ctx, x); err != nil {
		return err
	}
	if err := s.repo.RevokeIdentitySubscriptions(ctx, x.UserID, x.ID); err != nil {
		return err
	}
	return s.repo.IdentityAudit(ctx, x.UserID, "session_revoked", x.ID, s.now())
}

func (s *Service) limitCredential(ctx context.Context, rawID string) error {
	raw, err := base64.RawURLEncoding.DecodeString(rawID)
	if err != nil {
		return identity.ErrAttempt
	}
	return s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		credential, err := s.repo.IdentityCredential(ctx, raw)
		if err == identity.ErrUnauthorized {
			return nil
		}
		if err != nil {
			return err
		}
		return s.repo.IdentityRateLimit(ctx, "user:"+string(credential.UserID), 30, s.now())
	})
}
