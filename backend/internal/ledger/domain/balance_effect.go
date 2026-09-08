package domain

import (
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type BalanceEffect struct {
	AccountID                      string
	At                             calendar.Instant
	Owned, Available, Locked, Debt reporting.Amount
}

func (r Revision) BalanceEffects() ([]BalanceEffect, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	result := []BalanceEffect{}
	if r.Accounting() == ExcludedFromAccounting || r.Type == Opening {
		return result, nil
	}
	for i, p := range r.Postings {
		state := r.ContributionState(i)
		if !r.Contributes(i) || !p.MovesMoney() || state != Posted && state != Pending {
			continue
		}
		zero, err := money.NewMoney("0", p.Money.Asset())
		if err != nil {
			return nil, err
		}
		z, _ := reporting.KnownAmount(zero)
		e := BalanceEffect{AccountID: p.AccountID, At: r.ContributionAt(i), Owned: z, Available: z, Locked: z, Debt: z}
		inverse, err := zero.Subtract(p.Money)
		if err != nil {
			return nil, err
		}
		value, _ := reporting.KnownAmount(p.Money)
		negative, _ := reporting.KnownAmount(inverse)
		unknown, _ := reporting.MissingAmount(reporting.Unknown, "funding_split_unknown")
		if state == Pending {
			if p.Money.Sign() >= 0 {
				continue
			}
			switch p.Funding {
			case UnknownFunds:
				e.Available, e.Locked = unknown, unknown
			case CreditFunds: // A hold on credit does not reserve owned family money.
			default:
				e.Available, e.Locked = value, negative
			}
		} else {
			switch p.Funding {
			case UnknownFunds:
				e.Owned, e.Available, e.Debt = unknown, unknown, unknown
			case CreditFunds:
				e.Debt = negative
			default:
				e.Owned, e.Available = value, value
			}
		}
		result = append(result, e)
	}
	return result, nil
}
