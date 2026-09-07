package domain

import (
	"time"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type Amounts struct{ Owned, Available, Locked, Debt reporting.Amount }

func (v Amounts) Fields() []reporting.Amount {
	return []reporting.Amount{v.Owned, v.Available, v.Locked, v.Debt}
}
func (v Amounts) Validate(asset money.Asset) error {
	for _, a := range v.Fields() {
		if err := a.Validate(); err != nil {
			return err
		}
		if m, ok := a.Value(); ok && m.Asset() != asset {
			return money.ErrAssetMismatch
		}
	}
	return nil
}
func CashAmounts(m money.Money) (Amounts, error) {
	if err := m.Validate(); err != nil {
		return Amounts{}, err
	}
	if m.Sign() < 0 {
		return Amounts{}, ErrInvalidAccount
	}
	value, err := reporting.KnownAmount(m)
	if err != nil {
		return Amounts{}, err
	}
	zero, _ := money.NewMoney("0", m.Asset())
	z, _ := reporting.KnownAmount(zero)
	return Amounts{value, value, z, z}, nil
}
func UnknownAmounts(reason string) Amounts {
	a, _ := reporting.MissingAmount(reporting.Unknown, reason)
	return Amounts{a, a, a, a}
}

type Opening struct {
	AccountID, OperationID string
	Revision               uint64
	Date                   calendar.Date
	Timezone               calendar.Timezone
	Confirmed              bool
	Amounts                Amounts
	ActorID                household.UserID
	Reason                 string
	At                     calendar.Instant
}

func (o Opening) Validate(asset money.Asset) error {
	if o.AccountID == "" || o.Revision < 1 || o.Revision > 9007199254740991 || o.Date.String() == "" || o.ActorID == "" || len(o.Reason) < 1 || len(o.Reason) > 2000 || o.At.String() == "" {
		return ErrInvalidAccount
	}
	if _, err := calendar.ParseTimezone(o.Timezone.String()); err != nil {
		return err
	}
	start, err := o.Instant()
	if err != nil {
		return err
	}
	if start.Time().After(o.At.Time()) {
		return calendar.ErrInvalidTime
	}
	if err := o.Amounts.Validate(asset); err != nil {
		return err
	}
	if o.Confirmed {
		if _, ok := o.Amounts.Owned.Value(); !ok {
			return ErrInvalidAccount
		}
		if o.OperationID == "" {
			return ErrInvalidAccount
		}
	}
	for _, v := range []reporting.Amount{o.Amounts.Available, o.Amounts.Locked, o.Amounts.Debt} {
		if m, ok := v.Value(); ok && m.Sign() < 0 {
			return ErrInvalidAccount
		}
	}
	return nil
}
func (o Opening) Instant() (calendar.Instant, error) {
	zone, err := time.LoadLocation(o.Timezone.String())
	if err != nil {
		return calendar.Instant{}, calendar.ErrInvalidTime
	}
	at, err := time.ParseInLocation(time.DateOnly, o.Date.String(), zone)
	if err != nil || at.Format(time.DateOnly) != o.Date.String() {
		return calendar.Instant{}, calendar.ErrInvalidTime
	}
	return calendar.ParseInstant(at.UTC().Format(time.RFC3339Nano))
}

type Effect struct {
	OperationID string
	Revision    uint64
	At          calendar.Instant
	Amount      money.Money
}

// Projection only includes the current posted revisions. Source observations never enter this calculation.
func (o Opening) Project(asset money.Asset, effects []Effect) (Amounts, error) {
	if err := o.Validate(asset); err != nil {
		return Amounts{}, err
	}
	if !o.Confirmed {
		return UnknownAmounts("opening_unconfirmed"), nil
	}
	start, err := o.Instant()
	if err != nil {
		return Amounts{}, err
	}
	delta, _ := money.NewMoney("0", asset)
	for _, e := range effects {
		if e.At.String() == "" {
			return Amounts{}, ErrInvalidAccount
		}
		if !e.At.Time().Before(start.Time()) {
			delta, err = delta.Add(e.Amount)
			if err != nil {
				return Amounts{}, err
			}
		}
	}
	result := o.Amounts
	for _, field := range []*reporting.Amount{&result.Owned, &result.Available} {
		if m, known := field.Value(); known {
			m, err = m.Add(delta)
			if err != nil {
				return Amounts{}, err
			}
			*field, err = reporting.KnownAmount(m)
			if err != nil {
				return Amounts{}, err
			}
		}
	}
	return result, nil
}
