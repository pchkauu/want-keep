package domain

import (
	"errors"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

var (
	ErrInvalidCategory  = errors.New("invalid category")
	ErrCategoryArchived = errors.New("category is archived")
	ErrCategoryHasChild = errors.New("category has active children")
	ErrCategoryConflict = errors.New("category conflicts with catalog")
	ErrMerchantConflict = errors.New("merchant alias conflicts with catalog")
	ErrNotFound         = errors.New("catalog resource not found")
	ErrNoChange         = errors.New("catalog change has no effect")
	ErrVersionConflict  = errors.New("catalog revision conflict")
)

const MaxRevision uint64 = 9007199254740991

type State string

const (
	Active   State = "active"
	Archived State = "archived"
)

func (s State) Valid() bool { return s == Active || s == Archived }

type Origin string

const (
	Starter Origin = "starter"
	Custom  Origin = "custom"
)

func (o Origin) Valid() bool { return o == Starter || o == Custom }

type Category struct {
	HouseholdID household.HouseholdID
	ID          string
	Revision    uint64
	ParentID    string
	Key         string
	NameRU      string
	NameEN      string
	CustomName  string
	State       State
	Origin      Origin
}

func (c Category) Validate() error {
	if c.HouseholdID == "" || c.ID == "" || c.Revision < 1 || c.Revision > MaxRevision || !c.State.Valid() || !c.Origin.Valid() {
		return ErrInvalidCategory
	}
	if c.ParentID == c.ID || !validName(c.DisplayName()) {
		return ErrInvalidCategory
	}
	if c.Origin == Starter {
		if c.Key == "" || !validName(c.NameRU) || !validName(c.NameEN) {
			return ErrInvalidCategory
		}
	} else if c.Key != "" || c.NameRU != "" || c.NameEN != "" || !validName(c.CustomName) {
		return ErrInvalidCategory
	}
	return nil
}

func (c Category) DisplayName() string {
	if c.CustomName != "" {
		return c.CustomName
	}
	if c.NameRU != "" {
		return c.NameRU
	}
	return c.NameEN
}

func (c Category) Rename(name string) (Category, error) {
	name = strings.TrimSpace(name)
	if !validName(name) {
		return c, ErrInvalidCategory
	}
	if c.CustomName == name {
		return c, ErrNoChange
	}
	next := c
	next.CustomName = name
	return next.nextRevision()
}

func (c Category) RestoreDefaultName() (Category, error) {
	if c.Origin != Starter || c.CustomName == "" {
		return c, ErrNoChange
	}
	next := c
	next.CustomName = ""
	return next.nextRevision()
}

func (c Category) Reparent(parent string) (Category, error) {
	if parent == c.ID {
		return c, ErrInvalidCategory
	}
	if c.ParentID == parent {
		return c, ErrNoChange
	}
	next := c
	next.ParentID = parent
	return next.nextRevision()
}

func (c Category) ChangeState(state State) (Category, error) {
	if !state.Valid() {
		return c, ErrInvalidCategory
	}
	if c.State == state {
		return c, ErrNoChange
	}
	next := c
	next.State = state
	return next.nextRevision()
}

func (c Category) nextRevision() (Category, error) {
	if c.Revision >= MaxRevision {
		return c, ErrVersionConflict
	}
	c.Revision++
	return c, c.Validate()
}

func validName(value string) bool {
	value = strings.TrimSpace(value)
	return utf8.ValidString(value) && utf8.RuneCountInString(value) >= 1 && utf8.RuneCountInString(value) <= 200
}

func Normalize(value string) (string, error) {
	if !validName(value) {
		return "", ErrInvalidCategory
	}
	var b strings.Builder
	space := false
	for _, r := range strings.TrimSpace(value) {
		if unicode.IsSpace(r) {
			space = b.Len() > 0
			continue
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String(), nil
}

type Filter struct {
	State    State
	ParentID string
	Search   string
}

func (f Filter) Validate() error {
	if f.State != "" && !f.State.Valid() || utf8.RuneCountInString(f.Search) > 200 {
		return ErrInvalidCategory
	}
	return nil
}

type Change struct {
	NameAction string
	Name       string
	ParentSet  bool
	ParentID   string
	State      State
}

func (c Change) Validate() error {
	if !slices.Contains([]string{"", "set", "restore_default"}, c.NameAction) || c.State != "" && !c.State.Valid() {
		return ErrInvalidCategory
	}
	if c.NameAction == "set" && !validName(c.Name) || c.NameAction != "set" && c.Name != "" {
		return ErrInvalidCategory
	}
	if c.NameAction == "" && !c.ParentSet && c.State == "" {
		return ErrNoChange
	}
	return nil
}

func (c Category) Apply(change Change) (Category, error) {
	if err := change.Validate(); err != nil {
		return c, err
	}
	next := c
	changed := false
	switch change.NameAction {
	case "set":
		name := strings.TrimSpace(change.Name)
		if name != c.CustomName {
			next.CustomName = name
			changed = true
		}
	case "restore_default":
		if c.Origin != Starter {
			return c, ErrInvalidCategory
		}
		if c.CustomName != "" {
			next.CustomName = ""
			changed = true
		}
	}
	if change.ParentSet && change.ParentID != c.ParentID {
		if change.ParentID == c.ID {
			return c, ErrInvalidCategory
		}
		next.ParentID = change.ParentID
		changed = true
	}
	if change.State != "" && change.State != c.State {
		next.State = change.State
		changed = true
	}
	if !changed {
		return c, ErrNoChange
	}
	return next.nextRevision()
}
