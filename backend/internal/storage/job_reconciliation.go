package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	domain "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func (s *Store) ReconcileJob(ctx context.Context, p household.Principal, r jobs.Reconciliation, apply jobs.Effect) error {
	if r.EvidenceRef == "" || len(r.EvidenceRef) > 2000 || (r.Outcome != "confirmed" && r.Outcome != "absent") || (r.Outcome == "absent" && apply != nil) {
		return domain.ErrInvalidJob
	}
	if r.Job.HouseholdID != p.HouseholdID() || r.Job.ActorID != p.UserID() {
		return household.ErrForbidden
	}
	run := func(ctx context.Context) error {
		return s.WithinHousehold(ctx, p, func(ctx context.Context) error {
			scope, _ := s.familyScope(ctx)
			var prior, evidence string
			err := scope.tx.QueryRow(ctx, `SELECT outcome,evidence_ref FROM want_keep.job_reconciliations WHERE household_id=$1 AND job_id=$2 AND lease_token=$3`, p.HouseholdID(), r.Job.ID, r.Job.LeaseToken).Scan(&prior, &evidence)
			if err == nil {
				if prior != r.Outcome || evidence != r.EvidenceRef {
					return domain.ErrStaleAttempt
				}
				return nil
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			current, err := scanJob(scope.tx.QueryRow(ctx, `SELECT `+jobColumns+` FROM want_keep.jobs WHERE household_id=$1 AND id=$2 FOR UPDATE`, p.HouseholdID(), r.Job.ID))
			if err != nil {
				return err
			}
			if current.State != domain.Unresolved || current.LeaseToken != r.Job.LeaseToken || current.Attempt != r.Job.Attempt || current.ActorID != p.UserID() || current.Kind != r.Job.Kind || current.Binding != r.Job.Binding || current.ResourceID != r.Job.ResourceID || current.ResourceRevision != r.Job.ResourceRevision {
				return domain.ErrStaleAttempt
			}
			if r.Outcome == "confirmed" && current.Kind == domain.Sync {
				a, _, err := s.Admission(ctx, current.Binding.Provider, current.Binding.Environment)
				if err != nil {
					return err
				}
				if err = a.RequireResult(current.Binding, current.AdmissionRevision); err != nil {
					return err
				}
				c, err := s.Connection(ctx, p, current.ConnectionID)
				if err != nil {
					return err
				}
				if !c.Authorized || c.Generation != current.ConnectionGeneration || current.CancelRequested {
					return domain.ErrStaleAttempt
				}
			}
			if apply != nil {
				if err = apply(ctx, p); err != nil {
					return err
				}
			}
			_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.job_reconciliations(household_id,job_id,lease_token,evidence_ref,outcome,actor_id) VALUES($1,$2,$3,$4,$5,$6)`, p.HouseholdID(), current.ID, current.LeaseToken, r.EvidenceRef, r.Outcome, p.UserID())
			if err != nil {
				return err
			}
			state := "succeeded"
			if r.Outcome == "absent" {
				state = "ready"
				if current.CancelRequested {
					state = "canceled"
				} else if current.Attempt >= current.MaxAttempts {
					state = "failed"
				}
			}
			_, err = scope.tx.Exec(ctx, `UPDATE want_keep.jobs SET state=$3,external_started=false,reason='',run_deadline=clock_timestamp()+INTERVAL '24 hours',available_at=clock_timestamp() WHERE household_id=$1 AND id=$2`, p.HouseholdID(), current.ID, state)
			return err
		})
	}
	if r.Job.Kind == domain.Sync {
		if err := r.Job.Binding.Validate(); err != nil {
			return err
		}
		return s.WithinAdmission(ctx, r.Job.Binding.Provider, r.Job.Binding.Environment, run)
	}
	return run(ctx)
}
