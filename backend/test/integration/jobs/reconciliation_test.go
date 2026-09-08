//go:build integration

package storage_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	app "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestConfirmedSyncPageReconciliation(t *testing.T) {
	for _, test := range []struct {
		name                  string
		complete, lastAttempt bool
	}{
		{name: "continue"},
		{name: "complete", complete: true},
		{name: "last_attempt", lastAttempt: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			connection := f.connection()
			b := binding()
			gate := f.admit(b)
			issued := f.issued(gate, connection, b)
			if test.lastAttempt {
				if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET attempt=max_attempts WHERE id=$1`, issued.ID); err != nil {
					t.Fatal(err)
				}
				issued.Attempt = issued.MaxAttempts
			}
			account := f.account(money.RUB, "1000")
			revision := f.revision(uuid.NewString(), account, "-100", money.RUB, 1)
			input := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: b.Provider, ExternalAccountID: "synthetic-stable", Product: "current", Log: "statement", RecordID: "confirmed"}, PayloadHash: strings.Repeat("a", 64), EvidenceRef: "synthetic:confirmed", Classification: "new", Operation: &revision, ConnectionID: connection, JobID: issued.ID, FetchedAt: f.now}
			if err := f.store.BeginExternal(testContext, f.p, issued); err != nil {
				t.Fatal(err)
			}
			if err := f.store.SetJobOutcome(testContext, f.p, issued, jobs.Unresolved, jobs.ExternalUnknown, 0); err != nil {
				t.Fatal(err)
			}
			replacement := uuid.NewString()
			_, err := f.admin.Exec(testContext, `INSERT INTO want_keep.jobs(household_id,id,actor_id,kind,connection_id,connection_generation,binding,admission_revision,state,max_attempts,available_at,deadline,secret_purpose) SELECT household_id,$2,actor_id,kind,connection_id,connection_generation,binding,admission_revision,'ready',5,clock_timestamp(),deadline,secret_purpose FROM want_keep.jobs WHERE id=$1`, issued.ID, replacement)
			if err != nil {
				t.Fatal(err)
			}
			r := app.Reconciliation{Job: issued, EvidenceRef: input.EvidenceRef, Outcome: "confirmed", Page: &admission.Page{EvidenceRef: input.EvidenceRef, NextCursor: "page2", Coverage: "complete", Complete: test.complete}}
			source := journal.NewSources(f.store, f.writer)
			apply := func(ctx context.Context, p household.Principal) error {
				_, err := source.Apply(ctx, p, input)
				return err
			}
			fail := errors.New("synthetic rollback")
			if err := f.store.ReconcileJob(testContext, f.p, r, func(ctx context.Context, p household.Principal) error {
				if err := apply(ctx, p); err != nil {
					return err
				}
				return fail
			}); !errors.Is(err, fail) {
				t.Fatal(err)
			}
			if f.count("source_records") != 0 || f.count("job_reconciliations") != 0 || f.available(account) != "1000" {
				t.Fatal("partial reconciliation committed")
			}
			for range 2 {
				if err := f.store.ReconcileJob(testContext, f.p, r, apply); err != nil {
					t.Fatal(err)
				}
			}
			if f.count("source_records") != 1 || f.count("source_provenance") != 1 || f.count("job_reconciliations") != 1 || f.available(account) != "900" {
				t.Fatal("confirmed effect lost or duplicated")
			}
			progress, err := f.store.SyncProgress(testContext, f.p, connection)
			if err != nil || progress.Cursor != "page2" || progress.Completed != test.complete || (progress.LastSuccessAt != nil) != test.complete {
				t.Fatal("reconciliation checkpoint", err)
			}
			current, err := f.store.Job(testContext, f.p, issued.ID)
			if err != nil || current.ExternalStarted {
				t.Fatal(err)
			}
			sibling, err := f.store.Job(testContext, f.p, replacement)
			if err != nil || sibling.State != jobs.Canceled || !sibling.CancelRequested {
				t.Fatal("obsolete continuation retained", err)
			}
			if test.complete {
				if current.State != jobs.Succeeded || f.count("job_receipts") != 1 {
					t.Fatal("terminal receipt missing")
				}
				rows, err := f.restart().ClaimJobs(testContext, "sync", 1, time.Minute)
				if err != nil || len(rows) != 0 {
					t.Fatal("completed source repeated", err)
				}
			} else {
				next := f.issued(gate, connection, b)
				if next.LeaseToken == issued.LeaseToken || next.Cursor != "page2" || next.ID == replacement {
					t.Fatal("confirmed page was repeated")
				}
				if test.lastAttempt {
					if current.State != jobs.Failed || current.Reason != "attempts_exhausted" || next.ID == issued.ID {
						t.Fatal("attempt limit ignored")
					}
				} else if next.ID != issued.ID {
					t.Fatal("unfinished attempt replaced")
				}
			}
		})
	}
}

func TestUnresolvedSurvivesSourceInvalidation(t *testing.T) {
	for _, action := range []string{"rebind", "disconnect"} {
		t.Run(action, func(t *testing.T) {
			f := newFixture(t)
			connection := f.connection()
			b := binding()
			gate := f.admit(b)
			issued := f.issued(gate, connection, b)
			// The retained ambiguous-result contract predates the external-start marker.
			if err := f.store.RetryJob(testContext, f.p, issued, 0, true); err != nil {
				t.Fatal(err)
			}
			if action == "rebind" {
				b.ContractVersion = "11"
				if _, err := gate.Rebind(testContext, b); err != nil {
					t.Fatal(err)
				}
				gate = f.admit(b)
			} else {
				if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.store.Disconnect(ctx, connection) }); err != nil {
					t.Fatal(err)
				}
			}
			current, err := f.store.Job(testContext, f.p, issued.ID)
			if err != nil || current.State != jobs.Unresolved || !current.CancelRequested || current.ExternalStarted {
				t.Fatal("uncertainty lost", err)
			}
			if _, err = gate.RequestSync(testContext, f.p, connection, b, time.Now().Add(time.Hour)); err == nil {
				t.Fatal("refresh crossed unresolved barrier")
			}
			rows, err := f.store.ClaimJobs(testContext, "sync", 10, time.Minute)
			if err != nil || len(rows) != 0 {
				t.Fatal("unknown action repeated", err)
			}
			r := app.Reconciliation{Job: issued, EvidenceRef: "synthetic:stale", Outcome: "confirmed", Page: &admission.Page{EvidenceRef: "synthetic:stale", Coverage: "complete", Complete: true}}
			calls := 0
			if err = f.store.ReconcileJob(testContext, f.p, r, func(context.Context, household.Principal) error { calls++; return nil }); err == nil || calls != 0 {
				t.Fatal("stale confirmed result applied")
			}
			r.Page = nil
			r.Outcome = "absent"
			for range 2 {
				if err = f.store.ReconcileJob(testContext, f.p, r, nil); err != nil {
					t.Fatal(err)
				}
			}
			current, err = f.store.Job(testContext, f.p, issued.ID)
			if err != nil || current.State != jobs.Canceled {
				t.Fatal("proven absence did not clear canceled uncertainty", err)
			}
		})
	}
}

func TestCommittedPageAcknowledgesItsExternalAction(t *testing.T) {
	f := newFixture(t)
	connection := f.connection()
	b := binding()
	gate := f.admit(b)
	issued := f.issued(gate, connection, b)
	if err := f.store.BeginExternal(testContext, f.p, issued); err != nil {
		t.Fatal(err)
	}
	if ok, err := gate.CommitPage(testContext, f.p, issued, admission.Page{EvidenceRef: "synthetic:page1", NextCursor: "page2", Coverage: "complete"}, func(context.Context) error { return nil }); err != nil || !ok {
		t.Fatal(err)
	}
	current, err := f.store.Job(testContext, f.p, issued.ID)
	if err != nil || current.ExternalStarted || current.Cursor != "page2" {
		t.Fatal("confirmed action still uncertain", err)
	}
	if _, err = f.admin.Exec(testContext, `UPDATE want_keep.jobs SET lease_until=clock_timestamp()-INTERVAL '1 second' WHERE id=$1`, issued.ID); err != nil {
		t.Fatal(err)
	}
	rows, err := f.restart().ClaimJobs(testContext, "sync", 1, time.Minute)
	if err != nil || len(rows) != 1 || rows[0].Cursor != "page2" {
		t.Fatal("confirmed page crash needs unnecessary reconciliation", err)
	}
	next := rows[0]
	if err = f.store.BeginExternal(testContext, f.p, next); err != nil {
		t.Fatal("next marked action blocked", err)
	}
	if err = f.store.RetryJob(testContext, f.p, next, 0, false); err != nil {
		t.Fatal(err)
	}
	current, err = f.store.Job(testContext, f.p, next.ID)
	if err != nil || current.State != jobs.Unresolved {
		t.Fatal("new action uncertainty lost", err)
	}
}

func TestJobDiagnosticsRetainContextWithoutPrivateContent(t *testing.T) {
	for _, stage := range []string{"prepare", "commit"} {
		t.Run(stage, func(t *testing.T) {
			f := newFixture(t)
			f.event()
			private := errors.New("synthetic private receipt and provider response")
			var diagnostic app.Diagnostic
			worker := f.worker(handlerFunc(func(context.Context, app.Execution) (app.Result, error) {
				if stage == "prepare" {
					return app.Result{}, private
				}
				return app.Result{State: jobs.Succeeded, Apply: func(context.Context, household.Principal) error { return private }}, nil
			}))
			worker.Report = func(d app.Diagnostic) { diagnostic = d }
			if err := worker.Step(testContext); !errors.Is(err, private) {
				t.Fatal(err)
			}
			if diagnostic.Kind != jobs.Outbox || diagnostic.JobID == "" || diagnostic.Stage != stage || diagnostic.Code != "execution_failed" || diagnostic.Duration <= 0 {
				t.Fatal("failure context lost", diagnostic)
			}
			data, err := json.Marshal(diagnostic)
			if err != nil || strings.Contains(string(data), private.Error()) {
				t.Fatal("private diagnostic content", err)
			}
			connection := f.connection()
			b := binding()
			gate := f.admit(b)
			if _, err = f.admin.Exec(testContext, `UPDATE want_keep.memberships SET active=false WHERE household_id=$1 AND user_id=$2`, f.family.ID, f.p.UserID()); err != nil {
				t.Fatal(err)
			}
			scheduler := app.Scheduler{Repository: f.store, Admission: gate, Bindings: []connections.Binding{b}, Report: func(d app.Diagnostic) { diagnostic = d }}
			if err = scheduler.Tick(testContext); err != nil {
				t.Fatal(err)
			}
			if diagnostic.ConnectionID != connection || diagnostic.Stage != "schedule" || diagnostic.Duration <= 0 {
				t.Fatal("source diagnostic context lost", diagnostic)
			}
		})
	}
}
