//go:build integration

package audit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	allocation "github.com/pchkauu/want-keep/backend/internal/allocation/application"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestReviewProposalEvidenceStalenessAndRollback(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	r := c.expense(f.create(money.RUB, "5000"), money.RUB, "500")
	name := "Confirmed merchant"
	review := func(in journal.ReviewInput) error {
		return f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.ledgerService().CompleteReview(ctx, f.p, in) })
	}
	in := journal.ReviewInput{OperationID: r.Id, Revision: 1, State: "reviewed", Rationale: "Synthetic evidence-based suggestion", Correction: &ledger.Correction{Merchant: &name}}
	if err := review(in); err != nil {
		t.Fatal(err)
	}
	if err := review(in); err != nil {
		t.Fatal("review replay", err)
	}
	r = c.transaction(r.Id)
	if r.Revision != 2 || r.AiState != "waiting" || r.Merchant == nil || *r.Merchant != name {
		t.Fatal("proposal or new review request missing", r)
	}
	history := decode[generated.TransactionHistoryPage](t, c.call("GET", "/transactions/"+r.Id+"/history", "", nil, 200))
	if len(history.Items[0].Evidence) != 1 || history.Items[0].Evidence[0].Kind != "review" || history.Items[0].Before.Review == nil {
		t.Fatal("review provenance missing", history)
	}
	var requests, revisions int
	if err := f.admin.QueryRow(testContext, `SELECT (SELECT count(*) FROM want_keep.ledger_review_requests),(SELECT count(*) FROM want_keep.operation_revisions)`).Scan(&requests, &revisions); err != nil {
		t.Fatal(err)
	}
	if requests != revisions {
		t.Fatal("not exactly one request per revision", requests, revisions)
	}
	oldRevision := r.Revision
	r = c.correct(r, map[string]any{"note": "Partner-independent note"})
	in.Revision = uint64(oldRevision)
	in.Correction = nil
	if err := review(in); err == nil {
		t.Fatal("stale result accepted")
	}
	in.Revision = uint64(r.Revision)
	in.Evidence = []ledger.Evidence{{Kind: "source", ID: uuid.NewString(), Revision: 1}}
	before := f.count("ledger_review_results")
	if err := review(in); err == nil {
		t.Fatal("unknown evidence accepted")
	}
	if f.count("ledger_review_results") != before {
		t.Fatal("partial review committed")
	}
	in.Evidence = nil
	principal := []ledger.Posting{{AccountID: r.Postings[0].AccountId, Money: cash("-900", money.RUB), Role: ledger.Principal}}
	in.Correction = &ledger.Correction{Principal: &principal}
	if err := review(in); err == nil {
		t.Fatal("model money applied")
	}
}

func TestReviewAllocationReplayHashesTheCompleteSplit(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	r := c.expense(f.create(money.RUB, "5000"), money.RUB, "500")
	service := journal.NewServiceWithAllocations(f.store, f.writer, allocation.NewService(f.store, func() calendar.Instant { return f.now }, uuid.NewString), func() calendar.Instant { return f.now }, uuid.NewString)
	review := func(in journal.ReviewInput) error {
		return f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return service.CompleteReview(ctx, f.p, in) })
	}
	allocationChange := func(first, second string) *ledger.Correction {
		return &ledger.Correction{Allocation: &ledger.AllocationChange{Allocation: ledger.AllocationInput{Mode: ledger.AllocationByShares, Purpose: ledger.AllocationShared, Members: []ledger.AllocationMemberInput{{MemberID: f.members[0].ID, Share: first}, {MemberID: f.members[1].ID, Share: second}}}}}
	}
	in := journal.ReviewInput{OperationID: r.Id, Revision: 1, State: "reviewed", Rationale: "Synthetic allocation suggestion", Correction: allocationChange("60", "40")}
	if err := review(in); err != nil {
		t.Fatal(err)
	}
	if err := review(in); err != nil {
		t.Fatal("identical review replay", err)
	}
	in.Correction = allocationChange("40", "60")
	var rejection commands.Rejection
	if err := review(in); !errors.As(err, &rejection) || rejection.Code != "idempotency_conflict" {
		t.Fatalf("changed allocation replay error = %v", err)
	}
}
