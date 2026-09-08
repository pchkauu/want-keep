package application

import (
	"context"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
)

type View struct {
	Group    matching.Group
	Evidence map[string][]ledger.Evidence
}

func (s *Service) Read(ctx context.Context, p household.Principal, id string) (View, error) {
	g, err := s.repository.MatchingGroup(ctx, p, id)
	if err != nil {
		return View{}, err
	}
	return s.view(ctx, p, g)
}
func (s *Service) List(ctx context.Context, p household.Principal, state matching.State, c Cursor, limit int) ([]View, *Cursor, error) {
	if state != "" && !state.Valid() {
		return nil, nil, matching.ErrInvalid
	}
	groups, next, err := s.repository.MatchingPage(ctx, p, state, c, limit)
	if err != nil {
		return nil, nil, err
	}
	result := []View{}
	for _, g := range groups {
		v, err := s.view(ctx, p, g)
		if err != nil {
			return nil, nil, err
		}
		result = append(result, v)
	}
	return result, next, nil
}
func (s *Service) view(ctx context.Context, p household.Principal, g matching.Group) (View, error) {
	v := View{Group: g, Evidence: map[string][]ledger.Evidence{}}
	for _, m := range g.Members {
		refs, err := s.repository.RevisionEvidence(ctx, p, m.OperationID, m.Revision)
		if err != nil {
			return View{}, err
		}
		v.Evidence[m.OperationID] = refs
	}
	return v, nil
}
