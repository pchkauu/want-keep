package domain

import money "github.com/pchkauu/want-keep/backend/internal/money/domain"

type Hold struct {
	AccountID string
	Amount    money.Money
	Funding   FundingKind
}

func (r Revision) FeeOnly() bool {
	if len(r.Postings) == 0 {
		return false
	}
	for _, p := range r.Postings {
		if p.Role != Fee || !p.MovesMoney() {
			return false
		}
	}
	return true
}

func (r Revision) Holds() ([]Hold, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	out := []Hold{}
	if r.Accounting() == ExcludedFromAccounting {
		return out, nil
	}
	for i, p := range r.Postings {
		if !r.Contributes(i) || r.ContributionState(i) != Pending || !p.MovesMoney() || p.Money.Sign() >= 0 {
			continue
		}
		zero, _ := money.NewMoney("0", p.Money.Asset())
		amount, err := zero.Subtract(p.Money)
		if err != nil {
			return nil, err
		}
		funding := p.Funding
		if funding == "" {
			funding = OwnFunds
		}
		out = append(out, Hold{AccountID: p.AccountID, Amount: amount, Funding: funding})
	}
	return out, nil
}

type ExecutedExchange struct{ Sent, Received money.Money }

func (r Revision) ExchangeAmounts() (*ExecutedExchange, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	if r.Type != Exchange || r.State != Posted {
		return nil, nil
	}
	out := &ExecutedExchange{}
	for _, p := range r.Postings {
		if p.Role != Principal {
			continue
		}
		if p.Money.Sign() > 0 {
			out.Received = p.Money
			continue
		}
		zero, _ := money.NewMoney("0", p.Money.Asset())
		var err error
		out.Sent, err = zero.Subtract(p.Money)
		if err != nil {
			return nil, err
		}
	}
	if out.Sent.Validate() != nil || out.Received.Validate() != nil {
		return nil, nil
	}
	return out, nil
}
