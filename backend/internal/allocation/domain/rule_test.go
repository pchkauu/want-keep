package domain_test

import (
	"testing"

	allocation "github.com/pchkauu/want-keep/backend/internal/allocation/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
)

func recordedAt(t *testing.T) calendar.Instant {
	t.Helper()
	value, err := calendar.ParseInstant("2026-09-08T12:00:00.123456789Z")
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestRulesUsePriorityAndRequireClarificationForTiedResults(t *testing.T) {
	base := allocation.Rule{ID: "merchant", HouseholdID: "family", Revision: 1, Priority: 10, State: allocation.Active, Condition: allocation.Condition{MerchantID: "market"}, Shares: []allocation.Share{{MemberID: "a", Value: "60"}, {MemberID: "b", Value: "40"}}, ActorID: "actor", RecordedAt: recordedAt(t)}
	same := base
	same.ID = "category"
	same.Condition = allocation.Condition{CategoryID: "food"}
	same.Shares = []allocation.Share{{MemberID: "b", Value: "40.0"}, {MemberID: "a", Value: "60.00"}}
	lower := base
	lower.ID, lower.Priority = "fallback", 20
	resolution := allocation.Resolve([]allocation.Rule{lower, same, base}, "market", "food")
	if resolution.State != "resolved" || len(resolution.Rules) != 2 || resolution.Rules[0].ID != "category" || resolution.Rules[1].ID != "merchant" {
		t.Fatalf("resolution = %+v", resolution)
	}
	conflicting := same
	conflicting.Shares = []allocation.Share{{MemberID: "a", Value: "50"}, {MemberID: "b", Value: "50"}}
	resolution = allocation.Resolve([]allocation.Rule{base, conflicting}, "market", "food")
	if resolution.State != "unresolved" || resolution.Reason != "rule_conflict" {
		t.Fatalf("conflict = %+v", resolution)
	}
}

func TestRuleChangeIsVersionedAndNoopDoesNotAdvance(t *testing.T) {
	rule := allocation.Rule{ID: "rule", HouseholdID: "family", Revision: 1, Priority: 1, State: allocation.Active, Condition: allocation.Condition{CategoryID: "food"}, Shares: []allocation.Share{{MemberID: "a", Value: "100"}}, ActorID: "a", RecordedAt: recordedAt(t)}
	change := allocation.Change{Priority: rule.Priority, State: rule.State, Condition: rule.Condition, Shares: rule.Shares}
	if _, err := rule.Apply(change, "b", recordedAt(t)); err != allocation.ErrRuleNoChange {
		t.Fatalf("noop err = %v", err)
	}
	change.Priority = 2
	next, err := rule.Apply(change, "b", recordedAt(t))
	if err != nil || next.Revision != 2 || next.ActorID != "b" || rule.Revision != 1 {
		t.Fatalf("next=%+v err=%v original=%+v", next, err, rule)
	}
}
