package application

import (
	"context"
	"errors"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

type Transactions interface {
	// Registration requires an independently committed transaction; an enclosing scope is rejected.
	WithinNewHousehold(context.Context, household.Principal, func(context.Context) error) error
	WithinHousehold(context.Context, household.Principal, func(context.Context) error) error
}

type Repository interface {
	RegisterCommand(context.Context, command.Command) (command.Command, error)
	LoadCommand(context.Context, household.Principal, string) (command.Command, error)
	SaveCommand(context.Context, command.Command) error
}

// Rejection is a confirmed business refusal. Infrastructure errors leave the command pending.
type Rejection struct{ Code string }

func (e Rejection) Error() string { return e.Code }

type Request struct{ ID, Kind, PayloadHash string }

type Executor struct {
	transactions Transactions
	repository   Repository
	now          func() calendar.Instant
}

func NewExecutor(t Transactions, r Repository, now func() calendar.Instant) *Executor {
	return &Executor{t, r, now}
}

func (s *Executor) Register(ctx context.Context, p household.Principal, r Request) (command.Command, error) {
	c, err := command.NewCommand(r.ID, r.Kind, r.PayloadHash, p, s.now())
	if err != nil {
		return command.Command{}, err
	}
	err = s.transactions.WithinNewHousehold(ctx, p, func(ctx context.Context) error {
		var e error
		c, e = s.repository.RegisterCommand(ctx, c)
		if e != nil {
			return e
		}
		return c.CheckReplay(p, r.Kind, r.PayloadHash, s.now())
	})
	return c, err
}

// Execute coordinates database-only effects. Callers must not perform external IO in apply.
// The transaction lock reconciles an earlier unknown local commit before any replay.
func (s *Executor) Execute(ctx context.Context, p household.Principal, r Request, apply func(context.Context) (command.Result, error)) (command.Command, error) {
	if _, err := s.Register(ctx, p, r); err != nil {
		return command.Command{}, err
	}
	var c command.Command
	err := s.transactions.WithinHousehold(ctx, p, func(ctx context.Context) error {
		var err error
		c, err = s.repository.LoadCommand(ctx, p, r.ID)
		if err != nil {
			return err
		}
		if err = c.CheckReplay(p, r.Kind, r.PayloadHash, s.now()); err != nil {
			return err
		}
		if c.Status() != command.Pending {
			return nil
		}
		result, err := apply(context.WithValue(ctx, commandContextKey{}, c.ID()))
		if err != nil {
			return err
		}
		c, err = c.Succeed(result, s.now())
		if err != nil {
			return err
		}
		return s.repository.SaveCommand(ctx, c)
	})
	var rejection Rejection
	if errors.As(err, &rejection) {
		// A rejected apply has rolled back all writes before recording its refusal.
		err = s.transactions.WithinHousehold(ctx, p, func(ctx context.Context) error {
			var e error
			c, e = s.repository.LoadCommand(ctx, p, r.ID)
			if e != nil {
				return e
			}
			if e = c.CheckReplay(p, r.Kind, r.PayloadHash, s.now()); e != nil {
				return e
			}
			if c.Status() != command.Pending {
				return nil
			}
			c, e = c.Fail(rejection.Code, s.now())
			if e != nil {
				return e
			}
			return s.repository.SaveCommand(ctx, c)
		})
	}
	return c, err
}
