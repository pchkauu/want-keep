package application

import (
	"context"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type FeeInput struct {
	AccountID string
	Amount    money.Money
	Funding   ledger.FundingKind
}
type TransferInput struct {
	FromAccountID, ToAccountID string
	At                         calendar.Instant
	Sent, Received             money.Money
	FromFunding, ToFunding     ledger.FundingKind
	Fees                       []FeeInput
	Existing                   bool
}

func (s *Service) Transfer(ctx context.Context, p household.Principal, in TransferInput) (command.Result, error) {
	if in.Existing {
		return command.Result{}, commands.Rejection{Code: "feature_unavailable"}
	}
	if len(in.Fees) > 998 || in.FromAccountID == in.ToAccountID {
		return command.Result{}, commands.Rejection{Code: "invalid_transaction"}
	}
	for _, v := range []money.Money{in.Sent, in.Received} {
		if err := v.Validate(); err != nil {
			return command.Result{}, s.reject(err)
		}
		if v.Sign() <= 0 {
			return command.Result{}, commands.Rejection{Code: "invalid_money"}
		}
	}
	r, err := s.manual(ctx, p, in.At)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	r.Type = ledger.Transfer
	if in.Sent.Asset() != in.Received.Asset() {
		r.Type = ledger.Exchange
	}
	zero, _ := money.NewMoney("0", in.Sent.Asset())
	out, err := zero.Subtract(in.Sent)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	r.Postings = []ledger.Posting{{AccountID: in.FromAccountID, Money: out, Role: ledger.Principal, Funding: in.FromFunding, Treatment: ledger.Movement}, {AccountID: in.ToAccountID, Money: in.Received, Role: ledger.Principal, Funding: in.ToFunding, Treatment: ledger.Movement}}
	for _, fee := range in.Fees {
		if err := fee.Amount.Validate(); err != nil {
			return command.Result{}, s.reject(err)
		}
		if fee.Amount.Sign() <= 0 {
			return command.Result{}, commands.Rejection{Code: "invalid_money"}
		}
		zero, _ := money.NewMoney("0", fee.Amount.Asset())
		amount, err := zero.Subtract(fee.Amount)
		if err != nil {
			return command.Result{}, s.reject(err)
		}
		r.Postings = append(r.Postings, ledger.Posting{AccountID: fee.AccountID, Money: amount, Role: ledger.Fee, Funding: fee.Funding, Treatment: ledger.Movement})
	}
	return s.append(ctx, p, r)
}
