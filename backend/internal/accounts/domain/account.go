package domain

import (
	"errors"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

var ErrInvalidAccount = errors.New("invalid account")

type Account struct {
	ID, Name, Product string
	Ownership         household.Ownership
	Asset             money.Asset
	Revision          uint64
	OpeningDate       calendar.Date
	ExternalAccountID string
}

func (a Account) Validate() error {
	if a.ID == "" || a.Name == "" || a.Product == "" || a.Revision < 1 || a.Revision > 9007199254740991 || a.OpeningDate.String() == "" {
		return ErrInvalidAccount
	}
	if err := a.Ownership.Validate(); err != nil {
		return err
	}
	_, err := money.ParseAsset(string(a.Asset))
	return err
}

type Balance struct {
	AccountID, Field string
	Amount           reporting.Amount
	Coverage         reporting.Coverage
	Freshness        reporting.Freshness
	ObservedAt       calendar.Instant
}

func (b Balance) Validate() error {
	if b.AccountID == "" || b.ObservedAt.String() == "" {
		return ErrInvalidAccount
	}
	switch b.Field {
	case "owned", "available", "locked", "debt":
	default:
		return ErrInvalidAccount
	}
	if err := b.Amount.Validate(); err != nil {
		return err
	}
	if _, err := reporting.NewCoverage(b.Coverage.State(), b.Coverage.Reasons()); err != nil {
		return err
	}
	_, err := reporting.ParseFreshness(string(b.Freshness))
	return err
}
