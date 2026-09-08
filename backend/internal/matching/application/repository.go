package application

import (
	"context"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
)

type Cursor struct {
	At calendar.Instant
	ID string
}

type Repository interface {
	journal.DecisionRepository
	AccountTimezone(context.Context, household.Principal) (calendar.Timezone, error)
	MatchingGroup(context.Context, household.Principal, string) (matching.Group, error)
	MatchingRejected(context.Context, household.Principal, []string) (bool, error)
	MatchingForOperation(context.Context, household.Principal, string) (matching.Group, bool, error)
	SaveMatchingGroup(context.Context, matching.Group, uint64) error
	SaveMatchingDecisionGroups(context.Context, string, []matching.Group) error
	MatchingDecisionGroups(context.Context, household.Principal, string) ([]matching.Group, error)
	MatchingReferences(context.Context, household.Principal, ledger.Revision, bool, int) ([]ledger.Revision, bool, error)
	MatchingPage(context.Context, household.Principal, matching.State, Cursor, int) ([]matching.Group, *Cursor, error)
	ReleaseMatchingCarriers(context.Context, string) error
	RestoreMatchingCarriers(context.Context, string) error
}

type Service struct {
	repository Repository
	writer     journal.JournalWriter
	now        func() calendar.Instant
	newID      func() string
}

func NewService(r Repository, w journal.JournalWriter, now func() calendar.Instant, newID func() string) *Service {
	return &Service{repository: r, writer: w, now: now, newID: newID}
}
