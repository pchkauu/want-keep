//go:build integration

package storage_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func TestExactMoneyAndObservationRoundTrip(t *testing.T) {
	f := newFixture(t)
	for _, asset := range []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH} {
		amount := "0." + strings.Repeat("0", 252) + "17"
		id := f.account(asset, amount)
		if got := f.available(id); got != amount {
			t.Fatalf("%s: %s", asset, got)
		}
		b, err := f.store.Balance(testContext, f.p, id, "available")
		if err != nil {
			t.Fatal(err)
		}
		if b.ObservedAt.String() != f.now.String() {
			t.Fatal("timestamp precision lost")
		}
		missing, _ := reporting.MissingAmount(reporting.Unknown, "source missing field")
		coverage, _ := reporting.NewCoverage(reporting.Partial, []string{"history_gap"})
		err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
			return f.store.RecordBalance(ctx, account.Balance{AccountID: id, Field: "available", Amount: missing, Coverage: coverage, Freshness: reporting.Stale, ObservedAt: f.now})
		})
		if err != nil {
			t.Fatal(err)
		}
		b, err = f.store.Balance(testContext, f.p, id, "available")
		if err != nil {
			t.Fatal(err)
		}
		if _, known := b.Amount.Value(); known || b.Amount.Knowledge() != reporting.Unknown || b.Coverage.State() != reporting.Partial || b.Freshness != reporting.Stale {
			t.Fatal("availability conflated")
		}
	}
	for _, bad := range []string{"NaN", "Infinity", "-Infinity", strings.Repeat("1", 257)} {
		_, err := f.admin.Exec(testContext, "SELECT $1::numeric::want_keep.amount", bad)
		if err == nil {
			t.Fatalf("accepted invalid numeric %s", bad)
		}
	}
}
func TestCommandsCrashReplayAndConcurrentRevision(t *testing.T) {
	f := newFixture(t)
	accountID := f.account(money.RUB, "1000")
	key := request()
	op := f.revision(uuid.NewString(), accountID, "-100", money.RUB, 1)
	if _, err := f.executor.Register(testContext, f.p, key); err != nil {
		t.Fatal(err)
	}
	restarted := f.restart()
	executor := commands.NewExecutor(restarted, restarted, func() calendar.Instant { return f.now })
	pending, err := restarted.LoadCommand(testContext, f.p, key.ID)
	if err != nil || pending.Status() != command.Pending {
		t.Fatalf("registration lost: %v", err)
	}
	injected := errors.New("simulated crash before commit")
	_, err = f.executor.Execute(testContext, f.p, key, func(ctx context.Context) (command.Result, error) {
		if e := f.writer.Append(ctx, f.p, op, 0); e != nil {
			return command.Result{}, e
		}
		return command.Result{}, injected
	})
	if !errors.Is(err, injected) || f.count("operations") != 0 || f.count("outbox") != 0 || f.available(accountID) != "1000" {
		t.Fatalf("partial rollback: %v", err)
	}
	start := make(chan struct{})
	var calls atomic.Int32
	var group sync.WaitGroup
	errs := make(chan error, 12)
	for range 12 {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, e := f.executor.Execute(testContext, f.p, key, func(ctx context.Context) (command.Result, error) {
				calls.Add(1)
				e := f.writer.Append(ctx, f.p, op, 0)
				return command.Result{ResourceType: "transaction", ResourceID: op.OperationID, Revision: 1}, e
			})
			errs <- e
		}()
	}
	close(start)
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 || f.count("postings") != 1 || f.count("outbox") != 1 || f.available(accountID) != "900" {
		t.Fatal("duplicate financial effect")
	}
	// The committed response was lost; the new process must recover before rechecking the old revision.
	c, err := executor.Execute(testContext, f.p, key, func(context.Context) (command.Result, error) {
		t.Error("replayed effect")
		return command.Result{}, nil
	})
	if err != nil || c.Status() != command.Succeeded {
		t.Fatalf("restart replay: %v", err)
	}
	altered := key
	altered.PayloadHash = strings.Repeat("b", 64)
	if _, err = executor.Execute(testContext, f.p, altered, func(context.Context) (command.Result, error) {
		t.Error("altered payload executed")
		return command.Result{}, nil
	}); !errors.Is(err, command.ErrDuplicateCommand) {
		t.Fatalf("payload changed: %v", err)
	}
	if _, err = f.store.LoadCommand(testContext, f.q, key.ID); !errors.Is(err, command.ErrCommandNotFound) {
		t.Fatal("partner saw private command status")
	}
	var applied atomic.Int32
	start = make(chan struct{})
	errs = make(chan error, 2)
	for _, amount := range []string{"-120", "-130"} {
		group.Add(1)
		go func(value string) {
			defer group.Done()
			<-start
			r := f.revision(op.OperationID, accountID, value, money.RUB, 2)
			r.HumanOverride = true
			c, e := f.write(r, request())
			if e == nil && c.Status() == command.Succeeded {
				applied.Add(1)
			}
			errs <- e
		}(amount)
	}
	close(start)
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if applied.Load() != 1 || f.count("operation_revisions") != 2 {
		t.Fatal("revision race overwrote history")
	}
	old, err := f.store.LedgerRevision(testContext, f.p, op.OperationID, 1)
	if err != nil || old.Postings[0].Money.Amount() != "-100" {
		t.Fatal("original changed")
	}
}
func TestMultiPostingAtomicityAndScope(t *testing.T) {
	f := newFixture(t)
	a := f.account(money.USD, "100")
	b := f.account(money.USD, "0")
	r := f.revision(uuid.NewString(), a, "-10", money.USD, 1)
	r.Type = "transfer"
	r.Postings = append(r.Postings, ledger.Posting{AccountID: b, Money: cash("10", money.USD), Role: "principal"})
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	if f.available(a) != "90" || f.available(b) != "10" {
		t.Fatal("transfer did not conserve funds")
	}
	foreign := household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Other household"}
	user := household.User{ID: household.UserID(uuid.NewString()), Name: "Other member"}
	member := household.Membership{ID: household.MembershipID(uuid.NewString()), UserID: user.ID, HouseholdID: foreign.ID, Active: true}
	zone, _ := calendar.ParseTimezone("UTC")
	if err := f.store.InitializeHousehold(testContext, foreign, []household.User{user}, []household.Membership{member}, zone, 2); err != nil {
		t.Fatal(err)
	}
	other, _ := member.Principal()
	if _, err := f.store.Account(testContext, other, a); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("foreign account exposed: %v", err)
	}
	forged := r
	forged.OperationID = uuid.NewString()
	forged.ActorID = other.UserID()
	if _, err := f.write(forged, request()); !errors.Is(err, household.ErrForbidden) {
		t.Fatalf("forged actor accepted: %v", err)
	}
	bad := r
	bad.OperationID = uuid.NewString()
	bad.Postings = []ledger.Posting{{AccountID: a, Money: cash("-1", money.RUB), Role: "principal"}}
	if _, err := f.write(bad, request()); err == nil {
		t.Fatal("asset mismatch accepted")
	}
	if f.count("operations") != 1 {
		t.Fatal("invalid posting partially committed")
	}
}
func TestHouseholdMemberCapAndHistoryPrivileges(t *testing.T) {
	f := newFixture(t)
	err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		user := household.User{ID: household.UserID(uuid.NewString()), Name: "Third"}
		return f.store.AddMember(ctx, user, household.Membership{ID: household.MembershipID(uuid.NewString()), UserID: user.ID, HouseholdID: f.family.ID, Active: true})
	})
	if !errors.Is(err, household.ErrInvalidMembership) {
		t.Fatalf("member limit: %v", err)
	}
	id := f.account(money.BTC, "1")
	if _, err = f.write(f.revision(uuid.NewString(), id, "-0.1", money.BTC, 1), request()); err != nil {
		t.Fatal(err)
	}
	tx, err := f.admin.Begin(testContext)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(testContext)
	for _, query := range []string{"UPDATE want_keep.postings SET amount=0", "DELETE FROM want_keep.operation_revisions"} {
		if _, err = tx.Exec(testContext, "SAVEPOINT probe"); err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(testContext, query); err == nil {
			t.Fatal("history mutation accepted")
		}
		if _, err = tx.Exec(testContext, "ROLLBACK TO SAVEPOINT probe"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = tx.Exec(testContext, "SET LOCAL ROLE want_keep_app"); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"CREATE TABLE want_keep.forbidden(id int)", "DELETE FROM want_keep.command_tombstones", "DELETE FROM want_keep.users"} {
		if _, err = tx.Exec(testContext, "SAVEPOINT privilege_probe"); err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(testContext, query); err == nil {
			t.Fatalf("app privilege too broad: %s", query)
		}
		if _, err = tx.Exec(testContext, "ROLLBACK TO SAVEPOINT privilege_probe"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNestedRollbackAndIndependentCommandRegistration(t *testing.T) {
	f := newFixture(t)
	account := f.account(money.RUB, "100")
	r := f.revision(uuid.NewString(), account, "-10", money.RUB, 1)
	err := f.store.WithinAdmission(testContext, binding().Provider, binding().Environment, func(ctx context.Context) error {
		nestedErr := f.store.WithinHousehold(ctx, f.p, func(ctx context.Context) error {
			if err := f.writer.Append(ctx, f.p, r, 0); err != nil {
				return err
			}
			return commands.Rejection{Code: "invalid_allocation"}
		})
		var rejection commands.Rejection
		if !errors.As(nestedErr, &rejection) {
			return fmt.Errorf("nested rejection not propagated: %v", nestedErr)
		}
		// Rolling back also restores lock metadata; the household lock no longer exists.
		return f.store.WithinAdmission(ctx, binding().Provider, binding().Environment, func(context.Context) error { return nil })
	})
	if err != nil || f.count("postings") != 0 || f.count("outbox") != 0 || f.available(account) != "100" {
		t.Fatalf("nested writes escaped rollback: %v", err)
	}
	key := request()
	err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		_, err := f.executor.Execute(ctx, f.p, key, func(context.Context) (command.Result, error) {
			t.Error("nested command executed")
			return command.Result{}, commands.Rejection{Code: "invalid_allocation"}
		})
		if err == nil {
			return fmt.Errorf("registration joined ambient transaction")
		}
		return nil
	})
	if err != nil || f.count("command_tombstones") != 0 || f.count("postings") != 0 {
		t.Fatalf("nested registration accepted: %v", err)
	}
	if _, err = f.write(r, key); err != nil || f.available(account) != "90" {
		t.Fatalf("top-level command failed: %v", err)
	}
}

func TestRecoveredNestedPanicRollsBackSavepoint(t *testing.T) {
	f := newFixture(t)
	account := f.account(money.RUB, "100")
	r := f.revision(uuid.NewString(), account, "-10", money.RUB, 1)
	err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("expected nested panic")
				}
			}()
			_ = f.store.WithinHousehold(ctx, f.p, func(ctx context.Context) error {
				if err := f.writer.Append(ctx, f.p, r, 0); err != nil {
					return err
				}
				panic("synthetic nested failure")
			})
		}()
		return nil
	})
	if err != nil || f.count("postings") != 0 || f.count("outbox") != 0 || f.available(account) != "100" {
		t.Fatalf("recovered panic retained nested writes: %v", err)
	}
}
