package domain_test

import (
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	"testing"
)

func TestFamilyAccessAndPersonalEditing(t *testing.T) {
	a, _ := (household.Membership{ID: "ma", UserID: "a", HouseholdID: "family", Active: true}).Principal()
	b, _ := (household.Membership{ID: "mb", UserID: "b", HouseholdID: "family", Active: true}).Principal()
	foreign, _ := (household.Membership{ID: "mc", UserID: "c", HouseholdID: "other", Active: true}).Principal()
	personal, _ := household.NewOwnership("family", household.Personal, "a")
	shared, _ := household.NewOwnership("family", household.Shared, "")
	if personal.RequireRead(b) != nil || personal.RequireOwnerEdit(a) != nil || shared.RequireOwnerEdit(b) != nil {
		t.Fatal("authorized action denied")
	}
	if personal.RequireOwnerEdit(b) == nil || personal.RequireRead(foreign) == nil || shared.RequireRead(household.Principal{}) == nil {
		t.Fatal("unauthorized action allowed")
	}
	if _, err := household.NewOwnership("family", household.Shared, "a"); err == nil {
		t.Fatal("shared owner accepted")
	}
	if _, err := household.NewOwnership("family", household.Personal, ""); err == nil {
		t.Fatal("missing personal owner accepted")
	}
	if _, err := (household.Membership{ID: "x", UserID: "a", HouseholdID: "family"}).Principal(); err == nil {
		t.Fatal("inactive membership accepted")
	}
}

func TestConfiguredMemberLimit(t *testing.T) {
	family := household.Household{ID: "family"}
	members := []household.Membership{{ID: "a", UserID: "a", HouseholdID: "family", Active: true}, {ID: "b", UserID: "b", HouseholdID: "family", Active: true}, {ID: "c", UserID: "c", HouseholdID: "family", Active: true}}
	if family.ValidateMemberLimit(members, 2) == nil {
		t.Fatal("cap ignored")
	}
	if family.ValidateMemberLimit(members, 3) != nil {
		t.Fatal("hardcoded two participants")
	}
	members[2].HouseholdID = "foreign"
	if family.ValidateMemberLimit(members, 3) == nil {
		t.Fatal("foreign member accepted")
	}
}
