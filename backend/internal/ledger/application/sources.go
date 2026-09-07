package application

import (
	"context"
	"errors"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

type SourceRepository interface {
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
		if found && previous.HumanOverride {
			result.PreservedOverride = true
			return result, nil
		}
		expected := uint64(0)
		if found {
			expected = previous.Revision
		}
		r := *input.Operation
		r.Postings = append([]ledger.Posting(nil), r.Postings...)
		r.Origin = "source"
		if err = s.writer.Append(ctx, p, r, expected); err != nil {
			return result, err
		}
	}
	return result, nil
}
