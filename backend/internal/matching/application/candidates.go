package application

import (
	"context"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func (s *Service) withoutRejected(ctx context.Context, p household.Principal, operationID string, candidates []ledger.Revision) ([]ledger.Revision, error) {
	retained := make([]ledger.Revision, 0, len(candidates))
	for _, candidate := range candidates {
		ids := []string{operationID, candidate.OperationID}
		group, found, err := s.repository.MatchingForOperation(ctx, p, candidate.OperationID)
		if err != nil {
			return nil, err
		}
		if found {
			for _, member := range group.Members {
				ids = append(ids, member.OperationID)
			}
		}
		rejected, err := s.repository.MatchingRejected(ctx, p, ids)
		if err != nil {
			return nil, err
		}
		if !rejected {
			retained = append(retained, candidate)
		}
	}
	return retained, nil
}
