//go:build integration

package matching_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestSeparateIndependentFee(t *testing.T) {
	f := newFixture(t)
	gate, connection := f.admit(), f.connection(f.p)
	from, to, fees := f.create(money.RUB, "5000"), f.create(money.RUB, "0"), f.create(money.BTC, "1")
	proof := ledger.Correspondence{Kind: "transfer", Namespace: "synthetic:transfer", Reference: "separate-fee", FromAccountID: from, ToAccountID: to}
	a, b := f.revision(uuid.NewString(), from, "-1000", money.RUB, 1), f.revision(uuid.NewString(), to, "1000", money.RUB, 1)
	a.Type, b.Type = ledger.Transfer, ledger.Transfer
	a.Correspondence, b.Correspondence = &proof, &proof
	a.Postings = append(a.Postings, ledger.Posting{FeeID: "base-fee", AccountID: fees, Money: cash("-0.00001", money.BTC), Role: ledger.Fee, Funding: ledger.OwnFunds})
	for _, r := range []*ledger.Revision{&a, &b} {
		if _, err := f.importSource(gate, connection, f.sourceInput(r, r.OperationID)); err != nil {
			t.Fatal(err)
		}
	}
	fee := f.revision(uuid.NewString(), fees, "-0.00001", money.BTC, 1)
	fee.Postings[0].Role = ledger.Fee
	fee.Correspondence = &proof
	if _, err := f.importSource(gate, connection, f.sourceInput(&fee, fee.OperationID)); err != nil {
		t.Fatal(err)
	}
	g, found, err := f.store.MatchingForOperation(testContext, f.p, fee.OperationID)
	if err != nil || !found || g.State != matching.Clarification {
		t.Fatal(g, found, err)
	}
	f.balance(fees, "owned", "0.99999")
	c := f.client(f.p)
	raw := c.call("POST", "/matching/"+g.ID+"/resolve", uuid.NewString(), map[string]any{"decision": "separate", "expectedRevision": g.Revision, "expectedRevisions": f.versions(fee.OperationID), "reason": "This is an additional separate fee"}, 202)
	var response map[string]any
	if err := json.Unmarshal(raw.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["status"] != "succeeded" {
		t.Fatalf("separate fee response: %s", raw.Body.String())
	}
	f.balance(fees, "owned", "0.99998")
}

func TestFeeArrivesBeforePrincipal(t *testing.T) {
	f := newFixture(t)
	gate, connection := f.admit(), f.connection(f.p)
	from, to, fees := f.create(money.RUB, "5000"), f.create(money.RUB, "0"), f.create(money.BTC, "1")
	proof := ledger.Correspondence{Kind: "transfer", Namespace: "synthetic:transfer", Reference: "fee-first", FromAccountID: from, ToAccountID: to}
	fee := f.revision(uuid.NewString(), fees, "-0.00001", money.BTC, 1)
	fee.Postings[0].Role = ledger.Fee
	fee.Postings[0].FeeID = "fee-1"
	fee.Correspondence = &proof
	if err := fee.Validate(); err != nil {
		t.Fatal("invalid test source", err)
	}
	_, err := f.importSource(gate, connection, f.sourceInput(&fee, fee.OperationID))
	if err != nil {
		t.Fatalf("fee-only first import failed: %v", err)
	}
	g, found, err := f.store.MatchingForOperation(testContext, f.p, fee.OperationID)
	if err != nil || !found || g.State != matching.WaitingSide || len(g.Members) != 1 {
		t.Fatal("fee must wait for the principal", g, err)
	}
	f.balance(fees, "owned", "0.99999")
	f.balance(from, "owned", "5000")
	f.balance(to, "owned", "0")
	for _, row := range []struct{ account, amount string }{{from, "-1000"}, {to, "1000"}} {
		r := f.revision(uuid.NewString(), row.account, row.amount, money.RUB, 1)
		r.Type, r.Correspondence = ledger.Transfer, &proof
		if row.account == from {
			r.Postings = append(r.Postings, fee.Postings[0])
		}
		if _, err := f.importSource(gate, connection, f.sourceInput(&r, r.OperationID)); err != nil {
			t.Fatal(err)
		}
	}
	g, found, err = f.store.MatchingForOperation(testContext, f.p, fee.OperationID)
	if err != nil || !found || g.State != matching.Linked || len(g.Members) != 3 {
		t.Fatal("late principal did not complete the movement", g, err)
	}
	before := f.count("operation_revisions")
	if out, err := f.importSource(gate, connection, f.sourceInput(&fee, fee.OperationID)); err != nil || !out.Duplicate {
		t.Fatal(out, err)
	}
	if f.count("operation_revisions") != before {
		t.Fatal("old fee page created another revision")
	}
	f.balance(fees, "owned", "0.99999")
	f.balance(from, "owned", "4000")
	f.balance(to, "owned", "1000")
}
