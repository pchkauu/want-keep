package domain

import "errors"

var ErrInvalidTransition = errors.New("invalid transaction transition")
var ErrNotFound = errors.New("transaction not found")
var ErrFeatureUnavailable = errors.New("transaction feature unavailable")

type State string

const (
	Draft     State = "draft"
	Pending   State = "pending"
	Posted    State = "posted"
	Cancelled State = "cancelled"
	Reversed  State = "reversed"
)

func (s State) Valid() bool {
	switch s {
	case Draft, Pending, Posted, Cancelled, Reversed:
		return true
	}
	return false
}

func (s State) RequireNext(next State) error {
	if !s.Valid() || !next.Valid() {
		return ErrInvalidTransition
	}
	if s == next {
		return nil
	}
	switch s {
	case Draft:
		if next == Pending || next == Posted || next == Cancelled {
			return nil
		}
	case Pending:
		if next == Posted || next == Cancelled {
			return nil
		}
	case Posted:
		if next == Reversed {
			return nil
		}
	}
	return ErrInvalidTransition
}

type Type string

const (
	Income      Type = "income"
	Expense     Type = "expense"
	Transfer    Type = "transfer"
	Exchange    Type = "exchange"
	Refund      Type = "refund"
	Opening     Type = "opening"
	Adjustment  Type = "adjustment"
	Yield       Type = "yield"
	TradeResult Type = "trade_result"
)

func (t Type) Valid() bool {
	switch t {
	case Income, Expense, Transfer, Exchange, Refund, Opening, Adjustment, Yield, TradeResult:
		return true
	}
	return false
}

type Role string

const (
	Principal Role = "principal"
	Fee       Role = "fee"
	Interest  Role = "interest"
	Funding   Role = "funding"
	PnL       Role = "pnl"
	Reward    Role = "reward"
)

func (r Role) Valid() bool {
	switch r {
	case Principal, Fee, Interest, Funding, PnL, Reward:
		return true
	}
	return false
}

// A posting's funding describes whose money moved, independently of its budget meaning.
type FundingKind string

const (
	OwnFunds     FundingKind = "own"
	CreditFunds  FundingKind = "credit"
	UnknownFunds FundingKind = "unknown"
)

type Treatment string

const (
	Movement  Treatment = "movement"
	Included  Treatment = "included"
	Valuation Treatment = "valuation"
)

type FeeKnowledge string

const (
	KnownFees   FeeKnowledge = "known"
	UnknownFees FeeKnowledge = "unknown"
)

type PnLBasis string

const (
	GrossPnL PnLBasis = "gross"
	NetPnL   PnLBasis = "net"
)
