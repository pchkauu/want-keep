package application

import (
	"context"

	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Service) LinkTransfer(ctx context.Context, p household.Principal, in journal.TransferInput, members []matching.Member) (result command.Result, err error) {
	defer func() { err = s.reject(err) }()
	if len(members) < 2 || len(members) > 100 || !in.Existing {
		return result, matching.ErrInvalid
	}
	facts, err := s.expected(ctx, p, members)
	if err != nil {
		return result, err
	}
	kind := matching.Transfer
	if in.Sent.Asset() != in.Received.Asset() {
		kind = matching.Exchange
	}
	g := s.newGroup(p, facts[0], kind, matching.Clarification, "confirmed_existing_transfer")
	g.PrimaryID = members[0].OperationID
	var primary ledger.Revision
	for _, r := range facts {
		if r.OperationID == g.PrimaryID {
			primary = r
		}
	}
	if primary.OccurredAt != in.At {
		return result, matching.ErrConflict
	}
	_, assigned, err := g.Assign(facts, false)
	if err != nil {
		return result, err
	}
	zero, _ := money.NewMoney("0", in.Sent.Asset())
	sent, err := zero.Subtract(in.Sent)
	if err != nil {
		return result, err
	}
	expected := []ledger.Posting{{AccountID: in.FromAccountID, Money: sent, Role: ledger.Principal, Funding: in.FromFunding}, {AccountID: in.ToAccountID, Money: in.Received, Role: ledger.Principal, Funding: in.ToFunding}}
	for _, f := range in.Fees {
		if f.Amount.Sign() <= 0 {
			return result, matching.ErrInvalid
		}
		zero, _ := money.NewMoney("0", f.Amount.Asset())
		amount, err := zero.Subtract(f.Amount)
		if err != nil {
			return result, err
		}
		expected = append(expected, ledger.Posting{AccountID: f.AccountID, Money: amount, Role: ledger.Fee, Funding: f.Funding})
	}
	seen := make([]bool, len(expected))
	count := 0
	for _, r := range assigned {
		for i, p := range r.Postings {
			if !r.Contributes(i) {
				continue
			}
			count++
			found := false
			for j, e := range expected {
				if !seen[j] && p.AccountID == e.AccountID && p.Role == e.Role && p.SameMoney(e) && (e.Funding == "" || p.Funding.SameBasis(e.Funding)) {
					seen[j] = true
					found = true
					break
				}
			}
			if !found {
				return result, matching.ErrConflict
			}
		}
	}
	if count != len(expected) {
		return result, matching.ErrConflict
	}
	return s.Link(ctx, p, LinkInput{Kind: kind, PrimaryID: g.PrimaryID, Members: members, Reason: "confirmed_existing_transfer"})
}
