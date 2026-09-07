package domain

import (
	"regexp"
	"strings"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type Observation struct {
	ID, AccountID, ConnectionID, JobID, EvidenceRef string
	AsOf, FetchedAt                                 calendar.Instant
	Amounts                                         Amounts
	CreditLimit                                     reporting.Amount
	OwnAvailable                                    bool
	Coverage                                        reporting.Coverage
	Freshness                                       reporting.Freshness
}

func (o Observation) Validate(asset money.Asset) error {
	if o.ID == "" || o.AccountID == "" || o.ConnectionID == "" || o.JobID == "" || len(o.EvidenceRef) < 1 || len(o.EvidenceRef) > 2000 || o.AsOf.String() == "" || o.FetchedAt.String() == "" || o.AsOf.Time().After(o.FetchedAt.Time()) {
		return ErrInvalidAccount
	}
	if err := o.Amounts.Validate(asset); err != nil {
		return err
	}
	if err := (Amounts{o.CreditLimit, o.CreditLimit, o.CreditLimit, o.CreditLimit}).Validate(asset); err != nil {
		return err
	}
	if m, ok := o.CreditLimit.Value(); ok && m.Sign() < 0 {
		return ErrInvalidAccount
	}
	if _, err := reporting.NewCoverage(o.Coverage.State(), o.Coverage.Reasons()); err != nil {
		return err
	}
	_, err := reporting.ParseFreshness(string(o.Freshness))
	return err
}

// OwnAvailable records a verified provider mapping; a credit-inclusive balance is never spendable own money.
func (o Observation) Spendable() reporting.Amount {
	if !o.OwnAvailable {
		return UnknownAmounts("own_availability_unverified").Available
	}
	if o.Coverage.State() != reporting.Complete || o.Freshness != reporting.Fresh {
		return UnknownAmounts("source_quality_insufficient").Available
	}
	owned, ownKnown := o.Amounts.Owned.Value()
	locked, lockedKnown := o.Amounts.Locked.Value()
	available, availableKnown := o.Amounts.Available.Value()
	if !ownKnown || !lockedKnown || !availableKnown || locked.Sign() < 0 || available.Sign() < 0 {
		return UnknownAmounts("own_availability_unverified").Available
	}
	ceiling, err := owned.Subtract(locked)
	if err != nil {
		return UnknownAmounts("own_availability_unverified").Available
	}
	cmp, err := available.Compare(ceiling)
	if err != nil || cmp > 0 {
		return UnknownAmounts("own_availability_unverified").Available
	}
	return o.Amounts.Available
}

// Funding never adds journal movements to a source snapshot. Newer movements require reconciliation first.
func (o Observation) Funding(effects []Effect) reporting.Amount {
	for _, e := range effects {
		if e.At.Time().After(o.AsOf.Time()) {
			return UnknownAmounts("history_not_reconciled").Available
		}
	}
	return o.Spendable()
}

type CardAlias struct{ ID, AccountID, Label, LastFour string }

var lastFourPattern = regexp.MustCompile(`^[0-9]{4}$`)
var longDigits = regexp.MustCompile(`[0-9][0-9 -]{5,}[0-9]`)

func (c CardAlias) Validate() error {
	if c.ID == "" || c.AccountID == "" || strings.TrimSpace(c.Label) == "" || len(c.Label) > 100 || longDigits.MatchString(c.Label) || !lastFourPattern.MatchString(c.LastFour) {
		return ErrInvalidAccount
	}
	return nil
}

// Equivalent compares financial meaning at one observation instant, independently of fetch/session provenance.
func (o Observation) Equivalent(other Observation) bool {
	if o.OwnAvailable != other.OwnAvailable {
		return false
	}
	values := append(o.Amounts.Fields(), o.CreditLimit)
	right := append(other.Amounts.Fields(), other.CreditLimit)
	for i, v := range values {
		r := right[i]
		if v.Knowledge() != r.Knowledge() || v.Reason() != r.Reason() {
			return false
		}
		if m, ok := v.Value(); ok {
			n, _ := r.Value()
			cmp, err := m.Compare(n)
			if err != nil || cmp != 0 {
				return false
			}
		}
	}
	return true
}
