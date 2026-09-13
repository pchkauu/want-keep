package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	aiapp "github.com/pchkauu/want-keep/backend/internal/ai/application"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "AI reconciliation failed")
		os.Exit(1)
	}
}

func run() error {
	requestID := flag.String("request-id", "", "persisted AI request ID")
	outcome := flag.String("outcome", "", "charged or not_charged")
	actualUSD := flag.String("actual-usd", "", "exact actual USD cost; required when charged")
	evidenceRef := flag.String("evidence-ref", "", "safe evidence reference")
	flag.Parse()
	if flag.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	costText := *actualUSD
	if *outcome == string(aiapp.NotCharged) && costText == "" {
		costText = "0"
	}
	cost, err := ai.NewCost(costText)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := storage.Open(ctx, storage.Config{
		DSN: os.Getenv("WANT_KEEP_MAINTENANCE_DATABASE_URL"), Environment: os.Getenv("WANT_KEEP_ENV"), MaxConnections: 2,
	})
	if err != nil {
		return err
	}
	defer db.Close()
	service := aiapp.NewReconciliationService(db)
	return service.Reconcile(ctx, *requestID, aiapp.ReconciliationOutcome(*outcome), cost, *evidenceRef)
}
