//go:build integration

package familyallocation_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	allocationapp "github.com/pchkauu/want-keep/backend/internal/allocation/application"
	allocationdomain "github.com/pchkauu/want-keep/backend/internal/allocation/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	categoryapp "github.com/pchkauu/want-keep/backend/internal/categories/application"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
)

func TestConcurrentRuleChangesUseOneRevision(t *testing.T) {
	fixture := newFixture(t)
	merchantCommand, err := fixture.executor.Execute(testContext, fixture.p, commands.Request{ID: uuid.NewString(), Kind: "merchants.create", PayloadHash: strings.Repeat("a", 64)}, func(ctx context.Context) (command.Result, error) {
		return categoryapp.NewService(fixture.store, uuid.NewString).CreateMerchant(ctx, fixture.p, categoryapp.MerchantInput{Name: "Concurrent merchant"})
	})
	if err != nil || merchantCommand.Status() != command.Succeeded {
		t.Fatal(err, merchantCommand.ErrorCode())
	}
	merchant, _ := merchantCommand.Result()
	service := allocationapp.NewService(fixture.store, func() calendar.Instant { return fixture.now }, uuid.NewString)
	initial := allocationapp.RuleInput{Priority: 10, State: allocationdomain.Active, Condition: allocationdomain.Condition{MerchantID: merchant.ResourceID}, Shares: []allocationdomain.Share{{MemberID: fixture.members[0].ID, Value: "50"}, {MemberID: fixture.members[1].ID, Value: "50"}}}
	created, err := fixture.executor.Execute(testContext, fixture.p, commands.Request{ID: uuid.NewString(), Kind: "allocation_rules.create", PayloadHash: strings.Repeat("b", 64)}, func(ctx context.Context) (command.Result, error) {
		return service.CreateRule(ctx, fixture.p, initial)
	})
	if err != nil || created.Status() != command.Succeeded {
		t.Fatal(err, created.ErrorCode())
	}
	resource, _ := created.Result()

	start := make(chan struct{})
	results := make(chan command.Command, 2)
	errors := make(chan error, 2)
	var group sync.WaitGroup
	for index, principal := range []struct {
		first, second string
		principal     int
	}{{"60", "40", 0}, {"40", "60", 1}} {
		group.Add(1)
		go func(index int, first, second string, principalIndex int) {
			defer group.Done()
			<-start
			principal := fixture.p
			if principalIndex == 1 {
				principal = fixture.q
			}
			input := allocationapp.RuleInput{Priority: 10, State: allocationdomain.Active, Condition: allocationdomain.Condition{MerchantID: merchant.ResourceID}, Shares: []allocationdomain.Share{{MemberID: fixture.members[0].ID, Value: first}, {MemberID: fixture.members[1].ID, Value: second}}}
			value, executeErr := fixture.executor.Execute(testContext, principal, commands.Request{ID: uuid.NewString(), Kind: "allocation_rules.change", PayloadHash: strings.Repeat(string(rune('c'+index)), 64)}, func(ctx context.Context) (command.Result, error) {
				return service.ChangeRule(ctx, principal, resource.ResourceID, 1, input)
			})
			results <- value
			errors <- executeErr
		}(index, principal.first, principal.second, principal.principal)
	}
	close(start)
	group.Wait()
	close(results)
	close(errors)
	for executeErr := range errors {
		if executeErr != nil {
			t.Fatal(executeErr)
		}
	}
	succeeded, conflicted := 0, 0
	for result := range results {
		switch {
		case result.Status() == command.Succeeded:
			succeeded++
		case result.Status() == command.Failed && result.ErrorCode() == "version_conflict":
			conflicted++
		default:
			t.Fatalf("unexpected command: status=%s error=%s", result.Status(), result.ErrorCode())
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("race result: succeeded=%d conflicted=%d", succeeded, conflicted)
	}
	current, err := fixture.store.AllocationRule(testContext, fixture.p, resource.ResourceID)
	if err != nil || current.Revision != 2 {
		t.Fatalf("current rule: %+v %v", current, err)
	}
}
