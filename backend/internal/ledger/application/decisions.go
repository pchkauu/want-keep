package application

import (
	"context"
	"errors"
	"slices"
	"sort"

	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

type DecisionRepository interface {
	Repository
	CurrentLedgerRevision(context.Context, household.Principal, string) (ledger.Revision, bool, error)
	SaveDecision(context.Context, ledger.Decision) error
	Decision(context.Context, household.Principal, string) (ledger.Decision, error)
	DecisionUndone(context.Context, household.Principal, string) (bool, error)
	LatestSourceFact(context.Context, household.Principal, string) (*ledger.Revision, error)
	DecisionSourceFact(context.Context, household.Principal, string, string) (*ledger.Revision, error)
	RevisionEvidence(context.Context, household.Principal, string, uint64) ([]ledger.Evidence, error)
}

type Change struct {
	OperationID string
	Expected    uint64
	Correction  ledger.Correction
	Exclude     bool
}
type ExpectedRevision struct {
	OperationID string
	Revision    uint64
}

func (s *Service) Correct(ctx context.Context, p household.Principal, change Change, reason string) (command.Result, error) {
	return s.ApplyChanges(ctx, p, []Change{change}, reason)
}

// ApplyChanges is the atomic decision boundary used by household commands. Its caller owns the transaction.
func (s *Service) ApplyChanges(ctx context.Context, p household.Principal, changes []Change, reason string) (command.Result, error) {
	return s.applyChanges(ctx, p, changes, reason, "correction")
}
func (s *Service) applyChanges(ctx context.Context, p household.Principal, changes []Change, reason, kind string) (command.Result, error) {
	if len(changes) < 1 || len(changes) > 100 {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	changes = append([]Change(nil), changes...)
	sort.Slice(changes, func(i, j int) bool { return changes[i].OperationID < changes[j].OperationID })
	d := ledger.Decision{ID: s.newID(), Kind: kind, Reason: reason, ActorID: p.UserID(), At: s.now()}
	next := make([]ledger.Revision, 0, len(changes))
	unchanged := map[string]string{}
	changedGroups := map[string]bool{}
	for i, in := range changes {
		if in.Exclude && in.Correction != (ledger.Correction{}) {
			return command.Result{}, commands.Rejection{Code: "invalid_request"}
		}
		if in.Exclude != changes[0].Exclude {
			return command.Result{}, commands.Rejection{Code: "invalid_request"}
		}
		if i > 0 && changes[i-1].OperationID == in.OperationID {
			return command.Result{}, commands.Rejection{Code: "invalid_request"}
		}
		r, err := s.current(ctx, p, in.OperationID, in.Expected)
		if err != nil {
			return command.Result{}, err
		}
		var fields []ledger.Field
		var updated ledger.Revision
		if in.Exclude {
			if kind == "automated" || r.Type == ledger.Opening {
				return command.Result{}, commands.Rejection{Code: "feature_unavailable"}
			}
			if r.Accounting() == ledger.ExcludedFromAccounting {
				if len(changes) == 1 || r.Participation.GroupID == "" {
					return command.Result{}, commands.Rejection{Code: "no_change"}
				}
				unchanged[r.OperationID] = r.Participation.GroupID
				updated = r.Clone()
				fields = []ledger.Field{ledger.MatchingField}
			} else {
				updated = r.Clone()
				updated.AccountingState = ledger.ExcludedFromAccounting
				fields = []ledger.Field{ledger.AccountingField}
				changedGroups[r.Participation.GroupID] = true
			}
			d.Kind = "exclusion"
		} else {
			updated, fields, err = r.Correct(in.Correction)
			if errors.Is(err, ledger.ErrNoChange) && len(changes) > 1 && r.Participation.GroupID != "" {
				unchanged[r.OperationID] = r.Participation.GroupID
				updated = r.Clone()
				fields = []ledger.Field{ledger.MatchingField}
				err = nil
			} else if err == nil {
				changedGroups[r.Participation.GroupID] = true
			}
			if err != nil {
				return command.Result{}, s.rejectDecision(err)
			}
		}
		if kind == "automated" {
			if r.HumanOverride && len(r.Protections) == 0 {
				return command.Result{}, commands.Rejection{Code: "protected_field"}
			}
			for _, f := range fields {
				if _, ok := r.Protections[f]; ok {
					return command.Result{}, commands.Rejection{Code: "protected_field"}
				}
			}
			if _, ok := r.Protections[ledger.LegacyField]; ok {
				return command.Result{}, commands.Rejection{Code: "protected_field"}
			}
			if in.Correction.Principal != nil || in.Correction.Fees != nil || in.Correction.OccurredAt != nil {
				return command.Result{}, commands.Rejection{Code: "source_conflict"}
			}
		}
		if updated.OccurredAt.Time().After(s.now().Time()) {
			return command.Result{}, commands.Rejection{Code: "invalid_request"}
		}
		next = append(next, updated.WithDecision(d, fields))
		d.Entries = append(d.Entries, ledger.DecisionEntry{OperationID: r.OperationID, Before: r.Revision, After: r.Revision + 1, Fields: fields})
	}
	for _, group := range unchanged {
		if !changedGroups[group] {
			return command.Result{}, commands.Rejection{Code: "no_change"}
		}
	}
	return s.persistDecision(ctx, p, d, next)
}

func (s *Service) Undo(ctx context.Context, p household.Principal, id string, expected []ExpectedRevision, reason string) (command.Result, error) {
	d, err := s.repository.Decision(ctx, p, id)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	undone, err := s.repository.DecisionUndone(ctx, p, id)
	if err != nil {
		return command.Result{}, err
	}
	if undone {
		return command.Result{}, commands.Rejection{Code: "decision_conflict"}
	}
	if len(expected) != len(d.Entries) {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	versions := map[string]uint64{}
	for _, v := range expected {
		if _, exists := versions[v.OperationID]; exists {
			return command.Result{}, commands.Rejection{Code: "invalid_request"}
		}
		versions[v.OperationID] = v.Revision
	}
	undo := ledger.Decision{ID: s.newID(), Kind: "undo", Reason: reason, UndoOf: d.ID, ActorID: p.UserID(), At: s.now()}
	next := []ledger.Revision{}
	sort.Slice(d.Entries, func(i, j int) bool { return d.Entries[i].OperationID < d.Entries[j].OperationID })
	for _, entry := range d.Entries {
		r, err := s.current(ctx, p, entry.OperationID, versions[entry.OperationID])
		if err != nil {
			return command.Result{}, err
		}
		before, err := s.repository.LedgerRevision(ctx, p, entry.OperationID, entry.Before)
		if err != nil {
			return command.Result{}, s.reject(err)
		}
		source, err := s.repository.LatestSourceFact(ctx, p, entry.OperationID)
		if err != nil {
			return command.Result{}, s.reject(err)
		}
		prior := r
		r, err = (decisionRestorer{s.repository}).restore(ctx, p, r, entry, before, source)
		if err != nil {
			return command.Result{}, s.rejectDecision(err)
		}
		for _, field := range []ledger.Field{ledger.PrincipalField, ledger.FeesField, ledger.DateField, ledger.PayerField, ledger.MerchantField, ledger.NoteField} {
			if !prior.FieldEqual(r, field) && !slices.Contains(entry.Fields, field) {
				entry.Fields = append(entry.Fields, field)
			}
		}
		next = append(next, r.WithDecision(undo, entry.Fields))
		undo.Entries = append(undo.Entries, ledger.DecisionEntry{OperationID: r.OperationID, Before: r.Revision, After: r.Revision + 1, Fields: entry.Fields})
	}
	return s.persistDecision(ctx, p, undo, next)
}
func (s *Service) current(ctx context.Context, p household.Principal, id string, expected uint64) (ledger.Revision, error) {
	r, ok, err := s.repository.CurrentLedgerRevision(ctx, p, id)
	if err != nil {
		return r, s.reject(err)
	}
	if !ok {
		return r, commands.Rejection{Code: "not_found"}
	}
	if r.Revision != expected || expected >= command.MaxRevision {
		return r, commands.Rejection{Code: "version_conflict"}
	}
	return r, nil
}
func (s *Service) persistDecision(ctx context.Context, p household.Principal, d ledger.Decision, next []ledger.Revision) (command.Result, error) {
	if err := s.writer.AppendDecision(ctx, p, d, next); err != nil {
		return command.Result{}, s.rejectDecision(err)
	}
	r := next[0]
	return command.Result{ResourceType: "transaction", ResourceID: r.OperationID, Revision: r.Revision}, nil
}
func (s *Service) rejectDecision(err error) error {
	if err == ledger.ErrDecisionConflict {
		return commands.Rejection{Code: "decision_conflict"}
	}
	if err == ledger.ErrNoChange {
		return commands.Rejection{Code: "no_change"}
	}
	return s.reject(err)
}
