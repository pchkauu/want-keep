package domain_test

import (
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	"testing"
)

func TestConnectionManagementAndBankAuthenticationAreSeparate(t *testing.T) {
	a, _ := (household.Membership{ID: "ma", UserID: "a", HouseholdID: "family", Active: true}).Principal()
	b, _ := (household.Membership{ID: "mb", UserID: "b", HouseholdID: "family", Active: true}).Principal()
	foreign, _ := (household.Membership{ID: "mc", UserID: "c", HouseholdID: "foreign", Active: true}).Principal()
	owner := connections.ExternalOwnership{HouseholdID: "family", OwnerID: "a"}
	if owner.RequireManage(b) != nil || owner.RequireAuthentication(a) != nil {
		t.Fatal("authorized action rejected")
	}
	if owner.RequireAuthentication(b) == nil || owner.RequireManage(foreign) == nil || owner.RequireManage(household.Principal{}) == nil {
		t.Fatal("external owner boundary bypassed")
	}
}
