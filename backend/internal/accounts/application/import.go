package application

import (
	"context"
	"errors"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type ImportInput struct {
	ConnectionID, JobID, Provider, ExternalID, Product, Network, ExternalAssetCode, Name, EvidenceRef string
	Asset                                                                                             money.Asset
	OpeningDate                                                                                       calendar.Date
	Observation                                                                                       account.Observation
	Aliases                                                                                           []account.CardAlias
	Origin                                                                                            string
}
type ImportResult struct {
	Account *account.Account
	Reason  string
}
type ImportRepository interface {
	WithinAccountImport(context.Context, func(context.Context) error) error
	RecordAccountImportIssue(context.Context, household.Principal, ImportInput, string) error
	ResolveImportedAccount(context.Context, household.Principal, ImportInput) (account.Account, bool, error)
	RecordObservation(context.Context, household.Principal, account.Observation) error
	RecordCardAlias(context.Context, household.Principal, account.CardAlias) error
}

// Import is database-only work inside the admitted CommitPage transaction. It never posts a bank balance as a transaction.
func (s *Service) applyImport(ctx context.Context, p household.Principal, r ImportRepository, input ImportInput) (account.Account, error) {
	a, created, err := r.ResolveImportedAccount(ctx, p, input)
	if err != nil {
		return a, err
	}
	if created {
		zone, err := s.repository.AccountTimezone(ctx, p)
		if err != nil {
			return a, err
		}
		o := account.Opening{AccountID: a.ID, OperationID: s.newID(), Revision: 1, Date: a.OpeningDate, Timezone: zone, Amounts: account.UnknownAmounts("opening_unconfirmed"), ActorID: p.UserID(), Reason: "source_opening_unconfirmed", At: s.now()}
		if err = s.writeOpening(ctx, p, a, o, 0); err != nil {
			return a, err
		}
		if _, err = s.result(ctx, p, a.ID, "created", o.Reason, input.Origin); err != nil {
			return a, err
		}
	}
	o := input.Observation
	o.AccountID = a.ID
	o.ConnectionID = input.ConnectionID
	o.JobID = input.JobID
	o.EvidenceRef = input.EvidenceRef
	if err = r.RecordObservation(ctx, p, o); err != nil {
		return a, err
	}
	for _, alias := range input.Aliases {
		alias.AccountID = a.ID
		if err = r.RecordCardAlias(ctx, p, alias); err != nil {
			return a, err
		}
	}
	if s.reconciler != nil {
		if err = s.reconciler.ReconcileAccount(ctx, p, a.ID); err != nil {
			return a, err
		}
	}
	return s.repository.Account(ctx, p, a.ID)
}

func (s *Service) Import(ctx context.Context, p household.Principal, r ImportRepository, input ImportInput) (ImportResult, error) {
	var result ImportResult
	err := r.WithinAccountImport(ctx, func(ctx context.Context) error {
		a, e := s.applyImport(ctx, p, r, input)
		if e == nil {
			result.Account = &a
		}
		return e
	})
	switch {
	case errors.Is(err, ledger.ErrSourceAmbiguous):
		result.Reason = "source_ambiguous"
	case errors.Is(err, money.ErrUnsupportedAsset):
		result.Reason = "unsupported_asset"
	default:
		return result, err
	}
	result.Account = nil
	return result, r.RecordAccountImportIssue(ctx, p, input, result.Reason)
}
