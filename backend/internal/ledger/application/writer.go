package application

import (
	"context"

	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type Journal interface {
	Repository
	CurrentLedgerRevision(context.Context, household.Principal, string) (ledger.Revision, bool, error)
	EmitEvent(context.Context, string, string, uint64, string) error
}
type Writer struct {
	journal  Journal
	accounts accounts.Repository
}

func NewWriter(j Journal, a accounts.Repository) *Writer { return &Writer{j, a} }

// Append must be called inside the household transaction, including the command or import fence.
func (w *Writer) Append(ctx context.Context, p household.Principal, r ledger.Revision, expected uint64) error {
	if r.ActorID != p.UserID() {
		return household.ErrForbidden
	}
	if err := r.Validate(); err != nil {
		return err
	}
	current, exists, err := w.journal.CurrentLedgerRevision(ctx, p, r.OperationID)
	if err != nil {
		return err
	}
	actual := uint64(0)
	var previous *ledger.Revision
	if exists {
		actual = current.Revision
		previous = &current
	}
	if expected != actual || actual >= command.MaxRevision || r.Revision != actual+1 {
		return commands.Rejection{Code: "version_conflict"}
	}
	deltas, err := r.Deltas(previous)
	if err != nil {
		return err
	}
	if err = w.journal.AppendRevision(ctx, r, expected); err != nil {
		return err
	}
	for id, delta := range deltas {
		a, err := w.accounts.Account(ctx, p, id)
		if err != nil {
			return err
		}
		if a.Asset != delta.Asset() {
			return ledger.ErrInvalidRevision
		}
		for _, field := range []string{"owned", "available"} {
			balance, err := w.accounts.Balance(ctx, p, id, field)
			if err != nil {
				return err
			}
			value, known := balance.Amount.Value()
			if !known {
				continue
			}
			value, err = value.Add(delta)
			if err != nil {
				return err
			}
			balance.Amount, err = reporting.KnownAmount(value)
			if err != nil {
				return err
			}
			// A journal projection does not change the timestamp or quality of the source observation.
			if err = w.accounts.RecordBalance(ctx, balance); err != nil {
				return err
			}
		}
	}
	return w.journal.EmitEvent(ctx, "transaction", r.OperationID, r.Revision, "transaction.changed")
}
