package domain

import household "github.com/pchkauu/want-keep/backend/internal/household/domain"

type ExternalOwnership struct {
	HouseholdID household.HouseholdID
	OwnerID     household.UserID
}

func (o ExternalOwnership) RequireManage(p household.Principal) error {
	if o.OwnerID == "" {
		return household.ErrForbidden
	}
	return p.RequireHousehold(o.HouseholdID)
}

func (o ExternalOwnership) RequireAuthentication(p household.Principal) error {
	if err := o.RequireManage(p); err != nil {
		return err
	}
	if p.UserID() != o.OwnerID {
		return household.ErrForbidden
	}
	return nil
}
