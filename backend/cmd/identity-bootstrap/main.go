package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func main() {
	output := flag.String("output", "", "new private file for one-time bootstrap token")
	flag.Parse()
	if err := issue(*output); err != nil {
		fmt.Fprintln(os.Stderr, "Bootstrap token was not confirmed. Check setup state and the private output file before retrying.")
		os.Exit(1)
	}
	fmt.Println("Bootstrap token written to the requested private file; expires in 30 minutes.")
}
func issue(output string) error {
	if output == "" {
		return identity.ErrBootstrap
	}
	file, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	token, err := identity.NewToken(32)
	if err != nil {
		return err
	}
	if _, err = file.WriteString(string(token) + "\n"); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	database, err := storage.Open(ctx, storage.Config{DSN: os.Getenv("WANT_KEEP_MIGRATION_DATABASE_URL"), Environment: os.Getenv("WANT_KEEP_ENV"), MaxConnections: 1})
	if err != nil {
		return err
	}
	defer database.Close()
	return database.IssueIdentityBootstrap(ctx, token, time.Now().UTC())
}
