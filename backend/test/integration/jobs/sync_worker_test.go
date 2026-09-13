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

func TestSyncWorkerCommitsAdmittedPages(t *testing.T) {
	f := newFixture(t)
	connection, b := f.connection(), binding()
	gate := f.admit(b)
	queued, err := gate.RequestSync(testContext, f.p, connection, b, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	account := f.account(money.RUB, "1000")
	source := journal.NewSources(f.store, f.writer, nil)
	calls := 0
	worker := app.Worker{Repository: f.store, Admission: gate, Config: app.DefaultWorkerConfig(jobs.Sync)}
	worker.Handler = handlerFunc(func(ctx context.Context, x app.Execution) (app.Result, error) {
		calls++
		cursor := x.Job.Cursor
		for i, record := range []string{"first", "last"} {
			if err := gate.BeforeRead(ctx, x.Principal, x.Job); err != nil {
				return app.Result{}, err
			}
			if err := x.BeginExternal(ctx); err != nil {
				return app.Result{}, err
			}
			revision := f.revision(uuid.NewString(), account, "-10", money.RUB, 1)
			input := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: b.Provider, ExternalAccountID: "synthetic-stable", Product: "current", Log: "statement", RecordID: record}, PayloadHash: strings.Repeat("a", 64), EvidenceRef: "synthetic:" + record, Classification: "new", Operation: &revision, ConnectionID: connection, JobID: x.Job.ID, FetchedAt: f.now}
			page := admission.Page{EvidenceRef: input.EvidenceRef, Cursor: cursor, NextCursor: record, Coverage: "complete", Complete: i == 1}
			applied, err := gate.CommitPage(ctx, x.Principal, x.Job, page, func(ctx context.Context) error {
				_, err := source.Apply(ctx, x.Principal, input)
				return err
			})
			if err != nil {
				return app.Result{}, err
			}
			if !applied {
				return app.Result{}, jobs.ErrStaleAttempt
			}
			cursor = page.NextCursor
		}
		return app.Result{State: jobs.Succeeded}, nil
	})
	for range 2 {
		if err := worker.Step(testContext); err != nil {
			t.Fatal(err)
		}
	}
	current, err := f.store.Job(testContext, f.p, queued.ID)
	if err != nil || current.State != jobs.Succeeded || current.ExternalStarted || calls != 1 {
		t.Fatal("source worker did not acknowledge its committed receipt", err)
	}
	progress, err := f.store.SyncProgress(testContext, f.p, connection)
	if err != nil || progress.Cursor != "last" || !progress.Completed || progress.LastSuccessAt == nil {
		t.Fatal("source worker progress lost", err)
	}
	if f.available(account) != "980" || f.count("source_records") != 2 || f.count("job_receipts") != 1 {
		t.Fatal("page effects lost or duplicated")
	}
}

func TestSyncWorkerRejectsUnfencedApply(t *testing.T) {
	f := newFixture(t)
	connection, b := f.connection(), binding()
	gate := f.admit(b)
	if _, err := gate.RequestSync(testContext, f.p, connection, b, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	applied := false
	worker := app.Worker{Repository: f.store, Admission: gate, Config: app.DefaultWorkerConfig(jobs.Sync), Handler: handlerFunc(func(context.Context, app.Execution) (app.Result, error) {
		return app.Result{State: jobs.Succeeded, Apply: func(context.Context, household.Principal) error { applied = true; return nil }}, nil
	})}
	if err := worker.Step(testContext); !errors.Is(err, jobs.ErrInvalidJob) {
		t.Fatal("generic sync completion accepted", err)
	}
	if applied || f.count("job_receipts") != 0 || f.count("source_records") != 0 {
		t.Fatal("source fence bypassed")
	}
}
