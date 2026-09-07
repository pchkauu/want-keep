//go:build integration

package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	app "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestReviewRevisionBindingAndReplay(t *testing.T) {
	f := newFixture(t)
	id := f.account(money.RUB, "100")
	r := f.revision(uuid.NewString(), id, "-10", money.RUB, 1)
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	outbox := f.worker(app.OutboxHandler{Repository: f.store})
	if err := outbox.Step(testContext); err != nil {
		t.Fatal(err)
	}
	rows, err := f.store.ClaimJobs(testContext, "ai", 1, time.Minute)
	if err != nil || len(rows) != 1 {
		t.Fatal(err)
	}
	issued := rows[0]
	execution := app.Execution{Job: issued, Principal: f.p}
	executor := app.Executor{Repository: f.store}
	service := ledger.NewService(f.store, f.writer, func() calendar.Instant { return f.now }, uuid.NewString)
	in := ledger.ReviewInput{OperationID: r.OperationID, Revision: 1, State: "reviewed", Rationale: "Synthetic checked result"}
	forged := in
	forged.OperationID = uuid.NewString()
	if err = executor.CompleteReview(testContext, execution, service, forged); err == nil {
		t.Fatal("wrong target accepted")
	}
	for range 2 {
		if err = executor.CompleteReview(testContext, execution, service, in); err != nil {
			t.Fatal(err)
		}
	}
	if f.count("ledger_review_results") != 1 {
		t.Fatal("duplicate review")
	}
	r.Revision = 2
	r.Reason = "Correction"
	if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.writer.Append(ctx, f.p, r, 1) }); err != nil {
		t.Fatal(err)
	}
	if err = outbox.Step(testContext); err != nil {
		t.Fatal(err)
	}
	rows, err = f.store.ClaimJobs(testContext, "ai", 1, time.Minute)
	if err != nil || len(rows) != 1 {
		t.Fatal(err)
	}
	current := rows[0]
	r.Revision = 3
	if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.writer.Append(ctx, f.p, r, 2) }); err != nil {
		t.Fatal(err)
	}
	in.Revision = 2
	if err = executor.CompleteReview(testContext, app.Execution{Job: current, Principal: f.p}, service, in); err == nil {
		t.Fatal("stale review accepted")
	}
	current.Kind = jobs.Outbox
	if err = executor.CompleteReview(testContext, app.Execution{Job: current, Principal: f.p}, service, in); err == nil {
		t.Fatal("forged queue accepted")
	}
}
