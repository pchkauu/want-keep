package storage

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

var ErrStorage = errors.New("storage operation failed")
var ErrTransactionRequired = errors.New("transaction scope required")
var ErrNotFound = errors.New("resource not found")

type Store struct{ pool *pgxpool.Pool }

type Config struct {
	DSN, Environment string
	MaxConnections   int32
}

func Open(ctx context.Context, c Config) (*Store, error) {
	if c.MaxConnections < 1 || c.MaxConnections > 20 {
		return nil, errors.New("invalid connection limit")
	}
	cfg, err := pgxpool.ParseConfig(c.DSN)
	if err != nil {
		return nil, errors.New("invalid database configuration")
	}
	switch c.Environment {
	case "production":
		tls := cfg.ConnConfig.TLSConfig
		if tls == nil || tls.InsecureSkipVerify || tls.ServerName == "" || len(cfg.ConnConfig.Fallbacks) > 0 {
			return nil, errors.New("verified database TLS required")
		}
	case "development", "test":
		host := cfg.ConnConfig.Host
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return nil, errors.New("local database required")
		}
	default:
		return nil, errors.New("invalid environment")
	}
	cfg.MaxConns = c.MaxConnections
	cfg.MinConns = 0
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	cfg.ConnConfig.RuntimeParams["timezone"] = "UTC"
	cfg.ConnConfig.RuntimeParams["statement_timeout"] = "10000"
	cfg.ConnConfig.RuntimeParams["lock_timeout"] = "5000"
	cfg.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = "15000"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, ErrStorage
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, ErrStorage
	}
	return &Store{pool: pool}, nil
}
func (s *Store) Close() { s.pool.Close() }

type transactionKey struct{}
type transactionScope struct {
	store            *Store
	tx               pgx.Tx
	principal        household.Principal
	householdLocked  bool
	admissionKey     string
	syncJobID        string
	syncConnectionID string
	root             *transactionScope
	rollbackFailure  error
}

func (s *Store) scope(ctx context.Context) (*transactionScope, error) {
	scope, ok := ctx.Value(transactionKey{}).(*transactionScope)
	if !ok || scope.store != s {
		return nil, ErrTransactionRequired
	}
	return scope, nil
}
func (s *Store) familyScope(ctx context.Context) (*transactionScope, error) {
	scope, err := s.scope(ctx)
	if err != nil {
		return nil, err
	}
	if !scope.householdLocked {
		return nil, ErrTransactionRequired
	}
	return scope, nil
}
func (s *Store) transact(ctx context.Context, fn func(context.Context, *transactionScope) error) error {
	if existing := ctx.Value(transactionKey{}); existing != nil {
		scope, err := s.scope(ctx)
		if err != nil {
			return err
		}
		return s.nestedTransaction(ctx, scope, fn)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	scope := &transactionScope{store: s, tx: tx}
	scope.root = scope
	ctx = context.WithValue(ctx, transactionKey{}, scope)
	if err = fn(ctx, scope); err != nil {
		return err
	}
	if scope.rollbackFailure != nil {
		return scope.rollbackFailure
	}
	return tx.Commit(ctx)
}

func (s *Store) nestedTransaction(ctx context.Context, parent *transactionScope, fn func(context.Context, *transactionScope) error) error {
	if parent.root.rollbackFailure != nil {
		return parent.root.rollbackFailure
	}
	tx, err := parent.tx.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if rollbackErr := tx.Rollback(cleanup); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			parent.root.rollbackFailure = rollbackErr
		}
	}()
	child := *parent
	child.tx = tx
	nestedContext := context.WithValue(ctx, transactionKey{}, &child)
	if err = fn(nestedContext, &child); err != nil {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if rollbackErr := tx.Rollback(cleanup); rollbackErr != nil {
			parent.root.rollbackFailure = rollbackErr
			return errors.Join(err, rollbackErr)
		}
		return err
	}
	if parent.root.rollbackFailure != nil {
		return parent.root.rollbackFailure
	}
	if err = tx.Commit(ctx); err != nil {
		// Unknown savepoint release must never be converted into a successful outer commit.
		parent.root.rollbackFailure = err
		return err
	}
	parent.principal, parent.householdLocked = child.principal, child.householdLocked
	parent.admissionKey = child.admissionKey
	parent.syncJobID, parent.syncConnectionID = child.syncJobID, child.syncConnectionID
	return nil
}

// WithinNewHousehold rejects ambient transactions: command registration must be durable first.
func (s *Store) WithinNewHousehold(ctx context.Context, p household.Principal, fn func(context.Context) error) error {
	if ctx.Value(transactionKey{}) != nil {
		return ErrTransactionRequired
	}
	return s.WithinHousehold(ctx, p, fn)
}

func (s *Store) WithinHousehold(ctx context.Context, p household.Principal, fn func(context.Context) error) error {
	if err := p.RequireHousehold(p.HouseholdID()); err != nil {
		return err
	}
	return s.transact(ctx, func(ctx context.Context, scope *transactionScope) error {
		if scope.householdLocked {
			if scope.principal != p {
				return household.ErrForbidden
			}
			return fn(ctx)
		}
		var id string
		if err := scope.tx.QueryRow(ctx, "SELECT id FROM want_keep.households WHERE id=$1 FOR UPDATE", p.HouseholdID()).Scan(&id); err != nil {
			return household.ErrForbidden
		}
		var active bool
		if err := scope.tx.QueryRow(ctx, "SELECT active FROM want_keep.memberships WHERE household_id=$1 AND user_id=$2", p.HouseholdID(), p.UserID()).Scan(&active); err != nil || !active {
			return household.ErrForbidden
		}
		scope.principal = p
		scope.householdLocked = true
		return fn(ctx)
	})
}
func (s *Store) requireRead(ctx context.Context, p household.Principal) error {
	if err := p.RequireHousehold(p.HouseholdID()); err != nil {
		return err
	}
	var active bool
	if err := s.pool.QueryRow(ctx, "SELECT active FROM want_keep.memberships WHERE household_id=$1 AND user_id=$2", p.HouseholdID(), p.UserID()).Scan(&active); err != nil || !active {
		return household.ErrForbidden
	}
	return nil
}
func (s *Store) WithinAdmission(ctx context.Context, provider, environment string, fn func(context.Context) error) error {
	key := fmt.Sprintf("%d:%s%s", len(provider), provider, environment)
	return s.transact(ctx, func(ctx context.Context, scope *transactionScope) error {
		if scope.householdLocked {
			return errors.New("admission lock must precede household lock")
		}
		if scope.admissionKey != "" && scope.admissionKey != key {
			return errors.New("multiple admission locks prohibited")
		}
		if _, err := scope.tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 13))", key); err != nil {
			return err
		}
		scope.admissionKey = key
		return fn(ctx)
	})
}

func (scope *transactionScope) holdsAdmission(provider, environment string) bool {
	return scope.admissionKey == fmt.Sprintf("%d:%s%s", len(provider), provider, environment)
}
