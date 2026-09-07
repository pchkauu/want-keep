package domain

import (
	"encoding/hex"
	"errors"
	"time"
)

const InvitationLifetime = 24 * time.Hour
const MaxInvitationRevision uint64 = 9007199254740991

var (
	ErrInvitation         = errors.New("invitation unavailable")
	ErrInvitationExpired  = errors.New("invitation expired")
	ErrInvitationUsed     = errors.New("invitation used")
	ErrInvitationRevoked  = errors.New("invitation revoked")
	ErrInvitationRevision = errors.New("invitation revision conflict")
	ErrMemberLimit        = errors.New("member limit reached")
)

type InvitationStatus string

const (
	InvitationActive   InvitationStatus = "active"
	InvitationRevoked  InvitationStatus = "revoked"
	InvitationAccepted InvitationStatus = "accepted"
)

type Invitation struct {
	ID                   string
	HouseholdID          HouseholdID
	InvitedBy            UserID
	TokenHash            string
	CreatedAt, ExpiresAt time.Time
	Status               InvitationStatus
	AcceptedUserID       UserID
}

func (i Invitation) Validate() error {
	hash, err := hex.DecodeString(i.TokenHash)
	if err != nil || len(hash) != 32 || hex.EncodeToString(hash) != i.TokenHash || i.ID == "" || i.HouseholdID == "" || i.InvitedBy == "" || i.CreatedAt.IsZero() || !i.ExpiresAt.Equal(i.CreatedAt.Add(InvitationLifetime)) {
		return ErrInvitation
	}
	if i.Status != InvitationActive && i.Status != InvitationRevoked && i.Status != InvitationAccepted {
		return ErrInvitation
	}
	if (i.Status == InvitationAccepted) != (i.AcceptedUserID != "") {
		return ErrInvitation
	}
	return nil
}

func (i Invitation) RequireActive(now time.Time) error {
	if err := i.Validate(); err != nil {
		return err
	}
	switch i.Status {
	case InvitationRevoked:
		return ErrInvitationRevoked
	case InvitationAccepted:
		return ErrInvitationUsed
	}
	if now.Before(i.CreatedAt) {
		return ErrInvitation
	}
	if !now.Before(i.ExpiresAt) {
		return ErrInvitationExpired
	}
	return nil
}

func (i Invitation) Revoke() (Invitation, error) {
	if err := i.Validate(); err != nil {
		return Invitation{}, err
	}
	if i.Status != InvitationActive {
		return Invitation{}, ErrInvitationRevision
	}
	i.Status = InvitationRevoked
	return i, nil
}

func (i Invitation) Accept(user UserID, now time.Time) (Invitation, error) {
	if err := i.RequireActive(now); err != nil {
		return Invitation{}, err
	}
	if user == "" || user == i.InvitedBy {
		return Invitation{}, ErrInvitation
	}
	i.Status, i.AcceptedUserID = InvitationAccepted, user
	return i, nil
}

type InvitationState struct {
	HouseholdID HouseholdID
	Revision    uint64
	Current     *Invitation
}

func (s InvitationState) Validate() error {
	if s.HouseholdID == "" || s.Revision < 1 || s.Revision > MaxInvitationRevision {
		return ErrInvitationRevision
	}
	if s.Current != nil {
		if s.Current.HouseholdID != s.HouseholdID {
			return ErrInvitation
		}
		return s.Current.Validate()
	}
	return nil
}

func (s InvitationState) Change(expected uint64, next Invitation) (InvitationState, error) {
	if err := s.Validate(); err != nil {
		return InvitationState{}, err
	}
	if s.Revision != expected || expected == MaxInvitationRevision {
		return InvitationState{}, ErrInvitationRevision
	}
	if err := next.Validate(); err != nil {
		return InvitationState{}, err
	}
	if next.HouseholdID != s.HouseholdID {
		return InvitationState{}, ErrInvitation
	}
	s.Revision++
	s.Current = &next
	return s, nil
}

type Member struct {
	User       User
	Membership Membership
}
type Details struct {
	Household Household
	Timezone  string
	Maximum   int
	Members   []Member
}

func (d Details) RequireMember(user UserID) (Principal, error) {
	for _, m := range d.Members {
		if m.Membership.UserID == user && m.Membership.HouseholdID == d.Household.ID {
			return m.Membership.Principal()
		}
	}
	return Principal{}, ErrForbidden
}

func (d Details) RequireSpace() error {
	if d.Household.ID == "" || d.Maximum < 1 {
		return ErrInvalidMembership
	}
	active := 0
	for _, m := range d.Members {
		if m.Membership.Active {
			active++
		}
	}
	if active >= d.Maximum {
		return ErrMemberLimit
	}
	return nil
}
