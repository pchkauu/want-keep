package application

import (
	"context"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

type Repository interface {
	CreateAccount(context.Context, account.Account) error
	Account(context.Context, household.Principal, string) (account.Account, error)
	RecordBalance(context.Context, account.Balance) error
	Balance(context.Context, household.Principal, string, string) (account.Balance, error)
}
