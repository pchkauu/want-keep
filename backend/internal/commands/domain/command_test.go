package domain_test

import (
	"errors"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	"strings"
	"testing"
)

func TestCommandReplayVisibilityAndUnknownOutcome(t *testing.T) {
	a, _ := (household.Membership{ID: "ma", UserID: "a", HouseholdID: "family", Active: true}).Principal()
	b, _ := (household.Membership{ID: "mb", UserID: "b", HouseholdID: "family", Active: true}).Principal()
	foreign, _ := (household.Membership{ID: "mc", UserID: "a", HouseholdID: "other", Active: true}).Principal()
	now, _ := calendar.ParseInstant("2026-09-07T00:00:00Z")
	hash := strings.Repeat("a", 64)
	cmd, err := command.NewCommand("10000000-0000-4000-8000-000000000001", "transactions.create", hash, a, now)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Status() != command.Pending || cmd.CheckReplay(a, cmd.Kind(), hash, now) != nil {
		t.Fatal("registration/replay failed")
	}
	if cmd.RequireVisible(b) == nil || cmd.RequireVisible(foreign) == nil {
		t.Fatal("command leaked to another principal")
	}
	if !errors.Is(cmd.CheckReplay(a, cmd.Kind(), strings.Repeat("b", 64), now), command.ErrDuplicateCommand) {
		t.Fatal("changed payload accepted")
	}
	if !errors.Is(cmd.CheckReplay(a, "goals.create", hash, now), command.ErrDuplicateCommand) {
		t.Fatal("changed command type accepted")
	}
	if !errors.Is(cmd.RequireRevision(1, 2), command.ErrVersionConflict) || cmd.Status() != command.Pending {
		t.Fatal("revision conflict mutated command")
	}
	if err := cmd.RequireRevision(2, 2); err != nil {
		t.Fatal(err)
	}
	completed, err := cmd.Succeed(command.Result{ResourceType: "transaction", ResourceID: "record", Revision: 2}, now)
	if err != nil || cmd.Status() != command.Pending || completed.Status() != command.Succeeded {
		t.Fatal("command state incorrect")
	}
	if completed.CheckReplay(a, cmd.Kind(), hash, now) != nil {
		t.Fatal("completed replay should return existing result")
	}
	if _, err := completed.Fail("internal_error", now); !errors.Is(err, command.ErrFinalCommand) {
		t.Fatal("final command changed")
	}
	if _, err := cmd.Fail("provider secret details", now); err == nil {
		t.Fatal("unsafe failure accepted")
	}
	failed, err := cmd.Fail("version_conflict", now)
	if err != nil || failed.Status() != command.Failed || failed.ErrorCode() != "version_conflict" {
		t.Fatal("failed status lost")
	}
	if _, err := command.NewCommand("bad", cmd.Kind(), hash, a, now); err == nil {
		t.Fatal("invalid ID accepted")
	}
	if _, err := command.NewCommand(cmd.ID(), cmd.Kind(), hash, household.Principal{}, now); err == nil {
		t.Fatal("untrusted empty principal accepted")
	}
}
