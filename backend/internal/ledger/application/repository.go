package application

import (
	"context"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

type Repository interface {
	AppendRevision(context.Context, ledger.Revision, uint64) error
	LedgerRevision(context.Context, household.Principal, string, uint64) (ledger.Revision, error)
}
