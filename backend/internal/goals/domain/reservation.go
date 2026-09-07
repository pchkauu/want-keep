package domain

import (
	"errors"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

var ErrInvalidReservation = errors.New("invalid reservation")
var ErrInsufficientFunds = errors.New("insufficient funds")

type Goal struct {
	ID        string
	Ownership household.Ownership
	Asset     money.Asset
	Revision  uint64
}
type Reservation struct {
	AccountID, Mode string
	Amount          money.Money
}
type Funding struct {
	AccountID string
	Asset     money.Asset
	Available reporting.Amount
	Other     []Reservation
}

func (g Goal) Validate() error {
	if g.ID == "" || g.Revision < 1 || g.Revision > 9007199254740991 {
		return ErrInvalidReservation
	}
	if err := g.Ownership.Validate(); err != nil {
		return err
	}
	_, err := money.ParseAsset(string(g.Asset))
	return err
}
func (g Goal) ValidateReservations(requested []Reservation, funding map[string]Funding) error {
	if err := g.Validate(); err != nil {
		return err
	}
	if len(requested) > 1000 {
		return ErrInvalidReservation
	}
	seen := map[string]bool{}
	for _, r := range requested {
		f, ok := funding[r.AccountID]
		if !ok || r.AccountID == "" || seen[r.AccountID] {
			return ErrInvalidReservation
		}
		seen[r.AccountID] = true
		if f.Asset != g.Asset {
			return money.ErrAssetMismatch
		}
		available, known := f.Available.Value()
		if !known {
			return ErrInsufficientFunds
		}
		if available.Asset() != g.Asset {
			return money.ErrAssetMismatch
		}
		switch r.Mode {
		case "dedicated":
			if available.Sign() < 0 {
				return ErrInsufficientFunds
			}
			if len(f.Other) > 0 {
				return ErrInsufficientFunds
			}
			if r.Amount.Asset() != "" {
				return ErrInvalidReservation
			}
		case "virtual":
			if err := r.Amount.Validate(); err != nil {
				return err
			}
			if r.Amount.Asset() != g.Asset {
				return money.ErrAssetMismatch
			}
			if r.Amount.Sign() < 0 {
				return ErrInvalidReservation
			}
			total := r.Amount
			for _, other := range f.Other {
				if other.Mode == "dedicated" {
					return ErrInsufficientFunds
				}
				var err error
				total, err = total.Add(other.Amount)
				if err != nil {
					return err
				}
			}
			cmp, err := total.Compare(available)
			if err != nil {
				return err
			}
			if cmp > 0 {
				return ErrInsufficientFunds
			}
		default:
			return ErrInvalidReservation
		}
	}
	return nil
}
