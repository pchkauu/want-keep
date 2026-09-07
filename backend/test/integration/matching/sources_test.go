//go:build integration

package matching_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (f *fixture) sourceInput(r *ledger.Revision, id string) ledger.SourceInput {
	return ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: "raiffeisen", ExternalAccountID: "synthetic-business", Product: "current", Log: "statement", RecordID: id}, PayloadHash: strings.Repeat("a", 64), EvidenceRef: "synthetic:matching-source", Classification: "new", Operation: r}
}
func TestFencedPartialTransferLifecycleAndLateThirdAssetFee(t *testing.T) {
	for _, incomingFirst := range []bool{false, true} {
		name := "outgoing-first"
		if incomingFirst {
			name = "incoming-first"
		}
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			gate := f.admit()
			connection := f.connection(f.p)
			from, to, fees := f.create(money.RUB, "5000"), f.create(money.USDT, "0"), f.create(money.BTC, "1")
			a, b := f.revision(uuid.NewString(), from, "-9000", money.RUB, 1), f.revision(uuid.NewString(), to, "100", money.USDT, 1)
			a.Type, b.Type = ledger.Exchange, ledger.Exchange
			a.State = ledger.Pending
			a.FeeKnowledge = ledger.UnknownFees
			b.FeeKnowledge = ledger.UnknownFees
			proof := ledger.Correspondence{Kind: "exchange", Namespace: "synthetic:exchange", Reference: "exchange-1", FromAccountID: from, ToAccountID: to}
			a.Correspondence = &proof
			b.Correspondence = &proof
			ai, bi := f.sourceInput(&a, "out"), f.sourceInput(&b, "in")
			first, second := ai, bi
			if incomingFirst {
				first, second = bi, ai
			}
			if _, err := f.importSource(gate, connection, first); err != nil {
				t.Fatal(err)
			}
			known := first.Operation
			if components, err := f.current(known.OperationID).Components(); err != nil || len(components) != 0 {
				t.Fatal(components, err)
			}
			if _, err := f.importSource(gate, connection, second); err != nil {
				t.Fatal(err)
			}
			f.balance(from, "owned", "5000")
			f.balance(from, "locked", "9000")
			f.balance(to, "owned", "100")
			a.State = ledger.Posted
			a.PostedAt = f.now
			ai.Classification = "correction"
			ai.ExpectedRevision = 1
			ai.PayloadHash = strings.Repeat("b", 64)
			if _, err := f.importSource(gate, connection, ai); err != nil {
				t.Fatal(err)
			}
			f.balance(from, "owned", "-4000")
			f.balance(from, "locked", "0")
			a.Postings = append(a.Postings, ledger.Posting{FeeID: "fee-1", AccountID: fees, Money: cash("-0.00001", money.BTC), Role: ledger.Fee, Funding: ledger.OwnFunds})
			a.FeeKnowledge = ledger.KnownFees
			ai.ExpectedRevision = 2
			ai.PayloadHash = strings.Repeat("c", 64)
			if _, err := f.importSource(gate, connection, ai); err != nil {
				t.Fatal(err)
			}
			f.balance(fees, "owned", "0.99999")
			count := f.count("operation_revisions")
			if out, err := f.importSource(gate, connection, ai); err != nil || !out.Duplicate {
				t.Fatal(out, err)
			}
			if f.count("operation_revisions") != count {
				t.Fatal("duplicate source generated revision")
			}
			a.State = ledger.Reversed
			ai.ExpectedRevision = 3
			ai.PayloadHash = strings.Repeat("d", 64)
			if _, err := f.importSource(gate, connection, ai); err != nil {
				t.Fatal(err)
			}
			f.balance(from, "owned", "5000")
			f.balance(fees, "owned", "1")
			f.balance(to, "owned", "100")
		})
	}
}
func TestExplicitFeeEvidenceIsCountedOnce(t *testing.T) {
	f := newFixture(t)
	gate := f.admit()
	connection := f.connection(f.p)
	from, to, fees := f.create(money.RUB, "5000"), f.create(money.RUB, "0"), f.create(money.ETH, "1")
	proof := ledger.Correspondence{Kind: "transfer", Namespace: "synthetic:transfer", Reference: "transfer-fee", FromAccountID: from, ToAccountID: to}
	a := f.revision(uuid.NewString(), from, "-1000", money.RUB, 1)
	a.Type = ledger.Transfer
	a.Correspondence = &proof
	a.Postings = append(a.Postings, ledger.Posting{FeeID: "fee-1", AccountID: fees, Money: cash("-0.000000000000000123", money.ETH), Role: ledger.Fee, Funding: ledger.OwnFunds})
	b := f.revision(uuid.NewString(), to, "1000", money.RUB, 1)
	b.Type = ledger.Transfer
	b.Correspondence = &proof
	fee := f.revision(uuid.NewString(), fees, "-0.000000000000000123", money.ETH, 1)
	fee.Postings[0].Role = ledger.Fee
	fee.Postings[0].FeeID = "fee-1"
	fee.Correspondence = &proof
	for i, r := range []*ledger.Revision{&a, &b, &fee} {
		if _, err := f.importSource(gate, connection, f.sourceInput(r, []string{"out", "in", "fee"}[i])); err != nil {
			t.Fatal(err)
		}
	}
	f.balance(from, "owned", "4000")
	f.balance(to, "owned", "1000")
	f.balance(fees, "owned", "0.999999999999999877")
	g, found, err := f.store.MatchingForOperation(testContext, f.p, fee.OperationID)
	if err != nil || !found || len(g.Members) != 3 {
		t.Fatal(g, err)
	}
}
