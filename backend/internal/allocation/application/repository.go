package application

import (
	"context"

	allocation "github.com/pchkauu/want-keep/backend/internal/allocation/domain"
	category "github.com/pchkauu/want-keep/backend/internal/categories/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

type Repository interface {
	AllocationRule(context.Context, household.Principal, string) (allocation.Rule, error)
	AllocationRules(context.Context, household.Principal, string, int) ([]allocation.Rule, string, error)
	MatchingAllocationRules(context.Context, household.Principal, string, string) ([]allocation.Rule, error)
	AllocationRulesAtBoundary(context.Context, household.Principal, []allocation.Condition, uint64) ([]allocation.Rule, error)
	CreateAllocationRule(context.Context, allocation.Rule) error
	SaveAllocationRule(context.Context, allocation.Rule, uint64) error
	HouseholdMemberships(context.Context, household.Principal) ([]household.Membership, error)
	Category(context.Context, household.Principal, string) (category.Category, error)
	Merchant(context.Context, household.Principal, string) (category.Merchant, error)
	MerchantAliasOwner(context.Context, household.Principal, string, string) (string, error)
	EmitEvent(context.Context, string, string, uint64, string) error
}

type Result = command.Result
