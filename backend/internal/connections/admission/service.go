package admission

import (
	"context"
	"errors"
	"time"

	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

type Connection struct {
	HouseholdID  household.HouseholdID
	ID, Provider string
	Owner        household.UserID
	Generation   uint64
	Authorized   bool
}
type Transactions interface {
	WithinAdmission(context.Context, string, string, func(context.Context) error) error
	WithinHousehold(context.Context, household.Principal, func(context.Context) error) error
}
type Repository interface {
	Admission(context.Context, string, string) (connections.Admission, bool, error)
	SaveAdmission(context.Context, connections.Admission) error
	InvalidateJobs(context.Context, connections.Admission) error
	Connection(context.Context, household.Principal, string) (Connection, error)
	CreateSyncJob(context.Context, Connection, connections.Admission, time.Time) (jobs.Job, error)
	Job(context.Context, household.Principal, string) (jobs.Job, error)
	DatabaseTime(context.Context) (time.Time, error)
	FenceSyncResult(context.Context, household.Principal, jobs.Job) error
	Quarantine(context.Context, jobs.Job, string, string) error
	SaveCheckpoint(context.Context, jobs.Job, string, string, []string) error
	FinishJob(context.Context, household.Principal, jobs.Job) error
}

// Service is the trusted server application boundary. User/AI commands cannot supply gate evidence.
type Service struct {
	transactions Transactions
	repository   Repository
}

func NewService(t Transactions, r Repository) *Service { return &Service{t, r} }
func (s *Service) RecordCheck(ctx context.Context, check connections.Check) (connections.Admission, error) {
	var a connections.Admission
	if err := check.Binding.Validate(); err != nil {
		return a, err
	}
	err := s.transactions.WithinAdmission(ctx, check.Binding.Provider, check.Binding.Environment, func(ctx context.Context) error {
		var exists bool
		var err error
		a, exists, err = s.repository.Admission(ctx, check.Binding.Provider, check.Binding.Environment)
		if err != nil {
			return err
		}
		if !exists {
			a, err = connections.NewAdmission(check.Binding)
			if err != nil {
				return err
			}
		}
		previous := a.Revision()
		a, err = a.RecordCheck(check)
		if err != nil {
			return err
		}
		if !exists || a.Revision() != previous {
			if err = s.repository.SaveAdmission(ctx, a); err != nil {
				return err
			}
			return s.repository.InvalidateJobs(ctx, a)
		}
		return nil
	})
	return a, err
}
func (s *Service) Rebind(ctx context.Context, b connections.Binding) (connections.Admission, error) {
	var a connections.Admission
	if err := b.Validate(); err != nil {
		return a, err
	}
	err := s.transactions.WithinAdmission(ctx, b.Provider, b.Environment, func(ctx context.Context) error {
		var exists bool
		var err error
		a, exists, err = s.repository.Admission(ctx, b.Provider, b.Environment)
		if err != nil {
			return err
		}
		previous := a.Revision()
		a, err = a.Rebind(b)
		if err != nil {
			return err
		}
		if !exists || a.Revision() != previous {
			if err = s.repository.SaveAdmission(ctx, a); err != nil {
				return err
			}
			return s.repository.InvalidateJobs(ctx, a)
		}
		return nil
	})
	return a, err
}
func (s *Service) RequestSync(ctx context.Context, p household.Principal, id string, b connections.Binding, deadline time.Time) (jobs.Job, error) {
	var result jobs.Job
	if err := b.Validate(); err != nil {
		return result, connections.ErrProviderNotAdmitted
	}
	err := s.transactions.WithinAdmission(ctx, b.Provider, b.Environment, func(ctx context.Context) error {
		a, _, err := s.repository.Admission(ctx, b.Provider, b.Environment)
		if err != nil {
			return err
		}
		if err = a.RequireSync(b); err != nil {
			return err
		}
		return s.transactions.WithinHousehold(ctx, p, func(ctx context.Context) error {
			c, err := s.repository.Connection(ctx, p, id)
			if err != nil {
				return err
			}
			if err = (connections.ExternalOwnership{HouseholdID: c.HouseholdID, OwnerID: c.Owner}).RequireManage(p); err != nil {
				return err
			}
			if !c.Authorized || c.Provider != b.Provider {
				return connections.ErrProviderNotAdmitted
			}
			result, err = s.repository.CreateSyncJob(ctx, c, a, deadline)
			return err
		})
	})
	return result, err
}
func (s *Service) BeforeRead(ctx context.Context, p household.Principal, issued jobs.Job) error {
	if err := issued.Binding.Validate(); err != nil {
		return connections.ErrProviderNotAdmitted
	}
	return s.transactions.WithinAdmission(ctx, issued.Binding.Provider, issued.Binding.Environment, func(ctx context.Context) error {
		return s.transactions.WithinHousehold(ctx, p, func(ctx context.Context) error { return s.requireCurrent(ctx, p, issued) })
	})
}
func (s *Service) requireCurrent(ctx context.Context, p household.Principal, issued jobs.Job) error {
	if issued.HouseholdID != p.HouseholdID() {
		return household.ErrForbidden
	}
	a, _, err := s.repository.Admission(ctx, issued.Binding.Provider, issued.Binding.Environment)
	if err != nil {
		return err
	}
	if err = a.RequireResult(issued.Binding, issued.AdmissionRevision); err != nil {
		return err
	}
	current, err := s.repository.Job(ctx, p, issued.ID)
	if err != nil {
		return err
	}
	now, err := s.repository.DatabaseTime(ctx)
	if err != nil {
		return err
	}
	if err = current.RequireAttempt(issued, now); err != nil {
		return err
	}
	c, err := s.repository.Connection(ctx, p, current.ConnectionID)
	if err != nil {
		return err
	}
	if !c.Authorized || c.Generation != issued.ConnectionGeneration || c.Provider != issued.Binding.Provider {
		return connections.ErrProviderNotAdmitted
	}
	return nil
}

type Page struct {
	EvidenceRef, Cursor, NextCursor, Coverage string
	Gaps                                      []string
	Complete                                  bool
}

func (s *Service) CommitPage(ctx context.Context, p household.Principal, issued jobs.Job, page Page, apply func(context.Context) error) (bool, error) {
	if issued.HouseholdID != p.HouseholdID() {
		return false, household.ErrForbidden
	}
	if (page.Coverage == "complete") != (len(page.Gaps) == 0) {
		return false, jobs.ErrInvalidJob
	}
	if page.EvidenceRef == "" || len(page.EvidenceRef) > 2000 || (page.Coverage != "complete" && page.Coverage != "partial" && page.Coverage != "unavailable") {
		return false, jobs.ErrInvalidJob
	}
	if err := issued.Binding.Validate(); err != nil {
		return false, connections.ErrProviderNotAdmitted
	}
	applied := false
	err := s.transactions.WithinAdmission(ctx, issued.Binding.Provider, issued.Binding.Environment, func(ctx context.Context) error {
		return s.transactions.WithinHousehold(ctx, p, func(ctx context.Context) error {
			current, err := s.repository.Job(ctx, p, issued.ID)
			if err != nil {
				return err
			}
			if err = s.requireCurrent(ctx, p, issued); err != nil {
				return err
			}
			if current.Cursor != page.Cursor {
				return jobs.ErrStaleAttempt
			}
			if err = s.repository.FenceSyncResult(ctx, p, issued); err != nil {
				return err
			}
			if err = apply(ctx); err != nil {
				return err
			}
			if err = s.repository.SaveCheckpoint(ctx, issued, page.NextCursor, page.Coverage, page.Gaps); err != nil {
				return err
			}
			if page.Complete {
				if err = s.repository.FinishJob(ctx, p, issued); err != nil {
					return err
				}
			}
			applied = true
			return nil
		})
	})
	if errors.Is(err, jobs.ErrStaleAttempt) || errors.Is(err, connections.ErrProviderNotAdmitted) {
		// The failed transaction has rolled back before retaining the stale evidence.
		return false, s.quarantineResult(ctx, p, issued.ID, page.EvidenceRef)
	}
	return applied && err == nil, err
}

func (s *Service) quarantineResult(ctx context.Context, p household.Principal, jobID, evidence string) error {
	return s.transactions.WithinHousehold(ctx, p, func(ctx context.Context) error {
		current, err := s.repository.Job(ctx, p, jobID)
		if err != nil {
			return err
		}
		return s.repository.Quarantine(ctx, current, evidence, "stale_result")
	})
}
