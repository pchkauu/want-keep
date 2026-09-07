package application

import (
	"context"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type Repository interface {
	CreateAccount(context.Context, account.Account) error
	Account(context.Context, household.Principal, string) (account.Account, error)
	RecordBalance(context.Context, account.Balance) error
	Balance(context.Context, household.Principal, string, string) (account.Balance, error)
	Opening(context.Context, household.Principal, string) (account.Opening, bool, error)
	AccountEffects(context.Context, household.Principal, string) ([]account.Effect, error)
}

type CatalogRepository interface {
	Repository
	Accounts(context.Context, household.Principal, string, int) ([]account.Account, string, error)
	SaveOpening(context.Context, account.Opening) error
	ChangeAccountOwnership(context.Context, household.Principal, string, uint64, household.Ownership) error
	AccountTimezone(context.Context, household.Principal) (calendar.Timezone, error)
	AccountEvent(context.Context, household.Principal, string, uint64, string, string, string, calendar.Instant) error
	CardAliases(context.Context, household.Principal, string) ([]account.CardAlias, error)
	LatestObservation(context.Context, household.Principal, string) (account.Observation, bool, error)
	AccountFunding(context.Context, household.Principal, string) (reporting.Amount, error)
}
