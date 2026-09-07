package application

import (
	"context"

	goals "github.com/pchkauu/want-keep/backend/internal/goals/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

type Repository interface {
	CreateGoal(context.Context, goals.Goal) error
	Goal(context.Context, household.Principal, string) (goals.Goal, error)
	WriteReservations(context.Context, goals.Goal, []goals.Reservation, string) error
}
