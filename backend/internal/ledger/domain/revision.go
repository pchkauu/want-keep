package domain

import (
	"errors"
	"unicode/utf8"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

var ErrInvalidRevision = errors.New("invalid ledger revision")

type Posting struct {
	AccountID string
	Money     money.Money
	Role      Role
	Funding   FundingKind
	Treatment Treatment
}
type Revision struct {
	OperationID                                    string
	Revision                                       uint64
	ActorID                                        household.UserID
	Reason                                         string
	Type                                           Type
	State                                          State
	PostedAt                                       calendar.Instant
	Timezone                                       calendar.Timezone
	Origin                                         string
	FeeKnowledge                                   FeeKnowledge
	PnLBasis                                       PnLBasis
	Merchant, Note, AttachmentID, AllocationReason string
	OccurredAt                                     calendar.Instant
	CashDate                                       calendar.Date
	ExpenseMonth                                   calendar.Month
	HumanOverride                                  bool
	PayerState                                     string
	PayerMemberID                                  household.MembershipID
	Postings                                       []Posting
}

func (r Revision) Validate() error {
	if r.OperationID == "" || r.ActorID == "" || len(r.Reason) < 1 || len(r.Reason) > 2000 || r.Revision < 1 || r.Revision > 9007199254740991 || r.OccurredAt.String() == "" || r.CashDate.String() == "" {
		return ErrInvalidRevision
	}
	if !r.Type.Valid() || !r.State.Valid() {
		return ErrInvalidRevision
	}
	if utf8.RuneCountInString(r.Merchant) > 2000 || utf8.RuneCountInString(r.Note) > 2000 || utf8.RuneCountInString(r.AllocationReason) > 2000 {
		return ErrInvalidRevision
	}
	if r.Origin != "" && r.Origin != "manual" && r.Origin != "source" {
		return ErrInvalidRevision
	}
	if r.FeeKnowledge != "" && r.FeeKnowledge != KnownFees && r.FeeKnowledge != UnknownFees {
		return ErrInvalidRevision
	}
	if r.PnLBasis != "" && r.PnLBasis != GrossPnL && r.PnLBasis != NetPnL {
		return ErrInvalidRevision
	}
	if r.Type != TradeResult && r.PnLBasis != "" {
		return ErrInvalidRevision
	}
	if r.PostedAt.String() != "" && (r.State != Posted && r.State != Reversed || r.PostedAt.Time().Before(r.OccurredAt.Time())) {
		return ErrInvalidRevision
	}
	if r.Timezone.String() != "" {
		date, err := r.OccurredAt.DateIn(r.Timezone)
		if err != nil || date != r.CashDate {
			return ErrInvalidRevision
		}
		if r.ExpenseMonth.String() != "" && r.ExpenseMonth.String() != date.String()[:7] {
			return ErrInvalidRevision
		}
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
		if !p.Role.Valid() {
			return ErrInvalidRevision
		}
		if p.Funding != "" && p.Funding != OwnFunds && p.Funding != CreditFunds && p.Funding != UnknownFunds {
			return ErrInvalidRevision
		}
		if p.Treatment != "" && p.Treatment != Movement && p.Treatment != Included && p.Treatment != Valuation {
			return ErrInvalidRevision
		}
		if p.Treatment == Included && (r.Type != TradeResult || r.PnLBasis != NetPnL || p.Role != Fee) {
			return ErrInvalidRevision
		}
		if p.Treatment == Valuation && (r.Type != TradeResult || p.Role != PnL) {
			return ErrInvalidRevision
		}
		if p.Role == Fee && p.Money.Sign() >= 0 {
			return ErrInvalidRevision
		}
		if p.Money.Sign() == 0 && r.Type != Opening && r.Type != TradeResult {
			return ErrInvalidRevision
		}
	}
	return r.validateEconomics()
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
			if !p.MovesMoney() {
				continue
			}
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
