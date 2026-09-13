package application

import (
	"context"
	"errors"
	"slices"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
)

func (s *Service) update(ctx context.Context, p household.Principal, incoming ledger.Revision, expected uint64, evidence []ledger.Evidence) error {
	g, found, err := s.repository.MatchingForOperation(ctx, p, incoming.OperationID)
	if err != nil {
		return err
	}
	if !found {
		previous, exists, err := s.repository.CurrentLedgerRevision(ctx, p, incoming.OperationID)
		if err != nil {
			return err
		}
		if !exists || previous.Revision != expected {
			return matching.ErrConflict
		}
		if incoming.Correspondence != nil && (previous.Correspondence == nil || *previous.Correspondence != *incoming.Correspondence) {
			return s.discover(ctx, p, incoming, expected, evidence)
		}
		return s.writer.Append(ctx, p, incoming, expected)
	}
	before, err := s.members(ctx, p, g)
	if err != nil {
		return err
	}
	index := slices.IndexFunc(before, func(r ledger.Revision) bool { return r.OperationID == incoming.OperationID })
	if index < 0 || before[index].Revision != expected {
		return matching.ErrConflict
	}
	if before[index].SameFacts(incoming) && g.State != matching.Conflict {
		return nil
	}
	facts := slices.Clone(before)
	facts[index] = incoming
	if g.State == matching.Clarification {
		return s.saveSourceUpdate(ctx, p, g, before, facts, false, "awaiting_matching_resolution")
	}
	// Reconcile all retained source evidence together: the other side may have
	// arrived during the conflict without becoming an accepted monetary revision.
	if g.State == matching.Conflict {
		facts, err = s.sourceFacts(ctx, p, facts, incoming.OperationID)
		if err != nil {
			if errors.Is(err, ledger.ErrMatchingConflict) {
				return s.saveSourceConflict(ctx, p, g, before, "matching_conflict")
			}
			return err
		}
	}
	for i, r := range facts {
		old := before[i]
		if (old.Correspondence == nil) != (r.Correspondence == nil) || old.Correspondence != nil && *old.Correspondence != *r.Correspondence {
			return s.saveSourceConflict(ctx, p, g, before, "matching_proof_changed")
		}
	}
	updated, assigned, err := g.Assign(facts, true)
	if err != nil {
		if !errors.Is(err, matching.ErrConflict) && !errors.Is(err, matching.ErrInvalid) {
			return err
		}
		return s.saveSourceConflict(ctx, p, g, before, "matching_conflict")
	}
	return s.saveSourceUpdate(ctx, p, updated, before, assigned, g.State != updated.State, "linked_source_update")
}

func (s *Service) saveSourceConflict(ctx context.Context, p household.Principal, g matching.Group, before []ledger.Revision, reason string) error {
	g.State = matching.Conflict
	// The new revision records changed matching evidence, retaining every last-good
	// posting. The journal atomically refreshes quality, audit, review and outbox.
	if err := s.saveSourceUpdate(ctx, p, g, before, before, true, reason); err != nil {
		return err
	}
	return ledger.ErrMatchingConflict
}

func (s *Service) saveSourceUpdate(ctx context.Context, p household.Principal, g matching.Group, before, assigned []ledger.Revision, material bool, reason string) error {
	var err error
	g, err = g.Next(p.UserID(), s.now(), reason)
	if err != nil {
		return err
	}
	g.Members = nil
	changes := []ledger.Revision{}
	for _, r := range assigned {
		index := slices.IndexFunc(before, func(v ledger.Revision) bool { return v.OperationID == r.OperationID })
		if index < 0 {
			return ledger.ErrInvalidRevision
		}
		old := before[index]
		r = r.Clone()
		r.Revision = old.Revision
		if material || !old.SameFacts(r) {
			r.Revision++
			r.ActorID, r.RecordedAt, r.Reason, r.DecisionID = p.UserID(), s.now(), reason, ""
			if r.FieldVersions == nil {
				r.FieldVersions = map[ledger.Field]uint64{}
			}
			for _, f := range []ledger.Field{ledger.PrincipalField, ledger.FeesField, ledger.DateField, ledger.PayerField, ledger.MerchantField, ledger.NoteField, ledger.CategoryField, ledger.MerchantIDField, ledger.ReceiptItemsField} {
				if !old.FieldEqual(r, f) {
					r.FieldVersions[f] = r.Revision
				}
			}
			if !old.FieldEqual(r, ledger.AllocationField) {
				r.FieldVersions[ledger.AllocationField] = r.Revision
			}
			// Bank lifecycle/time refreshes the contribution without replacing the
			// association decision. Real composition changes and conflicts still do.
			if material || !old.Participation.SameCarriers(r.Participation) {
				r.FieldVersions[ledger.MatchingField] = r.Revision
			}
			changes = append(changes, r)
		}
		g.Members = append(g.Members, matching.Member{OperationID: r.OperationID, Revision: r.Revision})
	}
	if len(changes) == 0 {
		return nil
	}
	if err = s.repository.SaveMatchingGroup(ctx, g, g.Revision-1); err != nil {
		return err
	}
	if err = s.repository.ReleaseMatchingCarriers(ctx, g.ID); err != nil {
		return err
	}
	if err = s.writer.AppendBatch(ctx, p, changes); err != nil {
		return err
	}
	return s.repository.RestoreMatchingCarriers(ctx, g.ID)
}
