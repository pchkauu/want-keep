//go:build integration

package matching_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestNormalizedReceiptBankAndManualEvidenceRetainOnePayment(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "5000")
	first, partner := f.client(f.p), f.client(f.q)
	input := map[string]any{"type": "expense", "accountId": account, "amount": map[string]string{"asset": "RUB", "amount": "300"}, "occurredAt": f.now.String(), "payer": map[string]string{"state": "unknown"}, "allocation": map[string]string{"mode": "unresolved", "reason": "Unresolved"}}
	result, _ := first.result("/transactions", input).AsCommandSucceeded()
	manual := result.Result.Id
	input["attachmentId"] = f.attachment(account, true)
	receipt, _ := partner.result("/transactions", input).AsCommandSucceeded()
	gate := f.admit()
	connection := f.connection(f.p)
	bank := f.revision(uuid.NewString(), account, "-300", money.RUB, 1)
	source := f.sourceInput(&bank, "receipt-payment")
	if _, err := f.importSource(gate, connection, source); err != nil {
		t.Fatal(err)
	}
	f.balance(account, "owned", "4700")
	first.result("/transactions/"+manual+"/links", map[string]any{"kind": "receipt_match", "reason": "Confirmed payment evidence", "expectedRevisions": f.versions(manual, receipt.Result.Id, bank.OperationID)})
	linked := first.transaction(manual)
	view := decode[generated.MatchingCase](t, first.call("GET", "/matching/"+linked.Participation.GroupId, "", nil, 200))
	kinds := map[string]bool{}
	for _, member := range view.Members {
		for _, e := range member.Evidence {
			kinds[string(e.Kind)] = true
		}
	}
	if !kinds["source"] || !kinds["attachment"] || len(view.Members) != 3 {
		t.Fatal("evidence lost", view)
	}
	if partner.transaction(receipt.Result.Id).ActorId != string(f.p.UserID()) { // Latest link actor differs from immutable original author.
		t.Fatal("link actor not recorded")
	}
	original := decode[generated.Transaction](t, first.call("GET", "/transactions/"+receipt.Result.Id+"/revisions/1", "", nil, 200))
	if original.ActorId != string(f.q.UserID()) {
		t.Fatal("receipt author overwritten")
	}
	count := f.count("operation_revisions")
	if out, err := f.importSource(gate, connection, source); err != nil || !out.Duplicate {
		t.Fatal(out, err)
	}
	if f.count("operation_revisions") != count {
		t.Fatal("duplicate import changed history")
	}
	f.balance(account, "owned", "4700")
}
func TestStaleImportCannotCreateMatchingEffectsOrAdvanceCheckpoint(t *testing.T) {
	f := newFixture(t)
	gate := f.admit()
	connection := f.connection(f.p)
	job := f.issued(gate, connection)
	if _, err := gate.RecordCheck(testContext, connections.Check{Kind: connections.HostCheck, Binding: binding(), Result: connections.CheckRevoked, At: instant("2026-09-07T12:00:01.123456789Z")}); err != nil {
		t.Fatal(err)
	}
	called := false
	applied, err := gate.CommitPage(testContext, f.p, job, admission.Page{EvidenceRef: "synthetic:stale-matching", Coverage: "complete", Complete: true, NextCursor: "must-not-advance"}, func(context.Context) error { called = true; return nil })
	if err != nil || applied || called {
		t.Fatal(applied, called, err)
	}
	if f.count("quarantine") != 1 || f.count("matching_cases") != 0 || f.count("source_records") != 0 {
		t.Fatal("stale import escaped quarantine")
	}
}
func TestSeparateConfirmationDoesNotAutoMergeOnNewSharedProof(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "5000")
	a, b := uuid.NewString(), uuid.NewString()
	c := f.client(f.p)
	for _, id := range []string{a, b} {
		if _, err := f.write(f.revision(id, account, "-300", money.RUB, 1), request()); err != nil {
			t.Fatal(err)
		}
	}
	g, _, err := f.store.MatchingForOperation(testContext, f.p, b)
	if err != nil {
		t.Fatal(err)
	}
	c.result("/matching/"+g.ID+"/resolve", map[string]any{"expectedRevision": g.Revision, "decision": "separate", "reason": "Known separate payments", "expectedRevisions": f.versions(b)})
	for _, id := range []string{a, b} {
		r := f.current(id)
		r.Revision++
		r.Correspondence = &ledger.Correspondence{Kind: "payment", Namespace: "synthetic:payments", Reference: "new-evidence"}
		if _, err := f.write(r, request()); err != nil {
			t.Fatal(err)
		}
	}
	incoming := f.revision(uuid.NewString(), account, "-300", money.RUB, 1)
	incoming.Correspondence = &ledger.Correspondence{Kind: "payment", Namespace: "synthetic:payments", Reference: "new-evidence"}
	if _, err := f.write(incoming, request()); err != nil {
		t.Fatal(err)
	}
	f.balance(account, "owned", "4400")
	g, found, err := f.store.MatchingForOperation(testContext, f.p, incoming.OperationID)
	if err != nil || !found || g.State != matching.Clarification {
		t.Fatal(g, err)
	}
	if !strings.Contains(g.Reason, "matching") {
		t.Fatal("missing clarification reason")
	}
}
