package application

import (
	"context"
	"errors"
	"slices"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
)

func (s *Service) Account(ctx context.Context, p household.Principal, id string) (account.Account, error) {
	return s.writer.Account(ctx, p, id)
}

// Append is the coordinated ingress used by manual entry and fenced import.
// The underlying journal remains the sole writer of monetary history.
func (s *Service) Append(ctx context.Context, p household.Principal, r ledger.Revision, expected uint64) error {
	return s.append(ctx, p, r, expected, nil)
}
func (s *Service) AppendSource(ctx context.Context, p household.Principal, r ledger.Revision, expected uint64, evidence ledger.Evidence) error {
	if evidence.Kind != "source" || evidence.Validate() != nil {
		return ledger.ErrInvalidRevision
	}
	return s.append(ctx, p, r, expected, []ledger.Evidence{evidence})
}
func (s *Service) append(ctx context.Context, p household.Principal, r ledger.Revision, expected uint64, evidence []ledger.Evidence) error {
	if r.ActorID != p.UserID() {
		return household.ErrForbidden
	}
	zone, err := s.repository.AccountTimezone(ctx, p)
	if err != nil {
		return err
	}
	if r.Timezone.String() != "" && r.Timezone != zone {
		return ledger.ErrInvalidRevision
	}
	r, err = r.InTimezone(zone)
	if err != nil {
		return err
	}
	for i, posting := range r.Postings {
		if posting.Funding == "" {
			a, err := s.writer.Account(ctx, p, posting.AccountID)
			if err != nil {
				return err
			}
			r.Postings[i].Funding = ledger.OwnFunds
			if a.Product == "credit_card" {
				r.Postings[i].Funding = ledger.UnknownFunds
			}
		}
	}
	validation := r.Clone()
	if expected > 0 {
		validation.Participation = ledger.Participation{}
	}
	if validation.Validate() != nil {
		return ledger.ErrInvalidRevision
	}
	if expected > 0 {
		return s.update(ctx, p, r, expected)
	}
	if r.Participation.GroupID != "" {
		return matching.ErrInvalid
	}
	if r.Correspondence != nil && r.Correspondence.Kind != "payment" {
		for _, id := range []string{r.Correspondence.FromAccountID, r.Correspondence.ToAccountID} {
			if _, err := s.writer.Account(ctx, p, id); err != nil {
				return err
			}
		}
		for _, posting := range r.Postings {
			if posting.Role == ledger.Principal && (posting.Money.Sign() < 0 && posting.AccountID != r.Correspondence.FromAccountID || posting.Money.Sign() > 0 && posting.AccountID != r.Correspondence.ToAccountID) {
				return matching.ErrInvalid
			}
		}
	}
	exact, complete, err := s.repository.MatchingReferences(ctx, p, r, true, 99)
	if err != nil {
		return err
	}
	if len(exact) > 0 && complete {
		return s.acceptProven(ctx, p, r, exact, evidence)
	}
	possible, possibleComplete, err := s.repository.MatchingReferences(ctx, p, r, false, 100)
	if err != nil {
		return err
	}
	if len(exact) > 0 {
		possible, possibleComplete = exact, false
	}
	if len(possible) > 0 || !possibleComplete {
		return s.hold(ctx, p, r, possible, possibleComplete)
	}
	if r.Correspondence != nil && r.Correspondence.Kind != "payment" {
		g := s.newGroup(p, r, matching.Kind(r.Correspondence.Kind), matching.WaitingSide, "confirmed_internal_side")
		g, facts, err := g.Assign([]ledger.Revision{r}, true)
		if err != nil {
			return err
		}
		if err = s.repository.SaveMatchingGroup(ctx, g, 0); err != nil {
			return err
		}
		return s.writer.Append(ctx, p, facts[0], 0)
	}
	return s.writer.Append(ctx, p, r, expected)
}

func (s *Service) newGroup(p household.Principal, r ledger.Revision, kind matching.Kind, state matching.State, reason string) matching.Group {
	return matching.Group{ID: s.newID(), PrimaryID: r.OperationID, Revision: 1, Kind: kind, State: state, ActorID: p.UserID(), At: s.now(), Reason: reason, Members: []matching.Member{{OperationID: r.OperationID, Revision: r.Revision}}, CandidatesComplete: true}
}

