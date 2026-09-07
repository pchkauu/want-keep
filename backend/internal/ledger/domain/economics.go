package domain

import (
	"sort"

	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (p Posting) MovesMoney() bool { return p.Treatment == "" || p.Treatment == Movement }

func (r Revision) validateEconomics() error {
	principal := []Posting{}
	for _, p := range r.Postings {
		if !p.MovesMoney() {
			if p.Treatment == Included {
				matched := false
				for _, net := range r.Postings {
					if net.Role == PnL && net.MovesMoney() && net.AccountID == p.AccountID && net.Money.Asset() == p.Money.Asset() {
						matched = true
					}
				}
				if !matched {
					return ErrInvalidRevision
				}
			}
			continue
		}
		if p.Role == Principal {
			principal = append(principal, p)
		}
		switch r.Type {
		case Income:
			if p.Role != Principal && p.Role != Fee {
				return ErrInvalidRevision
			}
			if p.Role == Principal && p.Money.Sign() <= 0 {
				return ErrInvalidRevision
			}
		case Expense:
			if p.Role != Principal && p.Role != Fee && p.Role != Interest {
				return ErrInvalidRevision
			}
			if p.Money.Sign() >= 0 {
				return ErrInvalidRevision
			}
		case Transfer, Exchange:
			if p.Role != Principal && p.Role != Fee && p.Role != Interest {
				return ErrInvalidRevision
			}
			if p.Role == Interest && p.Money.Sign() >= 0 {
				return ErrInvalidRevision
			}
		case Yield:
			if p.Role != Reward && p.Role != Interest && p.Role != Funding && p.Role != Fee {
				return ErrInvalidRevision
			}
			if (p.Role == Reward || p.Role == Interest) && p.Money.Sign() <= 0 {
				return ErrInvalidRevision
			}
		case TradeResult:
			if p.Role != PnL && p.Role != Funding && p.Role != Fee {
				return ErrInvalidRevision
			}
		case Opening:
			if p.Role != Principal {
				return ErrInvalidRevision
			}
		case Refund:
			if p.Role != Principal && p.Role != Fee {
				return ErrInvalidRevision
			}
			if p.Role == Principal && p.Money.Sign() <= 0 {
				return ErrInvalidRevision
			}
		}
	}
	if r.Type != Transfer && r.Type != Exchange {
		return nil
	}
	// Drafts may retain incomplete legs, but an executable movement must be complete.
	if r.State == Draft && len(principal) < 2 {
		return nil
	}
	if len(principal) != 2 || principal[0].AccountID == principal[1].AccountID || principal[0].Money.Sign()*principal[1].Money.Sign() != -1 {
		return ErrInvalidRevision
	}
	if r.Type == Exchange {
		if principal[0].Money.Asset() == principal[1].Money.Asset() {
			return ErrInvalidRevision
		}
		return nil
	}
	sum, err := principal[0].Money.Add(principal[1].Money)
	if err != nil {
		return err
	}
	if sum.Sign() != 0 {
		return ErrInvalidRevision
	}
	return nil
}

// CheckSuccessor validates the complete replacement snapshot without modifying either revision.
func (r Revision) CheckSuccessor(previous *Revision) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if previous == nil {
		if r.Revision != 1 || r.State == Reversed {
			return ErrInvalidTransition
		}
		return nil
	}
	if previous.OperationID != r.OperationID || previous.Revision >= 9007199254740991 || r.Revision != previous.Revision+1 {
		return ErrInvalidTransition
	}
	return previous.State.RequireNext(r.State)
}

func (r Revision) AffectedAccounts(previous *Revision) []string {
	seen := map[string]bool{}
	for _, v := range []*Revision{previous, &r} {
		if v == nil {
			continue
		}
		for _, p := range v.Postings {
			seen[p.AccountID] = true
		}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

type EconomicComponent struct {
	Kind      string
	Money     money.Money
	Treatment Treatment
}

// Components exposes explanatory native facts. Included fees and valuation never
// contribute a second cash effect. Consumer reports must retain this treatment.
func (r Revision) Components() ([]EconomicComponent, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	result := []EconomicComponent{}
	if r.State != Posted || r.Accounting() == ExcludedFromAccounting {
		return result, nil
	}
	for _, p := range r.Postings {
		kind := string(p.Role)
		switch p.Role {
		case Principal:
			switch r.Type {
			case Income:
				kind = "income"
			case Expense:
				kind = "expense"
			default:
				continue
			}
		case PnL:
			kind = "realized_pnl"
			if p.Treatment == Valuation {
				kind = "unrealized_pnl"
			}
		}
		treatment := p.Treatment
		if treatment == "" {
			treatment = Movement
		}
		result = append(result, EconomicComponent{Kind: kind, Money: p.Money, Treatment: treatment})
	}
	return result, nil
}
