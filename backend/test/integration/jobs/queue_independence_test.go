//go:build integration

package storage_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/connections/admission"
	app "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestWaitingReviewDoesNotBlockAccounting(t *testing.T) {
	for _, reason := range []jobs.Reason{jobs.HandlerUnavailable, jobs.GatewayUnavailable, jobs.BudgetWait} {
		t.Run(string(reason), func(t *testing.T) {
			f := newFixture(t)
			account := f.account(money.RUB, "1000")
			if _, err := f.write(f.revision(uuid.NewString(), account, "-10", money.RUB, 1), request()); err != nil {
				t.Fatal(err)
			}
			if err := f.worker(app.OutboxHandler{Repository: f.store}).Step(testContext); err != nil {
				t.Fatal(err)
			}
			var reviewID string
			if err := f.admin.QueryRow(testContext, `SELECT id FROM want_keep.jobs WHERE kind='ai'`).Scan(&reviewID); err != nil {
				t.Fatal(err)
			}
			reviewWorker := app.Worker{Repository: f.store, Config: app.DefaultWorkerConfig(jobs.AI)}
			if reason != jobs.HandlerUnavailable {
				reviewWorker.Handler = handlerFunc(func(context.Context, app.Execution) (app.Result, error) {
					return app.Result{State: jobs.Waiting, Reason: reason}, nil
				})
			}
			if err := reviewWorker.Step(testContext); err != nil {
				t.Fatal(err)
			}
			waiting, err := f.store.Job(testContext, f.p, reviewID)
			if err != nil || waiting.State != jobs.Waiting || waiting.Reason != reason || waiting.Attempt != 0 {
				t.Fatal("review did not enter dependency waiting", err)
			}

			connection, b := f.connection(), binding()
			gate := f.admit(b)
			queued, err := gate.RequestSync(testContext, f.p, connection, b, time.Now().Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			imported := f.revision(uuid.NewString(), account, "-20", money.RUB, 1)
			manual := f.revision(uuid.NewString(), account, "-30", money.RUB, 1)
			sources := journal.NewSources(f.store, f.writer)
			worker := app.Worker{Repository: f.store, Admission: gate, Config: app.DefaultWorkerConfig(jobs.Sync), Handler: handlerFunc(func(ctx context.Context, x app.Execution) (app.Result, error) {
				if err := gate.BeforeRead(ctx, x.Principal, x.Job); err != nil {
					return app.Result{}, err
				}
				input := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: b.Provider, ExternalAccountID: "synthetic-stable", Product: "current", Log: "statement", RecordID: "while-review-waits"}, PayloadHash: strings.Repeat("a", 64), EvidenceRef: "synthetic:independent", Classification: "new", Operation: &imported, ConnectionID: connection, JobID: x.Job.ID, FetchedAt: f.now}
				applied, err := gate.CommitPage(ctx, x.Principal, x.Job, admission.Page{EvidenceRef: input.EvidenceRef, NextCursor: "done", Coverage: "complete", Complete: true}, func(ctx context.Context) error {
					_, err := sources.Apply(ctx, x.Principal, input)
					return err
				})
				if err != nil {
					return app.Result{}, err
				}
				if !applied {
					return app.Result{}, jobs.ErrStaleAttempt
				}
				return app.Result{State: jobs.Succeeded}, nil
			})}
			ctx, cancel := context.WithTimeout(testContext, 10*time.Second)
			defer cancel()
			start, results := make(chan struct{}), make(chan error, 2)
			go func() { <-start; results <- worker.Step(ctx) }()
			go func() {
				<-start
				_, err := f.executor.Execute(ctx, f.p, request(), func(ctx context.Context) (command.Result, error) {
					err := f.writer.Append(ctx, f.p, manual, 0)
					return command.Result{ResourceType: "transaction", ResourceID: manual.OperationID, Revision: 1}, err
				})
				results <- err
			}()
			close(start)
			for range 2 {
				if err := <-results; err != nil {
					t.Error(err)
				}
			}
			current, err := f.store.Job(testContext, f.p, reviewID)
			if err != nil || current.State != waiting.State || current.Reason != waiting.Reason || current.Attempt != waiting.Attempt || current.LeaseToken != waiting.LeaseToken {
				t.Fatal("other queues consumed the waiting review", err)
			}
			synced, err := f.store.Job(testContext, f.p, queued.ID)
			if err != nil || synced.State != jobs.Succeeded || f.count("job_receipts") != 2 {
				t.Fatal("sync did not commit independently", err)
			}
			if f.available(account) != "940" || f.count("postings") != 3 || f.count("source_records") != 1 || f.count("outbox") != 3 {
				t.Fatal("waiting blocked or duplicated ordinary financial effects")
			}
		})
	}
}
