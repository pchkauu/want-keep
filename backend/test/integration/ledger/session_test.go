//go:build integration

package ledger_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	access "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type sessionBarrier struct {
	*identity.Service
	reached, release chan struct{}
}

func (s *sessionBarrier) WithinSession(ctx context.Context, token access.Token, apply func(context.Context, identity.Access) error) error {
	close(s.reached)
	<-s.release
	return s.Service.WithinSession(ctx, token, apply)
}

func TestRevokedSessionCannotExecuteRegisteredFinancialCommand(t *testing.T) {
	f := newFixture(t)
	id := f.create(money.RUB, "5000")
	client := f.client(f.p)
	a, err := client.sessions.Me(testContext, client.token)
	if err != nil {
		t.Fatal(err)
	}
	barrier := &sessionBarrier{Service: client.sessions, reached: make(chan struct{}), release: make(chan struct{})}
	coordinator := commands.NewAuthenticated(f.executor, barrier)
	key := request()
	r := f.revision(uuid.NewString(), id, "-500", money.RUB, 1)
	finished := make(chan error, 1)
	go func() {
		_, err := coordinator.Execute(testContext, a, key, func(ctx context.Context) (command.Result, error) {
			return command.Result{ResourceType: "transaction", ResourceID: r.OperationID, Revision: 1}, f.writer.Append(ctx, f.p, r, 0)
		})
		finished <- err
	}()
	<-barrier.reached
	c, err := f.store.LoadCommand(testContext, f.p, key.ID)
	if err != nil || c.Status() != command.Pending {
		t.Fatal("registration was not durable", err)
	}
	err = f.store.WithinIdentity(testContext, func(ctx context.Context) error {
		s, err := f.store.IdentitySession(ctx, client.token.Hash())
		if err != nil {
			return err
		}
		s.Revoked = true
		return f.store.SaveIdentitySession(ctx, s)
	})
	if err != nil {
		t.Fatal(err)
	}
	close(barrier.release)
	if err = <-finished; !errors.Is(err, access.ErrUnauthorized) {
		t.Fatal("revocation raced past mutation", err)
	}
	f.balance(id, "owned", "5000")
	if f.count("operation_revisions") != 1 {
		t.Fatal("unauthorized write")
	}
}

func TestPostedReversalReleasesEffectWithoutRefundOrDeletedHistory(t *testing.T) {
	f := newFixture(t)
	id := f.create(money.RUB, "5000")
	r := f.revision(uuid.NewString(), id, "-500", money.RUB, 1)
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	f.balance(id, "owned", "4500")
	r.Revision = 2
	r.State = ledger.Reversed
	key := request()
	if _, err := f.write(r, key); err != nil {
		t.Fatal(err)
	}
	if _, err := f.write(r, key); err != nil {
		t.Fatal(err)
	}
	f.balance(id, "owned", "5000")
	old, err := f.store.LedgerRevision(testContext, f.p, r.OperationID, 1)
	if err != nil || old.State != ledger.Posted || old.Type != ledger.Expense {
		t.Fatal("reversal rewrote history", err)
	}
	current, _, err := f.store.CurrentLedgerRevision(testContext, f.p, r.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	parts, err := current.Components()
	if err != nil || len(parts) != 0 || current.Type != ledger.Expense {
		t.Fatal("reversal treated as refund", err)
	}
}
