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
	for i, field := range []string{"owned", "available", "locked", "debt"} {
		if err = s.repository.RecordBalance(ctx, account.Balance{AccountID: id, Field: field, Amount: values.Fields()[i], Coverage: coverage, Freshness: reporting.UnknownFreshness, ObservedAt: o.At}); err != nil {
			return err
		}
	}
	return nil
}
func (s *Projector) Apply(ctx context.Context, p household.Principal, r ledger.Revision, previous *ledger.Revision) error {
	deltas, err := r.Deltas(previous)
	if err != nil {
		return err
	}
	for id, delta := range deltas {
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
		// Legacy projections have no proven opening or bank provenance. Preserve their arithmetic without inventing either.
		for _, field := range []string{"owned", "available"} {
			b, err := s.repository.Balance(ctx, p, id, field)
			if err != nil {
				return err
			}
			if m, known := b.Amount.Value(); known {
				m, err = m.Add(delta)
				if err != nil {
					return err
				}
				b.Amount, err = reporting.KnownAmount(m)
				if err != nil {
					return err
				}
				if err = s.repository.RecordBalance(ctx, b); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
