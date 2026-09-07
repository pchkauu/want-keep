package application

import (
	"context"

	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	access "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

type SessionTransactions interface {
	WithinSession(context.Context, access.Token, func(context.Context, identity.Access) error) error
}
type Authenticated struct {
	executor *Executor
	sessions SessionTransactions
}

func NewAuthenticated(e *Executor, s SessionTransactions) *Authenticated { return &Authenticated{e, s} }

// Registration is independently durable. Session revalidation then precedes the
// household lock, including command replay and terminal rejection persistence.
func (s *Authenticated) Execute(ctx context.Context, a identity.Access, r Request, apply func(context.Context) (command.Result, error)) (command.Command, error) {
	if _, err := s.executor.Register(ctx, a.Principal, r); err != nil {
		return command.Command{}, err
	}
	var c command.Command
	err := s.sessions.WithinSession(ctx, a.Token, func(ctx context.Context, current identity.Access) error {
		if current.Principal != a.Principal {
			return household.ErrForbidden
		}
		var err error
		c, err = s.executor.ExecuteRegistered(ctx, current.Principal, r, apply)
		return err
	})
	return c, err
}
