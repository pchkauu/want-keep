package domain

import (
	"sort"

	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type Position struct {
	AccountID string
	Asset     money.Asset
	Amounts   Amounts
}
type AggregateAmount struct {
	Total             reporting.Amount
	KnownSubtotal     money.Money
	MissingAccountIDs []string
}
type AssetTotal struct {
	Asset                  money.Asset
	Owned, Available, Debt AggregateAmount
}
type Portfolio []Position

func (p Portfolio) Totals() ([]AssetTotal, error) {
	byAsset := map[money.Asset][]Position{}
	seen := map[string]bool{}
	for _, v := range p {
		if v.AccountID == "" || seen[v.AccountID] {
			return nil, ErrInvalidAccount
		}
		seen[v.AccountID] = true
		if err := v.Amounts.Validate(v.Asset); err != nil {
			return nil, err
		}
		byAsset[v.Asset] = append(byAsset[v.Asset], v)
	}
	result := []AssetTotal{}
	for asset, positions := range byAsset {
		total := AssetTotal{Asset: asset}
		for i, target := range []*AggregateAmount{&total.Owned, &total.Available, &total.Debt} {
			sum, _ := money.NewMoney("0", asset)
			for _, p := range positions {
				values := []reporting.Amount{p.Amounts.Owned, p.Amounts.Available, p.Amounts.Debt}
				v := values[i]
				if m, ok := v.Value(); ok {
					var err error
					sum, err = sum.Add(m)
					if err != nil {
						return nil, err
					}
				} else {
					target.MissingAccountIDs = append(target.MissingAccountIDs, p.AccountID)
				}
			}
			sort.Strings(target.MissingAccountIDs)
			target.KnownSubtotal = sum
			if len(target.MissingAccountIDs) == 0 {
				target.Total, _ = reporting.KnownAmount(sum)
			} else {
				target.Total, _ = reporting.MissingAmount(reporting.Unknown, "incomplete_accounts")
			}
		}
		result = append(result, total)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Asset < result[j].Asset })
	return result, nil
}
