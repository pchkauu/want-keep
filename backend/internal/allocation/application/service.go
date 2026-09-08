package application

import (
	"context"
	"errors"
	"slices"

	allocation "github.com/pchkauu/want-keep/backend/internal/allocation/domain"
	category "github.com/pchkauu/want-keep/backend/internal/categories/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

type Service struct {
	repository Repository
	newID      func() string
}

func NewService(repository Repository, newID func() string) *Service {
	return &Service{repository: repository, newID: newID}
}

type RuleInput struct {
	Priority  int
	State     allocation.State
	Condition allocation.Condition
	Shares    []allocation.Share
}

func (s *Service) CreateRule(ctx context.Context, principal household.Principal, input RuleInput) (command.Result, error) {
	rule := allocation.Rule{ID: s.newID(), HouseholdID: principal.HouseholdID(), Revision: 1, Priority: input.Priority, State: input.State, Condition: input.Condition, Shares: slices.Clone(input.Shares), ActorID: principal.UserID()}
	if err := s.validateRule(ctx, principal, rule); err != nil {
		return command.Result{}, err
	}
	if err := s.repository.CreateAllocationRule(ctx, rule); err != nil {
		return command.Result{}, s.reject(err)
	}
	if err := s.repository.EmitEvent(ctx, "allocation_rule", rule.ID, rule.Revision, "allocation_rule.created"); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "allocation_rule", ResourceID: rule.ID, Revision: rule.Revision}, nil
}

func (s *Service) ChangeRule(ctx context.Context, principal household.Principal, id string, expected uint64, input RuleInput) (command.Result, error) {
	current, err := s.repository.AllocationRule(ctx, principal, id)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	if current.Revision != expected || expected >= allocation.MaxRevision {
		return command.Result{}, commands.Rejection{Code: "version_conflict"}
	}
	next, err := current.Apply(allocation.Change{Priority: input.Priority, State: input.State, Condition: input.Condition, Shares: slices.Clone(input.Shares)}, principal.UserID())
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	if err = s.validateRule(ctx, principal, next); err != nil {
		return command.Result{}, err
	}
	if err = s.repository.SaveAllocationRule(ctx, next, expected); err != nil {
		return command.Result{}, s.reject(err)
	}
	if err = s.repository.EmitEvent(ctx, "allocation_rule", next.ID, next.Revision, "allocation_rule.changed"); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "allocation_rule", ResourceID: next.ID, Revision: next.Revision}, nil
}

func (s *Service) Rules(ctx context.Context, principal household.Principal, after string, limit int) ([]allocation.Rule, string, error) {
	if limit < 1 || limit > 100 {
		return nil, "", commands.Rejection{Code: "invalid_request"}
	}
	return s.repository.AllocationRules(ctx, principal, after, limit)
}

func (s *Service) Rule(ctx context.Context, principal household.Principal, id string) (allocation.Rule, error) {
	rule, err := s.repository.AllocationRule(ctx, principal, id)
	if err != nil {
		return allocation.Rule{}, s.reject(err)
	}
	return rule, nil
}

func (s *Service) Preview(ctx context.Context, principal household.Principal, merchantID, categoryID string) (allocation.Resolution, error) {
	if err := s.requireConditions(ctx, principal, allocation.Condition{MerchantID: merchantID, CategoryID: categoryID}); err != nil {
		return allocation.Resolution{}, err
	}
	rules, err := s.repository.MatchingAllocationRules(ctx, principal, merchantID, categoryID)
	if err != nil {
		return allocation.Resolution{}, s.reject(err)
	}
	return allocation.Resolve(rules, merchantID, categoryID), nil
}

func (s *Service) Resolve(ctx context.Context, principal household.Principal, merchantID, categoryID string) (ledger.AllocationInput, bool, error) {
	if merchantID == "" && categoryID == "" {
		return ledger.AllocationInput{Mode: ledger.AllocationUnknown, Reason: "no_matching_rule"}, false, nil
	}
	resolution, err := s.Preview(ctx, principal, merchantID, categoryID)
	if err != nil {
		return ledger.AllocationInput{}, false, err
	}
	if resolution.State != "resolved" {
		return ledger.AllocationInput{Mode: ledger.AllocationUnknown, Reason: resolution.Reason}, false, nil
	}
	input := ledger.AllocationInput{Mode: ledger.AllocationByShares, Purpose: ledger.AllocationShared, Origin: ledger.AllocationRule, Reason: "allocation_rule"}
	if len(resolution.Shares) == 1 {
		input.Purpose = ledger.AllocationPersonal
	}
	for _, share := range resolution.Shares {
		input.Members = append(input.Members, ledger.AllocationMemberInput{MemberID: share.MemberID, Share: share.Value})
	}
	for _, rule := range resolution.Rules {
		input.RuleRefs = append(input.RuleRefs, ledger.AllocationRuleRef{ID: rule.ID, Revision: rule.Revision})
	}
	return input, true, nil
}

func (s *Service) ActiveMemberIDs(ctx context.Context, principal household.Principal) ([]household.MembershipID, error) {
	members, err := s.repository.ActiveMemberships(ctx, principal)
	if err != nil {
		return nil, err
	}
	result := make([]household.MembershipID, 0, len(members))
	for _, member := range members {
		if member.Active && member.HouseholdID == principal.HouseholdID() {
			result = append(result, member.ID)
		}
	}
	slices.Sort(result)
	return result, nil
}

func (s *Service) validateRule(ctx context.Context, principal household.Principal, rule allocation.Rule) error {
	if err := rule.Validate(); err != nil {
		return s.reject(err)
	}
	if err := s.requireConditions(ctx, principal, rule.Condition); err != nil {
		return err
	}
	members, err := s.ActiveMemberIDs(ctx, principal)
	if err != nil {
		return err
	}
	active := map[household.MembershipID]bool{}
	for _, id := range members {
		active[id] = true
	}
	for _, share := range rule.Shares {
		if !active[share.MemberID] {
			return commands.Rejection{Code: "invalid_allocation"}
		}
	}
	if len(rule.Shares) != 1 && len(rule.Shares) != len(members) {
		return commands.Rejection{Code: "invalid_allocation"}
	}
	return nil
}

func (s *Service) requireConditions(ctx context.Context, principal household.Principal, condition allocation.Condition) error {
	if condition.MerchantID == "" && condition.CategoryID == "" {
		return commands.Rejection{Code: "invalid_request"}
	}
	if condition.MerchantID != "" {
		merchant, err := s.repository.Merchant(ctx, principal, condition.MerchantID)
		if err != nil || merchant.State != category.Active {
			return commands.Rejection{Code: "not_found"}
		}
	}
	if condition.CategoryID != "" {
		entry, err := s.repository.Category(ctx, principal, condition.CategoryID)
		if err != nil || entry.State != category.Active {
			return commands.Rejection{Code: "not_found"}
		}
	}
	return nil
}

func (s *Service) reject(err error) error {
	switch {
	case errors.Is(err, allocation.ErrRuleNotFound), errors.Is(err, category.ErrNotFound):
		return commands.Rejection{Code: "not_found"}
	case errors.Is(err, allocation.ErrRuleNoChange):
		return commands.Rejection{Code: "no_change"}
	case errors.Is(err, allocation.ErrRuleConflict):
		return commands.Rejection{Code: "decision_conflict"}
	case errors.Is(err, allocation.ErrInvalidRule):
		return commands.Rejection{Code: "invalid_allocation"}
	case errors.Is(err, household.ErrForbidden):
		return commands.Rejection{Code: "forbidden"}
	default:
		return err
	}
}
