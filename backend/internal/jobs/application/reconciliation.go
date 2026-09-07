package application

import (
	"context"

	"github.com/pchkauu/want-keep/backend/internal/connections/admission"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

type Reconciliation struct {
	Job         jobs.Job
	EvidenceRef string
	Outcome     string
	Page        *admission.Page
}

// This port is available only to trusted reconciliation code, never user/model commands.
type ReconciliationRepository interface {
	ReconcileJob(context.Context, household.Principal, Reconciliation, Effect) error
}

func (r Reconciliation) Validate() error {
	if r.EvidenceRef == "" || len(r.EvidenceRef) > 2000 || (r.Outcome != "confirmed" && r.Outcome != "absent") {
		return jobs.ErrInvalidJob
	}
	if r.Job.Kind == jobs.Sync && r.Outcome == "confirmed" {
		if r.Page == nil || r.Page.EvidenceRef != r.EvidenceRef {
			return jobs.ErrInvalidJob
		}
		return r.Page.Validate()
	}
	if r.Page != nil {
		return jobs.ErrInvalidJob
	}
	return nil
}
