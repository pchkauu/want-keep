package domain

import (
	"errors"
	"strings"
	"unicode/utf8"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

var ErrInvalidAccount = errors.New("invalid account")
var ErrNotFound = errors.New("account not found")

type Account struct {
	ID, Name, Product          string
	Ownership                  household.Ownership
	Asset                      money.Asset
	Revision                   uint64
	OpeningDate                calendar.Date
	ExternalAccountID          string
	ExternalOwnerID            household.UserID
	Network, ExternalAssetCode string
}

func (a Account) Validate() error {
	if a.ID == "" || !validAccountText(a.Name, 2000, true) || !Product(a.Product).Valid() || a.Revision < 1 || a.Revision > 9007199254740991 || a.OpeningDate.String() == "" || !validAccountText(a.Network, 2000, false) || !validAccountText(a.ExternalAssetCode, 2000, false) {
		return ErrInvalidAccount
	}
	if err := a.Ownership.Validate(); err != nil {
		return err
	}
	_, err := money.ParseAsset(string(a.Asset))
	return err
}

func validAccountText(value string, maximum int, required bool) bool {
	if value == "" {
		return !required
	}
	return !strings.ContainsRune(value, 0) && utf8.ValidString(value) && utf8.RuneCountInString(value) <= maximum && (!required || strings.TrimSpace(value) != "")
}

type Product string

func (p Product) Valid() bool {
	switch p {
	case "cash", "current", "debit_card", "credit_card", "savings", "deposit", "wallet", "funding", "spot", "earn", "coinhold", "crypto_card", "futures", "mining":
		return true
	}
	return false
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
