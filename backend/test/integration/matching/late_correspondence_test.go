//go:build integration

package matching_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestLateInternalCorrespondenceCreatesWaitingCase(t *testing.T) {
	f := newFixture(t)
	gate := f.admit()
	connection := f.connection(f.p)
	from, to := f.create(money.RUB, "5000"), f.create(money.RUB, "0")
	a := f.revision(uuid.NewString(), from, "-1000", money.RUB, 1)
	if _, err := f.importSource(gate, connection, f.sourceInput(&a, "late-proof")); err != nil {
		t.Fatal(err)
	}
	a.Type = ledger.Transfer
	a.Correspondence = &ledger.Correspondence{Kind: "transfer", Namespace: "synthetic:transfer", Reference: "late-proof", FromAccountID: from, ToAccountID: to}
	correction := f.sourceInput(&a, "late-proof")
	correction.Classification = "correction"
	correction.ExpectedRevision = 1
	correction.PayloadHash = strings.Repeat("b", 64)
	if _, err := f.importSource(gate, connection, correction); err != nil {
		t.Fatal(err)
	}
	current := f.current(a.OperationID)
	g, found, err := f.store.MatchingForOperation(testContext, f.p, a.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	funding, err := f.store.AccountFunding(testContext, f.p, from)
	if err != nil {
		t.Fatal(err)
	}
	_, known := funding.Value()
	t.Logf("accepted revision=%d type=%s proof=%t matching found=%t state=%s participation=%s funding known=%t", current.Revision, current.Type, current.Correspondence != nil, found, g.State, current.Participation.State, known)
	if !found || g.State != matching.WaitingSide {
		t.Fatal("accepted full internal correspondence did not create a missing-side case")
	}
	if known {
		t.Fatal("unresolved internal side did not gate funding")
	}
	before := f.count("operation_revisions")
	if out, err := f.importSource(gate, connection, correction); err != nil || !out.Duplicate {
		t.Fatal(out, err)
	}
	if f.count("operation_revisions") != before {
		t.Fatal("late proof replay changed history")
	}
	other := f.revision(uuid.NewString(), to, "1000", money.RUB, 1)
	other.Type, other.Correspondence = ledger.Transfer, a.Correspondence
	if _, err := f.importSource(gate, connection, f.sourceInput(&other, "late-proof-other")); err != nil {
		t.Fatal(err)
	}
	linked, found, err := f.store.MatchingForOperation(testContext, f.p, a.OperationID)
	if err != nil || !found || linked.State != matching.Linked || len(linked.Members) != 2 {
		t.Fatal(linked, found, err)
	}
	f.balance(from, "owned", "4000")
	f.balance(to, "owned", "1000")
}

func TestLateProofFindsExistingFinancialEvidence(t *testing.T) {
	for _, kind := range []matching.Kind{matching.Payment, matching.Transfer} {
		t.Run(string(kind), func(t *testing.T) {
			f := newFixture(t)
			gate, connection := f.admit(), f.connection(f.p)
			from, to := f.create(money.RUB, "5000"), f.create(money.RUB, "0")
			proof := ledger.Correspondence{Kind: string(kind), Namespace: "synthetic:late-proof", Reference: "existing-evidence"}
			first := f.revision(uuid.NewString(), from, "-300", money.RUB, 1)
			incoming := f.revision(uuid.NewString(), from, "-600", money.RUB, 1)
			if kind == matching.Transfer {
				proof.FromAccountID, proof.ToAccountID = from, to
				first = f.revision(first.OperationID, to, "300", money.RUB, 1)
				first.Type = ledger.Transfer
			}
			first.Correspondence = &proof
			if _, err := f.importSource(gate, connection, f.sourceInput(&first, "existing")); err != nil {
				t.Fatal(err)
			}
			if _, err := f.importSource(gate, connection, f.sourceInput(&incoming, "new-proof")); err != nil {
				t.Fatal(err)
			}
			if _, found, err := f.store.MatchingForOperation(testContext, f.p, incoming.OperationID); err != nil || found {
				t.Fatal("initial record must be independent", found, err)
			}
			incoming.Postings[0].Money = cash("-300", money.RUB)
			incoming.Correspondence = &proof
			if kind == matching.Transfer {
				incoming.Type = ledger.Transfer
			}
			input := f.sourceInput(&incoming, "new-proof")
			input.Classification, input.ExpectedRevision, input.PayloadHash = "correction", 1, strings.Repeat("b", 64)
			if _, err := f.importSource(gate, connection, input); err != nil {
				t.Fatal(err)
			}
			g, found, err := f.store.MatchingForOperation(testContext, f.p, incoming.OperationID)
			if err != nil || !found || g.State != matching.Linked || len(g.Members) != 2 {
				t.Fatal(g, found, err)
			}
			f.balance(from, "owned", "4700")
			if kind == matching.Transfer {
				f.balance(to, "owned", "300")
			}
			before := f.count("operation_revisions")
			if out, err := f.importSource(gate, connection, input); err != nil || !out.Duplicate {
				t.Fatal(out, err)
			}
			if f.count("operation_revisions") != before {
				t.Fatal("late proof replay duplicated effect")
			}
			c := f.client(f.p)
			decision := f.current(incoming.OperationID).DecisionID
			c.result("/transactions/"+incoming.OperationID+"/undo", map[string]any{"decisionId": decision, "expectedRevisions": f.versions(first.OperationID, incoming.OperationID), "reason": "Reconsider late proof"})
			if kind == matching.Payment {
				f.balance(from, "owned", "4400")
			} else {
				f.balance(from, "owned", "4700")
				f.balance(to, "owned", "300")
			}
		})
	}
}

func TestLateIncompatibleProofRetainsAcceptedEffect(t *testing.T) {
	for _, state := range []ledger.State{ledger.Posted, ledger.Pending} {
		t.Run(string(state), func(t *testing.T) {
			f := newFixture(t)
			gate, connection := f.admit(), f.connection(f.p)
			account := f.create(money.RUB, "1000")
			first := f.revision(uuid.NewString(), account, "-300", money.RUB, 1)
			incoming := f.revision(uuid.NewString(), account, "-600", money.RUB, 1)
			incoming.State = state
			if state == ledger.Pending {
				incoming.PostedAt = ledger.Revision{}.PostedAt
			}
			first.Correspondence = &ledger.Correspondence{Kind: "payment", Namespace: "synthetic:late-proof", Reference: "incompatible"}
			for _, r := range []*ledger.Revision{&first, &incoming} {
				if _, err := f.importSource(gate, connection, f.sourceInput(r, r.OperationID)); err != nil {
					t.Fatal(err)
				}
			}
			owned, locked := "100", "0"
			if state == ledger.Pending {
				owned, locked = "700", "600"
			}
			f.balance(account, "owned", owned)
			f.balance(account, "locked", locked)
			incoming.Correspondence = first.Correspondence
			input := f.sourceInput(&incoming, incoming.OperationID)
			input.Classification, input.ExpectedRevision, input.PayloadHash = "correction", 1, strings.Repeat("b", 64)
			if _, err := f.importSource(gate, connection, input); err != nil {
				t.Fatal(err)
			}
			current := f.current(incoming.OperationID)
			if current.Participation.State != "retained" || !current.Contributes(0) {
				t.Fatal("accepted effect not retained", current.Participation)
			}
			g, found, err := f.store.MatchingForOperation(testContext, f.p, incoming.OperationID)
			if err != nil || !found || g.State != matching.Clarification {
				t.Fatal(g, found, err)
			}
			f.balance(account, "owned", owned)
			f.balance(account, "locked", locked)
			var conflict string
			if err := f.admin.QueryRow(testContext, `SELECT conflict FROM want_keep.ledger_source_facts WHERE household_id=$1 AND operation_id=$2 ORDER BY source_revision DESC LIMIT 1`, f.family.ID, incoming.OperationID).Scan(&conflict); err != nil || conflict != "matching_conflict" {
				t.Fatal(conflict, err)
			}
			funding, err := f.store.AccountFunding(testContext, f.p, account)
			if err != nil {
				t.Fatal(err)
			}
			if _, known := funding.Value(); known {
				t.Fatal("unresolved contribution funded a reserve")
			}
			count := f.count("operation_revisions")
			if out, err := f.importSource(gate, connection, input); err != nil || !out.Duplicate {
				t.Fatal(out, err)
			}
			if f.count("operation_revisions") != count {
				t.Fatal("replay changed history")
			}
			c := f.client(f.p)
			if v := c.transaction(incoming.OperationID); v.Participation == nil || v.Participation.State != "retained" {
				t.Fatal("transport lost retained effect", v)
			}
			c.result("/matching/"+g.ID+"/resolve", map[string]any{"decision": "separate", "expectedRevision": g.Revision, "expectedRevisions": f.versions(incoming.OperationID), "reason": "Independent payment"})
			decision := f.current(incoming.OperationID).DecisionID
			f.balance(account, "owned", owned)
			f.balance(account, "locked", locked)
			c.result("/transactions/"+incoming.OperationID+"/undo", map[string]any{"decisionId": decision, "expectedRevisions": f.versions(incoming.OperationID), "reason": "Reconsider evidence"})
			restored, found, err := f.store.MatchingForOperation(testContext, f.p, incoming.OperationID)
			if err != nil || !found || restored.ID != g.ID || restored.State != matching.Clarification || f.current(incoming.OperationID).Participation.State != "retained" {
				t.Fatal("undo lost retained case", restored, err)
			}
			f.balance(account, "owned", owned)
			f.balance(account, "locked", locked)
			// Provider lifecycle still updates the retained independent contribution.
			incoming.State = ledger.Cancelled
			if state == ledger.Posted {
				incoming.State = ledger.Reversed
			}
			input.ExpectedRevision, input.PayloadHash = 2, strings.Repeat("c", 64)
			if _, err := f.importSource(gate, connection, input); err != nil {
				t.Fatal(err)
			}
			f.balance(account, "owned", "700")
			f.balance(account, "locked", "0")
		})
	}
}
