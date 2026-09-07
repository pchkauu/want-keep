package contract

import (
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func (b *Boundary) OwnershipFromDTO(value generated.Ownership) (household.Ownership, error) {
	if err := b.validateDTO("Ownership", value); err != nil {
		return household.Ownership{}, err
	}
	personal, err := value.AsPersonalOwnership()
	if err != nil {
		return household.Ownership{}, ErrInvalidRequest
	}
	return household.NewOwnership(household.HouseholdID(personal.HouseholdId), household.Scope(personal.Scope), household.UserID(personal.PersonalOwnerId))
}

func (b *Boundary) OwnershipToDTO(value household.Ownership) (generated.Ownership, error) {
	if err := value.Validate(); err != nil {
		return generated.Ownership{}, err
	}
	var result generated.Ownership
	var err error
	if value.Scope() == household.Personal {
		err = result.FromPersonalOwnership(generated.PersonalOwnership{HouseholdId: string(value.HouseholdID()), Scope: "personal", PersonalOwnerId: string(value.PersonalOwnerID())})
	} else {
		err = result.FromSharedOwnership(generated.SharedOwnership{HouseholdId: string(value.HouseholdID()), Scope: "household"})
	}
	if err != nil {
		return result, err
	}
	return result, b.validateDTO("Ownership", result)
}
