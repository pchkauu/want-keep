//go:build integration

package audit_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"testing"
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
