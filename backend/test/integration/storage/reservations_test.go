//go:build integration

package storage_test

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	goalapp "github.com/pchkauu/want-keep/backend/internal/goals/application"
	goals "github.com/pchkauu/want-keep/backend/internal/goals/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (f *fixture) goal(p household.Principal, asset money.Asset, personal bool) string {
	f.t.Helper()
	id := uuid.NewString()
	scope := household.Shared
	var owner household.UserID
	if personal {
		scope = household.Personal
		owner = p.UserID()
	}
	ownership, _ := household.NewOwnership(f.family.ID, scope, owner)
	if err := f.store.WithinHousehold(testContext, p, func(ctx context.Context) error {
		return f.store.CreateGoal(ctx, goals.Goal{ID: id, Ownership: ownership, Asset: asset, Revision: 1})
	}); err != nil {
		f.t.Fatal(err)
	}
	return id
}
func (f *fixture) reserve(p household.Principal, goal string, expected uint64, reservations []goals.Reservation) (command.Command, error) {
	key := request()
	key.Kind = "goal.reserve"
	service := goalapp.NewService(f.store)
	return f.executor.Execute(testContext, p, key, func(ctx context.Context) (command.Result, error) {
		return service.Replace(ctx, p, goal, expected, reservations, "Synthetic reservation")
	})
}
func TestConcurrentReservationsAndPersonalPermissions(t *testing.T) {
	f := newFixture(t)
	account := f.account(money.RUB, "1000")
	a, b := f.goal(f.p, money.RUB, true), f.goal(f.q, money.RUB, true)
	start := make(chan struct{})
	outcomes := make(chan command.Command, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i, p := range []household.Principal{f.p, f.q} {
		wg.Add(1)
		go func(i int, p household.Principal) {
			defer wg.Done()
			<-start
			goal := a
			if i == 1 {
				goal = b
			}
			c, e := f.reserve(p, goal, 1, []goals.Reservation{{AccountID: account, Mode: "virtual", Amount: cash("800", money.RUB)}})
			outcomes <- c
			errs <- e
		}(i, p)
	}
	close(start)
	wg.Wait()
	close(outcomes)
	close(errs)
	successes, failures := 0, 0
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	for c := range outcomes {
		if c.Status() == command.Succeeded {
			successes++
		} else if c.Status() == command.Failed && c.ErrorCode() == "invalid_availability" {
			failures++
		}
	}
	if successes != 1 || failures != 1 || f.count("reservations") != 1 {
		t.Fatalf("concurrent reserves: %d/%d", successes, failures)
	}
	g, err := f.store.Goal(testContext, f.q, a)
	if err != nil {
		t.Fatal(err)
	}
	denied, err := f.reserve(f.q, a, g.Revision, nil)
	if err != nil || denied.Status() != command.Failed || denied.ErrorCode() != "forbidden" {
		t.Fatalf("personal ownership bypass: %v", err)
	}
	// Confirmed financial facts remain valid even when existing reserves become underfunded.
	if _, err = f.write(f.revision(uuid.NewString(), account, "-500", money.RUB, 1), request()); err != nil {
		t.Fatal(err)
	}
	if f.available(account) != "500" || f.count("reservations") != 1 {
		t.Fatal("bank fact rejected or reserve silently changed")
	}
}
func TestReservationModesCurrenciesAndAtomicSwitch(t *testing.T) {
	f := newFixture(t)
	account := f.account(money.USD, "100")
	otherCurrency := f.account(money.RUB, "1000")
	a, b := f.goal(f.p, money.USD, false), f.goal(f.q, money.USD, false)
	c, err := f.reserve(f.p, a, 1, []goals.Reservation{{AccountID: account, Mode: "virtual", Amount: cash("80", money.USD)}})
	if err != nil || c.Status() != command.Succeeded {
		t.Fatal(err)
	}
	c, err = f.reserve(f.q, b, 1, []goals.Reservation{{AccountID: account, Mode: "virtual", Amount: cash("30", money.USD)}})
	if err != nil || c.Status() != command.Failed {
		t.Fatal("overallocated USD")
	}
	c, err = f.reserve(f.q, b, 1, []goals.Reservation{{AccountID: otherCurrency, Mode: "virtual", Amount: cash("1", money.USD)}})
	if err != nil || c.ErrorCode() != "asset_mismatch" {
		t.Fatal("cross-currency reserve")
	}
	c, err = f.reserve(f.p, a, 2, []goals.Reservation{{AccountID: account, Mode: "dedicated"}})
	if err != nil || c.Status() != command.Succeeded {
		t.Fatalf("switch failed: %v", err)
	}
	c, err = f.reserve(f.q, b, 1, []goals.Reservation{{AccountID: account, Mode: "virtual", Amount: cash("1", money.USD)}})
	if err != nil || c.Status() != command.Failed {
		t.Fatal("dedicated money reused")
	}
	// Old virtual revision remains audit-only, then the latest dedicated revision is released.
	c, err = f.reserve(f.q, a, 3, nil)
	if err != nil || c.Status() != command.Succeeded {
		t.Fatal("shared goal permission")
	}
	c, err = f.reserve(f.q, b, 1, []goals.Reservation{{AccountID: account, Mode: "virtual", Amount: cash("100", money.USD)}})
	if err != nil || c.Status() != command.Succeeded {
		t.Fatal("released funds unavailable")
	}
	before := f.count("reservation_revisions")
	key := request()
	key.Kind = "goal.reserve"
	c, err = f.executor.Execute(testContext, f.p, key, func(ctx context.Context) (command.Result, error) {
		g, e := f.store.Goal(ctx, f.p, b)
		if e != nil {
			return command.Result{}, e
		}
		g.Revision++
		if e = f.store.WriteReservations(ctx, g, nil, "will rollback"); e != nil {
			return command.Result{}, e
		}
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	})
	if err != nil || c.Status() != command.Failed || f.count("reservation_revisions") != before {
		t.Fatalf("partial rejected reserve: %v", err)
	}
}
