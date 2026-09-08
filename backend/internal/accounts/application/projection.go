package application

import (
	"context"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type Projector struct{ repository Repository }

func NewProjector(r Repository) *Projector { return &Projector{r} }
func (s *Projector) Rebuild(ctx context.Context, p household.Principal, id string) error {
	a, err := s.repository.Account(ctx, p, id)
	if err != nil {
		return err
	}
	o, exists, err := s.repository.Opening(ctx, p, id)
	if err != nil {
		return err
	}
	if !exists {
		return account.ErrInvalidAccount
	}
	effects, err := s.repository.AccountEffects(ctx, p, id)
	if err != nil {
		return err
	}
	values, err := o.Project(a.Asset, effects)
	if err != nil {
		return err
	}
	coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
	if a.Product != "cash" {
		coverage, _ = reporting.NewCoverage(reporting.Partial, []string{"history_not_reconciled"})
	}
	if !o.Confirmed {
		coverage, _ = reporting.NewCoverage(reporting.Partial, []string{"opening_unconfirmed"})
	}
	for _, v := range values.Fields() {
		if _, ok := v.Value(); !ok {
			coverage, _ = reporting.NewCoverage(reporting.Partial, []string{"balance_components_incomplete"})
			break
		}
	}
	unresolved, err := s.repository.AccountUnresolvedMatching(ctx, p, id)
	if err != nil {
		return err
	}
	if unresolved {
		coverage, _ = reporting.NewCoverage(reporting.Partial, append(coverage.Reasons(), "matching_unresolved"))
	}
	for i, field := range []string{"owned", "available", "locked", "debt"} {
		if err = s.repository.RecordBalance(ctx, account.Balance{AccountID: id, Field: field, Amount: values.Fields()[i], Coverage: coverage, Freshness: reporting.UnknownFreshness, ObservedAt: o.At}); err != nil {
			return err
		}
	}
	return nil
}
func (s *Projector) Apply(ctx context.Context, p household.Principal, r ledger.Revision, previous *ledger.Revision) error {
	for _, id := range r.AffectedAccounts(previous) {
		_, exists, err := s.repository.Opening(ctx, p, id)
		if err != nil {
			return err
		}
		if exists {
			if err = s.Rebuild(ctx, p, id); err != nil {
				return err
			}
			continue
		}
		if err = s.applyLegacy(ctx, p, id, r, previous); err != nil {
			return err
		}
	}
	return nil
}

// Historical projections without an opening cannot be reconstructed. Preserve their
// original basis and update only proven components; an unknown component stays unknown.
func (s *Projector) applyLegacy(ctx context.Context, p household.Principal, id string, r ledger.Revision, previous *ledger.Revision) error {
	for _, entry := range []struct {
		revision *ledger.Revision
		subtract bool
	}{{previous, true}, {&r, false}} {
		if entry.revision == nil {
			continue
		}
		effects, err := entry.revision.BalanceEffects()
		if err != nil {
			return err
		}
		for _, e := range effects {
			if e.AccountID != id {
				continue
			}
			values := []reporting.Amount{e.Owned, e.Available, e.Locked, e.Debt}
			for i, field := range []string{"owned", "available", "locked", "debt"} {
				b, err := s.repository.Balance(ctx, p, id, field)
				if err != nil {
					return err
				}
				delta, known := values[i].Value()
				if !known {
					b.Amount = values[i]
				} else if value, ok := b.Amount.Value(); ok {
					if entry.subtract {
						value, err = value.Subtract(delta)
					} else {
						value, err = value.Add(delta)
					}
					if err != nil {
						return err
					}
					b.Amount, err = reporting.KnownAmount(value)
					if err != nil {
						return err
					}
				}
				if _, known := b.Amount.Value(); !known {
					b.Coverage, _ = reporting.NewCoverage(reporting.Partial, []string{"balance_components_incomplete"})
				}
				if err = s.repository.RecordBalance(ctx, b); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
