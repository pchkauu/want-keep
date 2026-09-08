package application

import (
	"context"
	"slices"
	"sort"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
)

func (s *Service) AppendDecision(ctx context.Context, p household.Principal, d ledger.Decision, next []ledger.Revision) error {
	if d.Kind == "undo" {
		original, err := s.repository.Decision(ctx, p, d.UndoOf)
		if err != nil {
			return err
		}
		if original.Kind == "matching" {
			return s.undoMatching(ctx, p, d, next)
		}
	}
	groups := map[string]matching.Group{}
	for _, r := range next {
		g, found, err := s.repository.MatchingForOperation(ctx, p, r.OperationID)
		if err != nil {
			return err
		}
		if !found && r.Participation.GroupID != "" {
			g, err = s.repository.MatchingGroup(ctx, p, r.Participation.GroupID)
			if err != nil {
				return err
			}
			found = true
		}
		if found {
			groups[g.ID] = g
		}
	}
	next = slices.Clone(next)
	zone, err := s.repository.AccountTimezone(ctx, p)
	if err != nil {
		return err
	}
	for i, r := range next {
		if r.Timezone.String() != "" && r.Timezone != zone {
			return ledger.ErrInvalidRevision
		}
		next[i], err = r.InTimezone(zone)
		if err != nil {
			return err
		}
	}
	ids := []string{}
	for id := range groups {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		g := groups[id]
		facts, err := s.members(ctx, p, g)
		if err != nil {
			return err
		}
		financial := false
		for i, r := range facts {
			for _, v := range next {
				if r.OperationID == v.OperationID {
					financial = financial || !r.FieldEqual(v, ledger.PrincipalField) || !r.FieldEqual(v, ledger.FeesField) || !r.FieldEqual(v, ledger.AccountingField) || !r.Participation.SameCarriers(v.Participation)
					facts[i] = v
				}
			}
		}
		if financial {
			for _, r := range facts {
				if !slices.ContainsFunc(next, func(v ledger.Revision) bool { return v.OperationID == r.OperationID }) {
					return matching.ErrConflict
				}
			}
		}
		if g.State == matching.Linked || g.State == matching.WaitingSide || g.State == matching.Conflict {
			conflicted := g.State == matching.Conflict
			if conflicted && financial {
				if err := s.requireAcceptedSources(ctx, p, facts); err != nil {
					return err
				}
			}
			updated, assigned, err := g.Assign(facts, true)
			if err != nil {
				return err
			}
			g = updated
			if conflicted && !financial {
				g.State = matching.Conflict
			}
			for _, r := range assigned {
				index := slices.IndexFunc(next, func(v ledger.Revision) bool { return v.OperationID == r.OperationID })
				if index >= 0 {
					if !next[index].FieldEqual(r, ledger.MatchingField) {
						field := ledger.ContributionField
						if !next[index].Participation.SameCarriers(r.Participation) {
							field = ledger.MatchingField
						}
						for i, e := range d.Entries {
							if e.OperationID == r.OperationID && !slices.Contains(e.Fields, field) {
								d.Entries[i].Fields = append(d.Entries[i].Fields, field)
							}
						}
						r.FieldVersions[field] = r.Revision
					}
					next[index] = r
				} else {
					old, _, err := s.repository.CurrentLedgerRevision(ctx, p, r.OperationID)
					if err != nil {
						return err
					}
					if !r.FieldEqual(old, ledger.MatchingField) {
						field := ledger.ContributionField
						if !old.Participation.SameCarriers(r.Participation) {
							field = ledger.MatchingField
						}
						d.Entries = append(d.Entries, ledger.DecisionEntry{OperationID: r.OperationID, Before: r.Revision, After: r.Revision + 1, Fields: []ledger.Field{field}})
						r = r.WithDecision(d, []ledger.Field{field})
						next = append(next, r)
					}
				}
			}
			g.Members = nil
			for _, r := range assigned {
				for _, n := range next {
					if r.OperationID == n.OperationID {
						r = n
						break
					}
				}
				g.Members = append(g.Members, matching.Member{OperationID: r.OperationID, Revision: r.Revision})
			}
		} else {
			g.Members = nil
			for _, r := range facts {
				g.Members = append(g.Members, matching.Member{OperationID: r.OperationID, Revision: r.Revision})
			}
		}
		old := g.Revision
		g, err = g.Next(p.UserID(), s.now(), d.Reason)
		if err != nil {
			return err
		}
		g.DecisionID = d.ID
		if err = s.repository.SaveMatchingGroup(ctx, g, old); err != nil {
			return err
		}
		if err = s.repository.ReleaseMatchingCarriers(ctx, g.ID); err != nil {
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
