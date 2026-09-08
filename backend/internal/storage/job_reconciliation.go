package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/pchkauu/want-keep/backend/internal/connections/admission"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	domain "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func (s *Store) ReconcileJob(ctx context.Context, p household.Principal, r jobs.Reconciliation, apply jobs.Effect) error {
	if r.Validate() != nil || (r.Outcome == "absent" && apply != nil) || (r.Page != nil && apply == nil) {
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
			if current.State != domain.Unresolved || current.LeaseToken != r.Job.LeaseToken || current.Attempt != r.Job.Attempt || current.ActorID != p.UserID() || current.Kind != r.Job.Kind || current.Binding != r.Job.Binding || current.ResourceID != r.Job.ResourceID || current.ResourceRevision != r.Job.ResourceRevision || current.ConnectionID != r.Job.ConnectionID || current.ConnectionGeneration != r.Job.ConnectionGeneration || current.AdmissionRevision != r.Job.AdmissionRevision || current.SecretPurpose != r.Job.SecretPurpose {
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
				if !c.Authorized || c.Generation != current.ConnectionGeneration || c.Provider != current.Binding.Provider || c.SecretPurpose != current.SecretPurpose || current.CancelRequested || current.Cursor != r.Page.Cursor {
					return domain.ErrStaleAttempt
				}
				scope.syncJobID, scope.syncConnectionID = current.ID, current.ConnectionID
			}
			if apply != nil {
				if err = apply(ctx, p); err != nil {
					return err
				}
			}
			if r.Page != nil {
				if err = s.saveReconciledPage(ctx, p, current, *r.Page); err != nil {
					return err
				}
			}
			_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.job_reconciliations(household_id,job_id,lease_token,evidence_ref,outcome,actor_id) VALUES($1,$2,$3,$4,$5,$6)`, p.HouseholdID(), current.ID, current.LeaseToken, r.EvidenceRef, r.Outcome, p.UserID())
			if err != nil {
				return err
			}
			resolution := domain.EffectConfirmed
			if r.Outcome == "absent" {
				resolution = domain.EffectAbsent
			} else if r.Page != nil && !r.Page.Complete {
				resolution = domain.PageConfirmed
			}
			var replacement bool
			if current.Kind == domain.Sync && r.Outcome == "absent" {
				// Legacy queues could already have a replacement; preserve its continuation.
				err = scope.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.jobs WHERE household_id=$1 AND connection_id=$2 AND id!=$3 AND state IN ('ready','running','waiting') AND NOT cancel_requested)`, p.HouseholdID(), current.ConnectionID, current.ID).Scan(&replacement)
				if err != nil {
					return err
				}
			}
			outcome, err := current.Reconcile(resolution, replacement)
			if err != nil {
				return err
			}
			if current.Kind == domain.Sync && r.Outcome == "confirmed" {
				// Every confirmed page owns the newest checkpoint, including terminal attempts.
				rows, err := scope.tx.Query(ctx, `SELECT `+jobColumns+` FROM want_keep.jobs WHERE household_id=$1 AND connection_id=$2 AND id!=$3 AND state IN ('ready','running','waiting') AND NOT cancel_requested ORDER BY id FOR UPDATE`, p.HouseholdID(), current.ConnectionID, current.ID)
				if err != nil {
					return err
				}
				if err = s.cancelJobs(ctx, scope.tx, rows); err != nil {
					return err
				}
			}
			if outcome.State == domain.Succeeded {
				if err = s.insertJobReceipt(ctx, p, current); err != nil {
					return err
				}
			}
			_, err = scope.tx.Exec(ctx, `UPDATE want_keep.jobs SET state=$3,external_started=false,reason=$4,run_deadline=clock_timestamp()+INTERVAL '24 hours',available_at=clock_timestamp() WHERE household_id=$1 AND id=$2`, p.HouseholdID(), current.ID, outcome.State, outcome.Reason)
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

// The caller holds admission, household and the verified unresolved attempt in one transaction.
func (s *Store) saveReconciledPage(ctx context.Context, p household.Principal, j domain.Job, page admission.Page) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	omissions, err := s.ImportOmissions(ctx, p, j.ID)
	if err != nil {
		return err
	}
	page = page.WithOmissions(append(append([]string{}, j.Gaps...), omissions...))
	_, err = scope.tx.Exec(ctx, `UPDATE want_keep.jobs SET cursor=$3,coverage=$4,gaps=$5 WHERE household_id=$1 AND id=$2`, p.HouseholdID(), j.ID, page.NextCursor, page.Coverage, page.Gaps)
	if err != nil {
		return err
	}
	if err = s.saveSyncProgress(ctx, j, page.NextCursor, page.Coverage, page.Gaps); err != nil {
		return err
	}
	if page.Complete {
		_, err = scope.tx.Exec(ctx, `UPDATE want_keep.sync_progress SET completed=true,last_success_at=clock_timestamp() WHERE household_id=$1 AND connection_id=$2`, p.HouseholdID(), j.ConnectionID)
	}
	return err
}
