package storage

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrate is an operator command, never part of an application request or startup fallback.
func Migrate(ctx context.Context, pool *pgxpool.Pool, files fs.FS) error {
	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return err
	}
	slices.Sort(names)
	if len(names) == 0 {
		return fmt.Errorf("no migrations")
	}
	for i, name := range names {
		version, err := strconv.Atoi(strings.SplitN(name, "_", 2)[0])
		if err != nil || version != i+1 {
			return fmt.Errorf("nonsequential migration %s", name)
		}
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err = conn.Exec(ctx, "SELECT pg_advisory_lock(130013)"); err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, e := conn.Exec(cleanup, "SELECT pg_advisory_unlock(130013)"); e != nil {
			_ = conn.Conn().Close(cleanup)
		}
	}()
	if _, err = conn.Exec(ctx, "CREATE TABLE IF NOT EXISTS public.want_keep_schema_migrations (version integer PRIMARY KEY, name text NOT NULL, checksum text NOT NULL)"); err != nil {
		return err
	}
	rows, err := conn.Query(ctx, "SELECT version,name,checksum FROM public.want_keep_schema_migrations ORDER BY version")
	if err != nil {
		return err
	}
	applied := 0
	for rows.Next() {
		var version int
		var name, checksum string
		if err = rows.Scan(&version, &name, &checksum); err != nil {
			rows.Close()
			return err
		}
		if version != applied+1 || version > len(names) || name != names[version-1] {
			rows.Close()
			return fmt.Errorf("unknown migration history")
		}
		data, e := fs.ReadFile(files, name)
		if e != nil {
			rows.Close()
			return e
		}
		if fmt.Sprintf("%x", sha256.Sum256(data)) != checksum {
			rows.Close()
			return fmt.Errorf("migration checksum mismatch: %s", name)
		}
		applied++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for i := applied; i < len(names); i++ {
		data, err := fs.ReadFile(files, names[i])
		if err != nil {
			return err
		}
		err = func() error {
			tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return err
			}
			defer tx.Rollback(context.Background())
			if _, err = tx.Exec(ctx, string(data)); err != nil {
				return fmt.Errorf("migration %s: %w", names[i], err)
			}
			if _, err = tx.Exec(ctx, "INSERT INTO public.want_keep_schema_migrations(version,name,checksum) VALUES($1,$2,$3)", i+1, names[i], fmt.Sprintf("%x", sha256.Sum256(data))); err != nil {
				return err
			}
			return tx.Commit(ctx)
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ApplyMigrations(ctx context.Context, files fs.FS) error {
	return Migrate(ctx, s.pool, files)
}
