package application

import (
	"context"
	"errors"
	"regexp"
	"unicode/utf8"

	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
)

var ErrInvalidReconciliation = errors.New("invalid AI reconciliation")

var evidenceReference = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/?#=&%+~-]*$`)

type ReconciliationOutcome string

const (
	Charged    ReconciliationOutcome = "charged"
	NotCharged ReconciliationOutcome = "not_charged"
)

type ReconciliationRepository interface {
	ReconcileAI(context.Context, string, ReconciliationOutcome, ai.Cost, string) error
}

type ReconciliationService struct{ repository ReconciliationRepository }

func NewReconciliationService(repository ReconciliationRepository) *ReconciliationService {
	return &ReconciliationService{repository: repository}
}

func (s *ReconciliationService) Reconcile(ctx context.Context, requestID string, outcome ReconciliationOutcome, actual ai.Cost, evidenceRef string) error {
	if s.repository == nil || requestID == "" || utf8.RuneCountInString(evidenceRef) < 1 || utf8.RuneCountInString(evidenceRef) > 2000 || !evidenceReference.MatchString(evidenceRef) || (outcome != Charged && outcome != NotCharged) {
		return ErrInvalidReconciliation
	}
	if outcome == NotCharged {
		actual = ai.MustCost("0")
	} else if err := actual.Validate(); err != nil {
		return ErrInvalidReconciliation
	}
	return s.repository.ReconcileAI(ctx, requestID, outcome, actual, evidenceRef)
}
