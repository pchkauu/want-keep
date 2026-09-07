package application

import (
	"context"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

// Initialization is called only by the future closed bootstrap/invitation use cases.
type Repository interface {
	InitializeHousehold(context.Context, household.Household, []household.User, []household.Membership, calendar.Timezone, int) error
	AddMember(context.Context, household.User, household.Membership) error
	Membership(context.Context, household.HouseholdID, household.UserID) (household.Membership, error)
}
