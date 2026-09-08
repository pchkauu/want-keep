package application

import (
	"context"
	"errors"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

type SourceRepository interface {
	AccountTimezone(context.Context, household.Principal) (calendar.Timezone, error)
	SaveSourceFact(context.Context, ledger.SourceRecord, ledger.Revision, string) error
	Source(context.Context, household.Principal, ledger.SourceKey) (ledger.SourceRecord, bool, error)
	SaveSource(context.Context, ledger.SourceRecord, ledger.SourceInput) (ledger.SourceRecord, error)
	RecordProvenance(context.Context, ledger.SourceRecord, ledger.SourceInput) error
	RecordSourceAmbiguity(context.Context, ledger.SourceInput) error
	RecordUnresolvedTransaction(context.Context, ledger.SourceInput) error
	CurrentLedgerRevision(context.Context, household.Principal, string) (ledger.Revision, bool, error)
	EmitEvent(context.Context, string, string, uint64, string) error
}
type Sources struct {
	repository SourceRepository
	writer     *Writer
}

func NewSources(r SourceRepository, w *Writer) *Sources { return &Sources{r, w} }

// Apply belongs inside CommitPage's transaction, after the deployment and lease fences.
func (s *Sources) Apply(ctx context.Context, p household.Principal, input ledger.SourceInput) (ledger.SourceOutcome, error) {
	if err := input.Validate(); err != nil {
		return ledger.SourceOutcome{}, err
	}
	if err := p.RequireHousehold(input.Key.HouseholdID); err != nil {
		return ledger.SourceOutcome{}, err
	}
	if input.Operation != nil {
		raw := input.Operation.Clone()
		raw.Revision = 1
		raw.ActorID = p.UserID()
		raw.HumanOverride = false
		raw.Protections = map[ledger.Field]ledger.Protection{}
		raw.FieldVersions = map[ledger.Field]uint64{}
		raw.AccountingState = ledger.IncludedInAccounting
		raw.CategoryID, raw.MerchantID, raw.ReceiptItems = "", "", nil
		raw.DecisionID, raw.ReviewState = "", ""
		input.Operation = &raw
	}
	if input.Operation != nil && input.Operation.Validate() != nil {
		input.Operation = nil
		input.UnresolvedReason = "invalid_transaction"
	}
	if input.Operation == nil && input.UnresolvedReason == "" {
		input.UnresolvedReason = "transaction_not_normalized"
	}
	current, exists, err := s.repository.Source(ctx, p, input.Key)
	if errors.Is(err, ledger.ErrSourceAmbiguous) {
		return ledger.SourceOutcome{Record: ledger.SourceRecord{Key: input.Key, Ambiguous: true}}, s.repository.RecordSourceAmbiguity(ctx, input)
	}
	if err != nil {
		return ledger.SourceOutcome{}, err
	}
	var duplicate bool
	if exists {
		current, duplicate, err = current.Next(input)
		if err != nil {
			return ledger.SourceOutcome{}, err
		}
	} else {
		current = ledger.SourceRecord{Key: input.Key, Revision: 1, PayloadHash: input.PayloadHash, Ambiguous: input.Classification != "new"}
		if input.Operation != nil && !current.Ambiguous {
			current.OperationID = input.Operation.OperationID
		}
	}
	if !duplicate && input.Operation != nil && current.OperationID != "" && current.OperationID != input.Operation.OperationID {
		current.Ambiguous = true
	}
	if !duplicate && !current.Ambiguous && current.OperationID == "" && input.Operation != nil {
		current.OperationID = input.Operation.OperationID
	}
	if !duplicate {
		current, err = s.repository.SaveSource(ctx, current, input)
		if errors.Is(err, ledger.ErrSourceAmbiguous) {
			return ledger.SourceOutcome{Record: ledger.SourceRecord{Key: input.Key, Ambiguous: true}}, s.repository.RecordSourceAmbiguity(ctx, input)
		}
		if err != nil {
			return ledger.SourceOutcome{}, err
		}
	}
	if err = s.repository.RecordProvenance(ctx, current, input); err != nil {
		if errors.Is(err, ledger.ErrSourceAmbiguous) {
			return ledger.SourceOutcome{Record: ledger.SourceRecord{Key: input.Key, Ambiguous: true}}, s.repository.RecordSourceAmbiguity(ctx, input)
		}
		return ledger.SourceOutcome{}, err
	}
	result := ledger.SourceOutcome{Record: current, Duplicate: duplicate}
	if input.UnresolvedReason != "" {
		if err = s.repository.RecordUnresolvedTransaction(ctx, input); err != nil {
			return result, err
		}
		return result, nil
	}
	if duplicate {
		return result, nil
	}
	if current.Ambiguous {
		err = s.repository.EmitEvent(ctx, "source", current.ID, current.Revision, "source.ambiguous")
		return result, err
	}
	if input.Operation != nil {
		if current.OperationID != input.Operation.OperationID {
			return result, ledger.ErrSourceAmbiguous
		}
		previous, found, err := s.repository.CurrentLedgerRevision(ctx, p, current.OperationID)
		if err != nil {
			return result, err
		}
		expected := uint64(0)
		raw := input.Operation.Clone()
		raw.Origin = "source"
		raw.ActorID = p.UserID()
		zone, e := s.repository.AccountTimezone(ctx, p)
		if e != nil {
			return result, e
		}
		raw, e = raw.InTimezone(zone)
		if e != nil {
			return result, e
		}
		for i, posting := range raw.Postings {
			a, e := s.writer.accounts.Account(ctx, p, posting.AccountID)
			if e != nil {
				return result, e
			}
			if posting.Funding == "" {
				raw.Postings[i].Funding = ledger.OwnFunds
				if a.Product == "credit_card" {
					raw.Postings[i].Funding = ledger.UnknownFunds
				}
			}
		}
		r := raw.Clone()
		conflict := ""
		if found {
			expected = previous.Revision
			var differs bool
			r, differs, err = previous.MergeSource(raw)
			if err != nil {
				return result, err
			}
			result.PreservedOverride = len(previous.Protections) > 0 || previous.HumanOverride
			if differs {
				conflict = "protected_fields"
			}
			if _, legacy := previous.Protections[ledger.LegacyField]; legacy || previous.HumanOverride && len(previous.Protections) == 0 {
				return result, s.repository.SaveSourceFact(ctx, current, raw, "legacy_protection")
			}
		}
		r.Revision = expected + 1
		r.ActorID = p.UserID()
		r.DecisionID = ""
		// The storage boundary records commit-time observation separately from provider fetch time.
		r.RecordedAt = calendar.Instant{}
		r, err = r.InTimezone(zone)
		if err != nil {
			return result, err
		}
		if r.FieldVersions == nil {
			r.FieldVersions = map[ledger.Field]uint64{}
		}
		for _, f := range []ledger.Field{ledger.PrincipalField, ledger.FeesField, ledger.DateField, ledger.PayerField, ledger.MerchantField, ledger.NoteField} {
			if !found || !previous.FieldEqual(r, f) {
				r.FieldVersions[f] = r.Revision
			}
		}
		var prior *ledger.Revision
		if found {
			prior = &previous
		}
		if r.CheckSuccessor(prior) != nil {
			if !found {
				return result, s.repository.RecordUnresolvedTransaction(ctx, input)
			}
			return result, s.repository.SaveSourceFact(ctx, current, raw, "invalid_merge")
		}
		if found && previous.SameFacts(r) {
			return result, s.repository.SaveSourceFact(ctx, current, raw, conflict)
		}
		if err = s.writer.Append(ctx, p, r, expected); err != nil {
			return result, err
		}
		if err = s.repository.SaveSourceFact(ctx, current, raw, conflict); err != nil {
			return result, err
		}
	}
	return result, nil
}
