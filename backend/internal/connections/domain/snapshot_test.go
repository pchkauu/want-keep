package domain_test

import (
	"strings"
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
)

func TestRestoreAdmissionKeepsRevisionAndRejectsContradictions(t *testing.T) {
	b := connections.Binding{Provider: "bybit", Environment: "test", AdapterBuildDigest: "sha256:" + strings.Repeat("a", 64), CollectorImageDigest: "sha256:" + strings.Repeat("b", 64), ContractVersion: "10", AllowlistRevision: "1", NonSecretConfigRevision: "1", OperatorPermissionRevision: "1"}
	at, _ := calendar.ParseInstant("2026-09-07T12:00:00.123456789Z")
	snapshot := connections.Snapshot{Binding: b, Revision: 9007199254740991, Provider: connections.Check{Kind: connections.ProviderCheck, Binding: b, Result: connections.CheckPassed, At: at}, Host: connections.Check{Kind: connections.HostCheck, Binding: b, Result: connections.CheckPassed, At: at}}
	a, err := connections.Restore(snapshot)
	if err != nil || a.Snapshot() != snapshot || a.Status() != connections.Admitted {
		t.Fatalf("restore: %v", err)
	}
	same, err := a.RecordCheck(snapshot.Provider)
	if err != nil || same.Snapshot() != snapshot {
		t.Fatal("no-op changed max revision")
	}
	changed := snapshot.Provider
	changed.At, _ = calendar.ParseInstant("2026-09-07T12:00:00.123456790Z")
	if _, err = a.RecordCheck(changed); err == nil {
		t.Fatal("revision overflow")
	}
	if a.Snapshot() != snapshot {
		t.Fatal("restore or overflow mutated input")
	}
	for _, mutate := range []func(*connections.Snapshot){func(s *connections.Snapshot) { s.Revision = 0 }, func(s *connections.Snapshot) { s.Revision = 1 }, func(s *connections.Snapshot) { s.Revision++ }, func(s *connections.Snapshot) { s.Host.Kind = connections.ProviderCheck }, func(s *connections.Snapshot) { s.Provider.Binding.ContractVersion = "11" }, func(s *connections.Snapshot) { s.Provider.At = calendar.Instant{} }} {
		invalid := snapshot
		mutate(&invalid)
		if _, err = connections.Restore(invalid); err == nil {
			t.Fatal("invalid durable admission accepted")
		}
	}
}
