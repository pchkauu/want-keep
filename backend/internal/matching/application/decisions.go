package application

import (
	"context"
	"errors"
	"slices"
	"sort"

	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
)

type LinkInput struct {
	GroupID          string
	ExpectedRevision uint64
	Kind             matching.Kind
	PrimaryID        string
	Members          []matching.Member
	Reason           string
}

func (s *Service) Link(ctx context.Context, p household.Principal, in LinkInput) (result command.Result, err error) {
	defer func() { err = s.reject(err) }()
	if len(in.Members) < 2 || len(in.Members) > 100 {
		return command.Result{}, matching.ErrInvalid
	}
	facts, err := s.expected(ctx, p, in.Members)
	if err != nil {
		return command.Result{}, err
	}
	groups := map[string]matching.Group{}
	for _, r := range facts {
		existing, found, e := s.repository.MatchingForOperation(ctx, p, r.OperationID)
		if e != nil {
			return result, e
		}
		if found {
			groups[existing.ID] = existing
		}
	}
	var g matching.Group
	if in.GroupID != "" {
		current, e := s.repository.MatchingGroup(ctx, p, in.GroupID)
		if e != nil {
			return result, e
		}
		if current.Revision != in.ExpectedRevision {
			return result, command.ErrVersionConflict
		}
		if _, ok := groups[current.ID]; !ok {
			return result, matching.ErrConflict
		}
		g = current
	}
	ids := []string{}
	for id := range groups {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		candidate := groups[id]
		if candidate.State == matching.Separate || candidate.State == matching.Unlinked {
			return result, matching.ErrConflict
		}
		for _, m := range candidate.Members {
			if !slices.ContainsFunc(in.Members, func(v matching.Member) bool { return v.OperationID == m.OperationID }) {
				return result, matching.ErrConflict
			}
		}
		if candidate.State == matching.Linked {
			if g.ID != "" && g.State == matching.Linked && g.ID != candidate.ID {
				return result, matching.ErrConflict
			}
			g = candidate
		} else if g.ID == "" {
			g = candidate
		}
	}
	if g.ID == "" {
		g = s.newGroup(p, facts[0], in.Kind, matching.Clarification, in.Reason)
		g.PrimaryID = in.PrimaryID
		g.Members = slices.Clone(in.Members)
		if err = s.repository.SaveMatchingGroup(ctx, g, 0); err != nil {
			return result, err
		}
	} else if g.State == matching.Clarification {
		g.PrimaryID = in.PrimaryID
		g.Kind = in.Kind
	} else if g.Kind != in.Kind {
		return result, matching.ErrConflict
	}
	if !slices.ContainsFunc(in.Members, func(v matching.Member) bool { return v.OperationID == in.PrimaryID }) {
		return result, matching.ErrInvalid
	}
	if g.State == matching.Conflict {
		if err = s.requireAcceptedSources(ctx, p, facts); err != nil {
			return result, err
		}
	}
	if _, _, err = g.Assign(facts, false); err != nil {
		return result, err
	}
	for _, id := range ids {
		if id != g.ID {
			old := groups[id]
			old, err = old.Next(p.UserID(), s.now(), "matching_cases_combined")
			if err != nil {
				return result, err
			}
			old.State = matching.Unlinked
			old.DecisionID = ""
			if err = s.repository.SaveMatchingGroup(ctx, old, old.Revision-1); err != nil {
				return result, err
			}
			if err = s.repository.ReleaseMatchingCarriers(ctx, id); err != nil {
				return result, err
			}
		}
	}

	return s.link(ctx, p, g, facts, in.Reason, false, nil)
}

func (s *Service) expected(ctx context.Context, p household.Principal, members []matching.Member) ([]ledger.Revision, error) {
	seen := map[string]bool{}
	facts := []ledger.Revision{}
	members = slices.Clone(members)
	sort.Slice(members, func(i, j int) bool { return members[i].OperationID < members[j].OperationID })
	for _, m := range members {
		if seen[m.OperationID] || m.Revision < 1 {
			return nil, matching.ErrInvalid
		}
		seen[m.OperationID] = true
		r, found, err := s.repository.CurrentLedgerRevision(ctx, p, m.OperationID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, matching.ErrNotFound
		}
		if r.Revision != m.Revision || r.Revision >= command.MaxRevision {
			return nil, command.ErrVersionConflict
		}
		facts = append(facts, r)
	}
	return facts, nil
}

