package domain_test

import (
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type fixture struct{ t *testing.T }

func (f fixture) fact(id, account, amount string, asset money.Asset) ledger.Revision {
	f.t.Helper()
	at, _ := calendar.ParseInstant("2026-09-01T12:00:00.123456789Z")
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	m, err := money.NewMoney(amount, asset)
	if err != nil {
		f.t.Fatal(err)
	}
	kind := ledger.Expense
	if m.Sign() > 0 {
		kind = ledger.Income
	}
	r := ledger.Revision{OperationID: id, ActorID: "member-a", Revision: 1, Reason: "synthetic", Origin: "source", State: ledger.Posted, Type: kind, OccurredAt: at, PayerState: "unknown", FeeKnowledge: ledger.KnownFees, Postings: []ledger.Posting{{AccountID: account, Money: m, Role: ledger.Principal, Funding: ledger.OwnFunds, Treatment: ledger.Movement}}}
	r, err = r.InTimezone(zone)
	if err != nil {
		f.t.Fatal(err)
	}
	return r
}
func (f fixture) group(kind matching.Kind, primary string) matching.Group {
	at, _ := calendar.ParseInstant("2026-09-07T12:00:00.987654321Z")
	return matching.Group{ID: "group", PrimaryID: primary, Revision: 1, Kind: kind, State: matching.Clarification, ActorID: "member-a", At: at, Reason: "confirmed", CandidatesComplete: true}
}

func TestPaymentHasOneEffectAndPreservesOriginals(t *testing.T) {
	f := fixture{t}
	for _, asset := range []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH} {
		a := f.fact("a", "account", "-0.1234567890123456789", asset)
		a.Origin = "manual"
		b := f.fact("b", "account", "-0.1234567890123456789", asset)
		g, revs, err := f.group(matching.Payment, "a").Assign([]ledger.Revision{b, a}, false)
		if err != nil || g.State != matching.Linked {
			t.Fatalf("%s: %v", asset, err)
		}
		count := 0
		for _, r := range revs {
			effects, err := r.BalanceEffects()
			if err != nil {
				t.Fatal(err)
			}
			count += len(effects)
			if r.OperationID == "b" && r.Allocation.State != ledger.AllocationNotApplicable {
				t.Fatalf("%s non-carrier allocation = %+v", asset, r.Allocation)
			}
			if len(effects) > 0 {
				m, _ := effects[0].Owned.Value()
				if m.Amount() != "-0.1234567890123456789" {
					t.Fatal(m.Amount())
				}
			}
		}
		if count != 1 || a.Participation.GroupID != "" || b.Participation.GroupID != "" {
			t.Fatal("duplicate effect or input mutation")
		}
	}
}

