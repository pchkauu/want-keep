//go:build integration

package household_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	goals "github.com/pchkauu/want-keep/backend/internal/goals/application"
	goal "github.com/pchkauu/want-keep/backend/internal/goals/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func TestFamilyReadsAndPersonalResourceMutations(t *testing.T) {
	f := newFixture(t)
	id := f.account("RUB", "1000")
	if _, err := f.store.Account(testContext, f.q, id); err != nil {
		t.Fatal("partner cannot read", err)
	}
	personal, _ := household.NewOwnership(f.family.ID, household.Personal, f.p.UserID())
	personalGoal := goal.Goal{ID: uuid.NewString(), Ownership: personal, Asset: "RUB", Revision: 1}
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.store.CreateGoal(ctx, personalGoal) }); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Goal(testContext, f.q, personalGoal.ID); err != nil {
		t.Fatal("partner read rejected", err)
	}
	service := goals.NewService(f.store)
	run := func(p household.Principal, key commands.Request) (command.Command, error) {
		return f.executor.Execute(testContext, p, key, func(ctx context.Context) (command.Result, error) {
			return service.Replace(ctx, p, personalGoal.ID, 1, []goal.Reservation{{AccountID: id, Mode: "virtual", Amount: cash("100", "RUB")}}, "Synthetic change")
		})
	}
	denied, err := run(f.q, request())
	if err != nil || denied.Snapshot().ErrorCode != "forbidden" {
		t.Fatal("partner changed personal goal", denied, err)
	}
	allowed, err := run(f.p, request())
	if err != nil || allowed.Snapshot().Status != "succeeded" {
		t.Fatal("owner denied", allowed, err)
	}
	wrong := personalGoal
	wrong.ID = uuid.NewString()
	if err = f.store.WithinHousehold(testContext, f.q, func(ctx context.Context) error { return f.store.CreateGoal(ctx, wrong) }); !errors.Is(err, household.ErrForbidden) {
		t.Fatal("personal goal created for another member", err)
	}
	date, _ := calendar.ParseDate("2026-09-01")
	if err = f.store.WithinHousehold(testContext, f.q, func(ctx context.Context) error {
		return f.store.CreateAccount(ctx, accounts.Account{ID: uuid.NewString(), Name: "Private", Ownership: personal, Asset: "RUB", Product: "cash", Revision: 1, OpeningDate: date})
	}); !errors.Is(err, household.ErrForbidden) {
		t.Fatal("personal account ownership spoof", err)
	}
}

