//go:build integration

package ledger_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (f *fixture) balance(id, field, want string) {
	f.t.Helper()
	b, err := f.store.Balance(testContext, f.p, id, field)
	if err != nil {
		f.t.Fatal(err)
	}
	v, ok := b.Amount.Value()
	if want == "unknown" {
		if ok {
			f.t.Fatal("expected unknown", v.Amount())
		}
		return
	}
	if !ok || v.Amount() != want {
		f.t.Fatalf("%s=%s known=%v want %s", field, v.Amount(), ok, want)
	}
}

func TestPendingPostingCancellationAndPurchaseMonth(t *testing.T) {
	f := newFixture(t)
	id := f.create(money.RUB, "5000")
	r := f.revision(uuid.NewString(), id, "-500", money.RUB, 1)
	r.State = ledger.Pending
	r.OccurredAt = instant("2026-08-31T20:59:59.999999999Z")
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	f.balance(id, "owned", "5000")
	f.balance(id, "available", "4500")
	f.balance(id, "locked", "500")
	r.Revision = 2
	r.State = ledger.Posted
	r.PostedAt = instant("2026-09-02T10:00:00.123456789Z")
	key := request()
	if _, err := f.write(r, key); err != nil {
		t.Fatal(err)
	}
	if _, err := f.write(r, key); err != nil {
		t.Fatal("replay", err)
	}
	f.balance(id, "owned", "4500")
	f.balance(id, "available", "4500")
	f.balance(id, "locked", "0")
	saved, _, err := f.store.CurrentLedgerRevision(testContext, f.p, r.OperationID)
	if err != nil || saved.ExpenseMonth.String() != "2026-08" || saved.PostedAt != r.PostedAt {
		t.Fatal("purchase month/posted nanos", err)
	}
	r.Revision = 3
	r.State = ledger.Pending
	r.PostedAt = calendar.Instant{}
	if _, err = f.write(r, request()); !errors.Is(err, ledger.ErrInvalidTransition) {
		t.Fatal("backwards transition", err)
	}
	f.balance(id, "owned", "4500")
	r = f.revision(uuid.NewString(), id, "-300", money.RUB, 1)
	r.State = ledger.Pending
	if _, err = f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	r.Revision = 2
	r.State = ledger.Cancelled
	if _, err = f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	f.balance(id, "owned", "4500")
	f.balance(id, "available", "4500")
	f.balance(id, "locked", "0")
	if f.count("account_observations") != 0 {
		t.Fatal("journal invented a source observation")
	}
}

func TestCreditPurchaseRepaymentAndUnknownSplit(t *testing.T) {
	f := newFixture(t)
	cashID := f.create(money.RUB, "5000")
	creditID := uuid.NewString()
	ownership, _ := f.store.Account(testContext, f.p, cashID)
	date, _ := calendar.ParseDate("2026-08-01")
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		a := account.Account{ID: creditID, Name: "Credit", Ownership: ownership.Ownership, Asset: money.RUB, Product: "credit_card", Revision: 1, OpeningDate: date}
		if err := f.store.CreateAccount(ctx, a); err != nil {
			return err
		}
		values, _ := account.CashAmounts(cash("0", money.RUB))
		openingID := uuid.NewString()
		opening := f.revision(openingID, creditID, "0", money.RUB, 1)
		opening.Type = ledger.Opening
		opening.ExpenseMonth = calendar.Month{}
		opening.Allocation = ledger.NotApplicableAllocation()
		if err := f.store.AppendRevision(ctx, opening, 0); err != nil {
			return err
		}
		return f.store.SaveOpening(ctx, account.Opening{AccountID: creditID, OperationID: openingID, Revision: 1, Date: date, Timezone: zone, Confirmed: true, Amounts: values, ActorID: f.p.UserID(), Reason: "Synthetic proven start", At: f.now})
	})
	if err != nil {
		t.Fatal(err)
	}
	r := f.revision(uuid.NewString(), creditID, "-1000", money.RUB, 1)
	r.Postings[0].Funding = ledger.CreditFunds
	if _, err = f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	f.balance(creditID, "owned", "0")
	f.balance(creditID, "debt", "1000")
	r = f.revision(uuid.NewString(), cashID, "-1000", money.RUB, 1)
	r.Type = ledger.Transfer
	r.Postings = append(r.Postings, ledger.Posting{AccountID: creditID, Money: cash("1000", money.RUB), Role: ledger.Principal, Funding: ledger.CreditFunds})
	if _, err = f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	f.balance(cashID, "owned", "4000")
	f.balance(creditID, "owned", "0")
	f.balance(creditID, "debt", "0")
	components, _ := r.Components()
	if len(components) != 0 {
		t.Fatal("repayment became expense")
	}
	r = f.revision(uuid.NewString(), creditID, "-100", money.RUB, 1)
	if _, err = f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	f.balance(creditID, "owned", "unknown")
	f.balance(creditID, "debt", "unknown")
	saved, _, err := f.store.CurrentLedgerRevision(testContext, f.p, r.OperationID)
	if err != nil || saved.Postings[0].Funding != ledger.UnknownFunds {
		t.Fatal("credit funding guessed", err)
	}
}

