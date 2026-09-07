package application

import (
	"context"
	"unicode/utf8"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type Filter struct {
	AccountID, CategoryID, MerchantID string
	Search, ItemSearch                string
	From, To                          calendar.Date
	Type                              ledger.Type
	State                             ledger.State
}

func (f Filter) Validate() error {
	if utf8.RuneCountInString(f.Search) > 200 || utf8.RuneCountInString(f.ItemSearch) > 200 || f.Type != "" && !f.Type.Valid() || f.State != "" && !f.State.Valid() || f.From.String() != "" && f.To.String() != "" && f.From.String() > f.To.String() {
		return ledger.ErrInvalidRevision
	}
	return nil
}

type Cursor struct {
	At calendar.Instant
	ID string
}
type SourceReference struct {
	Key          ledger.SourceKey
	Revision     uint64
	ConnectionID string
}
type View struct {
	Revision    ledger.Revision
	Sources     []SourceReference
	Coverage    reporting.Coverage
	SourceFacts []SourceFact
	Review      *ReviewResult
}
type SourceFact struct {
	SourceID string
	Revision uint64
	Fact     ledger.Revision
	Conflict string
}
type QueryRepository interface {
	HistoryRepository
	ReviewRepository
	TransactionCoverage(context.Context, household.Principal) (reporting.Coverage, error)
	CurrentLedgerRevision(context.Context, household.Principal, string) (ledger.Revision, bool, error)
	TransactionReferences(context.Context, household.Principal, Filter, Cursor, int) ([]ledger.Revision, *Cursor, error)
	TransactionSources(context.Context, household.Principal, string, uint64) ([]SourceReference, error)
	TransactionSourceFacts(context.Context, household.Principal, string, uint64) ([]SourceFact, error)
}
type Queries struct{ repository QueryRepository }

func NewQueries(r QueryRepository) *Queries { return &Queries{r} }
func (q *Queries) Read(ctx context.Context, p household.Principal, id string) (View, error) {
	r, exists, err := q.repository.CurrentLedgerRevision(ctx, p, id)
	if err != nil {
		return View{}, err
	}
	if !exists {
		return View{}, ledger.ErrNotFound
	}
	return q.view(ctx, p, r)
}
func (q *Queries) List(ctx context.Context, p household.Principal, f Filter, c Cursor, limit int) ([]View, *Cursor, error) {
	if err := f.Validate(); err != nil {
		return nil, nil, err
	}
	if limit < 1 || limit > 100 {
		return nil, nil, ledger.ErrInvalidRevision
	}
	revisions, next, err := q.repository.TransactionReferences(ctx, p, f, c, limit)
	if err != nil {
		return nil, nil, err
	}
	out := make([]View, 0, len(revisions))
	for _, r := range revisions {
		v, err := q.view(ctx, p, r)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, v)
	}
	return out, next, nil
}
func (q *Queries) view(ctx context.Context, p household.Principal, r ledger.Revision) (View, error) {
	review, reviewed, err := q.repository.ReviewResult(ctx, p, r.OperationID, r.Revision)
	if err != nil {
		return View{}, err
	}
	facts, err := q.repository.TransactionSourceFacts(ctx, p, r.OperationID, r.Revision)
	if err != nil {
		return View{}, err
	}
	sources, err := q.repository.TransactionSources(ctx, p, r.OperationID, r.Revision)
	if err != nil {
		return View{}, err
	}
	reasons := []string{}
	if r.SourceConflict {
		reasons = append(reasons, "source_conflict")
	}
	if r.Origin != "manual" {
		reasons = append(reasons, "source_history_not_reconciled")
	}
	if r.FeeKnowledge != ledger.KnownFees {
		reasons = append(reasons, "fees_unknown")
	}
	for _, posting := range r.Postings {
		if posting.Funding == ledger.UnknownFunds {
			reasons = append(reasons, "funding_split_unknown")
			break
		}
	}
	state := reporting.Complete
	if len(reasons) > 0 {
		state = reporting.Partial
	}
	coverage, err := reporting.NewCoverage(state, reasons)
	v := View{Revision: r, Sources: sources, Coverage: coverage, SourceFacts: facts}
	if reviewed {
		v.Review = &review
	}
	return v, err
}

func (q *Queries) Coverage(ctx context.Context, p household.Principal) (reporting.Coverage, error) {
	return q.repository.TransactionCoverage(ctx, p)
}
