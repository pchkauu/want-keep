package application

import (
	"context"
	"errors"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	attachment "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	category "github.com/pchkauu/want-keep/backend/internal/categories/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type FactsRepository interface {
	DecisionRepository
	ReviewRepository
	Account(context.Context, household.Principal, string) (account.Account, error)
	AccountTimezone(context.Context, household.Principal) (calendar.Timezone, error)
	RequireLedgerPayer(context.Context, household.Principal, household.MembershipID) error
	Attachment(context.Context, household.Principal, string) (attachment.Attachment, error)
	Category(context.Context, household.Principal, string) (category.Category, error)
	Merchant(context.Context, household.Principal, string) (category.Merchant, error)
}

type Service struct {
	repository  FactsRepository
	writer      JournalWriter
	allocations AllocationResolver
	now         func() calendar.Instant
	newID       func() string
}

func NewService(r FactsRepository, w JournalWriter, now func() calendar.Instant, newID func() string) *Service {
	return &Service{repository: r, writer: w, now: now, newID: newID}
}

type AllocationResolver interface {
	Resolve(context.Context, household.Principal, string, string) (ledger.AllocationInput, bool, error)
	ResolveSource(context.Context, household.Principal, string) (ledger.AllocationInput, bool, error)
	ActiveMemberIDs(context.Context, household.Principal) ([]household.MembershipID, error)
}

func NewServiceWithAllocations(r FactsRepository, w JournalWriter, allocations AllocationResolver, now func() calendar.Instant, newID func() string) *Service {
	return &Service{repository: r, writer: w, allocations: allocations, now: now, newID: newID}
}

type CreateInput struct {
	Type                                           ledger.Type
	AccountID                                      string
	At                                             calendar.Instant
	Amount                                         money.Money
	Funding                                        ledger.FundingKind
	PayerState                                     string
	PayerMemberID                                  household.MembershipID
	Merchant, Note, AttachmentID, AllocationReason string
	CategoryID, MerchantID                         string
	Allocation                                     ledger.AllocationInput
	Unsupported                                    bool
}

func (s *Service) Create(ctx context.Context, p household.Principal, in CreateInput) (command.Result, error) {
	if in.Unsupported {
		return command.Result{}, commands.Rejection{Code: "feature_unavailable"}
	}
	if in.Type != ledger.Income && in.Type != ledger.Expense {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	if in.Type == ledger.Income && in.Allocation.Mode != "" {
		return command.Result{}, commands.Rejection{Code: "invalid_allocation"}
	}
	if err := in.Amount.Validate(); err != nil {
		return command.Result{}, s.reject(err)
	}
	if in.Amount.Sign() <= 0 {
		return command.Result{}, commands.Rejection{Code: "invalid_money"}
	}
	if in.Type == ledger.Expense && in.PayerState == "not_applicable" || in.PayerState == "known" && in.PayerMemberID == "" {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	if in.PayerState == "known" {
		if err := s.repository.RequireLedgerPayer(ctx, p, in.PayerMemberID); err != nil {
			return command.Result{}, s.reject(err)
		}
	}
	if _, err := s.repository.Account(ctx, p, in.AccountID); err != nil {
		return command.Result{}, s.reject(err)
	}
	if in.AttachmentID != "" {
		a, err := s.repository.Attachment(ctx, p, in.AttachmentID)
		if err != nil {
			return command.Result{}, s.reject(err)
		}
		if a.Upload.AccountID != in.AccountID {
			return command.Result{}, commands.Rejection{Code: "invalid_attachment"}
		}
		if a.State != attachment.Accepted || !a.OriginalReady {
			return command.Result{}, commands.Rejection{Code: "attachment_not_ready"}
		}
	}
	r, err := s.manual(ctx, p, in.At)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	r.Type = in.Type
	r.PayerState, r.PayerMemberID = in.PayerState, in.PayerMemberID
	r.Merchant, r.Note, r.AttachmentID, r.AllocationReason = in.Merchant, in.Note, in.AttachmentID, in.AllocationReason
	r.CategoryID, r.MerchantID = in.CategoryID, in.MerchantID
	if in.Merchant != "" {
		r.Protections[ledger.MerchantField] = ledger.Protection{Revision: 1}
	}
	if in.Note != "" {
		r.Protections[ledger.NoteField] = ledger.Protection{Revision: 1}
	}
	if in.CategoryID != "" {
		r.Protections[ledger.CategoryField] = ledger.Protection{Revision: 1}
	}
	if in.MerchantID != "" {
		r.Protections[ledger.MerchantIDField] = ledger.Protection{Revision: 1}
	}
	amount := in.Amount
	if in.Type == ledger.Expense {
		zero, _ := money.NewMoney("0", amount.Asset())
		amount, err = zero.Subtract(amount)
		if err != nil {
			return command.Result{}, s.reject(err)
		}
	}
	r.Postings = []ledger.Posting{{AccountID: in.AccountID, Money: amount, Role: ledger.Principal, Funding: in.Funding, Treatment: ledger.Movement}}
	if in.Type == ledger.Income {
		r.Allocation = ledger.NotApplicableAllocation()
	} else {
		allocation := in.Allocation
		if allocation.Mode == "" {
			allocation = ledger.AllocationInput{Mode: ledger.AllocationUnknown, Reason: "allocation_unresolved"}
		}
		if allocation.Mode == ledger.AllocationUnknown && s.allocations != nil {
			resolved, ok, resolveErr := s.allocations.Resolve(ctx, p, in.MerchantID, in.CategoryID)
			if resolveErr != nil {
				return command.Result{}, resolveErr
			}
			if ok {
				allocation = resolved
			}
		}
		var members []household.MembershipID
		if allocation.Mode != ledger.AllocationUnknown {
			if s.allocations == nil {
				return command.Result{}, commands.Rejection{Code: "feature_unavailable"}
			}
			members, err = s.allocations.ActiveMemberIDs(ctx, p)
			if err != nil {
				return command.Result{}, err
			}
			if allocation.Mode == ledger.AllocationEqual && allocation.Purpose == ledger.AllocationShared && len(allocation.Members) == 0 {
				for _, memberID := range members {
					allocation.Members = append(allocation.Members, ledger.AllocationMemberInput{MemberID: memberID})
				}
			}
		}
		r, err = r.WithAllocation(allocation, nil, members)
		if err != nil {
			return command.Result{}, s.reject(err)
		}
		if allocation.Origin == ledger.AllocationExplicitPurchase || allocation.Origin == ledger.AllocationExplicitItem {
			r.Protections[ledger.AllocationField] = ledger.Protection{Revision: 1}
			r.FieldVersions[ledger.AllocationField] = 1
		}
	}
	if err = s.requireActiveClassification(ctx, p, r); err != nil {
		return command.Result{}, err
	}
	return s.append(ctx, p, r)
}

func (s *Service) requireActiveClassification(ctx context.Context, p household.Principal, r ledger.Revision) error {
	ids := map[string]bool{}
	if r.CategoryID != "" {
		ids[r.CategoryID] = true
	}
	for _, item := range r.ReceiptItems {
		if item.CategoryID != "" {
			ids[item.CategoryID] = true
		}
	}
	for id := range ids {
		entry, err := s.repository.Category(ctx, p, id)
		if err != nil {
			return s.reject(err)
		}
		if entry.State != category.Active {
			return commands.Rejection{Code: "category_archived"}
		}
	}
	if r.MerchantID != "" {
		entry, err := s.repository.Merchant(ctx, p, r.MerchantID)
		if err != nil {
			return s.reject(err)
		}
		if entry.State != category.Active {
			return commands.Rejection{Code: "not_found"}
		}
	}
	return nil
}

func (s *Service) manual(ctx context.Context, p household.Principal, at calendar.Instant) (ledger.Revision, error) {
	if at.String() == "" || at.Time().After(s.now().Time()) {
		return ledger.Revision{}, calendar.ErrInvalidTime
	}
	zone, err := s.repository.AccountTimezone(ctx, p)
	if err != nil {
		return ledger.Revision{}, err
	}
	date, err := at.DateIn(zone)
	if err != nil {
		return ledger.Revision{}, err
	}
	month, err := calendar.ParseMonth(date.String()[:7])
	if err != nil {
		return ledger.Revision{}, err
	}
	return ledger.Revision{OperationID: s.newID(), Revision: 1, ActorID: p.UserID(), Reason: "manual_record", State: ledger.Posted, OccurredAt: at, CashDate: date, ExpenseMonth: month, Timezone: zone, Origin: "manual", FeeKnowledge: ledger.KnownFees, Protections: map[ledger.Field]ledger.Protection{ledger.PrincipalField: {Revision: 1}, ledger.FeesField: {Revision: 1}, ledger.DateField: {Revision: 1}, ledger.PayerField: {Revision: 1}}, FieldVersions: map[ledger.Field]uint64{}, RecordedAt: s.now(), HumanOverride: true, PayerState: "not_applicable", AllocationReason: "allocation_unresolved"}, nil
}

func (s *Service) append(ctx context.Context, p household.Principal, r ledger.Revision) (command.Result, error) {
	if err := s.writer.Append(ctx, p, r, 0); err != nil {
		return command.Result{}, s.reject(err)
	}
	return command.Result{ResourceType: "transaction", ResourceID: r.OperationID, Revision: r.Revision}, nil
}

func (s *Service) reject(err error) error {
	switch {
	case errors.Is(err, ledger.ErrMatchingConflict):
		return commands.Rejection{Code: "matching_conflict"}
	case errors.Is(err, ledger.ErrFeatureUnavailable):
		return commands.Rejection{Code: "feature_unavailable"}
	case errors.Is(err, ledger.ErrSourceAmbiguous):
		return commands.Rejection{Code: "source_ambiguous"}
	case errors.Is(err, ledger.ErrInvalidRevision), errors.Is(err, ledger.ErrInvalidTransition):
		return commands.Rejection{Code: "invalid_transaction"}
	case errors.Is(err, ledger.ErrInvalidAllocation):
		return commands.Rejection{Code: "invalid_allocation"}
	case errors.Is(err, ledger.ErrClarificationRequired):
		return commands.Rejection{Code: "clarification_required"}
	case errors.Is(err, category.ErrNotFound):
		return commands.Rejection{Code: "not_found"}
	case errors.Is(err, ledger.ErrNotFound), errors.Is(err, account.ErrNotFound), errors.Is(err, attachment.ErrNotFound):
		return commands.Rejection{Code: "not_found"}
	case errors.Is(err, household.ErrForbidden):
		return commands.Rejection{Code: "forbidden"}
	case errors.Is(err, money.ErrAssetMismatch):
		return commands.Rejection{Code: "asset_mismatch"}
	case errors.Is(err, money.ErrInvalidMoney):
		return commands.Rejection{Code: "invalid_money"}
	case errors.Is(err, calendar.ErrInvalidTime):
		return commands.Rejection{Code: "invalid_time"}
	}
	return err
}
