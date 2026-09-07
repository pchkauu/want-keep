package application

import (
	"context"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type View struct {
	Account  account.Account
	Opening  *account.Opening
	Ledger   account.Amounts
	Coverage reporting.Coverage
	Source   *account.Observation
	Cards    []account.CardAlias
	Funding  reporting.Amount
}

func (s *Service) Read(ctx context.Context, p household.Principal, id string) (View, error) {
	a, err := s.repository.Account(ctx, p, id)
	if err != nil {
		return View{}, err
	}
	v := View{Account: a, Ledger: account.UnknownAmounts("opening_unconfirmed")}
	v.Coverage, _ = reporting.NewCoverage(reporting.Partial, []string{"opening_unconfirmed"})
	o, found, err := s.repository.Opening(ctx, p, id)
	if err != nil {
		return v, err
	}
	if found {
		v.Opening = &o
		values := make([]reporting.Amount, 4)
		for i, f := range []string{"owned", "available", "locked", "debt"} {
			b, err := s.repository.Balance(ctx, p, id, f)
			if err != nil {
				return v, err
			}
			values[i] = b.Amount
			v.Coverage = b.Coverage
		}
		v.Ledger = account.Amounts{Owned: values[0], Available: values[1], Locked: values[2], Debt: values[3]}
	}
	source, found, err := s.repository.LatestObservation(ctx, p, id)
	if err != nil {
		return v, err
	}
	if found {
		v.Source = &source
	}
	v.Cards, err = s.repository.CardAliases(ctx, p, id)
	if err != nil {
		return v, err
	}
	v.Funding, err = s.repository.AccountFunding(ctx, p, id)
	return v, err
}
func (s *Service) List(ctx context.Context, p household.Principal, after string, limit int) ([]View, string, error) {
	accounts, next, err := s.repository.Accounts(ctx, p, after, limit)
	if err != nil {
		return nil, "", err
	}
	views := make([]View, 0, len(accounts))
	for _, a := range accounts {
		v, err := s.Read(ctx, p, a.ID)
		if err != nil {
			return nil, "", err
		}
		views = append(views, v)
	}
	return views, next, nil
}

func (v View) FundingAvailability() reporting.Amount {
	return v.Funding
}

// NativeTotals consumes all pages in the caller's consistent read snapshot; it is independent of a UI member filter.
func (s *Service) NativeTotals(ctx context.Context, p household.Principal) ([]account.AssetTotal, error) {
	var portfolio account.Portfolio
	after := ""
	for {
		views, next, err := s.List(ctx, p, after, 100)
		if err != nil {
			return nil, err
		}
		for _, v := range views {
			amounts := v.Ledger
			if v.Source != nil {
				amounts = v.Source.Amounts
			}
			amounts.Available = v.FundingAvailability()
			portfolio = append(portfolio, account.Position{AccountID: v.Account.ID, Asset: v.Account.Asset, Amounts: amounts})
		}
		if next == "" {
			break
		}
		after = next
	}
	return portfolio.Totals()
}
