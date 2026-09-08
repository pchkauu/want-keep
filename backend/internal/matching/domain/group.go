package domain

import (
	"fmt"
	"slices"
	"unicode/utf8"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

var ErrInvalid = fmt.Errorf("invalid matching decision: %w", ledger.ErrInvalidRevision)
var ErrConflict = fmt.Errorf("matching decision conflicts with current facts: %w", ledger.ErrMatchingConflict)
var ErrNotFound = fmt.Errorf("matching case not found: %w", ledger.ErrNotFound)

type Kind string

const (
	Payment  Kind = "payment"
	Transfer Kind = "transfer"
	Exchange Kind = "exchange"
)

type State string

const (
	Clarification State = "clarification"
	WaitingSide   State = "waiting_side"
	Linked        State = "linked"
	Separate      State = "separate"
	Unlinked      State = "unlinked"
	Conflict      State = "conflict"
)

func (s State) Valid() bool {
	return slices.Contains([]State{Clarification, WaitingSide, Linked, Separate, Unlinked, Conflict}, s)
}

type Member struct {
	OperationID string
	Revision    uint64
}

type Candidate struct {
	Member
	Reason string
}

type Group struct {
	ID, PrimaryID, DecisionID string
	Revision                  uint64
	Kind                      Kind
	State                     State
	ActorID                   household.UserID
	At                        calendar.Instant
	Reason                    string
	Members                   []Member
	Candidates                []Candidate
	CandidatesComplete        bool
}

func (g Group) Validate() error {
	if g.ID == "" || g.PrimaryID == "" || g.Revision < 1 || g.Revision > 9007199254740991 || g.ActorID == "" || g.At.String() == "" || !utf8.ValidString(g.Reason) || utf8.RuneCountInString(g.Reason) < 1 || utf8.RuneCountInString(g.Reason) > 2000 || len(g.Members) < 1 || len(g.Members) > 100 || len(g.Candidates) > 100 {
		return ErrInvalid
	}
	if !slices.Contains([]Kind{Payment, Transfer, Exchange}, g.Kind) || !slices.Contains([]State{Clarification, WaitingSide, Linked, Separate, Unlinked, Conflict}, g.State) {
		return ErrInvalid
	}
	primary := false
	seen := map[string]bool{}
	for _, m := range g.Members {
		if m.OperationID == "" || seen[m.OperationID] || m.Revision < 1 || m.Revision > 9007199254740991 {
			return ErrInvalid
		}
		primary = primary || m.OperationID == g.PrimaryID
		seen[m.OperationID] = true
	}
	for _, c := range g.Candidates {
		if c.OperationID == "" || seen[c.OperationID] || c.Revision < 1 || c.Revision > 9007199254740991 || c.Reason == "" {
			return ErrInvalid
		}
		seen[c.OperationID] = true
	}
	if !primary {
		return ErrInvalid
	}
	return nil
}

func (g Group) Clone() Group {
	g.Members = slices.Clone(g.Members)
	g.Candidates = slices.Clone(g.Candidates)
	return g
}

func (g Group) Next(actor household.UserID, at calendar.Instant, reason string) (Group, error) {
	if g.Revision >= 9007199254740991 {
		return g, ErrConflict
	}
	g = g.Clone()
	g.Revision++
	g.ActorID, g.At, g.Reason = actor, at, reason
	return g, nil
}

// Assign preserves source facts and chooses carriers independently from the display primary.
func (g Group) Assign(facts []ledger.Revision, allowPartial bool) (Group, []ledger.Revision, error) {
	assignment := effectAssignment{group: g}
	return assignment.assign(facts, allowPartial)
}
