package application

import (
	"context"
	"errors"
	"slices"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
)

func (s *Service) sourceFacts(ctx context.Context, p household.Principal, facts []ledger.Revision, incomingID string) ([]ledger.Revision, error) {
	result := slices.Clone(facts)
	for i, r := range facts {
		if r.OperationID == incomingID {
			continue
		}
		source, err := s.repository.LatestSourceFact(ctx, p, r.OperationID)
		if err != nil {
			if errors.Is(err, ledger.ErrSourceAmbiguous) {
				return nil, matching.ErrConflict
			}
			return nil, err
		}
		if source == nil {
			continue
		}
		next, _, err := r.MergeSource(*source)
		if err != nil {
			return nil, err
		}
		next, err = next.InTimezone(r.Timezone)
		if err != nil {
			return nil, err
		}
		validation := next.Clone()
		validation.Participation = ledger.Participation{}
		if validation.Validate() != nil || r.State.RequireNext(next.State) != nil {
			return nil, matching.ErrConflict
		}
		result[i] = next
	}
	return result, nil
}

func (s *Service) requireAcceptedSources(ctx context.Context, p household.Principal, facts []ledger.Revision) error {
	observed, err := s.sourceFacts(ctx, p, facts, "")
	if err != nil {
		return err
	}
	for i, r := range facts {
		if !r.SameFacts(observed[i]) {
			return matching.ErrConflict
		}
	}
	return nil
}
