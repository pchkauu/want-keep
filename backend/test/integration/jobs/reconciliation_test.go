//go:build integration

package storage_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/connections/admission"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	app "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestConfirmedSyncPageReconciliation(t *testing.T) {
	for _, complete := range []bool{false, true} {
		t.Run(map[bool]string{false: "continue", true: "complete"}[complete], func(t *testing.T) {
			f := newFixture(t)
			connection := f.connection()
			b := binding()
			gate := f.admit(b)
			issued := f.issued(gate, connection, b)
			account := f.account(money.RUB, "1000")
			revision := f.revision(uuid.NewString(), account, "-100", money.RUB, 1)
			input := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: b.Provider, ExternalAccountID: "synthetic-stable", Product: "current", Log: "statement", RecordID: "confirmed"}, PayloadHash: strings.Repeat("a", 64), EvidenceRef: "synthetic:confirmed", Classification: "new", Operation: &revision, ConnectionID: connection, JobID: issued.ID, FetchedAt: f.now}
			if err := f.store.BeginExternal(testContext, f.p, issued); err != nil {
				t.Fatal(err)
			}
			if err := f.store.SetJobOutcome(testContext, f.p, issued, jobs.Unresolved, jobs.ExternalUnknown, 0); err != nil {
				t.Fatal(err)
			}
			r := app.Reconciliation{Job: issued, EvidenceRef: input.EvidenceRef, Outcome: "confirmed", Page: &admission.Page{EvidenceRef: input.EvidenceRef, NextCursor: "page2", Coverage: "complete", Complete: complete}}
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
			if err != nil || progress.Cursor != "page2" || progress.Completed != complete || (progress.LastSuccessAt != nil) != complete {
				t.Fatal("reconciliation checkpoint", err)
			}
			current, err := f.store.Job(testContext, f.p, issued.ID)
			if err != nil || current.ExternalStarted {
				t.Fatal(err)
			}
			if complete {
				if current.State != jobs.Succeeded || f.count("job_receipts") != 1 {
					t.Fatal("terminal receipt missing")
				}
			} else {
				next := f.issued(gate, connection, b)
				if next.ID != issued.ID || next.LeaseToken == issued.LeaseToken || next.Cursor != "page2" {
					t.Fatal("confirmed page was repeated")
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