func TestLedgerRollbackRestartReplayAndRevisionRace(t *testing.T) {
	f := newFixture(t)
	id := f.create(money.USDC, "1000")
	r := f.revision(uuid.NewString(), id, "-100", money.USDC, 1)
	key := request()
	if _, err := f.executor.Register(testContext, f.p, key); err != nil {
		t.Fatal(err)
	}
	boom := errors.New("synthetic rollback")
	_, err := f.executor.ExecuteRegistered(testContext, f.p, key, func(ctx context.Context) (command.Result, error) {
		if err := f.writer.Append(ctx, f.p, r, 0); err != nil {
			return command.Result{}, err
		}
		return command.Result{}, boom
	})
	if !errors.Is(err, boom) {
		t.Fatal(err)
	}
	f.balance(id, "owned", "1000")
	f.restart()
	pending, err := f.store.LoadCommand(testContext, f.p, key.ID)
	if err != nil || pending.Status() != command.Pending {
		t.Fatal("lost pending", err)
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; _, err := f.write(r, key); errs <- err }()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	f.balance(id, "owned", "900")
	changed := key
	changed.PayloadHash = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err = f.write(r, changed); !errors.Is(err, command.ErrDuplicateCommand) {
		t.Fatal("payload changed", err)
	}
	start = make(chan struct{})
	var succeeded atomic.Int32
	for _, amount := range []string{"-200", "-300"} {
		wg.Add(1)
		go func(amount string) {
			defer wg.Done()
			<-start
			next := r
			next.Postings = append([]ledger.Posting(nil), r.Postings...)
			next.Revision = 2
			next.Postings[0].Money = cash(amount, money.USDC)
			c, err := f.write(next, request())
			if err != nil {
				t.Error(err)
			}
			if c.Status() == command.Succeeded {
				succeeded.Add(1)
			}
		}(amount)
	}
	close(start)
	wg.Wait()
	if succeeded.Load() != 1 {
		t.Fatal("revision race")
	}
	old, err := f.store.LedgerRevision(testContext, f.p, r.OperationID, 1)
	if err != nil || old.Postings[0].Money.Amount() != "-100" {
		t.Fatal("history rewritten", err)
	}
	if f.count("operation_revisions") != 3 {
		t.Fatal("partial or duplicate revision")
	}
	// A saved terminal command is replayed before invoking any new financial application.
	if _, err = f.executor.Execute(testContext, f.p, key, func(context.Context) (command.Result, error) {
		t.Error("executed after commit")
		return command.Result{}, nil
	}); err != nil {
		t.Fatal(err)
	}
}
