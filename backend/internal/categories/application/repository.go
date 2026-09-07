package application

import (
	"context"

	category "github.com/pchkauu/want-keep/backend/internal/categories/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

type Repository interface {
	Category(context.Context, household.Principal, string) (category.Category, error)
	Categories(context.Context, household.Principal, category.Filter, string, int) ([]category.Category, string, error)
	CategoryNameExists(context.Context, household.Principal, string, string, string) (bool, error)
	CategoryHasActiveChildren(context.Context, household.Principal, string) (bool, error)
	CreateCategory(context.Context, category.Category) error
	SaveCategory(context.Context, category.Category, uint64) error

	Merchant(context.Context, household.Principal, string) (category.Merchant, error)
	Merchants(context.Context, household.Principal, category.Filter, string, int) ([]category.Merchant, string, error)
	MerchantNameExists(context.Context, household.Principal, string, string) (bool, error)
	MerchantAliasOwner(context.Context, household.Principal, string, string) (string, error)
	CreateMerchant(context.Context, category.Merchant) error
	SaveMerchant(context.Context, category.Merchant, uint64) error

	EmitEvent(context.Context, string, string, uint64, string) error
}
