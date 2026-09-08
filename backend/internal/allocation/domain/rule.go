package domain

import (
	"errors"
	"math/big"
	"regexp"
	"slices"
	"sort"
	"unicode/utf8"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

var shareSyntax = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`)

const MaxRevision = uint64(9007199254740991)

var (
	ErrInvalidRule  = errors.New("invalid allocation rule")
	ErrRuleNotFound = errors.New("allocation rule not found")
	ErrRuleNoChange = errors.New("allocation rule has no change")
	ErrRuleConflict = errors.New("allocation rules conflict")
)

type State string

const (
	Active   State = "active"
	Archived State = "archived"
)

type Condition struct {
	MerchantID string
	CategoryID string
}

type Share struct {
	MemberID household.MembershipID
	Value    string
}

type Rule struct {
	ID          string
	HouseholdID household.HouseholdID
	Revision    uint64
	Priority    int
	State       State
	Condition   Condition
	Shares      []Share
	ActorID     household.UserID
	RecordedAt  calendar.Instant
}

func (r Rule) Validate() error {
	if r.ID == "" || r.HouseholdID == "" || r.ActorID == "" || r.RecordedAt.String() == "" || r.Revision < 1 || r.Revision > MaxRevision || r.Priority < 1 || r.Priority > 1000 || !slices.Contains([]State{Active, Archived}, r.State) || r.Condition.MerchantID == "" && r.Condition.CategoryID == "" || len(r.Shares) == 0 || len(r.Shares) > 1000 {
		return ErrInvalidRule
	}
	seen := map[household.MembershipID]bool{}
	total := new(big.Rat)
	for _, share := range r.Shares {
		if share.MemberID == "" || seen[share.MemberID] || len(share.Value) == 0 || len(share.Value) > 256 || utf8.RuneCountInString(share.Value) != len(share.Value) || !shareSyntax.MatchString(share.Value) {
			return ErrInvalidRule
		}
		value, ok := new(big.Rat).SetString(share.Value)
		if !ok || value.Sign() <= 0 {
			return ErrInvalidRule
		}
		seen[share.MemberID] = true
		total.Add(total, value)
	}
	if total.Cmp(big.NewRat(100, 1)) != 0 {
		return ErrInvalidRule
	}
	return nil
}

func (r Rule) Matches(merchantID, categoryID string) bool {
	return r.State == Active && (r.Condition.MerchantID == "" || r.Condition.MerchantID == merchantID) && (r.Condition.CategoryID == "" || r.Condition.CategoryID == categoryID)
}

func (r Rule) SameResult(other Rule) bool {
	left, right := slices.Clone(r.Shares), slices.Clone(other.Shares)
	sort.Slice(left, func(i, j int) bool { return left[i].MemberID < left[j].MemberID })
	sort.Slice(right, func(i, j int) bool { return right[i].MemberID < right[j].MemberID })
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].MemberID != right[i].MemberID {
			return false
		}
		a, aok := new(big.Rat).SetString(left[i].Value)
		b, bok := new(big.Rat).SetString(right[i].Value)
		if !aok || !bok || a.Cmp(b) != 0 {
			return false
		}
	}
	return true
}

type Change struct {
	Priority  int
	State     State
	Condition Condition
	Shares    []Share
}

func (r Rule) Apply(change Change, actor household.UserID, recordedAt calendar.Instant) (Rule, error) {
	if actor == "" || recordedAt.String() == "" || r.Revision >= MaxRevision {
		return r, ErrInvalidRule
	}
	next := r
	next.Priority = change.Priority
	next.State = change.State
	next.Condition = change.Condition
	next.Shares = slices.Clone(change.Shares)
	next.ActorID = actor
	if r.Priority == next.Priority && r.State == next.State && r.Condition == next.Condition && r.SameResult(next) {
		return r, ErrRuleNoChange
	}
	next.Revision++
	next.RecordedAt = recordedAt
	if err := next.Validate(); err != nil {
		return r, err
	}
	return next, nil
}

type Resolution struct {
	State  string
	Reason string
	Rules  []Rule
	Shares []Share
}

func Resolve(rules []Rule, merchantID, categoryID string) Resolution {
	matched := []Rule{}
	for _, rule := range rules {
		if rule.Validate() == nil && rule.Matches(merchantID, categoryID) {
			matched = append(matched, rule)
		}
	}
	if len(matched) == 0 {
		return Resolution{State: "unresolved", Reason: "no_matching_rule"}
	}
	sort.Slice(matched, func(i, j int) bool {
		if matched[i].Priority == matched[j].Priority {
			return matched[i].ID < matched[j].ID
		}
		return matched[i].Priority < matched[j].Priority
	})
	best := matched[0]
	selected := []Rule{best}
	for _, candidate := range matched[1:] {
		if candidate.Priority != best.Priority {
			break
		}
		if !best.SameResult(candidate) {
			return Resolution{State: "unresolved", Reason: "rule_conflict", Rules: append(selected, candidate)}
		}
		selected = append(selected, candidate)
	}
	return Resolution{State: "resolved", Rules: selected, Shares: slices.Clone(best.Shares)}
}
