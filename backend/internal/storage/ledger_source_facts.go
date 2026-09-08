package storage

import (
	"context"
	"encoding/json"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

// Source facts are immutable normalization evidence. Decimal strings preserve the
// exact input; executable postings continue to use PostgreSQL NUMERIC.
type storedSourceFact struct {
	Correspondence                                             *ledger.Correspondence
	OperationID                                                string
	ActorID                                                    household.UserID
	Reason                                                     string
	Type                                                       ledger.Type
	State                                                      ledger.State
	PostedAt, OccurredAt, CashDate, ExpenseMonth, Timezone     string
	FeeKnowledge                                               ledger.FeeKnowledge
	PnLBasis                                                   ledger.PnLBasis
	Merchant, Note, AttachmentID, AllocationReason, PayerState string
	PayerMemberID                                              household.MembershipID
	Postings                                                   []storedSourcePosting
}
type storedSourcePosting struct {
	FeeID             string
	AccountID, Amount string
	Asset             money.Asset
	Role              ledger.Role
	Funding           ledger.FundingKind
	Treatment         ledger.Treatment
}

func (s *Store) SaveSourceFact(ctx context.Context, record ledger.SourceRecord, r ledger.Revision, conflict string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.syncJobID == "" {
		return ErrTransactionRequired
	}
	if err = r.Validate(); err != nil {
		return err
	}
	f := storedSourceFact{Correspondence: r.Correspondence, OperationID: r.OperationID, ActorID: r.ActorID, Reason: r.Reason, Type: r.Type, State: r.State, PostedAt: r.PostedAt.String(), OccurredAt: r.OccurredAt.String(), CashDate: r.CashDate.String(), ExpenseMonth: r.ExpenseMonth.String(), Timezone: r.Timezone.String(), FeeKnowledge: r.FeeKnowledge, PnLBasis: r.PnLBasis, Merchant: r.Merchant, Note: r.Note, AttachmentID: r.AttachmentID, AllocationReason: r.AllocationReason, PayerState: r.PayerState, PayerMemberID: r.PayerMemberID}
	for _, p := range r.Postings {
		f.Postings = append(f.Postings, storedSourcePosting{p.FeeID, p.AccountID, p.Money.Amount(), p.Money.Asset(), p.Role, p.Funding, p.Treatment})
	}
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_source_facts(household_id,source_id,source_revision,operation_id,fact,conflict) VALUES($1,$2,$3,$4,$5,$6)`, scope.principal.HouseholdID(), record.ID, record.Revision, r.OperationID, data, conflict)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_revision_sources(household_id,operation_id,revision,source_id,source_revision) SELECT household_id,id,revision,$3,$4 FROM want_keep.operations WHERE household_id=$1 AND id=$2 ON CONFLICT DO NOTHING`, scope.principal.HouseholdID(), r.OperationID, record.ID, record.Revision)
	return err
}
func (s *Store) LatestSourceFact(ctx context.Context, p household.Principal, id string) (*ledger.Revision, error) {
	return s.sourceFact(ctx, p, id, 0)
}
func (s *Store) sourceFact(ctx context.Context, p household.Principal, id string, revision uint64) (*ledger.Revision, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT DISTINCT ON(f.source_id) f.fact FROM want_keep.ledger_source_facts f WHERE f.household_id=$1 AND f.operation_id=$2 AND ($3::bigint=0 OR EXISTS(SELECT 1 FROM want_keep.ledger_revision_sources r WHERE (r.household_id,r.operation_id,r.source_id,r.source_revision)=(f.household_id,f.operation_id,f.source_id,f.source_revision) AND r.revision=$3)) ORDER BY f.source_id,f.source_revision DESC`, p.HouseholdID(), id, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result *ledger.Revision
	for rows.Next() {
		if result != nil {
			return nil, ledger.ErrSourceAmbiguous
		}
		var data []byte
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		var f storedSourceFact
		if err = json.Unmarshal(data, &f); err != nil {
			return nil, ErrStorage
		}
		r, err := f.domain()
		if err != nil {
			return nil, err
		}
		result = &r
	}
	return result, rows.Err()
}
func (f storedSourceFact) domain() (ledger.Revision, error) {
	r := ledger.Revision{Correspondence: f.Correspondence, OperationID: f.OperationID, Revision: 1, ActorID: f.ActorID, Reason: f.Reason, Type: f.Type, State: f.State, Origin: "source", FeeKnowledge: f.FeeKnowledge, PnLBasis: f.PnLBasis, Merchant: f.Merchant, Note: f.Note, AttachmentID: f.AttachmentID, AllocationReason: f.AllocationReason, PayerState: f.PayerState, PayerMemberID: f.PayerMemberID}
	var err error
	r.OccurredAt, err = calendar.ParseInstant(f.OccurredAt)
	if err != nil {
		return r, err
	}
	r.CashDate, err = calendar.ParseDate(f.CashDate)
	if err != nil {
		return r, err
	}
	if f.PostedAt != "" {
		r.PostedAt, err = calendar.ParseInstant(f.PostedAt)
		if err != nil {
			return r, err
		}
	}
	if f.ExpenseMonth != "" {
		r.ExpenseMonth, err = calendar.ParseMonth(f.ExpenseMonth)
		if err != nil {
			return r, err
		}
	}
	if f.Timezone != "" {
		r.Timezone, err = calendar.ParseTimezone(f.Timezone)
		if err != nil {
			return r, err
		}
	}
	for _, p := range f.Postings {
		m, e := money.NewMoney(p.Amount, p.Asset)
		if e != nil {
			return r, e
		}
		r.Postings = append(r.Postings, ledger.Posting{FeeID: p.FeeID, AccountID: p.AccountID, Money: m, Role: p.Role, Funding: p.Funding, Treatment: p.Treatment})
	}
	return r, r.Validate()
}
