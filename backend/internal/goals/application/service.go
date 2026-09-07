package application

import (
	"context"
	"errors"

	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	goals "github.com/pchkauu/want-keep/backend/internal/goals/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type ReservationRepository interface {
	Goal(context.Context, household.Principal, string) (goals.Goal, error)
	Funding(context.Context, household.Principal, string, []string) (map[string]goals.Funding, error)
	WriteReservations(context.Context, goals.Goal, []goals.Reservation, string) error
	EmitEvent(context.Context, string, string, uint64, string) error
}
type Service struct{ repository ReservationRepository }

func NewService(r ReservationRepository) *Service { return &Service{r} }
func (s *Service) Replace(ctx context.Context, p household.Principal, id string, expected uint64, reservations []goals.Reservation, reason string) (command.Result, error) {
	goal, err := s.repository.Goal(ctx, p, id)
	if err != nil {
		return command.Result{}, err
	}
	if err = goal.Ownership.RequireOwnerEdit(p); err != nil {
		return command.Result{}, commands.Rejection{Code: "forbidden"}
	}
	if expected != goal.Revision || expected >= command.MaxRevision {
		return command.Result{}, commands.Rejection{Code: "version_conflict"}
	}
	if len(reason) < 1 || len(reason) > 2000 {
		return command.Result{}, commands.Rejection{Code: "invalid_allocation"}
	}
	ids := make([]string, len(reservations))
	for i, r := range reservations {
		ids[i] = r.AccountID
	}
	funding, err := s.repository.Funding(ctx, p, id, ids)
	if err != nil {
		return command.Result{}, err
	}
	if err = goal.ValidateReservations(reservations, funding); err != nil {
		code := "invalid_allocation"
		if errors.Is(err, goals.ErrInsufficientFunds) {
			code = "invalid_availability"
		}
		if errors.Is(err, money.ErrAssetMismatch) {
			code = "asset_mismatch"
		}
		return command.Result{}, commands.Rejection{Code: code}
	}
	goal.Revision++
	if err = s.repository.WriteReservations(ctx, goal, reservations, reason); err != nil {
		return command.Result{}, err
	}
	if err = s.repository.EmitEvent(ctx, "goal", id, goal.Revision, "goal.reservations_changed"); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "goal", ResourceID: id, Revision: goal.Revision}, nil
}
