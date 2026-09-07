package application

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"

	households "github.com/pchkauu/want-keep/backend/internal/household/application"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type IssuedInvitation struct {
	State household.InvitationState
	Token identity.Token
}
type InvitedProfile struct{ Name, Locale, ReportingAsset string }

func (s *Service) Household(ctx context.Context, token identity.Token) (household.Details, error) {
	var result household.Details
	err := s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		a, err := s.access(ctx, token)
		if err != nil {
			return err
		}
		result, err = s.invitations.ReadHousehold(ctx, a.Principal)
		return err
	})
	return result, err
}

func (s *Service) Invitations(ctx context.Context, token identity.Token) (household.InvitationState, error) {
	var result household.InvitationState
	err := s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		a, err := s.access(ctx, token)
		if err != nil {
			return err
		}
		result, err = s.invitations.Read(ctx, a.Principal)
		return err
	})
	return result, err
}

func (s *Service) IssueInvitation(ctx context.Context, r RequestContext, expected uint64) (IssuedInvitation, error) {
	var result IssuedInvitation
	if err := s.Limit(ctx, r); err != nil {
		return result, err
	}
	err := s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		a, err := s.access(ctx, r.Session)
		if err != nil {
			return err
		}
		if err = a.Session.RequireFresh(s.now()); err != nil {
			return err
		}
		if err = s.repo.IdentityRateLimit(ctx, "user:"+string(a.Principal.UserID()), 30, s.now()); err != nil {
			return err
		}
		result.Token, err = identity.NewToken(32)
		if err != nil {
			return err
		}
		id, err := s.newID()
		if err != nil {
			return err
		}
		result.State, err = s.invitations.Issue(ctx, a.Principal, expected, id, result.Token.Hash(), s.now())
		return err
	})
	if err != nil {
		return IssuedInvitation{}, err
	}
	return result, nil
}

func (s *Service) RevokeInvitation(ctx context.Context, r RequestContext, id string, expected uint64) (household.InvitationState, error) {
	var result household.InvitationState
	if err := s.Limit(ctx, r); err != nil {
		return result, err
	}
	err := s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		a, err := s.access(ctx, r.Session)
		if err != nil {
			return err
		}
		result, err = s.invitations.Revoke(ctx, a.Principal, id, expected, s.now())
		return err
	})
	return result, err
}

func (s *Service) requireSignedOut(ctx context.Context, token identity.Token) error {
	if !token.Valid(32) {
		return nil
	}
	_, err := s.access(ctx, token)
	if err == nil {
		return identity.ErrAlreadyAuthenticated
	}
	if errors.Is(err, identity.ErrUnauthorized) {
		return nil
	}
	return err
}

func (s *Service) PreviewInvitation(ctx context.Context, r RequestContext, token identity.Token) (households.InvitationPreview, error) {
	var result households.InvitationPreview
	if err := s.Limit(ctx, r); err != nil {
		return result, err
	}
	if !token.Valid(32) {
		return result, household.ErrInvitation
	}
	err := s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		if err := s.requireSignedOut(ctx, r.Session); err != nil {
			return err
		}
		var err error
		result, err = s.invitations.Preview(ctx, token.Hash(), s.now())
		return err
	})
	return result, err
}

func (s *Service) BeginInvitation(ctx context.Context, r RequestContext, token identity.Token, input InvitedProfile) (Enrollment, error) {
	var result Enrollment
	if strings.TrimSpace(input.Name) == "" || len(input.Name) > 2000 || (input.Locale != "ru" && input.Locale != "en") {
		return result, identity.ErrAttempt
	}
	if _, err := money.ParseAsset(input.ReportingAsset); err != nil {
		return result, identity.ErrAttempt
	}
	if err := s.Limit(ctx, r); err != nil {
		return result, err
	}
	if !token.Valid(32) {
		return result, household.ErrInvitation
	}
	err := s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		if err := s.requireSignedOut(ctx, r.Session); err != nil {
			return err
		}
		preview, err := s.invitations.Preview(ctx, token.Hash(), s.now())
		if err != nil {
			return err
		}
		user, err := s.newID()
		if err != nil {
			return err
		}
		member, err := s.newID()
		if err != nil {
			return err
		}
		handle, err := identity.NewToken(32)
		if err != nil {
			return err
		}
		decoded, err := base64.RawURLEncoding.DecodeString(string(handle))
		if err != nil {
			return err
		}
		a, err := s.attempt(identity.Invitation, r)
		if err != nil {
			return err
		}
		a.GrantID, a.InvitationRevision = preview.Admission.InvitationID, preview.Admission.Revision
		a.Setup = &identity.Setup{Name: input.Name, Locale: input.Locale, ReportingAsset: input.ReportingAsset, UserID: household.UserID(user), HouseholdID: preview.Admission.HouseholdID, MembershipID: household.MembershipID(member), Handle: decoded}
		result = Enrollment{Attempt: a, Profile: s.setupProfile(*a.Setup)}
		return s.repo.SaveIdentityAttempt(ctx, a)
	})
	return result, err
}

func (s *Service) CompleteInvitation(ctx context.Context, r RequestContext, token identity.Token, attemptID, credentialName string, input Registration) (EnrollmentResult, error) {
	var result EnrollmentResult
	if strings.TrimSpace(credentialName) == "" || len(credentialName) > 2000 {
		return result, identity.ErrAttempt
	}
	if err := s.Limit(ctx, r); err != nil {
		return result, err
	}
	if !token.Valid(32) {
		return result, household.ErrInvitation
	}
	err := s.finish(ctx, r, attemptID, func(ctx context.Context, a identity.Attempt) error {
		if err := s.requireSignedOut(ctx, r.Session); err != nil {
			return err
		}
		if a.Purpose != identity.Invitation || a.Setup == nil || a.InvitationRevision == 0 {
			return identity.ErrAttempt
		}
		p := s.setupProfile(*a.Setup)
		preview, err := s.invitations.Preview(ctx, token.Hash(), s.now())
		if err != nil {
			return err
		}
		admission := households.InvitationAdmission{InvitationID: a.GrantID, HouseholdID: p.HouseholdID, Revision: a.InvitationRevision}
		if admission != preview.Admission {
			return household.ErrInvitationRevision
		}
		c, err := s.verifier.Register(p, a, input)
		if err != nil {
			return err
		}
		if _, err = s.repo.IdentityCredential(ctx, c.RawID); err == nil {
			return identity.ErrAttempt
		} else if !errors.Is(err, identity.ErrUnauthorized) {
			return err
		}
		c.ID, err = s.newID()
		if err != nil {
			return err
		}
		c.UserID, c.Name, c.CreatedAt = p.UserID, credentialName, s.now()
		member := household.Membership{ID: p.MembershipID, UserID: p.UserID, HouseholdID: p.HouseholdID, Active: true}
		if err = s.invitations.Accept(ctx, admission, token.Hash(), household.User{ID: p.UserID, Name: p.Name}, member, s.now()); err != nil {
			return err
		}
		if err = s.repo.CreateIdentityProfile(ctx, p); err != nil {
			return err
		}
		if err = s.repo.SaveIdentityCredential(ctx, c); err != nil {
			return err
		}
		result.Purpose = identity.Invitation
		result.Codes, err = s.replaceCodes(ctx, p)
		if err != nil {
			return err
		}
		result.Access, err = s.createSession(ctx, p, c.ID)
		if err != nil {
			return err
		}
		return s.repo.IdentityAudit(ctx, p.UserID, "invitation_enrollment_completed", c.ID, s.now())
	})
	if err != nil {
		return EnrollmentResult{}, err
	}
	return result, nil
}
