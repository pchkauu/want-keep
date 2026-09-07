package ledger

import (
	"strconv"

	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	application "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func (s *Server) transactionDTO(p household.Principal, v application.View) (generated.Transaction, error) {
	r := v.Revision
	if err := r.Validate(); err != nil {
		return generated.Transaction{}, err
	}
	out := generated.Transaction{Id: r.OperationID, Revision: int64(r.Revision), ActorId: string(r.ActorID), HouseholdId: string(p.HouseholdID()), Type: generated.TransactionType(r.Type), State: generated.TransactionState(r.State), OccurredAt: r.OccurredAt.String(), CashDate: r.CashDate.String(), AiState: "waiting", Origin: "legacy", FeeKnowledge: "unknown", Postings: []generated.Posting{}, Sources: []generated.SourceReference{}, BalanceEffects: []generated.TransactionBalanceEffect{}, EconomicComponents: []generated.EconomicComponent{}, Holds: []generated.TransactionHold{}}
	if r.Origin != "" {
		out.Origin = generated.TransactionOrigin(r.Origin)
	}
	if r.FeeKnowledge != "" {
		out.FeeKnowledge = generated.TransactionFeeKnowledge(r.FeeKnowledge)
	}
	if r.PostedAt.String() != "" {
		value := r.PostedAt.String()
		out.PostedAt = &value
	}
	if r.Timezone.String() != "" {
		value := r.Timezone.String()
		out.Timezone = &value
	}
	if r.ExpenseMonth.String() != "" {
		value := r.ExpenseMonth.String()
		out.ExpenseMonth = &value
	}
	if r.Merchant != "" {
		out.Merchant = &r.Merchant
	}
	if r.Note != "" {
		out.Note = &r.Note
	}
	if r.AttachmentID != "" {
		out.AttachmentId = &r.AttachmentID
	}
	if r.PnLBasis != "" {
		value := generated.TransactionPnlBasis(r.PnLBasis)
		out.PnlBasis = &value
	}
	reason := r.AllocationReason
	if reason == "" {
		reason = "allocation_unresolved"
	}
	if err := out.Allocation.FromUnresolvedAllocation(generated.UnresolvedAllocation{Mode: "unresolved", Reason: reason}); err != nil {
		return out, err
	}
	var err error
	if r.PayerState == "known" {
		err = out.Payer.FromKnownPayer(generated.KnownPayer{State: "known", MemberId: string(r.PayerMemberID)})
	} else {
		err = out.Payer.FromUnspecifiedPayer(generated.UnspecifiedPayer{State: generated.UnspecifiedPayerState(r.PayerState)})
	}
	if err != nil {
		return out, err
	}
	out.Quality.Coverage, err = s.boundary.CoverageToDTO(v.Coverage)
	if err != nil {
		return out, err
	}
	out.Quality.Freshness = "unknown"
	for _, posting := range r.Postings {
		amount, err := (contract.MoneyConverter{}).ToDTO(posting.Money)
		if err != nil {
			return out, err
		}
		dto := generated.Posting{AccountId: posting.AccountID, Money: amount, Role: generated.PostingRole(posting.Role)}
		if posting.Funding != "" {
			f := generated.PostingFunding(posting.Funding)
			dto.Funding = &f
		}
		if posting.Treatment != "" {
			t := generated.PostingTreatment(posting.Treatment)
			dto.Treatment = &t
		}
		out.Postings = append(out.Postings, dto)
	}
	for _, src := range v.Sources {
		out.Sources = append(out.Sources, generated.SourceReference{Provider: generated.Provider(src.Key.Provider), ExternalAccountId: src.Key.ExternalAccountID, Product: src.Key.Product, Log: src.Key.Log, SourceId: src.Key.RecordID, Revision: strconv.FormatUint(src.Revision, 10), ConnectionId: src.ConnectionID})
	}
	components, err := r.Components()
	if err != nil {
		return out, err
	}
	for _, c := range components {
		amount, e := (contract.MoneyConverter{}).ToDTO(c.Money)
		if e != nil {
			return out, e
		}
		out.EconomicComponents = append(out.EconomicComponents, generated.EconomicComponent{Kind: generated.EconomicComponentKind(c.Kind), Money: amount, Treatment: generated.PostingTreatment(c.Treatment)})
	}
	effects, err := r.BalanceEffects()
	if err != nil {
		return out, err
	}
	for _, e := range effects {
		dto, err := s.effectDTO(e)
		if err != nil {
			return out, err
		}
		out.BalanceEffects = append(out.BalanceEffects, dto)
	}
	holds, err := r.Holds()
	if err != nil {
		return out, err
	}
	for _, h := range holds {
		amount, e := s.positiveDTO(h.Amount)
		if e != nil {
			return out, e
		}
		out.Holds = append(out.Holds, generated.TransactionHold{AccountId: h.AccountID, Amount: amount, Funding: generated.PostingFunding(h.Funding)})
	}
	exchange, err := r.ExchangeAmounts()
	if err != nil {
		return out, err
	}
	if exchange != nil {
		x := &generated.ExecutedExchange{}
		x.Sent, err = s.positiveDTO(exchange.Sent)
		if err != nil {
			return out, err
		}
		x.Received, err = s.positiveDTO(exchange.Received)
		if err != nil {
			return out, err
		}
		out.Exchange = x
	}
	return out, nil
}
func (s *Server) effectDTO(e ledger.BalanceEffect) (generated.TransactionBalanceEffect, error) {
	out := generated.TransactionBalanceEffect{AccountId: e.AccountID, At: e.At.String()}
	var err error
	out.Owned, err = s.boundary.AmountToDTO(e.Owned)
	if err != nil {
		return out, err
	}
	out.Available, err = s.boundary.AmountToDTO(e.Available)
	if err != nil {
		return out, err
	}
	out.Locked, err = s.boundary.AmountToDTO(e.Locked)
	if err != nil {
		return out, err
	}
	out.Debt, err = s.boundary.AmountToDTO(e.Debt)
	return out, err
}
