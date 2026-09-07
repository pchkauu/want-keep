package storage

import (
	"context"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

type reader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (s *Store) reader(ctx context.Context, p household.Principal) (reader, error) {
	if ctx.Value(transactionKey{}) != nil {
		scope, err := s.scope(ctx)
		if err != nil {
			return nil, err
		}
		if !scope.householdLocked || scope.principal != p {
			return nil, household.ErrForbidden
		}
		return scope.tx, nil
	}
	if err := s.requireRead(ctx, p); err != nil {
		return nil, err
	}
	return s.pool, nil
}
