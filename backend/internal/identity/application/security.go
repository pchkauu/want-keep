package application

import (
	"context"

	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

func (s *Service) Passkeys(ctx context.Context, token identity.Token) ([]identity.Credential, error) {
	var result []identity.Credential
	err := s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		access, err := s.access(ctx, token)
		if err != nil {
			return err
		}
		keys, err := s.repo.IdentityCredentials(ctx, access.Profile.UserID)
		if err != nil {
			return err
		}
		for _, key := range keys {
			if !key.Revoked {
				result = append(result, key)
			}
		}
		return nil
	})
	return result, err
}
func (s *Service) Sessions(ctx context.Context, token identity.Token) ([]identity.Session, error) {
	var result []identity.Session
	err := s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		access, err := s.access(ctx, token)
		if err != nil {
			return err
		}
		sessions, err := s.repo.IdentitySessions(ctx, access.Profile.UserID)
		if err != nil {
			return err
		}
		for _, session := range sessions {
			if session.RequireActive(s.now()) == nil {
				result = append(result, session)
			}
		}
		return nil
	})
	return result, err
}
func (s *Service) RevokePasskey(ctx context.Context, token identity.Token, id string) error {
	return s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		access, err := s.access(ctx, token)
		if err != nil {
			return err
		}
		if err = access.Session.RequireFresh(s.now()); err != nil {
			return err
		}
		keys, err := s.repo.IdentityCredentials(ctx, access.Profile.UserID)
		if err != nil {
			return err
		}
		var target identity.Credential
		active := 0
		for _, key := range keys {
			if key.ID == id {
				target = key
			}
			if !key.Revoked {
				active++
			}
		}
		if target.ID == "" {
			return identity.ErrUnauthorized
		}
		if target.Revoked {
			return nil
		}
		if active < 2 {
			return identity.ErrLastPasskey
		}
		target.Revoked = true
		if err = s.repo.SaveIdentityCredential(ctx, target); err != nil {
			return err
		}
		sessions, err := s.repo.IdentitySessions(ctx, access.Profile.UserID)
		if err != nil {
			return err
		}
		for _, session := range sessions {
			if session.CredentialID == id && !session.Revoked {
				if err = s.revokeSession(ctx, session); err != nil {
					return err
				}
			}
		}
		return s.repo.IdentityAudit(ctx, access.Profile.UserID, "passkey_revoked", id, s.now())
	})
}
func (s *Service) RevokeSession(ctx context.Context, token identity.Token, id string) error {
	return s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		access, err := s.access(ctx, token)
		if err != nil {
			return err
		}
		sessions, err := s.repo.IdentitySessions(ctx, access.Profile.UserID)
		if err != nil {
			return err
		}
		for _, session := range sessions {
			if session.ID == id {
				if session.Revoked {
					return nil
				}
				return s.revokeSession(ctx, session)
			}
		}
		return identity.ErrUnauthorized
	})
}
