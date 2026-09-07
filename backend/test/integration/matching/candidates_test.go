//go:build integration

package matching_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestCandidateWindowAccountsAndDistinctBlockchainMovements(t *testing.T) {
	f := newFixture(t)
	account, other := f.create(money.USDT, "1000"), f.create(money.USDT, "1000")
	a := f.revision(uuid.NewString(), account, "-300", money.USDT, 1)
	a.OccurredAt, _ = calendar.ParseInstant(f.now.Time().AddDate(0, 0, -20).UTC().Format(time.RFC3339Nano))
	a.CashDate, _ = a.OccurredAt.DateIn(a.Timezone)
	a.Correspondence = &ledger.Correspondence{Kind: "payment", Namespace: "synthetic:blockchain:log", Reference: "transaction-hash", Network: "synthetic-chain", Movement: "log:1"}
	if _, err := f.write(a, request()); err != nil {
		t.Fatal(err)
	}
	b := f.revision(uuid.NewString(), account, "-300", money.USDT, 1)
	b.Correspondence = a.Correspondence
	if _, err := f.write(b, request()); err != nil {
		t.Fatal(err)
	}
	f.balance(account, "owned", "700") // Proven identifiers are searched beyond seven days.
	c := f.revision(uuid.NewString(), account, "-300", money.USDT, 1)
	proof := *a.Correspondence
	proof.Movement = "log:2"
	c.Correspondence = &proof
	if _, err := f.write(c, request()); err != nil {
		t.Fatal(err)
	}
	if f.current(c.OperationID).Participation.GroupID != "" {
		t.Fatal("different blockchain movement collapsed")
	}
	f.balance(account, "owned", "400")
	d := f.revision(uuid.NewString(), other, "-300", money.USDT, 1)
	if _, err := f.write(d, request()); err != nil {
		t.Fatal(err)
	}
	if f.current(d.OperationID).Participation.GroupID != "" {
		t.Fatal("different account considered duplicate")
	}
}
func TestSeparateDecisionUndoAndUnresolvedFunding(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "5000")
	a, b := uuid.NewString(), uuid.NewString()
	for _, id := range []string{a, b} {
		if _, err := f.write(f.revision(id, account, "-300", money.RUB, 1), request()); err != nil {
			t.Fatal(err)
		}
	}
	funding, err := f.store.AccountFunding(testContext, f.p, account)
	if err != nil {
		t.Fatal(err)
	}
	if _, known := funding.Value(); known {
		t.Fatal("unresolved money used to fund a new reserve")
	}
	c := f.client(f.p)
	g, _, err := f.store.MatchingForOperation(testContext, f.p, b)
	if err != nil {
		t.Fatal(err)
	}
	c.result("/matching/"+g.ID+"/resolve", map[string]any{"expectedRevision": g.Revision, "decision": "separate", "reason": "Different payment", "expectedRevisions": f.versions(b)})
	f.balance(account, "owned", "4400")
	funding, err = f.store.AccountFunding(testContext, f.p, account)
	if err != nil {
		t.Fatal(err)
	}
	if _, known := funding.Value(); !known {
		t.Fatal("resolved funds remained unavailable")
	}
	separated := c.transaction(b)
	c.result("/transactions/"+b+"/undo", map[string]any{"decisionId": *separated.DecisionId, "expectedRevisions": f.versions(b), "reason": "Reopen matching"})
	f.balance(account, "owned", "4700")
	g, found, err := f.store.MatchingForOperation(testContext, f.p, b)
	if err != nil || !found || g.State != matching.Clarification {
		t.Fatal(g, found, err)
	}
	if len(g.Candidates) != 1 || g.Candidates[0].OperationID != a || g.Candidates[0].Revision != f.current(a).Revision {
		t.Fatal("undo lost the candidate needed to resolve the reopened case", g.Candidates)
	}
}
func TestKnownDifferentPurchaseWindowDoesNotInventDuplicate(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "5000")
	a, b := f.revision(uuid.NewString(), account, "-300", money.RUB, 1), f.revision(uuid.NewString(), account, "-300", money.RUB, 1)
	a.OccurredAt, _ = calendar.ParseInstant(f.now.Time().AddDate(0, 0, -8).UTC().Format(time.RFC3339Nano))
	a.CashDate, _ = a.OccurredAt.DateIn(a.Timezone)
	for _, r := range []ledger.Revision{a, b} {
		if _, err := f.write(r, request()); err != nil {
			t.Fatal(err)
		}
	}
	f.balance(account, "owned", "4400")
	_, err := f.executor.Execute(testContext, f.p, request(), func(ctx context.Context) (command.Result, error) {
		return f.matching().Separate(ctx, f.p, uuid.NewString(), 1, nil, "Unknown case")
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTruncatedCandidatesStayUnresolved(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "100000")
	// Independent historical facts predate matching; they are not inferred matches.
	_, err := f.admin.Exec(testContext, `WITH ids AS (SELECT gen_random_uuid() id FROM generate_series(1,101)) INSERT INTO want_keep.operations(household_id,id,revision) SELECT $1,id,1 FROM ids`, f.family.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.admin.Exec(testContext, `INSERT INTO want_keep.operation_revisions(household_id,operation_id,revision,actor_id,reason,economic_type,state,occurred_at,occurred_ns,cash_date,payer_state,human_override) SELECT $1,id,1,$2,'Distinct retained expense','expense','posted','2026-09-07T12:00:00Z',0,'2026-09-07','unknown',false FROM want_keep.operations WHERE household_id=$1 AND NOT EXISTS(SELECT 1 FROM want_keep.operation_revisions r WHERE r.household_id=$1 AND r.operation_id=operations.id)`, f.family.ID, f.p.UserID())
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.admin.Exec(testContext, `INSERT INTO want_keep.postings(household_id,operation_id,revision,position,account_id,amount,asset,role) SELECT $1,operation_id,1,0,$2,-300,'RUB','principal' FROM want_keep.operation_revisions r WHERE household_id=$1 AND reason='Distinct retained expense'`, f.family.ID, account)
	if err != nil {
		t.Fatal(err)
	}
	r := f.revision(uuid.NewString(), account, "-300", money.RUB, 1)
	if _, err = f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	g, found, err := f.store.MatchingForOperation(testContext, f.p, r.OperationID)
	if err != nil || !found || g.CandidatesComplete || len(g.Candidates) != 100 || g.State != matching.Clarification {
		t.Fatal("truncated search claimed certainty", g, err)
	}
	if f.current(r.OperationID).Contributes(0) {
		t.Fatal("truncated candidates created second effect")
	}
}