func (s *Service) link(ctx context.Context, p household.Principal, g matching.Group, facts []ledger.Revision, reason string, automatic bool, evidence []ledger.Evidence) (command.Result, error) {
	assigned, revisions, err := g.Assign(facts, automatic)
	if err != nil {
		return command.Result{}, err
	}
	g, err = assigned.Next(p.UserID(), s.now(), reason)
	if err != nil {
		return command.Result{}, err
	}
	d := ledger.Decision{ID: s.newID(), Kind: "matching", Evidence: evidence, Reason: reason, ActorID: p.UserID(), At: s.now()}
	next := []ledger.Revision{}
	g.Members = nil
	changed := false
	for _, r := range revisions {
		var before ledger.Revision
		for _, f := range facts {
			if f.OperationID == r.OperationID {
				before = f
				break
			}
		}
		changed = changed || !r.FieldEqual(before, ledger.MatchingField)
		d.Entries = append(d.Entries, ledger.DecisionEntry{OperationID: r.OperationID, Before: r.Revision, After: r.Revision + 1, Fields: []ledger.Field{ledger.MatchingField}})
		r = r.WithDecision(d, []ledger.Field{ledger.MatchingField})
		next = append(next, r)
		g.Members = append(g.Members, matching.Member{OperationID: r.OperationID, Revision: r.Revision})
	}
	if !changed {
		return command.Result{}, ledger.ErrNoChange
	}
	g.DecisionID = d.ID
	if err = s.repository.SaveMatchingGroup(ctx, g, g.Revision-1); err != nil {
		return command.Result{}, err
	}
	if err = s.repository.ReleaseMatchingCarriers(ctx, g.ID); err != nil {
		return command.Result{}, err
	}
	if err = s.writer.AppendDecision(ctx, p, d, next); err != nil {
		return command.Result{}, err
	}
	for _, r := range next {
		if r.OperationID == g.PrimaryID {
			return command.Result{ResourceType: "transaction", ResourceID: r.OperationID, Revision: r.Revision}, nil
		}
	}
	return command.Result{}, matching.ErrInvalid
}

func (s *Service) Separate(ctx context.Context, p household.Principal, id string, revision uint64, members []matching.Member, reason string) (result command.Result, err error) {
	defer func() { err = s.reject(err) }()
	g, err := s.repository.MatchingGroup(ctx, p, id)
	if err != nil {
		return command.Result{}, err
	}
	if g.Revision != revision {
		return command.Result{}, command.ErrVersionConflict
	}
	if g.State != matching.Clarification || len(g.Members) != 1 || len(members) != 1 || members[0].OperationID != g.Members[0].OperationID {
		return command.Result{}, matching.ErrConflict
	}
	facts, err := s.expected(ctx, p, members)
	if err != nil {
		return command.Result{}, err
	}
	g, err = g.Next(p.UserID(), s.now(), reason)
	if err != nil {
		return command.Result{}, err
	}
	g.State = matching.Separate
	d := ledger.Decision{ID: s.newID(), Kind: "matching", ActorID: p.UserID(), At: s.now(), Reason: reason}
	r := facts[0].Clone()
	r.Participation = ledger.Participation{}
	d.Entries = []ledger.DecisionEntry{{OperationID: r.OperationID, Before: r.Revision, After: r.Revision + 1, Fields: []ledger.Field{ledger.MatchingField}}}
	r = r.WithDecision(d, []ledger.Field{ledger.MatchingField})
	g.DecisionID = d.ID
	g.Members = []matching.Member{{OperationID: r.OperationID, Revision: r.Revision}}
	if err = s.repository.SaveMatchingGroup(ctx, g, g.Revision-1); err != nil {
		return command.Result{}, err
	}
	if err = s.writer.AppendDecision(ctx, p, d, []ledger.Revision{r}); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "transaction", ResourceID: r.OperationID, Revision: r.Revision}, nil
}

func (s *Service) reject(err error) error {
	switch {
	case errors.Is(err, ledger.ErrMatchingConflict):
		return commands.Rejection{Code: "matching_conflict"}
	case errors.Is(err, ledger.ErrInvalidRevision):
		return commands.Rejection{Code: "invalid_transaction"}
	case errors.Is(err, ledger.ErrNotFound):
		return commands.Rejection{Code: "not_found"}
	case errors.Is(err, command.ErrVersionConflict):
		return commands.Rejection{Code: "version_conflict"}
	case errors.Is(err, ledger.ErrNoChange):
		return commands.Rejection{Code: "no_change"}
	}
	return err
}
