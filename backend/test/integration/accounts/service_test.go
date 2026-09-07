//go:build integration

package accounts_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (f *fixture) service() *accounts.Service {
	return accounts.NewService(f.store, f.store, func() calendar.Instant { return f.now }, uuid.NewString)
}
func (f *fixture) create(asset money.Asset, amount string) string {
	f.t.Helper()
	o, _ := household.NewOwnership(f.family.ID, household.Personal, f.p.UserID())
	date, _ := calendar.ParseDate("2026-08-01")
	if day := f.now.Time().Format("2006-01-02"); day < date.String() {
		date, _ = calendar.ParseDate(day)
	}
	c, err := f.executor.Execute(testContext, f.p, request(), func(ctx context.Context) (command.Result, error) {
		return f.service().Create(ctx, f.p, accounts.CreateInput{Name: "Cash", Asset: asset, Balance: cash(amount, asset), Ownership: o, Date: date})
	})
	if err != nil {
		f.t.Fatal(err)
	}
	result, ok := c.Result()
	if !ok {
		f.t.Fatal(c.ErrorCode())
	}
	return result.ResourceID
}
func (f *fixture) correct(id, date, amount string, p household.Principal) (command.Command, error) {
	a, err := f.store.Account(testContext, p, id)
	if err != nil {
		return command.Command{}, err
	}
	d, _ := calendar.ParseDate(date)
	v, _ := account.CashAmounts(cash(amount, a.Asset))
	return f.executor.Execute(testContext, p, request(), func(ctx context.Context) (command.Result, error) {
		return f.service().CorrectOpening(ctx, p, id, accounts.Correction{ExpectedRevision: a.Revision, Date: d, Amounts: v, Reason: "Confirmed correction"})
	})
}
func TestOpeningHistoryCorrectionAndDateMovement(t *testing.T) {
	f := newFixture(t)
	id := f.create(money.RUB, "5000")
	r := f.revision(uuid.NewString(), id, "-500", money.RUB, 1)
	r.OccurredAt = instant("2026-08-02T12:00:00Z")
	r.CashDate, _ = calendar.ParseDate("2026-08-02")
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	if f.available(id) != "4500" {
		t.Fatal("wrong balance")
	}
	if c, err := f.correct(id, "2026-08-03", "7000", f.q); err != nil || c.Status() != command.Succeeded {
		t.Fatal(c.ErrorCode(), err)
	}
	if f.available(id) != "7000" {
		t.Fatal("old effect included")
	}
	if _, err := f.correct(id, "2026-08-01", "6000", f.p); err != nil {
		t.Fatal(err)
	}
	if f.available(id) != "5500" {
		t.Fatal("history lost")
	}
	if f.count("operations") != 2 || f.count("account_openings") != 3 || f.count("account_observations") != 0 {
		t.Fatal("rewrote history or source")
	}
	var income int
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.operation_revisions WHERE economic_type='income'`).Scan(&income); err != nil || income != 0 {
		t.Fatal("opening became income", err)
	}
	r.Revision = 2
	r.State = "reversed"
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	if f.available(id) != "6000" {
		t.Fatal("reversal not projected")
	}
}
func TestAccountCommandsReplayRaceAndOwnership(t *testing.T) {
	f := newFixture(t)
	id := f.create(money.RUB, "1000")
	a, _ := f.store.Account(testContext, f.p, id)
	shared, _ := household.NewOwnership(f.family.ID, household.Shared, "")
	c, err := f.executor.Execute(testContext, f.q, request(), func(ctx context.Context) (command.Result, error) {
		return f.service().ChangeOwnership(ctx, f.q, id, a.Revision, shared, "Partner attempt")
	})
	if err != nil || c.ErrorCode() != "forbidden" {
		t.Fatal("partner changed ownership", err)
	}
	var group sync.WaitGroup
	start := make(chan struct{})
	results := make(chan command.Command, 2)
	failures := make(chan error, 2)
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			c, e := f.executor.Execute(testContext, f.p, request(), func(ctx context.Context) (command.Result, error) {
				return f.service().ChangeOwnership(ctx, f.p, id, a.Revision, shared, "Owner change")
			})
			results <- c
			failures <- e
		}()
	}
	close(start)
	group.Wait()
	close(results)
	close(failures)
	success := 0
	conflicts := 0
	for c := range results {
		if c.Status() == command.Succeeded {
			success++
		}
		if c.ErrorCode() == "version_conflict" {
			conflicts++
		}
	}
	for e := range failures {
		if e != nil {
			t.Fatal(e)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatal("race not fenced")
	}
	key := request()
	input := accounts.Correction{ExpectedRevision: a.Revision + 1, Date: a.OpeningDate, Reason: "Recount"}
	input.Amounts, _ = account.CashAmounts(cash("900", a.Asset))
	apply := func(ctx context.Context) (command.Result, error) {
		return f.service().CorrectOpening(ctx, f.p, id, input)
	}
	first, err := f.executor.Execute(testContext, f.p, key, apply)
	if err != nil {
		t.Fatal(err)
	}
	again, err := f.executor.Execute(testContext, f.p, key, func(context.Context) (command.Result, error) { t.Error("replayed apply"); return command.Result{}, nil })
	if err != nil || again.Status() != first.Status() {
		t.Fatal(err)
	}
	key.PayloadHash = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err = f.executor.Execute(testContext, f.p, key, apply); !errors.Is(err, command.ErrDuplicateCommand) {
		t.Fatal("changed payload allowed", err)
	}
}
func TestOpeningRollbackAndMoneyPrecision(t *testing.T) {
	f := newFixture(t)
	for _, asset := range []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH} {
		id := f.create(asset, "0.0000000000000000000000000123")
		if f.available(id) != "0.0000000000000000000000000123" {
			t.Fatal("precision lost")
		}
	}
	before := f.count("accounts")
	o, _ := household.NewOwnership(f.family.ID, household.Shared, "")
	date, _ := calendar.ParseDate("2026-08-01")
	crash := errors.New("injected rollback")
	_, err := f.executor.Execute(testContext, f.p, request(), func(ctx context.Context) (command.Result, error) {
		_, e := f.service().Create(ctx, f.p, accounts.CreateInput{Name: "Rolled back", Asset: money.RUB, Balance: cash("5000", money.RUB), Ownership: o, Date: date})
		if e != nil {
			return command.Result{}, e
		}
		return command.Result{}, crash
	})
	if !errors.Is(err, crash) || f.count("accounts") != before {
		t.Fatal("partial account after rollback", err)
	}
}
