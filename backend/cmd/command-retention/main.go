package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func main() {
	mode := flag.String("mode", "", "details or tombstones; one bounded batch")
	batch := flag.Int("batch", 100, "rows per batch (1..1000)")
	flag.Parse()
	if (*mode != "details" && *mode != "tombstones") || *batch < 1 || *batch > 1000 {
		fmt.Fprintln(os.Stderr, "Valid mode and batch are required")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := storage.Open(ctx, storage.Config{DSN: os.Getenv("WANT_KEEP_MAINTENANCE_DATABASE_URL"), Environment: os.Getenv("WANT_KEEP_ENV"), MaxConnections: 1})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot open maintenance database")
		os.Exit(1)
	}
	defer db.Close()
	now, err := calendar.ParseInstant(time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		os.Exit(1)
	}
	retention := jobs.Retention{Repository: db}
	var count int64
	if *mode == "details" {
		count, err = retention.RunDetails(ctx, now, *batch)
	} else {
		count, err = retention.RunTombstones(ctx, now, *batch)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Retention batch failed")
		os.Exit(1)
	}
	fmt.Printf("Retention batch complete: %d rows\n", count)
}