func TestPartnerCorrectionsKeepActorAndDetectConcurrentRevision(t *testing.T) {
	f := newFixture(t)
	accountID := f.account("RUB", "1000")
	id := uuid.NewString()
	r := f.revision(id, accountID, "-100", "RUB", 1)
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan command.Command, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, p := range []household.Principal{f.p, f.q} {
		next := f.revision(id, accountID, "-150", "RUB", 2)
		next.ActorID = p.UserID()
		key := request()
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := f.executor.Execute(testContext, p, key, func(ctx context.Context) (command.Result, error) {
				err := f.writer.Append(ctx, p, next, 1)
				return command.Result{ResourceType: "transaction", ResourceID: id, Revision: 2}, err
			})
			results <- result
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	success, conflict := 0, 0
	for c := range results {
		if c.Snapshot().Status == "succeeded" {
			success++
		} else if c.Snapshot().ErrorCode == "version_conflict" {
			conflict++
		} else {
			t.Fatal(c)
		}
	}
	if success != 1 || conflict != 1 || f.available(accountID) != "850" {
		t.Fatal("concurrent correction", success, conflict, f.available(accountID))
	}
	current, _, err := f.store.CurrentLedgerRevision(testContext, f.q, id)
	if err != nil {
		t.Fatal(err)
	}
	// The other member can correct the winner; payer and original author remain independently recorded.
	actor := f.q
	if current.ActorID == f.q.UserID() {
		actor = f.p
	}
	next := f.revision(id, accountID, "-120", "RUB", 3)
	next.ActorID = actor.UserID()
	key := request()
	execute := func() (command.Command, error) {
		return f.executor.Execute(testContext, actor, key, func(ctx context.Context) (command.Result, error) {
			err := f.writer.Append(ctx, actor, next, 2)
			return command.Result{ResourceType: "transaction", ResourceID: id, Revision: 3}, err
		})
	}
	if _, err = execute(); err != nil {
		t.Fatal(err)
	}
	if _, err = execute(); err != nil {
		t.Fatal("replay", err)
	}
	stored, _, err := f.store.CurrentLedgerRevision(testContext, f.p, id)
	if err != nil || stored.ActorID != actor.UserID() || stored.PayerMemberID != f.members[0].ID || f.available(accountID) != "880" {
		t.Fatal("attribution or replay", stored, err)
	}
	spoof := next
	spoof.Revision = 4
	spoof.ActorID = f.p.UserID()
	if err = f.store.WithinHousehold(testContext, f.q, func(ctx context.Context) error { return f.writer.Append(ctx, f.q, spoof, 3) }); !errors.Is(err, household.ErrForbidden) {
		t.Fatal("actor spoof accepted", err)
	}
}

func TestHouseholdIsolationCurrentMembershipAndLockOrder(t *testing.T) {
	f := newFixture(t)
	id := f.account("USD", "100")
	family := household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Other synthetic family"}
	user := household.User{ID: household.UserID(uuid.NewString()), Name: "C"}
	m := household.Membership{ID: household.MembershipID(uuid.NewString()), HouseholdID: family.ID, UserID: user.ID, Active: true}
	zone, _ := calendar.ParseTimezone("UTC")
	if err := f.store.InitializeHousehold(testContext, family, []household.User{user}, []household.Membership{m}, zone, 2); err != nil {
		t.Fatal(err)
	}
	foreign, _ := m.Principal()
	if _, err := f.store.Account(testContext, foreign, id); !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("foreign account exposed", err)
	}
	if err := f.store.WithinInvitationHousehold(testContext, f.family.ID, func(context.Context) error { return nil }); !errors.Is(err, storage.ErrTransactionRequired) {
		t.Fatal("invitation without identity scope", err)
	}
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		return f.store.WithinIdentity(ctx, func(context.Context) error { return nil })
	}); !errors.Is(err, storage.ErrTransactionRequired) {
		t.Fatal("reversed lock order", err)
	}
	if _, err := f.admin.Exec(testContext, "UPDATE want_keep.memberships SET active=false WHERE household_id=$1 AND user_id=$2", f.family.ID, f.q.UserID()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Account(testContext, f.q, id); !errors.Is(err, household.ErrForbidden) {
		t.Fatal("stale membership reads", err)
	}
	if err := f.store.WithinHousehold(testContext, f.q, func(context.Context) error { return nil }); !errors.Is(err, household.ErrForbidden) {
		t.Fatal("stale membership writes", err)
	}
}

func TestHouseholdLimitAndApplicationRole(t *testing.T) {
	f := newFixture(t)
	u := household.User{ID: household.UserID(uuid.NewString()), Name: "Third"}
	m := household.Membership{ID: household.MembershipID(uuid.NewString()), HouseholdID: f.family.ID, UserID: u.ID, Active: true}
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.store.AddMember(ctx, u, m) }); err == nil {
		t.Fatal("member limit ignored")
	}
	if f.count("users") != 2 || f.count("memberships") != 2 {
		t.Fatal("partial membership")
	}
	if err := f.store.WithinIdentity(testContext, func(ctx context.Context) error {
		return f.store.WithinInvitationHousehold(ctx, f.family.ID, func(ctx context.Context) error { return f.store.JoinHousehold(ctx, u, m) })
	}); !errors.Is(err, household.ErrMemberLimit) {
		t.Fatal("join ignored configured cap", err)
	}
}
