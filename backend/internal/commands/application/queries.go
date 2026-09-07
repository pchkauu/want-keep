package application

import (
	"context"
	"errors"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

type ReadRepository interface {
	LoadCommand(context.Context, household.Principal, string) (command.Command, error)
	RecentCommands(context.Context, household.Principal, calendar.Instant, string, int) ([]command.Command, string, error)
}
type ResultAuthorizer func(context.Context, household.Principal, command.Result) error
type Queries struct {
	repository ReadRepository
	authorize  ResultAuthorizer
}

func NewQueries(r ReadRepository, a ResultAuthorizer) *Queries { return &Queries{r, a} }
func (q *Queries) Read(ctx context.Context, p household.Principal, id string, now calendar.Instant) (command.Command, error) {
	c, err := q.repository.LoadCommand(ctx, p, id)
	if err != nil {
		return command.Command{}, err
	}
	detailErr := c.RequireDetail(p, now)
	if detailErr != nil && !errors.Is(detailErr, command.ErrCommandExpired) {
		return command.Command{}, detailErr
	}
	if err = q.authorizeResult(ctx, p, c); err != nil {
		return command.Command{}, err
	}
	return c, detailErr
}
func (q *Queries) Recent(ctx context.Context, p household.Principal, now calendar.Instant, after string, limit int) ([]command.Command, string, error) {
	if now.String() == "" {
		return nil, "", command.ErrInvalidCommand
	}
	entries, next, err := q.repository.RecentCommands(ctx, p, now, after, limit)
	if err != nil {
		return nil, "", err
	}
	for _, c := range entries {
		eligible, err := c.InRecent(p, now)
		if err != nil {
			return nil, "", err
		}
		if !eligible {
			return nil, "", command.ErrInvalidCommand
		}
		if err = q.authorizeResult(ctx, p, c); err != nil {
			return nil, "", err
		}
	}
	return entries, next, nil
}
func (q *Queries) authorizeResult(ctx context.Context, p household.Principal, c command.Command) error {
	if result, ok := c.Result(); ok {
		if q.authorize == nil {
			return household.ErrForbidden
		}
		return q.authorize(ctx, p, result)
	}
	return nil
}
