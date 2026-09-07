package application

import (
	"context"

	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type Journal interface {
	AccountTimezone(context.Context, household.Principal) (calendar.Timezone, error)
	Repository
	CurrentLedgerRevision(context.Context, household.Principal, string) (ledger.Revision, bool, error)
	RequireLedgerPayer(context.Context, household.Principal, household.MembershipID) error
	EmitEvent(context.Context, string, string, uint64, string) error
}
type ReconciliationTrigger interface {
	ReconcileAccount(context.Context, household.Principal, string) error
}
type Writer struct {
	journal    Journal
	accounts   accounts.Repository
	reconciler ReconciliationTrigger
}

func NewWriter(j Journal, a accounts.Repository) *Writer { return &Writer{journal: j, accounts: a} }
func NewWriterWithReconciliation(j Journal, a accounts.Repository, reconciler ReconciliationTrigger) *Writer {
	return &Writer{journal: j, accounts: a, reconciler: reconciler}
}

// Append must be called inside the household transaction, including the command or import fence.
func (w *Writer) Append(ctx context.Context, p household.Principal, r ledger.Revision, expected uint64) error {
	if r.ActorID != p.UserID() {
		return household.ErrForbidden
	}
	zone, err := w.journal.AccountTimezone(ctx, p)
	if err != nil {
		return err
	}
	if r.Timezone.String() != "" && r.Timezone != zone {
		return ledger.ErrInvalidRevision
	}
	r, err = r.InTimezone(zone)
	if err != nil {
		return err
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
	if err = r.CheckSuccessor(previous); err != nil {
		return err
	}
	if r.PayerState == "known" {
		if err = w.journal.RequireLedgerPayer(ctx, p, r.PayerMemberID); err != nil {
			return err
		}
	}
	r.Postings = append([]ledger.Posting(nil), r.Postings...)
	affected := r.AffectedAccounts(previous)
	for _, id := range affected {
		a, e := w.accounts.Account(ctx, p, id)
		if e != nil {
			return e
		}
		for i, posting := range r.Postings {
			if posting.AccountID != id {
				continue
			}
			if posting.Money.Asset() != a.Asset {
				return money.ErrAssetMismatch
			}
			if posting.Funding == ledger.CreditFunds && a.Product != "credit_card" {
				return ledger.ErrInvalidRevision
			}
			if posting.Funding == "" {
				r.Postings[i].Funding = ledger.OwnFunds
				if a.Product == "credit_card" {
					r.Postings[i].Funding = ledger.UnknownFunds
				}
			}
		}
	}
	if err = w.journal.AppendRevision(ctx, r, expected); err != nil {
		return err
	}
	if err = accounts.NewProjector(w.accounts).Apply(ctx, p, r, previous); err != nil {
		return err
	}
	if w.reconciler != nil && r.Type != ledger.Opening {
		for _, id := range affected {
			if err = w.reconciler.ReconcileAccount(ctx, p, id); err != nil {
				return err
			}
		}
	}
	return nil
}