func (s *Service) hold(ctx context.Context, p household.Principal, r ledger.Revision, candidates []ledger.Revision, complete bool) error {
	kind := matching.Payment
	if r.Correspondence != nil {
		kind = matching.Kind(r.Correspondence.Kind)
	}
	g := s.newGroup(p, r, kind, matching.Clarification, "matching_unresolved")
	g.CandidatesComplete = complete
	for _, v := range candidates {
		g.Candidates = append(g.Candidates, matching.Candidate{Member: matching.Member{OperationID: v.OperationID, Revision: v.Revision}, Reason: "possible_same_payment"})
	}
	if err := s.repository.SaveMatchingGroup(ctx, g, 0); err != nil {
		return err
	}
	r.Participation = ledger.Participation{GroupID: g.ID, Kind: ledger.ParticipationKind(kind), State: "waiting"}
	return s.writer.Append(ctx, p, r, 0)
}

func (s *Service) acceptProven(ctx context.Context, p household.Principal, r ledger.Revision, existing []ledger.Revision, evidence []ledger.Evidence) error {
	ids := []string{}
	for _, v := range existing {
		ids = append(ids, v.OperationID)
	}
	rejected, err := s.repository.MatchingRejected(ctx, p, ids)
	if err != nil {
		return err
	}
	if rejected {
		return s.hold(ctx, p, r, existing, true)
	}
	// Existing user protections remain independent; automatic linking does not
	// overwrite contradictory confirmed values or hide competing fee evidence.
	for _, v := range existing {
		if v.Correspondence == nil || r.Correspondence == nil || *v.Correspondence != *r.Correspondence {
			return s.hold(ctx, p, r, existing, true)
		}
		for _, posting := range v.Postings {
			if posting.Role == ledger.Fee && posting.FeeID == "" {
				return s.hold(ctx, p, r, existing, true)
			}
		}
	}
	for _, posting := range r.Postings {
		if posting.Role == ledger.Fee && posting.FeeID == "" {
			return s.hold(ctx, p, r, existing, true)
		}
	}
	g, found, err := s.repository.MatchingForOperation(ctx, p, existing[0].OperationID)
	if err != nil {
		return err
	}
	if !found {
		g = s.newGroup(p, r, matching.Kind(r.Correspondence.Kind), matching.Clarification, "proven_correspondence")
		if err = s.repository.SaveMatchingGroup(ctx, g, 0); err != nil {
			return err
		}
	} else if g.State != matching.Linked && g.State != matching.WaitingSide {
		return s.hold(ctx, p, r, existing, true)
	}
	facts := append(slices.Clone(existing), r)
	if found {
		facts, err = s.members(ctx, p, g)
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, f := range facts {
			seen[f.OperationID] = true
		}
		for _, f := range existing {
			if !seen[f.OperationID] {
				facts = append(facts, f)
				seen[f.OperationID] = true
			}
		}
		facts = append(facts, r)
	}
	proposal := g.Clone()
	if !found {
		proposal.PrimaryID = existing[0].OperationID
	}
	if _, _, err = proposal.Assign(facts, true); err != nil {
		if found {
			return s.hold(ctx, p, r, existing, true)
		}
		g.Candidates = nil
		for _, v := range existing {
			g.Candidates = append(g.Candidates, matching.Candidate{Member: matching.Member{OperationID: v.OperationID, Revision: v.Revision}, Reason: "conflicting_correspondence"})
		}
		g, err = g.Next(p.UserID(), s.now(), "conflicting_correspondence")
		if err != nil {
			return err
		}
		if err = s.repository.SaveMatchingGroup(ctx, g, g.Revision-1); err != nil {
			return err
		}
		r.Participation = ledger.Participation{GroupID: g.ID, Kind: ledger.ParticipationKind(g.Kind), State: "waiting"}
		return s.writer.Append(ctx, p, r, 0)
	}
	r.Participation = ledger.Participation{GroupID: g.ID, Kind: ledger.ParticipationKind(g.Kind), State: "waiting"}
	if err = s.writer.Append(ctx, p, r, 0); err != nil {
		return err
	}
	_, err = s.link(ctx, p, proposal, facts, "proven_correspondence", true, evidence)
	return err
}

