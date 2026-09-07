package storage

import (
	"context"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

// WithinFinancialRead gives a consistent page without taking financial write locks.
func (s *Store) WithinFinancialRead(ctx context.Context, p household.Principal, fn func(context.Context) error) error {
	if ctx.Value(transactionKey{}) != nil {
		return ErrTransactionRequired
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	var active bool
	if err = tx.QueryRow(ctx, `SELECT active FROM want_keep.memberships WHERE household_id=$1 AND user_id=$2`, p.HouseholdID(), p.UserID()).Scan(&active); err != nil || !active {
		return household.ErrForbidden
	}
	scope := &transactionScope{store: s, tx: tx, principal: p, readOnly: true}
	scope.root = scope
	if err = fn(context.WithValue(ctx, transactionKey{}, scope)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
