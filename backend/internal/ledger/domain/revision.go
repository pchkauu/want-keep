package domain

import (
	"errors"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

var ErrInvalidRevision = errors.New("invalid ledger revision")

type Posting struct {
	AccountID string
	Money     money.Money
	Role      string
}
type Revision struct {
	OperationID         string
	Revision            uint64
	ActorID             household.UserID
	Reason, Type, State string
	OccurredAt          calendar.Instant
	CashDate            calendar.Date
	ExpenseMonth        calendar.Month
	HumanOverride       bool
	PayerState          string
	PayerMemberID       household.MembershipID
	Postings            []Posting
}

func (r Revision) Validate() error {
	if r.OperationID == "" || r.ActorID == "" || len(r.Reason) < 1 || len(r.Reason) > 2000 || r.Revision < 1 || r.Revision > 9007199254740991 || r.OccurredAt.String() == "" || r.CashDate.String() == "" {
		return ErrInvalidRevision
	}
	switch r.Type {
	case "income", "expense", "transfer", "exchange", "refund", "opening", "adjustment", "yield", "trade_result":
	default:
		return ErrInvalidRevision
	}
	switch r.State {
	case "draft", "pending", "posted", "reversed":
	default:
		return ErrInvalidRevision
	}
	if (r.PayerState == "known") != (r.PayerMemberID != "") {
		return ErrInvalidRevision
	}
	switch r.PayerState {
	case "known", "unknown", "not_applicable":
	default:
		return ErrInvalidRevision
	}
	if len(r.Postings) > 1000 || r.State == "posted" && len(r.Postings) == 0 {
		return ErrInvalidRevision
	}
	for _, p := range r.Postings {
		if p.AccountID == "" {
			return ErrInvalidRevision
		}
		if err := p.Money.Validate(); err != nil {
			return err
		}
		switch p.Role {
		case "principal", "fee", "interest", "funding", "pnl", "reward":
		default:
			return ErrInvalidRevision
		}
	}
	return nil
}

// Deltas compares complete revision snapshots. History remains immutable, while the
// current balance projection receives only the difference from the prior revision.
func (r Revision) Deltas(previous *Revision) (map[string]money.Money, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	result := map[string]money.Money{}
	for _, entry := range []struct {
		value    *Revision
		subtract bool
	}{{previous, true}, {&r, false}} {
		if entry.value == nil || entry.value.State != "posted" {
			continue
		}
		for _, p := range entry.value.Postings {
			amount, ok := result[p.AccountID]
			if !ok {
				amount, _ = money.NewMoney("0", p.Money.Asset())
			}
			var err error
			if entry.subtract {
				amount, err = amount.Subtract(p.Money)
			} else {
				amount, err = amount.Add(p.Money)
			}
			if err != nil {
				return nil, err
			}
			result[p.AccountID] = amount
		}
	}
	return result, nil
}
