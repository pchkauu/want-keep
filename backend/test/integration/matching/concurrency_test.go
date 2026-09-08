//go:build integration

package matching_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	application "github.com/pchkauu/want-keep/backend/internal/matching/application"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestConcurrentHouseholdResponsesCommitOneOutcome(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "5000")
	a, b := uuid.NewString(), uuid.NewString()
	for _, id := range []string{a, b} {
		if _, err := f.write(f.revision(id, account, "-300", money.RUB, 1), request()); err != nil {
			t.Fatal(err)
		}
	}
	g, _, err := f.store.MatchingForOperation(testContext, f.p, b)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan command.Command, 2)
	failures := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for _, p := range []bool{false, true} {
		go func(partner bool) {
			actor := f.p
			if partner {
				actor = f.q
			}
			ready.Done()
			<-start
			c, err := f.executor.Execute(testContext, actor, request(), func(ctx context.Context) (command.Result, error) {
				return f.matching().Separate(ctx, actor, g.ID, g.Revision, []matching.Member{{OperationID: b, Revision: 1}}, "Separate purchase")
			})
			failures <- err
			results <- c
		}(p)
	}
	ready.Wait()
	close(start)
	succeeded := 0
	failed := 0
	for range 2 {
		if err := <-failures; err != nil {
			t.Fatal(err)
		}
		r := <-results
		if r.Status() == command.Succeeded {
			succeeded++
		} else if r.Status() == command.Failed {
			failed++
		}
	}
	if succeeded != 1 || failed != 1 {
		t.Fatal(succeeded, failed)
	}
	f.balance(account, "owned", "4400")
}
func TestMatchingRollbackRestartAndLostResponseReplay(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "5000")
	a, b := uuid.NewString(), uuid.NewString()
	for _, id := range []string{a, b} {
		if _, err := f.write(f.revision(id, account, "-300", money.RUB, 1), request()); err != nil {
			t.Fatal(err)
		}
	}
	before := f.count("operation_revisions")
	key := request()
	in := application.LinkInput{Kind: matching.Payment, PrimaryID: a, Members: []matching.Member{{OperationID: a, Revision: 1}, {OperationID: b, Revision: 1}}, Reason: "Confirmed evidence"}
	failure := errors.New("synthetic failure before commit")
	_, err := f.executor.Execute(testContext, f.p, key, func(ctx context.Context) (command.Result, error) {
		_, err := f.matching().Link(ctx, f.p, in)
		if err != nil {
			return command.Result{}, err
		}
		return command.Result{}, failure
	})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if f.count("operation_revisions") != before {
		t.Fatal("partial revision survived rollback")
	}
	f.restart()
	calls := 0
	for range 2 {
		_, err = f.executor.Execute(testContext, f.p, key, func(ctx context.Context) (command.Result, error) { calls++; return f.matching().Link(ctx, f.p, in) })
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatal("lost acknowledgement caused replay", calls)
	}
	f.balance(account, "owned", "4700")
	key.PayloadHash = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err = f.executor.Execute(testContext, f.p, key, func(context.Context) (command.Result, error) {
		t.Fatal("changed payload executed")
		return command.Result{}, nil
	}); !errors.Is(err, command.ErrDuplicateCommand) {
		t.Fatal(err)
	}
}
