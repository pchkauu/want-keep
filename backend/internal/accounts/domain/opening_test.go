package domain

import (
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func TestOpeningProjectionBoundaryAndUnknown(t *testing.T) {
	asset := money.RUB
	m, _ := money.NewMoney("5000", asset)
	amounts, _ := CashAmounts(m)
	date, _ := calendar.ParseDate("2026-08-01")
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	now, _ := calendar.ParseInstant("2026-08-05T00:00:00Z")
	o := Opening{AccountID: "account", OperationID: "opening", Revision: 1, Date: date, Timezone: zone, Confirmed: true, Amounts: amounts, ActorID: "A", Reason: "test", At: now}
	before, _ := calendar.ParseInstant("2026-07-31T20:59:59.999999999Z")
	at, _ := calendar.ParseInstant("2026-07-31T21:00:00Z")
	expense, _ := money.NewMoney("-500", asset)
	out, err := o.Project(asset, []Effect{{At: before, Amount: expense}, {At: at, Amount: expense}})
	if err != nil {
		t.Fatal(err)
	}
	v, _ := out.Owned.Value()
	if v.Amount() != "4500" {
		t.Fatal(v.Amount())
	}
	old, _ := o.Amounts.Owned.Value()
	if old.Amount() != "5000" {
		t.Fatal("mutated input")
	}
	o.Confirmed = false
	out, err = o.Project(asset, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, known := out.Owned.Value(); known {
		t.Fatal("unconfirmed became zero")
	}
	o.Revision = 9007199254740992
	if o.Validate(asset) == nil {
		t.Fatal("revision overflow")
	}
}
func TestCashAssetsAndCardSafety(t *testing.T) {
	for _, asset := range []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH} {
		m, err := money.NewMoney("0.00000000000000000000000000001", asset)
		if err != nil {
			t.Fatal(err)
		}
		v, err := CashAmounts(m)
		if err != nil {
			t.Fatal(err)
		}
		if err = v.Validate(asset); err != nil {
			t.Fatal(err)
		}
	}
	for _, bad := range []CardAlias{
		{ID: "card", AccountID: "a", Label: "4242 4242 4242 4242", LastFour: "4242"},
		{ID: "card", AccountID: "a", Label: "4242.4242.4242.4242", LastFour: "4242"},
		{ID: "card", AccountID: "a", Label: "4242/4242/4242/4242", LastFour: "4242"},
		{ID: "card", AccountID: "a", Label: "4242\u200b4242\u200b4242\u200b4242", LastFour: "4242"},
		{ID: "card", AccountID: "a", Label: "٤٢٤٢", LastFour: "4242"},
		{ID: "c", AccountID: "a", Label: "Card", LastFour: "123"},
	} {
		if bad.Validate() == nil {
			t.Fatal("unsafe alias accepted")
		}
	}
	if err := (CardAlias{ID: "card", AccountID: "a", Label: "Primary card", LastFour: "1234"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (CardAlias{ID: "card", AccountID: "a", Label: "Основная карта 1234", LastFour: "1234"}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestPortfolioNeverConvertsUnknownOrCreditIntoOwnMoney(t *testing.T) {
	rub, _ := money.NewMoney("100", money.RUB)
	amounts, _ := CashAmounts(rub)
	debt, _ := money.NewMoney("300", money.RUB)
	amounts.Debt, _ = reporting.KnownAmount(debt)
	usd, _ := money.NewMoney("10", money.USD)
	dollars, _ := CashAmounts(usd)
	p := Portfolio{{AccountID: "bank", Asset: money.RUB, Amounts: amounts}, {AccountID: "cash", Asset: money.USD, Amounts: dollars}}
	totals, err := p.Totals()
	if err != nil || len(totals) != 2 {
		t.Fatal(err)
	}
	if totals[0].Owned.KnownSubtotal.Amount() != "100" || totals[0].Debt.KnownSubtotal.Amount() != "300" {
		t.Fatal("debt changed own funds")
	}
	p = append(p, Position{AccountID: "unknown", Asset: money.RUB, Amounts: UnknownAmounts("missing")})
	totals, err = p.Totals()
	if err != nil {
		t.Fatal(err)
	}
	if _, known := totals[0].Owned.Total.Value(); known || totals[0].Owned.KnownSubtotal.Amount() != "100" {
		t.Fatal("partial became total")
	}
	p = append(p, p[0])
	if _, err = p.Totals(); err == nil {
		t.Fatal("duplicate account doubled total")
	}
}

func TestSourceFundingRequiresOwnConsistentFreshFunds(t *testing.T) {
	money100, _ := money.NewMoney("100", money.RUB)
	amounts, _ := CashAmounts(money100)
	coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
	now, _ := calendar.ParseInstant("2026-09-07T12:00:00Z")
	source := Observation{Amounts: amounts, OwnAvailable: true, Coverage: coverage, Freshness: reporting.Fresh, AsOf: now}
	if _, known := source.Funding(nil).Value(); !known {
		t.Fatal("verified own funds lost")
	}
	for _, mutate := range []func(*Observation){
		func(o *Observation) { o.OwnAvailable = false },
		func(o *Observation) { o.Freshness = reporting.Stale },
		func(o *Observation) { o.Coverage, _ = reporting.NewCoverage(reporting.Partial, []string{"missing"}) },
		func(o *Observation) {
			m, _ := money.NewMoney("1100", money.RUB)
			o.Amounts.Available, _ = reporting.KnownAmount(m)
		},
		func(o *Observation) {
			m, _ := money.NewMoney("10", money.RUB)
			o.Amounts.Locked, _ = reporting.KnownAmount(m)
		},
	} {
		o := source
		mutate(&o)
		if _, known := o.Funding(nil).Value(); known {
			t.Fatal("unverified or credit funds were available")
		}
	}
	later, _ := calendar.ParseInstant("2026-09-07T12:00:00.000000001Z")
	if _, known := source.Funding([]Effect{{At: later}}).Value(); known {
		t.Fatal("newer journal effect missed")
	}
}

func TestOpeningStartsAtEarliestExistingLocalInstant(t *testing.T) {
	for _, tc := range []struct{ zone, date, want string }{
		{"America/Havana", "2026-03-08", "2026-03-08T05:00:00Z"},
		{"America/Sao_Paulo", "2018-11-04", "2018-11-04T03:00:00Z"},
		{"America/Havana", "2026-11-01", "2026-11-01T04:00:00Z"},
		{"Pacific/Apia", "2011-12-30", ""},
		{"Europe/Moscow", "2026-08-01", "2026-07-31T21:00:00Z"},
	} {
		t.Run(tc.zone+tc.date, func(t *testing.T) {
			date, _ := calendar.ParseDate(tc.date)
			zone, _ := calendar.ParseTimezone(tc.zone)
			got, err := (Opening{Date: date, Timezone: zone}).Instant()
			if tc.want == "" {
				if err == nil {
					t.Fatal("whole skipped date accepted")
				}
				return
			}
			if err != nil || got.String() != tc.want {
				t.Fatal(got.String(), err)
			}
		})
	}
}

func TestAccountEventEligibilityIsDomainOwned(t *testing.T) {
	at, _ := calendar.ParseInstant("2026-09-07T12:00:00Z")
	for _, kind := range []EventKind{Created, OpeningCorrected, OwnershipChanged} {
		for _, origin := range []EventOrigin{Interactive, LiveSync, HistoricalBackfill} {
			e := Event{AccountID: "account", Revision: 1, Kind: kind, Origin: origin, Reason: "Confirmed event", At: at}
			if err := e.Validate(); err != nil {
				t.Fatal(err)
			}
			if e.CelebrationEligible() != (kind == Created && origin != HistoricalBackfill) {
				t.Fatal("incorrect eligibility")
			}
		}
	}
	if (Event{Kind: "unverified"}).Validate() == nil {
		t.Fatal("invalid event accepted")
	}
}
