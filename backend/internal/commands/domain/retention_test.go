package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func TestRetentionBoundariesAndReconciliation(t *testing.T) {
	actor, _ := (household.Membership{ID: "member", UserID: "owner", HouseholdID: "family", Active: true}).Principal()
	other, _ := (household.Membership{ID: "partner", UserID: "other", HouseholdID: "family", Active: true}).Principal()
	start, _ := calendar.ParseInstant("2024-01-01T00:00:00Z")
	at := func(days int, offset time.Duration) calendar.Instant {
		value, err := calendar.ParseInstant(start.Time().Add(time.Duration(days)*24*time.Hour + offset).Format(time.RFC3339Nano))
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	hash := strings.Repeat("a", 64)
	pending, err := command.NewCommand("10000000-0000-4000-8000-000000000001", "transactions.create", hash, actor, start)
	if err != nil {
		t.Fatal(err)
	}
	result := command.Result{ResourceType: "transaction", ResourceID: "record", Revision: 1}
	terminal, err := pending.Succeed(result, start)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		days   int
		offset time.Duration
		recent bool
		detail error
		replay error
	}{
		{30, -time.Nanosecond, true, nil, nil}, {30, 0, false, nil, nil},
		{90, -time.Nanosecond, false, nil, nil}, {90, 0, false, command.ErrCommandExpired, nil},
		{400, -time.Nanosecond, false, command.ErrCommandExpired, nil}, {400, 0, false, command.ErrCommandNotFound, command.ErrCommandNotFound},
	} {
		now := at(tc.days, tc.offset)
		if err := terminal.RequireDetail(actor, now); !errors.Is(err, tc.detail) {
			t.Fatalf("day %d detail: %v", tc.days, err)
		}
		if recent, err := terminal.InRecent(actor, now); err != nil || recent != tc.recent {
			t.Fatalf("recent: %v %v", recent, err)
		}
		out, err := terminal.Recover(actor, terminal.Kind(), hash, now)
		if !errors.Is(err, tc.replay) {
			t.Fatalf("day %d replay: %v", tc.days, err)
		}
		if err == nil && (out.Result != result || out.Status != command.Succeeded) {
			t.Fatal("outcome lost")
		}
	}
	if _, err := terminal.Recover(actor, terminal.Kind(), strings.Repeat("b", 64), at(100, 0)); !errors.Is(err, command.ErrDuplicateCommand) {
		t.Fatal("expired detail allowed changed payload")
	}
	if err := terminal.RequireDetail(other, at(100, 0)); !errors.Is(err, household.ErrForbidden) {
		t.Fatal("expired state leaked")
	}
	if _, err := terminal.Recover(other, terminal.Kind(), hash, at(100, 0)); !errors.Is(err, household.ErrForbidden) {
		t.Fatal("outcome leaked")
	}
	if recent, err := pending.InRecent(actor, at(900, 0)); !recent || err != nil {
		t.Fatal("unresolved command expired")
	}
	if err := pending.RequireDetail(actor, at(900, 0)); err != nil {
		t.Fatal(err)
	}
	reconciled, err := pending.Fail("version_conflict", at(900, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := reconciled.RequireDetail(actor, at(989, 0)); err != nil {
		t.Fatal("retention used registration instead of resolution")
	}
	if err := reconciled.RequireDetail(actor, at(990, 0)); !errors.Is(err, command.ErrCommandExpired) {
		t.Fatal(err)
	}
	if out, err := reconciled.Recover(actor, pending.Kind(), hash, at(1299, 0)); err != nil || out.FailureCode != "version_conflict" {
		t.Fatal("failure outcome lost")
	}
	if _, err := reconciled.Recover(actor, pending.Kind(), hash, at(1300, 0)); !errors.Is(err, command.ErrCommandNotFound) {
		t.Fatal(err)
	}
	if pending.Status() != command.Pending {
		t.Fatal("resolution mutated source")
	}
	for _, invalid := range []calendar.Instant{{}, at(-1, 0)} {
		if _, err := pending.Succeed(result, invalid); err == nil {
			t.Fatal("invalid completion time")
		}
		if _, err := terminal.InRecent(actor, invalid); err == nil {
			t.Fatal("invalid read time")
		}
	}
	if _, err := command.NewCommand(pending.ID(), pending.Kind(), hash, actor, calendar.Instant{}); err == nil {
		t.Fatal("missing registration time")
	}
}
