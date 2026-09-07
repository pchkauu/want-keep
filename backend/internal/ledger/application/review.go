package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"unicode/utf8"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

type ReviewResult struct {
	OperationID                   string
	Revision                      uint64
	ActorID                       household.UserID
	State, Rationale, PayloadHash string
	At                            calendar.Instant
	Evidence                      []ledger.Evidence
}
type ReviewRepository interface {
	ReviewResult(context.Context, household.Principal, string, uint64) (ReviewResult, bool, error)
	SaveReviewResult(context.Context, ReviewResult) error
}
type ReviewInput struct {
	OperationID      string
	Revision         uint64
	State, Rationale string
	Correction       *ledger.Correction
	Evidence         []ledger.Evidence
}

// CompleteReview accepts a result from a trusted worker in a household transaction.
// The initiator is the trusted principal, never a model-supplied actor.
func (s *Service) CompleteReview(ctx context.Context, p household.Principal, in ReviewInput) error {
	if !slices.Contains([]string{"reviewed", "clarification", "failed"}, in.State) || utf8.RuneCountInString(in.Rationale) < 1 || utf8.RuneCountInString(in.Rationale) > 2000 {
		return commands.Rejection{Code: "invalid_request"}
	}
	if in.Correction != nil && (in.State != "reviewed" || in.Correction.Principal != nil || in.Correction.Fees != nil || in.Correction.OccurredAt != nil) {
		return commands.Rejection{Code: "source_conflict"}
	}
	if len(in.Evidence) > 100 {
		return commands.Rejection{Code: "invalid_request"}
	}
	seen := map[ledger.Evidence]bool{}
	for _, e := range in.Evidence {
		if e.Validate() != nil || seen[e] {
			return commands.Rejection{Code: "invalid_request"}
		}
		seen[e] = true
	}
	// Only text and payer changes reach this encoder; money is never JSON-marshaled from domain objects.
	payload := struct {
		State, Rationale string
		Merchant, Note   *string
		Payer            *ledger.PayerChange
		Evidence         []ledger.Evidence
	}{State: in.State, Rationale: in.Rationale, Evidence: in.Evidence}
	if in.Correction != nil {
		payload.Merchant = in.Correction.Merchant
		payload.Note = in.Correction.Note
		payload.Payer = in.Correction.Payer
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(data)
	digest := hex.EncodeToString(hash[:])
	prior, exists, err := s.repository.ReviewResult(ctx, p, in.OperationID, in.Revision)
	if err != nil {
		return err
	}
	if exists {
		if prior.ActorID != p.UserID() || prior.PayloadHash != digest {
			return commands.Rejection{Code: "idempotency_conflict"}
		}
		return nil
	}
	current, found, err := s.repository.CurrentLedgerRevision(ctx, p, in.OperationID)
	if err != nil {
		return err
	}
	if !found {
		return commands.Rejection{Code: "not_found"}
	}
	if current.Revision != in.Revision {
		return commands.Rejection{Code: "version_conflict"}
	}
	if err = s.repository.SaveReviewResult(ctx, ReviewResult{OperationID: in.OperationID, Revision: in.Revision, ActorID: p.UserID(), State: in.State, Rationale: in.Rationale, PayloadHash: digest, At: s.now(), Evidence: in.Evidence}); err != nil {
		return err
	}
	if in.Correction != nil {
		if _, err = s.applyChanges(ctx, p, []Change{{OperationID: in.OperationID, Expected: in.Revision, Correction: *in.Correction}}, in.Rationale, "automated"); err != nil {
			var rejection commands.Rejection
			if !errors.As(err, &rejection) || rejection.Code != "no_change" {
				return err
			}
		}
	}
	return nil
}
