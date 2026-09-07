package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func TestInvitationTransitionsAndRevisionBoundaries(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	i := household.Invitation{ID: "invite", HouseholdID: "family", InvitedBy: "a", TokenHash: strings.Repeat("a", 64), CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour), Status: household.InvitationActive}
	state := household.InvitationState{HouseholdID: "family", Revision: 1}
	issued, err := state.Change(1, i)
	if err != nil || issued.Revision != 2 || state.Current != nil {
		t.Fatal("issue mutated old state", err)
	}
	if _, err = issued.Change(1, i); !errors.Is(err, household.ErrInvitationRevision) {
		t.Fatal("stale revision accepted", err)
	}
	for _, invalid := range []uint64{0, household.MaxInvitationRevision + 1} {
		bad := issued
		bad.Revision = invalid
		if bad.Validate() == nil {
			t.Fatal("invalid revision", invalid)
		}
	}
	full := issued
	full.Revision = household.MaxInvitationRevision
	if _, err = full.Change(full.Revision, i); !errors.Is(err, household.ErrInvitationRevision) || full.Revision != household.MaxInvitationRevision {
		t.Fatal("overflow", err)
	}
	if i.RequireActive(now.Add(24*time.Hour-time.Nanosecond)) != nil || !errors.Is(i.RequireActive(now.Add(24*time.Hour)), household.ErrInvitationExpired) || i.RequireActive(now.Add(-time.Nanosecond)) == nil {
		t.Fatal("lifetime bounds")
	}
	revoked, err := i.Revoke()
	if err != nil || !errors.Is(revoked.RequireActive(now), household.ErrInvitationRevoked) || i.Status != household.InvitationActive {
		t.Fatal("revoke", err)
	}
	accepted, err := i.Accept("b", now)
	if err != nil || !errors.Is(accepted.RequireActive(now), household.ErrInvitationUsed) || accepted.AcceptedUserID != "b" || i.AcceptedUserID != "" {
		t.Fatal("accept", err)
	}
	if _, err = i.Accept("a", now); err == nil {
		t.Fatal("self invitation")
	}
	if _, err = revoked.Accept("b", now); err == nil {
		t.Fatal("revoked invitation")
	}
	bad := i
	bad.AcceptedUserID = "b"
	if bad.Validate() == nil {
		t.Fatal("accepted user in active invitation")
	}
	bad = i
	bad.TokenHash = "secret"
	if bad.Validate() == nil {
		t.Fatal("not a token digest")
	}
}

func TestOwnershipChangesCannotLaunderPersonalResources(t *testing.T) {
	a, _ := (household.Membership{ID: "ma", UserID: "a", HouseholdID: "family", Active: true}).Principal()
	b, _ := (household.Membership{ID: "mb", UserID: "b", HouseholdID: "family", Active: true}).Principal()
	aOwn, _ := household.NewOwnership("family", household.Personal, "a")
	bOwn, _ := household.NewOwnership("family", household.Personal, "b")
	shared, _ := household.NewOwnership("family", household.Shared, "")
	foreign, _ := household.NewOwnership("other", household.Shared, "")
	if aOwn.RequireCreate(b) == nil || aOwn.RequireChange(b, shared) == nil || aOwn.RequireChange(b, bOwn) == nil || aOwn.RequireChange(a, foreign) == nil {
		t.Fatal("ownership bypass")
	}
	if aOwn.RequireCreate(a) != nil || shared.RequireCreate(b) != nil || aOwn.RequireChange(a, shared) != nil || shared.RequireChange(b, bOwn) != nil {
		t.Fatal("authorized ownership change rejected")
	}
	// AI confirmation uses the same current-owner check, never the requested future scope.
	if aOwn.RequireOwnerEdit(b) == nil || aOwn.RequireRead(b) != nil {
		t.Fatal("read and proposal approval conflated")
	}
}
