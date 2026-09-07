package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	domain "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "Background process failed")
		os.Exit(1)
	}
}
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var bindings []connections.Binding
	// File content is deployment-owned and contains only non-secret exact artifact bindings.
	if path := os.Getenv("WANT_KEEP_JOB_BINDINGS_FILE"); path != "" {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		dec := json.NewDecoder(f)
		dec.DisallowUnknownFields()
		if err = dec.Decode(&bindings); err != nil {
			return err
		}
		if err = dec.Decode(new(any)); err != io.EOF {
			return errors.New("invalid binding file")
		}
		for _, b := range bindings {
			if b.Environment != os.Getenv("WANT_KEEP_ENV") {
				return errors.New("binding environment mismatch")
			}
			if err = b.Validate(); err != nil {
				return err
			}
		}
	}
	db, err := storage.Open(ctx, storage.Config{DSN: os.Getenv("WANT_KEEP_DATABASE_URL"), Environment: os.Getenv("WANT_KEEP_ENV"), MaxConnections: 8})
	if err != nil {
		return err
	}
	defer db.Close()
	report := func(kind domain.Kind, code string) { fmt.Fprintf(os.Stderr, "queue=%s code=%s\n", kind, code) }
	scheduler := jobs.Scheduler{Repository: db, Admission: admission.NewService(db, db), Bindings: bindings, Report: func(code string) { fmt.Fprintln(os.Stderr, code) }}
	var group sync.WaitGroup
	group.Add(1)
	go func() { defer group.Done(); _ = scheduler.Run(ctx) }()
	for _, kind := range []domain.Kind{domain.Sync, domain.Outbox, domain.AI} {
		var handler jobs.Handler
		if kind == domain.Outbox {
			handler = jobs.OutboxHandler{Repository: db}
		}
		worker := jobs.Worker{Repository: db, Handler: handler, Config: jobs.DefaultWorkerConfig(kind), Report: report}
		group.Add(1)
		go func() {
			defer group.Done()
			if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				report(worker.Config.Kind, "worker_stopped")
				stop()
			}
		}()
	}
	group.Wait()
	return ctx.Err()
}
