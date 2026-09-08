package application

import (
	"context"
	"slices"
	"sort"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
)

func (s *Service) undoMatching(ctx context.Context, p household.Principal, d ledger.Decision, next []ledger.Revision) error {
	prior, err := s.repository.MatchingDecisionGroups(ctx, p, d.UndoOf)
	if err != nil {
		return err
	}
	basis := map[string]matching.Group{}
	for _, g := range prior {
		basis[g.ID] = g
	}
	groups := map[string]matching.Group{}
	restored := map[string][]ledger.Revision{}
	for _, r := range next {
		g, found, err := s.repository.MatchingForOperation(ctx, p, r.OperationID)
		if err != nil {
			return err
		}
		if found {
			groups[g.ID] = g
		}
		if id := r.Participation.GroupID; id != "" {
			if _, known := basis[id]; !known {
				return matching.ErrConflict
			}
			if _, found := groups[id]; !found {
				g, err = s.repository.MatchingGroup(ctx, p, id)
				if err != nil {
					return err
				}
				groups[id] = g
			}
			restored[id] = append(restored[id], r)
		}
	}
	ids := []string{}
	for id := range groups {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		g := groups[id]
		if g.State != matching.Separate && g.State != matching.Unlinked {
			for _, m := range g.Members {
				if !slices.ContainsFunc(next, func(v ledger.Revision) bool { return v.OperationID == m.OperationID }) {
					return matching.ErrConflict
				}
			}
		}
		g, err := g.Next(p.UserID(), s.now(), d.Reason)
		if err != nil {
			return err
		}
		g.State = matching.Unlinked
		g.DecisionID = d.ID
		if err = s.repository.SaveMatchingGroup(ctx, g, g.Revision-1); err != nil {
			return err
		}
		if err = s.repository.ReleaseMatchingCarriers(ctx, id); err != nil {
			return err
		}
		groups[id] = g
	}
	next = slices.Clone(next)
	// Automatic matching may stage new evidence as waiting inside the existing
	// movement's case. Undo restores those two independent questions separately.
	for _, id := range ids {
		facts := restored[id]
		waiting, accepted := []ledger.Revision{}, []ledger.Revision{}
		for _, r := range facts {
			if r.Participation.State == "waiting" {
				waiting = append(waiting, r)
			} else {
				accepted = append(accepted, r)
			}
		}
		if len(waiting) == 0 || len(accepted) == 0 {
			continue
		}
		restored[id] = accepted
		g := groups[id].Clone()
		if !slices.ContainsFunc(accepted, func(r ledger.Revision) bool { return r.OperationID == g.PrimaryID }) {
			g.PrimaryID = accepted[0].OperationID
			groups[id] = g
		}
		g.ID, g.PrimaryID, g.Revision = s.newID(), waiting[0].OperationID, 0
		for i := range waiting {
			waiting[i] = waiting[i].Clone()
			waiting[i].Participation.GroupID = g.ID
			for j := range next {
				if next[j].OperationID == waiting[i].OperationID {
					next[j] = waiting[i]
				}
			}
		}
		groups[g.ID], restored[g.ID] = g, waiting
		if original, ok := basis[id]; ok {
			basis[g.ID] = original
		}
		ids = append(ids, g.ID)
	}
	for _, id := range ids {
		facts := restored[id]
		if len(facts) == 0 {
			continue
		}
		g := groups[id]
		if original, ok := basis[id]; ok {
			g.Candidates = slices.Clone(original.Candidates)
			g.CandidatesComplete = original.CandidatesComplete
			g.Kind = original.Kind
		}
		var err error
		if facts[0].Participation.State == "waiting" {
			g.State = matching.Clarification
			g.PrimaryID = facts[0].OperationID
			g.Members = nil
			candidates := g.Candidates
			g.Candidates = nil
			for _, r := range facts {
				if r.Participation.State != "waiting" {
					return matching.ErrConflict
				}
				g.Members = append(g.Members, matching.Member{OperationID: r.OperationID, Revision: r.Revision})
			}
			for _, r := range next {
				if r.Participation.GroupID != id {
					g.Candidates = append(g.Candidates, matching.Candidate{Member: matching.Member{OperationID: r.OperationID, Revision: r.Revision}, Reason: "restored_matching_candidate"})
				}
			}
			for _, c := range candidates {
				if slices.ContainsFunc(facts, func(r ledger.Revision) bool { return r.OperationID == c.OperationID }) || slices.ContainsFunc(g.Candidates, func(v matching.Candidate) bool { return v.OperationID == c.OperationID }) {
					continue
				}
				current, found, err := s.repository.CurrentLedgerRevision(ctx, p, c.OperationID)
				if err != nil {
					return err
				}
				if !found {
					return matching.ErrConflict
				}
				c.Revision = current.Revision
				g.Candidates = append(g.Candidates, c)
			}
			if len(g.Candidates) > 100 {
				g.Candidates = g.Candidates[:100]
				g.CandidatesComplete = false
			}
		} else {
			if !slices.ContainsFunc(facts, func(v ledger.Revision) bool { return v.OperationID == g.PrimaryID }) {
				return matching.ErrConflict
			}
			var assigned []ledger.Revision
			g, assigned, err = g.Assign(facts, true)
			if err != nil {
				return err
			}
			for _, r := range assigned {
				for i := range next {
					if r.OperationID == next[i].OperationID {
						next[i] = r
					}
				}
			}
		}
		g, err = g.Next(p.UserID(), s.now(), d.Reason)
		if err != nil {
			return err
		}
		g.DecisionID = d.ID
		if err = s.repository.SaveMatchingGroup(ctx, g, g.Revision-1); err != nil {
			return err
		}
	}
	if err := s.writer.AppendDecision(ctx, p, d, next); err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.repository.RestoreMatchingCarriers(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
