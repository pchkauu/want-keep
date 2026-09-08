package domain_test

import (
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"testing"
)

func TestProtectedClassificationAcrossThreeEvidenceRecords(t *testing.T) {
	f := fixture{t}
	a, b, c := f.fact("a", "account", "-500", money.RUB), f.fact("b", "account", "-500", money.RUB), f.fact("c", "account", "-500", money.RUB)
	a.Origin, b.Origin, c.Origin = "manual", "manual", "manual"
	b.CategoryID, c.CategoryID = "category-food", "category-sport"
	b = b.WithDecision(ledger.Decision{ID: "classify-b", Kind: "correction", ActorID: "member-a", Reason: "Confirmed food", At: b.OccurredAt}, []ledger.Field{ledger.CategoryField})
	c = c.WithDecision(ledger.Decision{ID: "classify-c", Kind: "correction", ActorID: "member-a", Reason: "Confirmed sport", At: c.OccurredAt}, []ledger.Field{ledger.CategoryField})
	b.Participation = ledger.Participation{GroupID: "wait-b", Kind: "payment", State: "waiting"}
	c.Participation = ledger.Participation{GroupID: "wait-c", Kind: "payment", State: "waiting"}
	for _, r := range []ledger.Revision{a, b, c} {
		if err := r.Validate(); err != nil {
			t.Fatal("invalid fixture", err)
		}
	}
	if _, _, err := f.group(matching.Payment, "b").Assign([]ledger.Revision{b, c}, false); err == nil {
		t.Fatal("control: distinct protected categories accepted")
	}

	for _, order := range [][]ledger.Revision{{a, b, c}, {a, c, b}, {b, a, c}, {b, c, a}, {c, a, b}, {c, b, a}} {
		if _, _, err := f.group(matching.Payment, "a").Assign(order, false); err == nil {
			t.Fatal("unprotected carrier hid conflicting protected categories")
		}
	}
	group, assigned, err := f.group(matching.Payment, "a").Assign([]ledger.Revision{a, b}, false)
	if err != nil {
		t.Fatal("one protected classification should be accepted", err)
	}
	if _, _, err = group.Assign(append(assigned, c), false); err == nil {
		t.Fatal("group expansion hid a protected conflict")
	}
	c.CategoryID = b.CategoryID
	if _, _, err = f.group(matching.Payment, "a").Assign([]ledger.Revision{a, b, c}, false); err != nil {
		t.Fatal("agreeing protected categories were rejected", err)
	}
}
