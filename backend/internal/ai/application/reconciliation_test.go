package application

import (
	"context"
	"errors"
	"testing"

	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
)

type reconciliationRecorder struct {
	called      bool
	evidenceRef string
}

func (r *reconciliationRecorder) ReconcileAI(_ context.Context, _ string, _ ReconciliationOutcome, _ ai.Cost, evidenceRef string) error {
	r.called = true
	r.evidenceRef = evidenceRef
	return nil
}

func TestReconciliationAcceptsOnlyStructuralEvidenceReference(t *testing.T) {
	for _, reference := range []string{"provider-dashboard:request-1", "https://provider.example/evidence?id=1"} {
		repository := &reconciliationRecorder{}
		service := NewReconciliationService(repository)
		if err := service.Reconcile(context.Background(), "request-1", Charged, ai.MustCost("0.25"), reference); err != nil {
			t.Fatalf("valid reference %q: %v", reference, err)
		}
		if !repository.called || repository.evidenceRef != reference {
			t.Fatalf("reference was not forwarded: %#v", repository)
		}
	}

	for _, reference := range []string{"provider dashboard: charged 0.25", "provider\nrequest", "provider/request(1)"} {
		repository := &reconciliationRecorder{}
		service := NewReconciliationService(repository)
		if err := service.Reconcile(context.Background(), "request-1", Charged, ai.MustCost("0.25"), reference); !errors.Is(err, ErrInvalidReconciliation) {
			t.Fatalf("unsafe reference %q accepted: %v", reference, err)
		}
		if repository.called {
			t.Fatalf("unsafe reference %q reached repository", reference)
		}
	}
}
