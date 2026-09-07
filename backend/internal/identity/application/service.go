package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	households "github.com/pchkauu/want-keep/backend/internal/household/application"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

type Service struct {
	repo         Repository
	households   households.Repository
	invitations  *households.Invitations
	verifier     Verifier
	rpID, origin string
	maximum      int
	clock        func() time.Time
}

func NewService(repo Repository, h households.Repository, v Verifier, rpID, origin string, maximum int, clock func() time.Time) (*Service, error) {
	if repo == nil || h == nil || v == nil || rpID == "" || origin == "" || maximum < 1 || clock == nil {
		return nil, identity.ErrUnavailable
	}
	return &Service{repo: repo, households: h, invitations: households.NewInvitations(h), verifier: v, rpID: rpID, origin: origin, maximum: maximum, clock: clock}, nil
}
func (s *Service) now() time.Time { return s.clock().UTC().Truncate(time.Microsecond) }
func (s *Service) access(ctx context.Context, token identity.Token) (Access, error) {
	if !token.Valid(32) {
		return Access{}, identity.ErrUnauthorized
	}
	session, err := s.repo.IdentitySession(ctx, token.Hash())
	if err != nil {
		return Access{}, err
	}
	if err = session.RequireActive(s.now()); err != nil {
		return Access{}, err
	}
	profile, err := s.repo.IdentityProfile(ctx, session.UserID)
	if err != nil {
		return Access{}, err
	}
	principal, err := s.repo.IdentityMembership(ctx, profile)
	if err != nil {
		return Access{}, err
	}
	return Access{Profile: profile, Principal: principal, Session: session, Token: token}, nil
}
func (s *Service) Me(ctx context.Context, token identity.Token) (Access, error) {
	var result Access
	err := s.repo.WithinIdentity(ctx, func(ctx context.Context) error { var err error; result, err = s.access(ctx, token); return err })
	return result, err
}

// WithinSession serializes a dependent security change with logout and recovery.
// The callback must contain database-only work and preserve identity-before-household lock order.
func (s *Service) WithinSession(ctx context.Context, token identity.Token, apply func(context.Context, Access) error) error {
	return s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		access, err := s.access(ctx, token)
		if err != nil {
			return err
		}
		return apply(ctx, access)
	})
}

func (s *Service) Limit(ctx context.Context, r RequestContext) error {
	if !r.Browser.Valid(32) || r.Source == "" {
		return identity.ErrAttempt
	}
	return s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		if err := s.repo.IdentityRateLimit(ctx, "deployment", 600, s.now()); err != nil {
			return err
		}
		if err := s.repo.IdentityRateLimit(ctx, "source:"+identity.Token(r.Source).Hash(), 120, s.now()); err != nil {
			return err
		}
		return s.repo.IdentityRateLimit(ctx, "browser:"+r.Browser.Hash(), 60, s.now())
	})
}

func (s *Service) attempt(purpose identity.Purpose, r RequestContext) (identity.Attempt, error) {
	id, err := s.newID()
	if err != nil {
		return identity.Attempt{}, err
	}
	challenge, err := identity.NewToken(32)
	if err != nil {
		return identity.Attempt{}, err
	}
	now := s.now()
	return identity.Attempt{ID: id, Purpose: purpose, BrowserHash: r.Browser.Hash(), Challenge: string(challenge), RPID: s.rpID, Origin: s.origin, CreatedAt: now, ExpiresAt: now.Add(identity.CeremonyLifetime)}, nil
}
func (s *Service) newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	x := hex.EncodeToString(b)
	return x[:8] + "-" + x[8:12] + "-" + x[12:16] + "-" + x[16:20] + "-" + x[20:], nil
}

// A rejected ceremony is consumed and audited without retaining the supplied authenticator response.
func (s *Service) finish(ctx context.Context, r RequestContext, id string, fn func(context.Context, identity.Attempt) error) error {
	var rejection error
	err := s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		a, err := s.repo.IdentityAttempt(ctx, id)
		if err != nil {
			return err
		}
		if err = a.Require(r.Browser.Hash(), s.now(), a.Purpose); err != nil {
			return err
		}
		a.Consumed = true
		if err = s.repo.SaveIdentityAttempt(ctx, a); err != nil {
			return err
		}
		// The inner scope is a savepoint: a rejected flow cannot commit partial access changes.
		err = s.repo.WithinIdentity(ctx, func(ctx context.Context) error { return fn(ctx, a) })
		if err == nil {
			return nil
		}
		if errors.Is(err, household.ErrInvitation) || errors.Is(err, household.ErrInvitationExpired) || errors.Is(err, household.ErrInvitationUsed) || errors.Is(err, household.ErrInvitationRevoked) || errors.Is(err, household.ErrInvitationRevision) || errors.Is(err, household.ErrMemberLimit) || errors.Is(err, identity.ErrAlreadyAuthenticated) || errors.Is(err, identity.ErrAttempt) || errors.Is(err, identity.ErrUnauthorized) || errors.Is(err, identity.ErrBootstrap) || errors.Is(err, identity.ErrFreshAuthentication) || errors.Is(err, identity.ErrCounter) {
			rejection = err
			event := "ceremony_rejected"
			if errors.Is(err, identity.ErrCounter) {
				event = "counter_rejected"
			}
			return s.repo.IdentityAudit(ctx, a.UserID, event, a.ID, s.now())
		}
		return err
	})
	if err != nil {
		return err
	}
	return rejection
}
