package ledger

import (
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	application "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Server) createInput(in generated.TransactionCreate) (application.CreateInput, error) {
	out := application.CreateInput{Type: ledger.Type(in.Type), AccountID: in.AccountId}
	var err error
	out.At, err = calendar.ParseInstant(in.OccurredAt)
	if err != nil {
		return out, err
	}
	out.Amount, err = money.NewMoney(in.Amount.Amount, money.Asset(in.Amount.Asset))
	if err != nil {
		return out, err
	}
	allocation, err := in.Allocation.AsUnresolvedAllocation()
	if err != nil {
		return out, err
	}
	if allocation.Mode != "unresolved" {
		out.Unsupported = true
	}
	out.AllocationReason = allocation.Reason
	if in.Funding != nil {
		out.Funding = ledger.FundingKind(*in.Funding)
	}
	payer, err := in.Payer.AsKnownPayer()
	if err != nil {
		return out, err
	}
	if payer.State == "known" {
		out.PayerState = "known"
		out.PayerMemberID = household.MembershipID(payer.MemberId)
	} else {
		v, e := in.Payer.AsUnspecifiedPayer()
		if e != nil {
			return out, e
		}
		out.PayerState = string(v.State)
	}
	if in.Merchant != nil {
		out.Merchant = *in.Merchant
	}
	if in.CategoryId != nil {
		out.CategoryID = *in.CategoryId
	}
	if in.MerchantId != nil {
		out.MerchantID = *in.MerchantId
	}
	if in.Note != nil {
		out.Note = *in.Note
	}
	if in.AttachmentId != nil {
		out.AttachmentID = *in.AttachmentId
	}
	return out, nil
}
func (s *Server) transferInput(in generated.TransferCreate) (application.TransferInput, error) {
	out := application.TransferInput{FromAccountID: in.FromAccountId, ToAccountID: in.ToAccountId, Existing: len(in.ExistingTransactions) > 0}
	var err error
	out.At, err = calendar.ParseInstant(in.OccurredAt)
	if err != nil {
		return out, err
	}
	out.Sent, err = money.NewMoney(in.Sent.Amount, money.Asset(in.Sent.Asset))
	if err != nil {
		return out, err
	}
	out.Received, err = money.NewMoney(in.Received.Amount, money.Asset(in.Received.Asset))
	if err != nil {
		return out, err
	}
	if in.FromFunding != nil {
		out.FromFunding = ledger.FundingKind(*in.FromFunding)
	}
	if in.ToFunding != nil {
		out.ToFunding = ledger.FundingKind(*in.ToFunding)
	}
	for _, f := range in.Fees {
		amount, err := money.NewMoney(f.Amount.Amount, money.Asset(f.Amount.Asset))
		if err != nil {
			return out, err
		}
		fee := application.FeeInput{AccountID: f.AccountId, Amount: amount}
		if f.Funding != nil {
			fee.Funding = ledger.FundingKind(*f.Funding)
		}
		out.Fees = append(out.Fees, fee)
	}
	return out, nil
}

func (s *Server) positiveDTO(m money.Money) (generated.PositiveMoney, error) {
	out, err := (contract.MoneyConverter{}).ToDTO(m)
	if err != nil {
		return generated.PositiveMoney{}, err
	}
	if m.Sign() <= 0 {
		return generated.PositiveMoney{}, money.ErrInvalidMoney
	}
	return generated.PositiveMoney{Amount: out.Amount, Asset: out.Asset}, nil
}
