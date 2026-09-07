package domain

import (
	"errors"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

var ErrInvalidAvailability = errors.New("invalid availability")

type Knowledge string

const (
	Known       Knowledge = "known"
	Unknown     Knowledge = "unknown"
	Unavailable Knowledge = "unavailable"
)

type Amount struct {
	knowledge Knowledge
	value     money.Money
	reason    string
}

func KnownAmount(value money.Money) (Amount, error) {
	if err := value.Validate(); err != nil {
		return Amount{}, err
	}
	return Amount{knowledge: Known, value: value}, nil
}

func MissingAmount(knowledge Knowledge, reason string) (Amount, error) {
	if (knowledge != Unknown && knowledge != Unavailable) || reason == "" {
		return Amount{}, ErrInvalidAvailability
	}
	return Amount{knowledge: knowledge, reason: reason}, nil
}

func (a Amount) Knowledge() Knowledge       { return a.knowledge }
func (a Amount) Reason() string             { return a.reason }
func (a Amount) Value() (money.Money, bool) { return a.value, a.knowledge == Known }
func (a Amount) Validate() error {
	if a.knowledge == Known {
		if a.reason != "" {
			return ErrInvalidAvailability
		}
		return a.value.Validate()
	}
	_, err := MissingAmount(a.knowledge, a.reason)
	return err
}

type CoverageState string

const (
	Complete   CoverageState = "complete"
	Partial    CoverageState = "partial"
	NoCoverage CoverageState = "unavailable"
)

type Coverage struct {
	state   CoverageState
	reasons []string
}

func NewCoverage(state CoverageState, reasons []string) (Coverage, error) {
	if state != Complete && state != Partial && state != NoCoverage {
		return Coverage{}, ErrInvalidAvailability
	}
	if (state == Complete && len(reasons) != 0) || (state != Complete && len(reasons) == 0) {
		return Coverage{}, ErrInvalidAvailability
	}
	for _, reason := range reasons {
		if reason == "" {
			return Coverage{}, ErrInvalidAvailability
		}
	}
	return Coverage{state: state, reasons: append([]string(nil), reasons...)}, nil
}

func (c Coverage) State() CoverageState { return c.state }
func (c Coverage) Reasons() []string    { return append([]string(nil), c.reasons...) }

type Freshness string

const (
	Fresh            Freshness = "fresh"
	Stale            Freshness = "stale"
	UnknownFreshness Freshness = "unknown"
)

func ParseFreshness(value string) (Freshness, error) {
	switch Freshness(value) {
	case Fresh, Stale, UnknownFreshness:
		return Freshness(value), nil
	default:
		return "", ErrInvalidAvailability
	}
}
