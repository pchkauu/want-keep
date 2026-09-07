package accounts

import (
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func (s *Server) amountsDTO(v account.Amounts) (generated.OpeningAmounts, error) {
	fields := []*generated.AmountValue{}
	var out generated.OpeningAmounts
	fields = append(fields, &out.Owned, &out.Available, &out.Locked, &out.Debt)
	for i, target := range fields {
		x, err := s.boundary.AmountToDTO(v.Fields()[i])
		if err != nil {
			return out, err
		}
		*target = x
	}
	return out, nil
}
func (s *Server) amountsFromDTO(v generated.OpeningAmounts) (account.Amounts, error) {
	var out account.Amounts
	fields := []*reporting.Amount{&out.Owned, &out.Available, &out.Locked, &out.Debt}
	for i, input := range []generated.AmountValue{v.Owned, v.Available, v.Locked, v.Debt} {
		x, err := s.boundary.AmountFromDTO(input)
		if err != nil {
			return out, err
		}
		*fields[i] = x
	}
	return out, nil
}
func (s *Server) quality(c reporting.Coverage, f reporting.Freshness) (generated.DataQuality, error) {
	v, err := s.boundary.CoverageToDTO(c)
	return generated.DataQuality{Coverage: v, Freshness: generated.Freshness(f)}, err
}
func (s *Server) accountDTO(v accounts.View) (generated.Account, error) {
	a := v.Account
	out := generated.Account{Id: a.ID, Name: a.Name, Asset: generated.Asset(a.Asset), Product: generated.AccountProduct(a.Product), Revision: int64(a.Revision), OpeningDate: a.OpeningDate.String(), CardAliases: []generated.CardAlias{}}
	var err error
	if a.ExternalOwnerID != "" {
		id := string(a.ExternalOwnerID)
		out.ExternalAccountOwnerId = &id
	}
	out.Ownership, err = s.boundary.OwnershipToDTO(a.Ownership)
	if err != nil {
		return out, err
	}
	out.FundingAvailability, err = s.boundary.AmountToDTO(v.FundingAvailability())
	if err != nil {
		return out, err
	}
	amounts, err := s.amountsDTO(v.Ledger)
	if err != nil {
		return out, err
	}
	quality, err := s.quality(v.Coverage, reporting.UnknownFreshness)
	if err != nil {
		return out, err
	}
	out.Balance = generated.Balance{Owned: amounts.Owned, Available: amounts.Available, Locked: amounts.Locked, Debt: amounts.Debt, Quality: quality}
	if a.Network != "" {
		out.Network = &a.Network
	}
	if a.ExternalAssetCode != "" {
		out.ExternalAssetCode = &a.ExternalAssetCode
	}
	for _, c := range v.Cards {
		out.CardAliases = append(out.CardAliases, generated.CardAlias{Id: c.ID, Label: c.Label, LastFour: c.LastFour})
	}
	if o := v.Opening; o != nil {
		values, err := s.amountsDTO(o.Amounts)
		if err != nil {
			return out, err
		}
		out.Opening = &generated.OpeningPoint{Revision: int64(o.Revision), Date: o.Date.String(), Timezone: o.Timezone.String(), Confirmed: o.Confirmed, Balances: values, ActorId: string(o.ActorID), Reason: o.Reason, RecordedAt: o.At.String()}
	}
	if o := v.Source; o != nil {
		values, err := s.amountsDTO(o.Amounts)
		if err != nil {
			return out, err
		}
		quality, err := s.quality(o.Coverage, o.Freshness)
		if err != nil {
			return out, err
		}
		asof, fetched := o.AsOf.String(), o.FetchedAt.String()
		quality.AsOf = &asof
		quality.FetchedAt = &fetched
		credit, err := s.boundary.AmountToDTO(o.CreditLimit)
		if err != nil {
			return out, err
		}
		out.ConnectionId = &o.ConnectionID
		out.SourceBalance = &generated.SourceBalance{Id: o.ID, ConnectionId: o.ConnectionID, EvidenceRef: o.EvidenceRef, CreditLimit: credit, OwnAvailabilityVerified: o.OwnAvailable, Balances: generated.Balance{Owned: values.Owned, Available: values.Available, Locked: values.Locked, Debt: values.Debt, Quality: quality}}
	}
	return out, nil
}
func (s *Server) ownershipFromDTO(v generated.ScopeInput, p household.Principal) (household.Ownership, error) {
	x, err := v.AsPersonalScopeInput()
	if err != nil {
		return household.Ownership{}, contract.ErrInvalidRequest
	}
	return household.NewOwnership(p.HouseholdID(), household.Scope(x.Scope), household.UserID(x.PersonalOwnerId))
}