func TestPaymentReallocatesPreviouslyUnresolvedParticipants(t *testing.T) {
	f := fixture{t}
	a, b := f.fact("a", "account", "-300", money.RUB), f.fact("b", "account", "-300", money.RUB)
	var err error
	for _, revision := range []*ledger.Revision{&a, &b} {
		*revision, err = revision.WithAllocation(ledger.AllocationInput{Mode: ledger.AllocationUnknown, Reason: "allocation_unresolved"}, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	b.Participation = ledger.Participation{GroupID: "candidate", Kind: "payment", State: "waiting"}
	_, revisions, err := f.group(matching.Payment, "a").Assign([]ledger.Revision{a, b}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, revision := range revisions {
		if err := revision.Validate(); err != nil {
			t.Fatalf("%s: %v allocation=%+v participation=%+v", revision.OperationID, err, revision.Allocation, revision.Participation)
		}
	}
}

func TestTransferAndExchangeKeepNativeEffects(t *testing.T) {
	f := fixture{t}
	for _, tc := range []struct {
		kind           matching.Kind
		sent, received string
		asset          money.Asset
	}{{matching.Transfer, "-1000", "1000", money.RUB}, {matching.Exchange, "-9000", "100", money.USDT}} {
		a, b := f.fact("a", "from", tc.sent, money.RUB), f.fact("b", "to", tc.received, tc.asset)
		fee, _ := money.NewMoney("-0.00001", money.BTC)
		a.Postings = append(a.Postings, ledger.Posting{AccountID: "fee-account", Money: fee, Role: ledger.Fee, Funding: ledger.OwnFunds, Treatment: ledger.Movement})
		g, rs, err := f.group(tc.kind, "a").Assign([]ledger.Revision{a, b}, false)
		if err != nil || g.State != matching.Linked {
			t.Fatal(err)
		}
		components := 0
		for _, r := range rs {
			cs, err := r.Components()
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range cs {
				if c.Kind != "fee" {
					t.Fatal("principal became income or expense")
				}
			}
			components += len(cs)
			if r.OperationID == "a" && (r.Allocation.State != ledger.AllocationUnresolved || len(r.Allocation.Unallocated) != 1 || r.Allocation.Unallocated[0].Asset() != money.BTC) {
				t.Fatalf("fee-only allocation = %+v", r.Allocation)
			}
			if r.OperationID == "b" && r.Allocation.State != ledger.AllocationNotApplicable {
				t.Fatalf("incoming allocation = %+v", r.Allocation)
			}
		}
		if components != 1 {
			t.Fatal("fee duplicated")
		}
	}
}

func TestPartialSideUsesItsOwnLifecycle(t *testing.T) {
	f := fixture{t}
	a := f.fact("a", "from", "-1000", money.RUB)
	a.State = ledger.Pending
	g, rs, err := f.group(matching.Transfer, "a").Assign([]ledger.Revision{a}, true)
	if err != nil || g.State != matching.WaitingSide {
		t.Fatal(err)
	}
	holds, _ := rs[0].Holds()
	cs, _ := rs[0].Components()
	if len(holds) != 1 || len(cs) != 0 {
		t.Fatal("partial side became expense")
	}
	b := f.fact("b", "to", "1000", money.RUB)
	g, rs, err = g.Assign([]ledger.Revision{a, b}, false)
	if err != nil || g.State != matching.Linked {
		t.Fatal(err)
	}
	for _, r := range rs {
		if r.OperationID == "a" && r.ContributionState(0) != ledger.Pending {
			t.Fatal("incoming posted fabricated outgoing posting")
		}
	}
	if _, _, err = f.group(matching.Transfer, "a").Assign([]ledger.Revision{a}, false); err == nil {
		t.Fatal("missing side accepted as complete")
	}
}

func TestBankReversalRemovesPaymentWithoutRewritingManualState(t *testing.T) {
	f := fixture{t}
	a := f.fact("a", "account", "-300", money.RUB)
	a.Origin = "manual"
	b := f.fact("b", "account", "-300", money.RUB)
	b.State = ledger.Reversed
	_, rs, err := f.group(matching.Payment, "a").Assign([]ledger.Revision{a, b}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rs {
		e, _ := r.BalanceEffects()
		if len(e) != 0 {
			t.Fatal("reversed payment still affects cash")
		}
		if r.OperationID == "a" && r.State != ledger.Posted {
			t.Fatal("original lifecycle overwritten")
		}
	}
}

func TestInvalidGroupsNeverMergeDistinctMoney(t *testing.T) {
	f := fixture{t}
	a := f.fact("a", "one", "-300", money.RUB)
	for _, b := range []ledger.Revision{f.fact("b", "two", "-300", money.RUB), f.fact("b", "one", "-301", money.RUB), f.fact("b", "one", "-300", money.USD), a} {
		if _, _, err := f.group(matching.Payment, "a").Assign([]ledger.Revision{a, b}, false); err == nil {
			t.Fatal("incompatible payment merged")
		}
	}
	for _, b := range []ledger.Revision{f.fact("b", "two", "299", money.RUB), f.fact("b", "one", "300", money.RUB), f.fact("b", "two", "300", money.USD)} {
		if _, _, err := f.group(matching.Transfer, "a").Assign([]ledger.Revision{a, b}, false); err == nil {
			t.Fatal("invalid principal pair accepted")
		}
	}
}

func TestWaitingAndUserExclusionRemainIndependent(t *testing.T) {
	f := fixture{t}
	a := f.fact("a", "one", "-300", money.RUB)
	a.Participation = ledger.Participation{GroupID: "group", Kind: "payment", State: "waiting"}
	e, _ := a.BalanceEffects()
	cs, _ := a.Components()
	h, _ := a.Holds()
	if len(e)+len(cs)+len(h) != 0 || a.Accounting() != ledger.IncludedInAccounting || a.State != ledger.Posted {
		t.Fatal("waiting altered original status")
	}
	a.Participation = ledger.Participation{}
	a.AccountingState = ledger.ExcludedFromAccounting
	b := f.fact("b", "one", "-300", money.RUB)
	_, rs, err := f.group(matching.Payment, "a").Assign([]ledger.Revision{a, b}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rs {
		e, _ := r.BalanceEffects()
		if len(e) != 0 {
			t.Fatal("link reactivated user exclusion")
		}
	}
}

func TestCorrespondenceRequiresNetworkMovementIdentity(t *testing.T) {
	c := ledger.Correspondence{Kind: "transfer", Namespace: "chain", Reference: "tx", Network: "ethereum", FromAccountID: "a", ToAccountID: "b"}
	if c.Validate() == nil {
		t.Fatal("transaction hash accepted without movement")
	}
	c.Movement = "log:1"
	if c.Validate() != nil {
		t.Fatal("complete correspondence rejected")
	}
	other := c
	other.Movement = "log:2"
	if c.Digest() == other.Digest() {
		t.Fatal("different chain movements collapsed")
	}
}

func TestDistinctFeesOnOneSideRemainDistinct(t *testing.T) {
	f := fixture{t}
	a, b := f.fact("a", "from", "-1000", money.RUB), f.fact("b", "to", "1000", money.RUB)
	for _, value := range []string{"-10", "-20"} {
		fee := f.fact("fee", "from", value, money.RUB).Postings[0]
		fee.Role = ledger.Fee
		a.Postings = append(a.Postings, fee)
	}
	_, revisions, err := f.group(matching.Transfer, "a").Assign([]ledger.Revision{a, b}, false)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, r := range revisions {
		cs, err := r.Components()
		if err != nil {
			t.Fatal(err)
		}
		count += len(cs)
	}
	if count != 2 {
		t.Fatal("distinct fees collapsed", count)
	}
}
