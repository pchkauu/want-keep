package ledger

import (
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	application "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func (s *Server) sourceFactDTO(v application.SourceFact) (generated.SourceTransactionFact, error) {
	r := v.Fact
	out := generated.SourceTransactionFact{SourceId: v.SourceID, SourceRevision: int64(v.Revision), ConflictAtImport: "none", Type: generated.SourceTransactionFactType(r.Type), State: generated.SourceTransactionFactState(r.State), OccurredAt: r.OccurredAt.String(), FeeKnowledge: "unknown", Merchant: r.Merchant, Note: r.Note, Postings: []generated.Posting{}}
	if v.Conflict != "" {
		out.ConflictAtImport = generated.SourceTransactionFactConflictAtImport(v.Conflict)
	}
	if r.FeeKnowledge != "" {
		out.FeeKnowledge = generated.SourceTransactionFactFeeKnowledge(r.FeeKnowledge)
	}
	if r.PostedAt.String() != "" {
		at := r.PostedAt.String()
		out.PostedAt = &at
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
	for _, posting := range r.Postings {
		p, err := s.postingDTO(posting)
		if err != nil {
			return out, err
		}
		out.Postings = append(out.Postings, p)
	}
	return out, nil
}

func (s *Server) postingDTO(p ledger.Posting) (generated.Posting, error) {
	amount, err := (contract.MoneyConverter{}).ToDTO(p.Money)
	if err != nil {
		return generated.Posting{}, err
	}
	out := generated.Posting{AccountId: p.AccountID, Money: amount, Role: generated.PostingRole(p.Role)}
	if p.Funding != "" {
		v := generated.PostingFunding(p.Funding)
		out.Funding = &v
	}
	if p.Treatment != "" {
		v := generated.PostingTreatment(p.Treatment)
		out.Treatment = &v
	}
	return out, nil
}
