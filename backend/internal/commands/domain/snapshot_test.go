package domain_test

import (
	"strings"
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
)

func TestRestoreCommandValidatesDurableShape(t *testing.T) {
	at, _ := calendar.ParseInstant("2026-09-07T12:00:00.123456789Z")
	base := command.Snapshot{ID: "11111111-1111-4111-8111-111111111111", Kind: "transaction.create", PayloadHash: strings.Repeat("a", 64), HouseholdID: "household", ActorID: "user", Status: command.Pending, RegisteredAt: at}
	for _, status := range []command.Status{command.Pending, command.Succeeded, command.Failed} {
		state := base
		state.Status = status
		if status != command.Pending {
			state.CompletedAt = at
		}
		if status == command.Succeeded {
			state.Result = command.Result{ResourceType: "transaction", ResourceID: "result", Revision: command.MaxRevision}
		}
		if status == command.Failed {
			state.ErrorCode = "invalid_money"
		}
		restored, err := command.Restore(state)
		if err != nil || restored.Snapshot() != state {
			t.Fatalf("roundtrip %s: %v", status, err)
		}
	}
	for _, mutate := range []func(*command.Snapshot){func(s *command.Snapshot) { s.CompletedAt = at }, func(s *command.Snapshot) { s.Status = command.Succeeded }, func(s *command.Snapshot) { s.PayloadHash = "invalid" }, func(s *command.Snapshot) { s.ActorID = "" }, func(s *command.Snapshot) { s.Status = command.Failed; s.CompletedAt = at; s.ErrorCode = "bad code" }} {
		invalid := base
		mutate(&invalid)
		if _, err := command.Restore(invalid); err == nil {
			t.Fatal("invalid snapshot accepted")
		}
	}
}