func (s *Service) members(ctx context.Context, p household.Principal, g matching.Group) ([]ledger.Revision, error) {
	out := make([]ledger.Revision, 0, len(g.Members))
	for _, m := range g.Members {
		r, found, err := s.repository.CurrentLedgerRevision(ctx, p, m.OperationID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, matching.ErrNotFound
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Service) update(ctx context.Context, p household.Principal, r ledger.Revision, expected uint64) error {
	g, found, err := s.repository.MatchingForOperation(ctx, p, r.OperationID)
	if err != nil {
		return err
	}
	if !found {
		return s.writer.Append(ctx, p, r, expected)
	}
	facts, err := s.members(ctx, p, g)
	if err != nil {
		return err
	}
	for i, v := range facts {
		if v.OperationID == r.OperationID {
			if v.Revision != expected {
				return matching.ErrConflict
			}
			if (v.Correspondence == nil) != (r.Correspondence == nil) || v.Correspondence != nil && *v.Correspondence != *r.Correspondence {
				g, err = g.Next(p.UserID(), s.now(), "matching_proof_changed")
				if err != nil {
					return err
				}
				g.State = matching.Conflict
				if err = s.repository.SaveMatchingGroup(ctx, g, g.Revision-1); err != nil {
					return err
				}
				return ledger.ErrMatchingConflict
			}
			facts[i] = r
		}
	}
	if g.State == matching.Clarification {
		g, err = g.Next(p.UserID(), s.now(), "awaiting_matching_resolution")
		if err != nil {
			return err
		}
		g.Members = nil
		for _, v := range facts {
			g.Members = append(g.Members, matching.Member{OperationID: v.OperationID, Revision: v.Revision})
		}
		if err = s.repository.SaveMatchingGroup(ctx, g, g.Revision-1); err != nil {
			return err
		}
		return s.writer.Append(ctx, p, r, expected)
	}
	updated, assigned, err := g.Assign(facts, true)
	if err != nil {
		if !errors.Is(err, matching.ErrConflict) && !errors.Is(err, matching.ErrInvalid) {
			return err
		}
		g, err = g.Next(p.UserID(), s.now(), "matching_conflict")
		if err != nil {
			return err
		}
		g.State = matching.Conflict
		if err = s.repository.SaveMatchingGroup(ctx, g, g.Revision-1); err != nil {
			return err
		}
		return ledger.ErrMatchingConflict
	}
	return s.saveSourceUpdate(ctx, p, updated, facts, assigned, r.OperationID, expected)
}

func (s *Service) saveSourceUpdate(ctx context.Context, p household.Principal, g matching.Group, before, assigned []ledger.Revision, sourceID string, expected uint64) error {
	var err error
	g, err = g.Next(p.UserID(), s.now(), "linked_source_update")
	if err != nil {
		return err
	}
	g.Members = nil
	changes := []ledger.Revision{}
	for i, r := range assigned {
		var old ledger.Revision
		for _, v := range before {
			if v.OperationID == r.OperationID {
				old = v
				break
			}
		}
		if r.OperationID != sourceID && !r.FieldEqual(old, ledger.MatchingField) {
			r.Revision++
			r.ActorID = p.UserID()
			r.Reason = "linked_source_update"
			r.RecordedAt = s.now()
			r.DecisionID = ""
			assigned[i] = r
		}
		g.Members = append(g.Members, matching.Member{OperationID: r.OperationID, Revision: r.Revision})
		if r.OperationID == sourceID || r.Revision != old.Revision {
			changes = append(changes, r)
		}
	}
	if err = s.repository.SaveMatchingGroup(ctx, g, g.Revision-1); err != nil {
		return err
	}
	if err = s.repository.ReleaseMatchingCarriers(ctx, g.ID); err != nil {
		return err
	}
	// Every member is rewritten only when its effective participation changed;
	// unchanged carrier claims are restored by the repository after this batch.
	for _, r := range changes {
		want := r.Revision - 1
		if r.OperationID == sourceID {
			want = expected
		}
		if err = s.writer.Append(ctx, p, r, want); err != nil {
			return err
		}
	}
	return s.repository.RestoreMatchingCarriers(ctx, g.ID)
}
