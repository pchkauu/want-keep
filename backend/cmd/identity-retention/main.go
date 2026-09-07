package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database, err := storage.Open(ctx, storage.Config{DSN: os.Getenv("WANT_KEEP_MAINTENANCE_DATABASE_URL"), Environment: os.Getenv("WANT_KEEP_ENV"), MaxConnections: 1})
	if err == nil {
		defer database.Close()
		err = database.CleanupIdentity(ctx, time.Now().UTC(), 1000)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Identity maintenance failed.")
		os.Exit(1)
	}
}
