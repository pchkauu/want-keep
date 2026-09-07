package storage

import (
	"context"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func (s *Store) AccountFunding(ctx context.Context, p household.Principal, id string) (reporting.Amount, error) {
	a, err := s.Account(ctx, p, id)
	if err != nil {
		return reporting.Amount{}, err
	}
	unresolved, err := s.AccountUnresolvedMatching(ctx, p, id)
	if err != nil {
		return reporting.Amount{}, err
	}
	if unresolved {
		return account.UnknownAmounts("matching_unresolved").Available, nil
	}
	if a.Product == "cash" {
		b, err := s.Balance(ctx, p, id, "available")
		return b.Amount, err
	}
	o, found, err := s.LatestObservation(ctx, p, id)
	if err != nil {
		return reporting.Amount{}, err
	}
	if !found {
		return account.UnknownAmounts("source_unavailable").Available, nil
	}
	effects, err := s.AccountEffects(ctx, p, id)
	if err != nil {
		return reporting.Amount{}, err
	}
	return o.Funding(effects), nil
}
