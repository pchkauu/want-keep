package collector

import (
	"context"
	"time"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

const (
	reconciliationBatch    = 100
	reconciliationInterval = 30 * time.Second
)

type StagedReconciler struct {
	Principals func(context.Context, int) ([]household.Principal, error)
	Reconcile  func(context.Context, household.Principal, int) (int, error)
	Report     func(error)
}

func (r StagedReconciler) Step(ctx context.Context) error {
	principals, err := r.Principals(ctx, reconciliationBatch)
	if err != nil {
		return err
	}
	for _, principal := range principals {
		if _, err = r.Reconcile(ctx, principal, reconciliationBatch); err != nil {
			return err
		}
	}
	return nil
}

func (r StagedReconciler) Run(ctx context.Context) error {
	if r.Principals == nil || r.Reconcile == nil || r.Report == nil {
		return ErrUnavailable
	}
	ticker := time.NewTicker(reconciliationInterval)
	defer ticker.Stop()
	for {
		if err := r.Step(ctx); err != nil && ctx.Err() == nil {
			r.Report(err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
