package domain

import "errors"

type Scope string

const (
	Personal Scope = "personal"
	Shared   Scope = "household"
)

var ErrInvalidOwnership = errors.New("invalid ownership")

type Ownership struct {
	householdID HouseholdID
	scope       Scope
	owner       UserID
}

func NewOwnership(householdID HouseholdID, scope Scope, owner UserID) (Ownership, error) {
	if householdID == "" || (scope != Personal && scope != Shared) || (scope == Personal && owner == "") || (scope == Shared && owner != "") {
		return Ownership{}, ErrInvalidOwnership
	}
	return Ownership{householdID: householdID, scope: scope, owner: owner}, nil
}

func (o Ownership) HouseholdID() HouseholdID { return o.householdID }
func (o Ownership) Scope() Scope             { return o.scope }
func (o Ownership) PersonalOwnerID() UserID  { return o.owner }
func (o Ownership) Validate() error {
	_, err := NewOwnership(o.householdID, o.scope, o.owner)
	return err
}

func (o Ownership) RequireRead(principal Principal) error {
	if err := o.Validate(); err != nil {
		return err
	}
	return principal.RequireHousehold(o.householdID)
}

// Personal plans/goals and account ownership use this policy. Accounting facts use family access.
func (o Ownership) RequireOwnerEdit(principal Principal) error {
	if err := o.RequireRead(principal); err != nil {
		return err
	}
	if o.scope == Personal && o.owner != principal.UserID() {
		return ErrForbidden
	}
	return nil
}
