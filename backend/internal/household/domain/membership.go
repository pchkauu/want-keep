package domain

import "errors"

type UserID string
type HouseholdID string
type MembershipID string

var (
	ErrInvalidMembership = errors.New("invalid membership")
	ErrForbidden         = errors.New("forbidden")
)

type User struct {
	ID   UserID
	Name string
}
type Household struct {
	ID   HouseholdID
	Name string
}

type Membership struct {
	ID          MembershipID
	UserID      UserID
	HouseholdID HouseholdID
	Active      bool
}

// Principal derives from membership loaded using an authenticated identity, never request JSON.
type Principal struct {
	userID      UserID
	householdID HouseholdID
}

func (m Membership) Principal() (Principal, error) {
	if m.ID == "" || m.UserID == "" || m.HouseholdID == "" || !m.Active {
		return Principal{}, ErrInvalidMembership
	}
	return Principal{userID: m.UserID, householdID: m.HouseholdID}, nil
}

func (p Principal) UserID() UserID           { return p.userID }
func (p Principal) HouseholdID() HouseholdID { return p.householdID }
func (p Principal) RequireHousehold(id HouseholdID) error {
	if p.userID == "" || p.householdID == "" || p.householdID != id {
		return ErrForbidden
	}
	return nil
}

// ValidateMemberLimit validates the configured cap; task-1.3 must enforce it transactionally.
func (h Household) ValidateMemberLimit(members []Membership, maximum int) error {
	if h.ID == "" || maximum < 1 {
		return ErrInvalidMembership
	}
	seen := make(map[UserID]bool)
	ids := make(map[MembershipID]bool)
	active := 0
	for _, member := range members {
		if member.ID == "" || member.UserID == "" || member.HouseholdID != h.ID || ids[member.ID] || seen[member.UserID] {
			return ErrInvalidMembership
		}
		ids[member.ID], seen[member.UserID] = true, true
		if member.Active {
			active++
		}
	}
	if active > maximum {
		return ErrInvalidMembership
	}
	return nil
}
