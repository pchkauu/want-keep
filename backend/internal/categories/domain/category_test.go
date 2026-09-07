package domain

import (
	"testing"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func TestStarterCatalogAndLocalizedRename(t *testing.T) {
	householdID := household.HouseholdID("24972db1-ed6c-4b19-a307-e99724d65860")
	items := StarterCategories(householdID)
	if len(items) != 31 {
		t.Fatalf("starter catalog has %d entries", len(items))
	}
	for _, item := range items {
		if err := item.Validate(); err != nil {
			t.Fatalf("%s: %v", item.Key, err)
		}
	}
	food := items[5]
	renamed, err := food.Apply(Change{NameAction: "set", Name: "  Семейная еда  "})
	if err != nil || renamed.DisplayName() != "Семейная еда" || renamed.NameEN != "Food" || renamed.Revision != 2 {
		t.Fatalf("rename lost starter identity: %+v %v", renamed, err)
	}
	restored, err := renamed.Apply(Change{NameAction: "restore_default"})
	if err != nil || restored.DisplayName() != "Еда" || restored.Revision != 3 {
		t.Fatalf("restore failed: %+v %v", restored, err)
	}
}

func TestNormalizePreservesPunctuationAndFoldsWhitespace(t *testing.T) {
	got, err := Normalize("  Dizengoff\t—  Cafe!  ")
	if err != nil || got != "dizengoff — cafe!" {
		t.Fatalf("normalize = %q, %v", got, err)
	}
}

func TestCategoryRevisionOverflowAndNoChange(t *testing.T) {
	item := Category{HouseholdID: "24972db1-ed6c-4b19-a307-e99724d65860", ID: "24972db1-ed6c-4b19-a307-e99724d65861", Revision: MaxRevision, CustomName: "Food", State: Active, Origin: Custom}
	if _, err := item.Apply(Change{NameAction: "set", Name: "Groceries"}); err != ErrVersionConflict {
		t.Fatalf("overflow accepted: %v", err)
	}
	item.Revision = 1
	if _, err := item.Apply(Change{NameAction: "set", Name: "Food"}); err != ErrNoChange {
		t.Fatalf("no-op accepted: %v", err)
	}
}
