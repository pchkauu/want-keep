package application

import (
	"context"
	"errors"
	"slices"
	"strings"

	category "github.com/pchkauu/want-keep/backend/internal/categories/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

type Service struct {
	repository Repository
	newID      func() string
}

func NewService(repository Repository, newID func() string) *Service {
	return &Service{repository: repository, newID: newID}
}

type CategoryInput struct {
	Name, ParentID string
}

func (s *Service) CreateCategory(ctx context.Context, p household.Principal, in CategoryInput) (command.Result, error) {
	if s.repository == nil || s.newID == nil || p.RequireHousehold(p.HouseholdID()) != nil {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	c := category.Category{HouseholdID: p.HouseholdID(), ID: s.newID(), Revision: 1, ParentID: in.ParentID, CustomName: strings.TrimSpace(in.Name), State: category.Active, Origin: category.Custom}
	if err := c.Validate(); err != nil {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	if err := s.requireParent(ctx, p, c.ID, c.ParentID, true); err != nil {
		return command.Result{}, err
	}
	if err := s.requireCategoryName(ctx, p, c, ""); err != nil {
		return command.Result{}, err
	}
	if err := s.repository.CreateCategory(ctx, c); err != nil {
		return command.Result{}, s.reject(err)
	}
	if err := s.repository.EmitEvent(ctx, "category", c.ID, c.Revision, "category.created"); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "category", ResourceID: c.ID, Revision: c.Revision}, nil
}

func (s *Service) ChangeCategory(ctx context.Context, p household.Principal, id string, expected uint64, change category.Change) (command.Result, error) {
	current, err := s.repository.Category(ctx, p, id)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	if current.Revision != expected || expected >= category.MaxRevision {
		return command.Result{}, commands.Rejection{Code: "version_conflict"}
	}
	next, err := current.Apply(change)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	if current.ParentID == "" && next.ParentID != "" {
		hasChildren, e := s.repository.CategoryHasActiveChildren(ctx, p, id)
		if e != nil {
			return command.Result{}, e
		}
		if hasChildren {
			return command.Result{}, commands.Rejection{Code: "category_has_active_children"}
		}
	}
	if current.State == category.Active && next.State == category.Archived {
		hasChildren, e := s.repository.CategoryHasActiveChildren(ctx, p, id)
		if e != nil {
			return command.Result{}, e
		}
		if hasChildren {
			return command.Result{}, commands.Rejection{Code: "category_has_active_children"}
		}
	}
	if err = s.requireParent(ctx, p, next.ID, next.ParentID, next.State == category.Active); err != nil {
		return command.Result{}, err
	}
	if next.State == category.Active {
		if err = s.requireCategoryName(ctx, p, next, next.ID); err != nil {
			return command.Result{}, err
		}
	}
	if err = s.repository.SaveCategory(ctx, next, expected); err != nil {
		return command.Result{}, s.reject(err)
	}
	if err = s.repository.EmitEvent(ctx, "category", next.ID, next.Revision, "category.changed"); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "category", ResourceID: next.ID, Revision: next.Revision}, nil
}

func (s *Service) requireParent(ctx context.Context, p household.Principal, childID, parentID string, active bool) error {
	if parentID == "" {
		return nil
	}
	if childID == parentID {
		return commands.Rejection{Code: "invalid_request"}
	}
	parent, err := s.repository.Category(ctx, p, parentID)
	if err != nil {
		return s.reject(err)
	}
	if parent.ParentID != "" {
		return commands.Rejection{Code: "invalid_request"}
	}
	if active && parent.State != category.Active {
		return commands.Rejection{Code: "category_archived"}
	}
	return nil
}

func (s *Service) requireCategoryName(ctx context.Context, p household.Principal, c category.Category, exclude string) error {
	claims, err := c.ActiveNameClaims()
	if err != nil {
		return commands.Rejection{Code: "invalid_request"}
	}
	found, err := s.repository.CategoryNamesExist(ctx, p, c.ParentID, claims, exclude)
	if err != nil {
		return err
	}
	if found {
		return commands.Rejection{Code: "invalid_request"}
	}
	return nil
}

type MerchantInput struct {
	Name    string
	Aliases []string
}

func (s *Service) CreateMerchant(ctx context.Context, p household.Principal, in MerchantInput) (command.Result, error) {
	if s.repository == nil || s.newID == nil || p.RequireHousehold(p.HouseholdID()) != nil {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	aliases := append([]string{in.Name}, in.Aliases...)
	m := category.Merchant{HouseholdID: p.HouseholdID(), ID: s.newID(), Revision: 1, Name: strings.TrimSpace(in.Name), State: category.Active}
	seen := map[string]bool{}
	for _, name := range aliases {
		a, err := category.NewConfirmedAlias(s.newID(), name)
		if err != nil {
			return command.Result{}, commands.Rejection{Code: "invalid_request"}
		}
		if seen[a.Normalized] {
			continue
		}
		seen[a.Normalized] = true
		m.Aliases = append(m.Aliases, a)
	}
	if err := m.Validate(); err != nil {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	if err := s.requireMerchantUnique(ctx, p, m, ""); err != nil {
		return command.Result{}, err
	}
	if err := s.repository.CreateMerchant(ctx, m); err != nil {
		return command.Result{}, s.reject(err)
	}
	if err := s.repository.EmitEvent(ctx, "merchant", m.ID, m.Revision, "merchant.created"); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "merchant", ResourceID: m.ID, Revision: m.Revision}, nil
}

type MerchantChange struct {
	Name              string
	State             category.State
	AliasesToAdd      []string
	AliasIDsToArchive []string
}

func (s *Service) ChangeMerchant(ctx context.Context, p household.Principal, id string, expected uint64, in MerchantChange) (command.Result, error) {
	current, err := s.repository.Merchant(ctx, p, id)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	if current.Revision != expected || expected >= category.MaxRevision {
		return command.Result{}, commands.Rejection{Code: "version_conflict"}
	}
	change := category.MerchantChange{Name: in.Name, State: in.State, AliasIDsToArchive: slices.Clone(in.AliasIDsToArchive)}
	for _, name := range in.AliasesToAdd {
		alias, e := category.NewConfirmedAlias(s.newID(), name)
		if e != nil {
			return command.Result{}, commands.Rejection{Code: "invalid_request"}
		}
		change.AliasesToAdd = append(change.AliasesToAdd, alias)
	}
	next, err := current.Apply(change)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	if next.State == category.Active {
		if err = s.requireMerchantUnique(ctx, p, next, next.ID); err != nil {
			return command.Result{}, err
		}
	}
	if err = s.repository.SaveMerchant(ctx, next, expected); err != nil {
		return command.Result{}, s.reject(err)
	}
	if err = s.repository.EmitEvent(ctx, "merchant", next.ID, next.Revision, "merchant.changed"); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "merchant", ResourceID: next.ID, Revision: next.Revision}, nil
}

func (s *Service) requireMerchantUnique(ctx context.Context, p household.Principal, m category.Merchant, exclude string) error {
	normalized, err := category.Normalize(m.Name)
	if err != nil {
		return commands.Rejection{Code: "invalid_request"}
	}
	if found, err := s.repository.MerchantNameExists(ctx, p, normalized, exclude); err != nil {
		return err
	} else if found {
		return commands.Rejection{Code: "invalid_request"}
	}
	for _, alias := range m.Aliases {
		if !alias.Confirmed() {
			continue
		}
		owner, err := s.repository.MerchantAliasOwner(ctx, p, alias.Normalized, exclude)
		if err != nil {
			return err
		}
		if owner != "" {
			return commands.Rejection{Code: "merchant_alias_conflict"}
		}
	}
	return nil
}

func (s *Service) reject(err error) error {
	switch {
	case errors.Is(err, category.ErrNotFound):
		return commands.Rejection{Code: "not_found"}
	case errors.Is(err, category.ErrCategoryArchived):
		return commands.Rejection{Code: "category_archived"}
	case errors.Is(err, category.ErrCategoryHasChild):
		return commands.Rejection{Code: "category_has_active_children"}
	case errors.Is(err, category.ErrMerchantConflict):
		return commands.Rejection{Code: "merchant_alias_conflict"}
	case errors.Is(err, category.ErrNoChange):
		return commands.Rejection{Code: "no_change"}
	case errors.Is(err, category.ErrVersionConflict):
		return commands.Rejection{Code: "version_conflict"}
	case errors.Is(err, category.ErrInvalidCategory), errors.Is(err, category.ErrCategoryConflict):
		return commands.Rejection{Code: "invalid_request"}
	default:
		return err
	}
}

func (s *Service) Categories(ctx context.Context, p household.Principal, filter category.Filter, after string, limit int) ([]category.Category, string, error) {
	if err := filter.Validate(); err != nil || limit < 1 || limit > 100 {
		return nil, "", category.ErrInvalidCategory
	}
	if filter.Search != "" {
		normalized, err := category.Normalize(filter.Search)
		if err != nil {
			return nil, "", category.ErrInvalidCategory
		}
		filter.Search = normalized
	}
	return s.repository.Categories(ctx, p, filter, after, limit)
}

func (s *Service) Merchants(ctx context.Context, p household.Principal, filter category.Filter, after string, limit int) ([]category.Merchant, string, error) {
	if err := filter.Validate(); err != nil || filter.ParentID != "" || limit < 1 || limit > 100 {
		return nil, "", category.ErrInvalidCategory
	}
	if filter.Search != "" {
		normalized, err := category.Normalize(filter.Search)
		if err != nil {
			return nil, "", category.ErrInvalidCategory
		}
		filter.Search = normalized
	}
	return s.repository.Merchants(ctx, p, filter, after, limit)
}
