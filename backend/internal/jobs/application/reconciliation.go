package application

import (
	"context"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

type Reconciliation struct {
	Job         jobs.Job
	EvidenceRef string
	Outcome     string
}

// This port is available only to trusted reconciliation code, never user/model commands.
type ReconciliationRepository interface {
	ReconcileJob(context.Context, household.Principal, Reconciliation, Effect) error
}
