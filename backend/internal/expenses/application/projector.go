package application

import (
	"context"
	"errors"

	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

type Projector struct{ repository Repository }

func NewProjector(repository Repository) *Projector { return &Projector{repository: repository} }

func (p *Projector) ProjectRefunds(ctx context.Context, principal household.Principal, current ledger.Revision, _ *ledger.Revision) error {
	links, err := p.repository.RefundsForOperation(ctx, principal, current.OperationID)
	if err != nil {
		return err
	}
	for _, link := range links {
		purchase, found, err := p.repository.CurrentLedgerRevision(ctx, principal, link.PurchaseID)
		if err != nil || !found {
			return errOr(err, ledger.ErrNotFound)
		}
		refund, found, err := p.repository.CurrentLedgerRevision(ctx, principal, link.OperationID)
		if err != nil || !found {
			return errOr(err, ledger.ErrNotFound)
		}
		refunded, items, err := p.repository.ActiveRefundTotals(ctx, principal, purchase.OperationID, refund.OperationID, principalAsset(purchase))
		if err != nil {
			return err
		}
		basis, err := p.repository.PurchaseValuation(ctx, principal, purchase.OperationID, purchase.Revision)
		if err != nil {
			return err
		}
		next, err := expenses.Calculate(purchase, refund, link.Items, refunded, items, basis, link.Revision+1, "refund_recalculation", current.ActorID, current.RecordedAt)
		if errors.Is(err, expenses.ErrClarificationRequired) {
			next, err = expenses.Clarify(purchase, refund, link.Items, refunded, link.Revision+1, "refund_recalculation", current.ActorID, current.RecordedAt)
		}
		if err != nil {
			return err
		}
		if link.SameCalculation(next) {
			continue
		}
		if err = p.repository.SaveRefund(ctx, principal, next, link.Revision); err != nil {
			return err
		}
		if err = p.repository.EmitEvent(ctx, "refund", next.OperationID, next.Revision, "refund.changed"); err != nil {
			return err
		}
	}
	return nil
}
