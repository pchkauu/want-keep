package domain_test

import (
	"errors"
	"strings"
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	connection "github.com/pchkauu/want-keep/backend/internal/connections/domain"
)

func TestAdmissionRequiresBothChecksAndExactBinding(t *testing.T) {
	binding := connection.Binding{Provider: "raiffeisen", Environment: "production", AdapterBuildDigest: "sha256:" + strings.Repeat("a", 64), CollectorImageDigest: "sha256:" + strings.Repeat("b", 64), ContractVersion: "10", AllowlistRevision: "1", NonSecretConfigRevision: "1", OperatorPermissionRevision: "1"}
	first, _ := calendar.ParseInstant("2026-09-07T00:00:00Z")
	later, _ := calendar.ParseInstant("2026-09-07T01:00:00Z")
	pending, err := connection.NewAdmission(binding)
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(pending.RequireSync(binding), connection.ErrProviderNotAdmitted) {
		t.Fatal("default admission open")
	}
	provider, err := pending.RecordCheck(connection.Check{Kind: connection.ProviderCheck, Binding: binding, Result: connection.CheckPassed, At: first})
	if err != nil || provider.Status() != connection.Pending {
		t.Fatal("provider check admitted without host")
	}
	admitted, err := provider.RecordCheck(connection.Check{Kind: connection.HostCheck, Binding: binding, Result: connection.CheckPassed, At: first})
	if err != nil || admitted.RequireSync(binding) != nil || len(admitted.Reasons()) != 0 {
		t.Fatal("matching combined admission failed")
	}
	if pending.Status() != connection.Pending || provider.Status() != connection.Pending {
		t.Fatal("check mutated previous snapshot")
	}
	for _, mutate := range []func(*connection.Binding){
		func(b *connection.Binding) { b.Provider = "ozon" }, func(b *connection.Binding) { b.Environment = "test" },
		func(b *connection.Binding) { b.AdapterBuildDigest = "sha256:" + strings.Repeat("c", 64) }, func(b *connection.Binding) { b.CollectorImageDigest = "sha256:" + strings.Repeat("d", 64) },
		func(b *connection.Binding) { b.ContractVersion = "11" }, func(b *connection.Binding) { b.AllowlistRevision = "2" },
		func(b *connection.Binding) { b.NonSecretConfigRevision = "2" }, func(b *connection.Binding) { b.OperatorPermissionRevision = "2" },
	} {
		changed := binding
		mutate(&changed)
		if !errors.Is(admitted.RequireSync(changed), connection.ErrProviderNotAdmitted) {
			t.Fatal("changed binding accepted")
		}
		rebound, err := admitted.Rebind(changed)
		if err != nil || rebound.Status() != connection.Pending || rebound.CheckedAt().String() != "" {
			t.Fatal("rebind retained old evidence")
		}
		if _, err := rebound.RecordCheck(connection.Check{Kind: connection.ProviderCheck, Binding: binding, Result: connection.CheckPassed, At: later}); err == nil {
			t.Fatal("old evidence accepted after rebind")
		}
	}
	for _, kind := range []connection.CheckKind{connection.ProviderCheck, connection.HostCheck} {
		for _, result := range []connection.CheckResult{connection.CheckFailed, connection.CheckRevoked} {
			blocked, err := admitted.RecordCheck(connection.Check{Kind: kind, Binding: binding, Result: result, At: later})
			if err != nil || blocked.Status() != connection.Blocked || blocked.RequireSync(binding) == nil {
				t.Fatal("failure/revocation accepted")
			}
			if _, err := blocked.RecordCheck(connection.Check{Kind: kind, Binding: binding, Result: connection.CheckPassed, At: first}); err == nil {
				t.Fatal("stale check restored admission")
			}
		}
	}
	if (connection.Admission{}).RequireSync(binding) == nil {
		t.Fatal("missing admission accepted")
	}
	if _, err := connection.NewAdmission(connection.Binding{}); err == nil {
		t.Fatal("invalid binding")
	}
	if _, err := pending.RecordCheck(connection.Check{Kind: "user", Binding: binding, Result: connection.CheckPassed, At: first}); err == nil {
		t.Fatal("unsupported evidence")
	}
	if _, err := pending.RecordCheck(connection.Check{Kind: connection.HostCheck, Binding: binding, Result: connection.CheckPassed}); err == nil {
		t.Fatal("undated evidence")
	}
}
