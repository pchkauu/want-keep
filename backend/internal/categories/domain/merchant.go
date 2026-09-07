package domain

import (
	"slices"
	"strings"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

type AliasOrigin string

const (
	UserConfirmed  AliasOrigin = "user_confirmed"
	ReviewProposed AliasOrigin = "review_proposed"
)

type Alias struct {
	ID         string
	Name       string
	Normalized string
	State      State
	Origin     AliasOrigin
}

func NewConfirmedAlias(id, name string) (Alias, error) {
	normalized, err := Normalize(name)
	if err != nil || id == "" {
		return Alias{}, ErrInvalidCategory
	}
	return Alias{ID: id, Name: strings.TrimSpace(name), Normalized: normalized, State: Active, Origin: UserConfirmed}, nil
}

func (a Alias) Validate() error {
	normalized, err := Normalize(a.Name)
	if err != nil || a.ID == "" || normalized != a.Normalized || !a.State.Valid() || !slices.Contains([]AliasOrigin{UserConfirmed, ReviewProposed}, a.Origin) {
		return ErrInvalidCategory
	}
	return nil
}

func (a Alias) Confirmed() bool { return a.State == Active && a.Origin == UserConfirmed }

type Merchant struct {
	HouseholdID household.HouseholdID
	ID          string
	Revision    uint64
	Name        string
	State       State
	Aliases     []Alias
}

func (m Merchant) Validate() error {
	if m.HouseholdID == "" || m.ID == "" || m.Revision < 1 || m.Revision > MaxRevision || !m.State.Valid() || !validName(m.Name) || len(m.Aliases) > 1000 {
		return ErrInvalidCategory
	}
	ids := map[string]bool{}
	keys := map[string]bool{}
	for _, alias := range m.Aliases {
		if alias.Validate() != nil || ids[alias.ID] || keys[alias.Normalized] {
			return ErrInvalidCategory
		}
		ids[alias.ID], keys[alias.Normalized] = true, true
	}
	return nil
}

type MerchantChange struct {
	Name              string
	State             State
	AliasesToAdd      []Alias
	AliasIDsToArchive []string
}

func (c MerchantChange) Validate() error {
	if c.Name != "" && !validName(c.Name) || c.State != "" && !c.State.Valid() || len(c.AliasesToAdd)+len(c.AliasIDsToArchive) > 1000 {
		return ErrInvalidCategory
	}
	if c.Name == "" && c.State == "" && len(c.AliasesToAdd) == 0 && len(c.AliasIDsToArchive) == 0 {
		return ErrNoChange
	}
	return nil
}

func (m Merchant) Apply(c MerchantChange) (Merchant, error) {
	if err := c.Validate(); err != nil {
		return m, err
	}
	next := m
	next.Aliases = slices.Clone(m.Aliases)
	changed := false
	if c.Name != "" && c.Name != m.Name {
		next.Name = strings.TrimSpace(c.Name)
		changed = true
	}
	if c.State != "" && c.State != m.State {
		next.State = c.State
		changed = true
	}
	archive := map[string]bool{}
	for _, id := range c.AliasIDsToArchive {
		if id == "" || archive[id] {
			return m, ErrInvalidCategory
		}
		archive[id] = true
	}
	for i := range next.Aliases {
		if archive[next.Aliases[i].ID] && next.Aliases[i].State != Archived {
			next.Aliases[i].State = Archived
			changed = true
			delete(archive, next.Aliases[i].ID)
		}
	}
	if len(archive) > 0 {
		return m, ErrInvalidCategory
	}
	for _, alias := range c.AliasesToAdd {
		if alias.Validate() != nil || alias.Origin != UserConfirmed {
			return m, ErrInvalidCategory
		}
		next.Aliases = append(next.Aliases, alias)
		changed = true
	}
	if !changed {
		return m, ErrNoChange
	}
	if m.Revision >= MaxRevision {
		return m, ErrVersionConflict
	}
	next.Revision++
	return next, next.Validate()
}
