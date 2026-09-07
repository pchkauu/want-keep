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
	if pending.Revision() != 1 {
		t.Fatal("initial admission revision")
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
	if provider.Revision() != 2 || admitted.Revision() != 3 || admitted.RequireResult(binding, 3) != nil {
		t.Fatal("evidence did not advance revision")
	}
	replayed, err := admitted.RecordCheck(connection.Check{Kind: connection.HostCheck, Binding: binding, Result: connection.CheckPassed, At: first})
	if err != nil || replayed != admitted {
		t.Fatal("exact evidence replay changed admission")
	}
	unchanged, err := admitted.Rebind(binding)
	if err != nil || unchanged != admitted {
		t.Fatal("same binding changed admission")
	}
	refreshed, err := admitted.RecordCheck(connection.Check{Kind: connection.HostCheck, Binding: binding, Result: connection.CheckPassed, At: later})
	if err != nil || refreshed.Status() != connection.Admitted || refreshed.Revision() != 4 || refreshed.RequireResult(binding, 3) == nil {
		t.Fatal("new evidence did not invalidate old result")
	}
	for _, revision := range []int64{-1, 0, 1, 2, 4, 9007199254740992} {
		if admitted.RequireResult(binding, revision) == nil {
			t.Fatal("wrong issued revision accepted", revision)
		}
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
		if admitted.RequireResult(changed, admitted.Revision()) == nil {
			t.Fatal("changed result binding accepted")
		}
		rebound, err := admitted.Rebind(changed)
		if err != nil || rebound.Status() != connection.Pending || rebound.CheckedAt().String() != "" {
			t.Fatal("rebind retained old evidence")
		}
		if rebound.Revision() != 4 || rebound.RequireResult(changed, rebound.Revision()) == nil {
			t.Fatal("rebind reset revision or permitted a pending result")
		}
		returned, err := rebound.Rebind(binding)
		if err != nil || returned.Revision() != 5 || returned.RequireResult(binding, admitted.Revision()) == nil {
			t.Fatal("A to B to A restored an old admission")
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
			if blocked.Revision() != 4 || blocked.RequireResult(binding, 3) == nil {
				t.Fatal("revocation did not fence the result")
			}
			next, _ := calendar.ParseInstant("2026-09-07T02:00:00Z")
			restored, err := blocked.RecordCheck(connection.Check{Kind: kind, Binding: binding, Result: connection.CheckPassed, At: next})
			if err != nil || restored.Revision() != 5 || restored.RequireResult(binding, 3) == nil || restored.RequireResult(binding, 5) != nil {
				t.Fatal("re-admission accepted the pre-revocation result")
			}
			if _, err := blocked.RecordCheck(connection.Check{Kind: kind, Binding: binding, Result: connection.CheckPassed, At: first}); err == nil {
				t.Fatal("stale check restored admission")
			}
		}
	}
	if (connection.Admission{}).RequireSync(binding) == nil {
		t.Fatal("missing admission accepted")
	}
	if (connection.Admission{}).RequireResult(binding, 0) == nil {
		t.Fatal("missing admission accepted a result")
	}
	if pending.Revision() != 1 || admitted.Revision() != 3 {
		t.Fatal("transitions mutated their source snapshot")
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
