package domain

import (
	"errors"
	"strings"
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
)

func TestAdmissionRevisionOverflowLeavesSnapshotUnchanged(t *testing.T) {
	binding := Binding{Provider: "bybit", Environment: "production", AdapterBuildDigest: "sha256:" + strings.Repeat("a", 64), CollectorImageDigest: "sha256:" + strings.Repeat("b", 64), ContractVersion: "10", AllowlistRevision: "1", NonSecretConfigRevision: "1", OperatorPermissionRevision: "1"}
	admission, err := NewAdmission(binding)
	if err != nil {
		t.Fatal(err)
	}
	// Reach the persisted counter boundary without quadrillions of transitions.
	admission.revision = 9007199254740990
	at, _ := calendar.ParseInstant("2026-09-07T00:00:00Z")
	check := Check{Kind: ProviderCheck, Binding: binding, Result: CheckPassed, At: at}
	last, err := admission.RecordCheck(check)
	if err != nil || last.Revision() != 9007199254740991 {
		t.Fatal("last exact revision is unreachable")
	}
	before := last
	hostCheck := Check{Kind: HostCheck, Binding: binding, Result: CheckPassed, At: at}
	if _, err := last.RecordCheck(hostCheck); !errors.Is(err, ErrInvalidAdmission) {
		t.Fatal("evidence overflow accepted")
	}
	changed := binding
	changed.AllowlistRevision = "2"
	if _, err := last.Rebind(changed); !errors.Is(err, ErrInvalidAdmission) {
		t.Fatal("binding overflow accepted")
	}
	if last != before || admission.Revision() != 9007199254740990 {
		t.Fatal("overflow mutated original snapshots")
	}
	if replay, err := last.RecordCheck(check); err != nil || replay != last {
		t.Fatal("evidence no-op failed at maximum revision")
	}
	if replay, err := last.Rebind(binding); err != nil || replay != last {
		t.Fatal("binding no-op failed at maximum revision")
	}
}
