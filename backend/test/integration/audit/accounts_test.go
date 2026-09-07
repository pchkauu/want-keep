//go:build integration

package audit_test

import (
	"context"

	"github.com/google/uuid"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
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

func (f *fixture) ledgerService() *journal.Service {
	return journal.NewService(f.store, f.writer, func() calendar.Instant { return f.now }, uuid.NewString)
}

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
