package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	dsn := os.Getenv("WANT_KEEP_MIGRATION_DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "WANT_KEEP_MIGRATION_DATABASE_URL is required")
		os.Exit(2)
	}
	database, err := storage.Open(ctx, storage.Config{DSN: dsn, Environment: os.Getenv("WANT_KEEP_ENV"), MaxConnections: 1})
	if err == nil {
		defer database.Close()
		err = database.ApplyMigrations(ctx, migrations.Files)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Database migration failed; inspect private operator diagnostics.")
		os.Exit(1)
	}
	fmt.Println("Database migrations are current.")
}
