//go:build integration

package audit_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestCompoundDecisionRollbackUndoAndReplayAfterRestart(t *testing.T) {
	f := newFixture(t)
	a, b := f.create(money.RUB, "5000"), f.create(money.RUB, "5000")
	first, second := f.revision(uuid.NewString(), a, "-500", money.RUB, 1), f.revision(uuid.NewString(), b, "-500", money.RUB, 1)
	if _, err := f.write(first, request()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.write(second, request()); err != nil {
		t.Fatal(err)
	}
	fields := func(r ledger.Revision, amount string) journal.Change {
		p := append([]ledger.Posting(nil), r.Postings...)
		p[0].Money = cash(amount, money.RUB)
		return journal.Change{OperationID: r.OperationID, Expected: r.Revision, Correction: ledger.Correction{Principal: &p}}
	}
	changes := []journal.Change{fields(first, "-700"), fields(second, "-700")}
	key := request()
	boom := errors.New("synthetic before commit")
	_, err := f.executor.Execute(testContext, f.p, key, func(ctx context.Context) (command.Result, error) {
		r, e := f.ledgerService().ApplyChanges(ctx, f.p, changes, "Compound correction")
		if e != nil {
			return r, e
		}
		return r, boom
	})
	if !errors.Is(err, boom) || f.count("ledger_decisions") != 0 {
		t.Fatal("partial decision", err)
	}
	f.balance(a, "owned", "4500")
	f.balance(b, "owned", "4500")
	f.restart()
	execute := func() (command.Command, error) {
		return f.executor.Execute(testContext, f.p, key, func(ctx context.Context) (command.Result, error) {
			return f.ledgerService().ApplyChanges(ctx, f.p, changes, "Compound correction")
		})
	}
	c, err := execute()
	if err != nil || c.Status() != command.Succeeded {
		t.Fatal(c.ErrorCode(), err)
	}
	f.balance(a, "owned", "4300")
	f.balance(b, "owned", "4300")
	// Simulate a lost acknowledgement: recreate the service and replay the same registered command.
	f.restart()
	if _, err = execute(); err != nil {
		t.Fatal(err)
	}
	if f.count("ledger_decisions") != 1 {
		t.Fatal("replay duplicated decision")
	}
	r, _, err := f.store.CurrentLedgerRevision(testContext, f.p, first.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.executor.Execute(testContext, f.q, request(), func(ctx context.Context) (command.Result, error) {
		return f.ledgerService().Undo(ctx, f.q, r.DecisionID, []journal.ExpectedRevision{{first.OperationID, 2}, {second.OperationID, 2}}, "Undo both")
	})
	if err != nil {
		t.Fatal(err)
	}
	f.balance(a, "owned", "4500")
	f.balance(b, "owned", "4500")
}

func TestConcurrentCorrectionsKeepOneFinancialEffect(t *testing.T) {
	f := newFixture(t)
	id := f.create(money.RUB, "5000")
	r := f.revision(uuid.NewString(), id, "-500", money.RUB, 1)
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	ready := make(chan struct{}, 2)
	results := make(chan command.Command, 2)
	failures := make(chan error, 2)
	var wg sync.WaitGroup
	for _, amount := range []string{"-700", "-900"} {
		wg.Add(1)
		go func(amount string) {
			defer wg.Done()
			posts := append([]ledger.Posting(nil), r.Postings...)
			posts[0].Money = cash(amount, money.RUB)
			ready <- struct{}{}
			<-start
			c, err := f.executor.Execute(testContext, f.p, request(), func(ctx context.Context) (command.Result, error) {
				return f.ledgerService().Correct(ctx, f.p, journal.Change{OperationID: r.OperationID, Expected: 1, Correction: ledger.Correction{Principal: &posts}}, "Concurrent change")
			})
			results <- c
			failures <- err
		}(amount)
	}
	<-ready
	<-ready
	close(start)
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	succeeded, failed := 0, 0
	for c := range results {
		if c.Status() == command.Succeeded {
			succeeded++
		} else if c.Status() == command.Failed && c.ErrorCode() == "version_conflict" {
			failed++
		}
	}
	if succeeded != 1 || failed != 1 || f.count("ledger_decisions") != 1 {
		t.Fatal("concurrent effect", succeeded, failed)
	}
}
