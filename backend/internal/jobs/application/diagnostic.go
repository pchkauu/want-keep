package application

import (
	"context"
	"errors"
	"time"

	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

// Diagnostics contain only persisted identifiers and closed codes, never error text or payloads.
type Diagnostic struct {
	Kind                               jobs.Kind
	JobID, ConnectionID, TransactionID string
	Stage, Code                        string
	Duration                           time.Duration
}

func (d Diagnostic) Failure(err error, elapsed time.Duration) Diagnostic {
	d.Duration = elapsed
	switch {
	case errors.Is(err, jobs.ErrStaleAttempt):
		d.Code = "stale_attempt"
	case errors.Is(err, household.ErrForbidden):
		d.Code = "forbidden"
	case errors.Is(err, connections.ErrProviderNotAdmitted):
		d.Code = "provider_not_admitted"
	case errors.Is(err, jobs.ErrInvalidJob), errors.Is(err, connections.ErrInvalidAdmission):
		d.Code = "invalid_input"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		d.Code = "canceled"
	default:
		d.Code = "execution_failed"
	}
	return d
}
