package domain_test

import (
	"reflect"
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type examples struct{}

func (examples) money(value string, asset money.Asset) money.Money {
	v, e := money.NewMoney(value, asset)
	if e != nil {
		panic(e)
	}
	return v
}
func (e examples) expense() ledger.Revision {
	at, _ := calendar.ParseInstant("2026-08-31T19:00:00.123456789Z")
	posted, _ := calendar.ParseInstant("2026-09-02T10:00:00Z")
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	date, _ := at.DateIn(zone)
	month, _ := calendar.ParseMonth("2026-08")
	return ledger.Revision{OperationID: "purchase", Revision: 1, ActorID: "a", Reason: "Synthetic purchase", Type: ledger.Expense, State: ledger.Posted, OccurredAt: at, PostedAt: posted, Timezone: zone, CashDate: date, ExpenseMonth: month, Origin: "source", FeeKnowledge: ledger.KnownFees, PayerState: "unknown", Postings: []ledger.Posting{{AccountID: "cash", Money: e.money("-500", money.RUB), Role: ledger.Principal, Funding: ledger.OwnFunds}}}
}
func (examples) amount(t *testing.T, a reporting.Amount, want string) {
	t.Helper()
	v, ok := a.Value()
	if !ok || v.Amount() != want {
		t.Fatalf("amount=%v known=%v want=%s", v.Amount(), ok, want)
	}
}

func TestLedgerTransitions(t *testing.T) {
	states := []ledger.State{ledger.Draft, ledger.Pending, ledger.Posted, ledger.Cancelled, ledger.Reversed}
	allowed := map[ledger.State][]ledger.State{ledger.Draft: {ledger.Draft, ledger.Pending, ledger.Posted, ledger.Cancelled}, ledger.Pending: {ledger.Pending, ledger.Posted, ledger.Cancelled}, ledger.Posted: {ledger.Posted, ledger.Reversed}, ledger.Cancelled: {ledger.Cancelled}, ledger.Reversed: {ledger.Reversed}}
	for _, from := range states {
		for _, to := range states {
			want := false
			for _, v := range allowed[from] {
				want = want || v == to
			}
			if (from.RequireNext(to) == nil) != want {
				t.Fatalf("%s -> %s", from, to)
			}
		}
	}
	e := examples{}
	r := e.expense()
	before := r
	next := r
	next.Revision = 2
	next.State = ledger.Pending
	next.PostedAt = calendar.Instant{}
	if next.CheckSuccessor(&r) == nil || !reflect.DeepEqual(r, before) {
		t.Fatal("invalid transition mutated state")
	}
	next = r
	next.Revision = 2
	if next.CheckSuccessor(&r) != nil {
		t.Fatal("same-state correction rejected")
	}
	next.Revision = 9007199254740992
	if next.CheckSuccessor(&r) == nil {
		t.Fatal("overflow accepted")
	}
	r.State = "unknown"
	if r.Validate() == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestHoldsCreditAndEconomicFacts(t *testing.T) {
	e := examples{}
	r := e.expense()
	for _, funding := range []ledger.FundingKind{ledger.OwnFunds, ledger.CreditFunds, ledger.UnknownFunds} {
		r.Postings[0].Funding = funding
		fx, err := r.BalanceEffects()
		if err != nil {
			t.Fatal(err)
		}
		components, err := r.Components()
		if err != nil || len(components) != 1 || components[0].Kind != "expense" || components[0].Money.Amount() != "-500" {
			t.Fatal(components, err)
		}
		switch funding {
		case ledger.OwnFunds:
			e.amount(t, fx[0].Owned, "-500")
			e.amount(t, fx[0].Debt, "0")
		case ledger.CreditFunds:
			e.amount(t, fx[0].Owned, "0")
			e.amount(t, fx[0].Debt, "500")
		case ledger.UnknownFunds:
			if _, ok := fx[0].Owned.Value(); ok {
				t.Fatal("invented own split")
			}
		}
		pending := r
		pending.PostedAt = calendar.Instant{}
		pending.State = ledger.Pending
		fx, err = pending.BalanceEffects()
		if err != nil {
			t.Fatal(err)
		}
		e.amount(t, fx[0].Owned, "0")
		e.amount(t, fx[0].Debt, "0")
		if funding == ledger.OwnFunds {
			e.amount(t, fx[0].Available, "-500")
			e.amount(t, fx[0].Locked, "500")
		}
		c, _ := pending.Components()
		if len(c) != 0 {
			t.Fatal("pending became expense")
		}
		holds, err := pending.Holds()
		if err != nil || len(holds) != 1 || holds[0].Amount.Amount() != "500" {
			t.Fatal(holds, err)
		}
		for _, state := range []ledger.State{ledger.Cancelled, ledger.Reversed} {
			done := pending
			done.State = state
			fx, err = done.BalanceEffects()
			if err != nil || len(fx) != 0 {
				t.Fatal("cancelled effect", err)
			}
		}
	}
	if r.ExpenseMonth.String() != "2026-08" || r.PostedAt.Time().Month() != 9 {
		t.Fatal("recognition dates conflated")
	}
}

func TestTransferExchangeFeesAndPnL(t *testing.T) {
	e := examples{}
	r := e.expense()
	r.Type = ledger.Transfer
	r.Postings = []ledger.Posting{{AccountID: "a", Money: e.money("-1000", money.RUB), Role: ledger.Principal}, {AccountID: "b", Money: e.money("1000", money.RUB), Role: ledger.Principal}, {AccountID: "a", Money: e.money("-10", money.RUB), Role: ledger.Fee}}
	c, err := r.Components()
	if err != nil || len(c) != 1 || c[0].Kind != "fee" || c[0].Money.Amount() != "-10" {
		t.Fatal(c, err)
	}
	r.Postings[1].Money = e.money("999", money.RUB)
	if r.Validate() == nil {
		t.Fatal("unequal transfer accepted")
	}
	r.Type = ledger.Exchange
	r.Postings[0].Money = e.money("-9000", money.RUB)
	r.Postings[1].Money = e.money("100", money.USDT)
	r.Postings[2].Money = e.money("-0.00000000001", money.BTC)
	r.Postings[2].AccountID = "fee"
	x, err := r.ExchangeAmounts()
	if err != nil || x.Sent.Amount() != "9000" || x.Received.Amount() != "100" {
		t.Fatal(x, err)
	}
	r.Postings[1].AccountID = "a"
	if r.Validate() == nil {
		t.Fatal("self exchange accepted")
	}
	r = e.expense()
	r.Type = ledger.TradeResult
	r.PnLBasis = ledger.NetPnL
	r.Postings = []ledger.Posting{{AccountID: "a", Money: e.money("10", money.USDT), Role: ledger.PnL}, {AccountID: "a", Money: e.money("-1", money.USDT), Role: ledger.Fee, Treatment: ledger.Included}, {AccountID: "a", Money: e.money("5", money.USDT), Role: ledger.PnL, Treatment: ledger.Valuation}}
	d, err := r.Deltas(nil)
	if err != nil || d["a"].Amount() != "10" {
		t.Fatal("fee or valuation counted twice", err)
	}
	c, err = r.Components()
	if err != nil || len(c) != 3 || c[1].Treatment != ledger.Included || c[2].Treatment != ledger.Valuation {
		t.Fatal(c, err)
	}
	r.PnLBasis = ledger.GrossPnL
	if r.Validate() == nil {
		t.Fatal("included fee without net basis")
	}
	r.Postings[1].Treatment = ledger.Movement
	d, err = r.Deltas(nil)
	if err != nil || d["a"].Amount() != "9" {
		t.Fatal(d, err)
	}
}
