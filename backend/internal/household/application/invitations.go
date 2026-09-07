package application

import (
	"context"
	"time"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

type InvitationRepository interface {
	WithinInvitationHousehold(context.Context, household.HouseholdID, func(context.Context) error) error
	HouseholdDetails(context.Context, household.HouseholdID) (household.Details, error)
	InvitationState(context.Context, household.HouseholdID) (household.InvitationState, error)
	InvitationByHash(context.Context, string) (household.Invitation, error)
	SaveInvitation(context.Context, household.Invitation) error
	SaveInvitationState(context.Context, household.InvitationState, uint64) error
	JoinHousehold(context.Context, household.User, household.Membership) error
	InvitationAudit(context.Context, household.HouseholdID, household.UserID, string, string, uint64, time.Time) error
	MembershipJoined(context.Context, household.Membership, uint64) error
}

type Invitations struct{ repository InvitationRepository }

func NewInvitations(r InvitationRepository) *Invitations { return &Invitations{repository: r} }

type InvitationAdmission struct {
	InvitationID string
	HouseholdID  household.HouseholdID
	Revision     uint64
}
type InvitationPreview struct {
	Admission                  InvitationAdmission
	HouseholdName, InviterName string
	ExpiresAt                  time.Time
}

func (s *Invitations) ReadHousehold(ctx context.Context, p household.Principal) (household.Details, error) {
	d, err := s.repository.HouseholdDetails(ctx, p.HouseholdID())
	if err != nil {
		return d, err
	}
	_, err = d.RequireMember(p.UserID())
	return d, err
}

func (s *Invitations) Read(ctx context.Context, p household.Principal) (household.InvitationState, error) {
	if _, err := s.ReadHousehold(ctx, p); err != nil {
		return household.InvitationState{}, err
	}
	return s.repository.InvitationState(ctx, p.HouseholdID())
}

func (s *Invitations) Issue(ctx context.Context, p household.Principal, expected uint64, id, hash string, now time.Time) (household.InvitationState, error) {
	var result household.InvitationState
	err := s.repository.WithinInvitationHousehold(ctx, p.HouseholdID(), func(ctx context.Context) error {
		d, err := s.ReadHousehold(ctx, p)
		if err != nil {
			return err
		}
		if err = d.RequireSpace(); err != nil {
			return err
		}
		state, err := s.repository.InvitationState(ctx, p.HouseholdID())
		if err != nil {
			return err
		}
		next := household.Invitation{ID: id, HouseholdID: p.HouseholdID(), InvitedBy: p.UserID(), TokenHash: hash, CreatedAt: now, ExpiresAt: now.Add(household.InvitationLifetime), Status: household.InvitationActive}
		result, err = state.Change(expected, next)
		if err != nil {
			return err
		}
		if state.Current != nil && state.Current.Status == household.InvitationActive {
			old, err := state.Current.Revoke()
			if err != nil {
				return err
			}
			if err = s.repository.SaveInvitation(ctx, old); err != nil {
				return err
			}
		}
		if err = s.repository.SaveInvitation(ctx, next); err != nil {
			return err
		}
		if err = s.repository.SaveInvitationState(ctx, result, expected); err != nil {
			return err
		}
		return s.repository.InvitationAudit(ctx, p.HouseholdID(), p.UserID(), "invitation_issued", id, result.Revision, now)
	})
	return result, err
}

func (s *Invitations) Revoke(ctx context.Context, p household.Principal, id string, expected uint64, now time.Time) (household.InvitationState, error) {
	var result household.InvitationState
	err := s.repository.WithinInvitationHousehold(ctx, p.HouseholdID(), func(ctx context.Context) error {
		if _, err := s.ReadHousehold(ctx, p); err != nil {
			return err
		}
		state, err := s.repository.InvitationState(ctx, p.HouseholdID())
		if err != nil {
			return err
		}
		if state.Current == nil || state.Current.ID != id {
			return household.ErrInvitation
		}
		next, err := state.Current.Revoke()
		if err != nil {
			return err
		}
		result, err = state.Change(expected, next)
		if err != nil {
			return err
		}
		if err = s.repository.SaveInvitation(ctx, next); err != nil {
			return err
		}
		if err = s.repository.SaveInvitationState(ctx, result, expected); err != nil {
			return err
		}
		return s.repository.InvitationAudit(ctx, p.HouseholdID(), p.UserID(), "invitation_revoked", id, result.Revision, now)
	})
	return result, err
}

func (s *Invitations) Preview(ctx context.Context, hash string, now time.Time) (InvitationPreview, error) {
	i, err := s.repository.InvitationByHash(ctx, hash)
	if err != nil {
		return InvitationPreview{}, err
	}
	if err = i.RequireActive(now); err != nil {
		return InvitationPreview{}, err
	}
	d, err := s.repository.HouseholdDetails(ctx, i.HouseholdID)
	if err != nil {
		return InvitationPreview{}, err
	}
	if _, err = d.RequireMember(i.InvitedBy); err != nil {
		return InvitationPreview{}, household.ErrInvitation
	}
	if err = d.RequireSpace(); err != nil {
		return InvitationPreview{}, err
	}
	state, err := s.repository.InvitationState(ctx, i.HouseholdID)
	if err != nil {
		return InvitationPreview{}, err
	}
	if state.Current == nil || state.Current.ID != i.ID {
		return InvitationPreview{}, household.ErrInvitation
	}
	result := InvitationPreview{Admission: InvitationAdmission{InvitationID: i.ID, HouseholdID: i.HouseholdID, Revision: state.Revision}, HouseholdName: d.Household.Name, ExpiresAt: i.ExpiresAt}
	for _, m := range d.Members {
		if m.User.ID == i.InvitedBy {
			result.InviterName = m.User.Name
		}
	}
	return result, nil
}

// The identity coordinator has verified the credential before joining. The invitation is checked again under the family lock.
func (s *Invitations) Accept(ctx context.Context, admission InvitationAdmission, hash string, user household.User, member household.Membership, now time.Time) error {
	return s.repository.WithinInvitationHousehold(ctx, admission.HouseholdID, func(ctx context.Context) error {
		preview, err := s.Preview(ctx, hash, now)
		if err != nil {
			return err
		}
		if preview.Admission != admission {
			return household.ErrInvitationRevision
		}
		if member.HouseholdID != admission.HouseholdID || member.UserID != user.ID || !member.Active {
			return household.ErrInvalidMembership
		}
		state, err := s.repository.InvitationState(ctx, admission.HouseholdID)
		if err != nil {
			return err
		}
		next, err := state.Current.Accept(user.ID, now)
		if err != nil {
			return err
		}
		updated, err := state.Change(admission.Revision, next)
		if err != nil {
			return err
		}
		if err = s.repository.JoinHousehold(ctx, user, member); err != nil {
			return err
		}
		if err = s.repository.SaveInvitation(ctx, next); err != nil {
			return err
		}
		if err = s.repository.SaveInvitationState(ctx, updated, state.Revision); err != nil {
			return err
		}
		if err = s.repository.InvitationAudit(ctx, admission.HouseholdID, user.ID, "invitation_accepted", next.ID, updated.Revision, now); err != nil {
			return err
		}
		return s.repository.MembershipJoined(ctx, member, updated.Revision)
	})
}
