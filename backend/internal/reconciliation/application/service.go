package application

import (
	"context"
	"errors"
	"sort"
	"time"
	"unicode/utf8"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reconciliation "github.com/pchkauu/want-keep/backend/internal/reconciliation/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type Cursor struct {
	At calendar.Instant
	ID string
}

type Filter struct {
	AccountID string
	Lifecycle reconciliation.Lifecycle
	Result    reconciliation.Result
}

func (f Filter) Validate() error {
	if f.Lifecycle != "" && !f.Lifecycle.Valid() || f.Result != "" && !f.Result.Valid() {
		return reconciliation.ErrInvalidReconciliation
	}
	return nil
}

type Repository interface {
	Account(context.Context, household.Principal, string) (account.Account, error)
	Opening(context.Context, household.Principal, string) (account.Opening, bool, error)
	LastConfirmedReconciliation(context.Context, household.Principal, string, calendar.Instant) (calendar.Instant, bool, error)
	LatestObservation(context.Context, household.Principal, string) (account.Observation, bool, error)
	HistoricalProjection(context.Context, household.Principal, string, calendar.Instant) (account.Amounts, reporting.Coverage, []string, error)
	SyncJob(context.Context, household.Principal, string) (jobs.Job, error)
	ActiveReconciliation(context.Context, household.Principal, string) (reconciliation.Reconciliation, bool, error)
	Reconciliation(context.Context, household.Principal, string) (reconciliation.Reconciliation, error)
	Reconciliations(context.Context, household.Principal, Filter, Cursor, int) ([]reconciliation.Reconciliation, *Cursor, error)
	SaveReconciliation(context.Context, household.Principal, reconciliation.Reconciliation) error
	SaveResolution(context.Context, household.Principal, reconciliation.Reconciliation) error
	UpdateReplay(context.Context, household.Principal, string, reconciliation.ReplayStatus, string, string) (reconciliation.Reconciliation, bool, error)
	FenceReplayOutcome(context.Context, household.Principal, reconciliation.Replay, jobs.Job, reconciliation.ReplayStatus) error
	AccountTimezone(context.Context, household.Principal) (calendar.Timezone, error)
	EmitEvent(context.Context, string, string, uint64, string) error
}

type Transactions interface {
	WithinHousehold(context.Context, household.Principal, func(context.Context) error) error
}

type ReplayScheduler interface {
	RequestReplay(context.Context, household.Principal, string, connections.Binding, int64, uint64, time.Time, string, time.Time, time.Time) (jobs.Job, error)
}

type LedgerWriter interface {
	Append(context.Context, household.Principal, ledger.Revision, uint64) error
}

type Service struct {
	repository   Repository
	transactions Transactions
	writer       LedgerWriter
	scheduler    ReplayScheduler
	now          func() calendar.Instant
	newID        func() string
}

func NewService(repository Repository, transactions Transactions, writer LedgerWriter, scheduler ReplayScheduler, now func() calendar.Instant, newID func() string) *Service {
	return &Service{repository: repository, transactions: transactions, writer: writer, scheduler: scheduler, now: now, newID: newID}
}

func (s *Service) ReconcileAccount(ctx context.Context, principal household.Principal, accountID string) error {
	_, _, err := s.EvaluateAccount(ctx, principal, accountID)
	return err
}

func (s *Service) EvaluateAccount(ctx context.Context, principal household.Principal, accountID string) (reconciliation.Reconciliation, bool, error) {
	entry, err := s.repository.Account(ctx, principal, accountID)
	if err != nil {
		return reconciliation.Reconciliation{}, false, err
	}
	if entry.ExternalAccountID == "" {
		return reconciliation.Reconciliation{}, false, nil
	}
	observation, found, err := s.repository.LatestObservation(ctx, principal, accountID)
	if err != nil || !found {
		return reconciliation.Reconciliation{}, false, err
	}
	ledgerAmounts, ledgerCoverage, related, err := s.repository.HistoricalProjection(ctx, principal, accountID, observation.AsOf)
	if err != nil {
		return reconciliation.Reconciliation{}, false, err
	}
	coverage, err := mergeCoverage(observation.Coverage, ledgerCoverage)
	if err != nil {
		return reconciliation.Reconciliation{}, false, err
	}
	active, exists, err := s.repository.ActiveReconciliation(ctx, principal, accountID)
	if err != nil {
		return reconciliation.Reconciliation{}, false, err
	}
	if exists && active.ObservationID != observation.ID {
		superseded := active
		superseded.Revision++
		superseded.Lifecycle = reconciliation.Superseded
		superseded.EvaluatedAt = s.now()
		if err = s.repository.SaveReconciliation(ctx, principal, superseded); err != nil {
			return reconciliation.Reconciliation{}, false, err
		}
		exists = false
	}
	revision := uint64(1)
	id := s.newID()
	if exists {
		revision = active.Revision + 1
		id = active.ID
		if revision > command.MaxRevision {
			return reconciliation.Reconciliation{}, false, command.ErrVersionConflict
		}
	}
	candidate, err := reconciliation.Evaluate(reconciliation.Evaluation{
		ID: id, AccountID: accountID, ObservationID: observation.ID, Revision: revision,
		SourceAsOf: observation.AsOf, EvaluatedAt: s.now(), Asset: entry.Asset,
		Source: observation.Amounts, Ledger: ledgerAmounts, Coverage: coverage,
		Freshness: observation.Freshness, Replay: reconciliation.Replay{Status: reconciliation.ReplayNotRequired}, RelatedOperationIDs: related,
	})
	if err != nil {
		return reconciliation.Reconciliation{}, false, err
	}
	if candidate.Result != reconciliation.Balanced {
		replay, replayErr := s.replay(ctx, principal, observation, exists, active, entry)
		if replayErr != nil {
			return reconciliation.Reconciliation{}, false, replayErr
		}
		candidate.Replay = replay
		if err = candidate.Validate(entry.Asset); err != nil {
			return reconciliation.Reconciliation{}, false, err
		}
	}
	if exists && active.SameEvaluation(candidate) {
		return active, false, nil
	}
	if err = s.repository.SaveReconciliation(ctx, principal, candidate); err != nil {
		return reconciliation.Reconciliation{}, false, err
	}
	if err = s.repository.EmitEvent(ctx, "reconciliation", candidate.ID, candidate.Revision, "reconciliation.changed"); err != nil {
		return reconciliation.Reconciliation{}, false, err
	}
	return candidate, true, nil
}

func (s *Service) replay(ctx context.Context, principal household.Principal, observation account.Observation, exists bool, active reconciliation.Reconciliation, entry account.Account) (reconciliation.Replay, error) {
	if exists && active.Replay.Status != reconciliation.ReplayNotRequired {
		return active.Replay, nil
	}
	issued, err := s.repository.SyncJob(ctx, principal, observation.JobID)
	if err != nil {
		return reconciliation.Replay{}, err
	}
	from := observation.AsOf.Time().Add(-90 * 24 * time.Hour)
	if anchor, found, err := s.repository.LastConfirmedReconciliation(ctx, principal, entry.ID, observation.AsOf); err != nil {
		return reconciliation.Replay{}, err
	} else if found && anchor.Time().After(from) {
		from = anchor.Time()
	}
	if opening, found, err := s.repository.Opening(ctx, principal, entry.ID); err != nil {
		return reconciliation.Replay{}, err
	} else if found {
		anchor, err := opening.Instant()
		if err != nil {
			return reconciliation.Replay{}, err
		}
		if anchor.Time().After(from) {
			from = anchor.Time()
		}
	}
	if !from.Before(observation.AsOf.Time()) {
		from = observation.AsOf.Time().Add(-time.Nanosecond)
	}
	start, err := calendar.ParseInstant(from.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return reconciliation.Replay{}, err
	}
	return reconciliation.Replay{
		RequestID: s.newID(), ConnectionID: observation.ConnectionID,
		Status: reconciliation.ReplayPending, From: start, To: observation.AsOf,
		Binding: issued.Binding, AdmissionRevision: issued.AdmissionRevision,
		ConnectionGeneration: issued.ConnectionGeneration,
	}, nil
}

func mergeCoverage(values ...reporting.Coverage) (reporting.Coverage, error) {
	reasons := []string{}
	state := reporting.Complete
	for _, value := range values {
		if value.State() == reporting.NoCoverage {
			state = reporting.NoCoverage
		} else if value.State() == reporting.Partial && state == reporting.Complete {
			state = reporting.Partial
		}
		reasons = append(reasons, value.Reasons()...)
	}
	sort.Strings(reasons)
	reasons = unique(reasons)
	return reporting.NewCoverage(state, reasons)
}

func unique(values []string) []string {
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}

func (s *Service) Read(ctx context.Context, principal household.Principal, id string) (reconciliation.Reconciliation, error) {
	return s.repository.Reconciliation(ctx, principal, id)
}

func (s *Service) List(ctx context.Context, principal household.Principal, filter Filter, cursor Cursor, limit int) ([]reconciliation.Reconciliation, *Cursor, error) {
	if err := filter.Validate(); err != nil || limit < 1 || limit > 100 {
		return nil, nil, reconciliation.ErrInvalidReconciliation
	}
	return s.repository.Reconciliations(ctx, principal, filter, cursor, limit)
}

type ResolutionInput struct {
	ExpectedRevision uint64
	Reason           string
	Components       []reconciliation.ComponentName
}

func (s *Service) Resolve(ctx context.Context, principal household.Principal, id string, input ResolutionInput) (command.Result, error) {
	if input.ExpectedRevision < 1 || utf8.RuneCountInString(input.Reason) < 1 || utf8.RuneCountInString(input.Reason) > 2000 || len(input.Components) < 1 || len(input.Components) > 2 {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	current, err := s.repository.Reconciliation(ctx, principal, id)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	if current.Revision != input.ExpectedRevision {
		return command.Result{}, commands.Rejection{Code: "version_conflict"}
	}
	if current.Lifecycle != reconciliation.Open || current.Coverage.State() != reporting.Complete || current.Result != reconciliation.Discrepant || current.Replay.Status != reconciliation.ReplayCompleted && current.Replay.Status != reconciliation.ReplayUnavailable {
		return command.Result{}, commands.Rejection{Code: "reconciliation_not_ready"}
	}
	selected := map[reconciliation.ComponentName]bool{}
	for _, name := range input.Components {
		if !name.Adjustable() || selected[name] {
			return command.Result{}, commands.Rejection{Code: "component_not_adjustable"}
		}
		selected[name] = true
	}
	postings, err := s.prepareResolutionPostings(current, selected)
	if err != nil {
		return command.Result{}, err
	}
	zone, err := s.repository.AccountTimezone(ctx, principal)
	if err != nil {
		return command.Result{}, err
	}
	date, err := current.SourceAsOf.DateIn(zone)
	if err != nil {
		return command.Result{}, err
	}
	now := s.now()
	adjustmentID := s.newID()
	revision := ledger.Revision{
		OperationID: adjustmentID, Revision: 1, ActorID: principal.UserID(), Reason: input.Reason,
		Type: ledger.Adjustment, State: ledger.Posted, OccurredAt: current.SourceAsOf,
		PostedAt: now, RecordedAt: now, CashDate: date, Timezone: zone,
		Origin: "manual", FeeKnowledge: ledger.KnownFees, PayerState: "not_applicable",
		AllocationReason: "not_applicable", HumanOverride: true,
		Protections:   map[ledger.Field]ledger.Protection{ledger.PrincipalField: {Revision: 1}},
		FieldVersions: map[ledger.Field]uint64{ledger.PrincipalField: 1}, Postings: postings,
	}
	resolvedComponents, err := s.applyResolutionEffects(current, revision)
	if err != nil {
		return command.Result{}, err
	}
	if err = s.writer.Append(ctx, principal, revision, 0); err != nil {
		return command.Result{}, s.reject(err)
	}
	resolved := current
	resolved.Revision++
	resolved.Lifecycle = reconciliation.Resolved
	resolved.Result = reconciliation.Balanced
	resolved.EvaluatedAt = now
	resolved.RelatedOperationIDs = append(resolved.RelatedOperationIDs, adjustmentID)
	resolved.Resolution = &reconciliation.Resolution{ActorID: string(principal.UserID()), Reason: input.Reason, AdjustmentTransactionID: adjustmentID, At: now, Components: append([]reconciliation.ComponentName(nil), input.Components...)}
	resolved.Components = resolvedComponents
	if err = s.repository.SaveResolution(ctx, principal, resolved); err != nil {
		return command.Result{}, err
	}
	if err = s.repository.EmitEvent(ctx, "reconciliation", resolved.ID, resolved.Revision, "reconciliation.resolved"); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "reconciliation", ResourceID: resolved.ID, Revision: resolved.Revision}, nil
}

func (s *Service) prepareResolutionPostings(current reconciliation.Reconciliation, selected map[reconciliation.ComponentName]bool) ([]ledger.Posting, error) {
	postings := []ledger.Posting{}
	for _, component := range current.Components {
		name := component.Name
		if !selected[name] {
			continue
		}
		difference, known := component.Difference.Value()
		if !known {
			return nil, commands.Rejection{Code: "reconciliation_not_ready"}
		}
		if difference.Sign() == 0 {
			return nil, commands.Rejection{Code: "no_change"}
		}
		posting := ledger.Posting{AccountID: current.AccountID, Money: difference, Role: ledger.Principal, Funding: ledger.OwnFunds, Treatment: ledger.Movement}
		if name == reconciliation.Debt {
			zero, _ := money.NewMoney("0", difference.Asset())
			posting.Money, _ = zero.Subtract(difference)
			posting.Funding = ledger.CreditFunds
		}
		postings = append(postings, posting)
	}
	return postings, nil
}

func (s *Service) applyResolutionEffects(current reconciliation.Reconciliation, revision ledger.Revision) ([]reconciliation.Component, error) {
	effects, err := revision.BalanceEffects()
	if err != nil {
		return nil, err
	}
	deltas := map[reconciliation.ComponentName]money.Money{}
	for _, effect := range effects {
		if effect.AccountID != current.AccountID {
			return nil, reconciliation.ErrInvalidReconciliation
		}
		for _, item := range []struct {
			name  reconciliation.ComponentName
			value reporting.Amount
		}{{reconciliation.Owned, effect.Owned}, {reconciliation.Available, effect.Available}, {reconciliation.Locked, effect.Locked}, {reconciliation.Debt, effect.Debt}} {
			value, known := item.value.Value()
			if !known {
				return nil, commands.Rejection{Code: "reconciliation_not_ready"}
			}
			if previous, exists := deltas[item.name]; exists {
				value, err = previous.Add(value)
				if err != nil {
					return nil, err
				}
			}
			deltas[item.name] = value
		}
	}
	resolved := make([]reconciliation.Component, 0, len(current.Components))
	for _, component := range current.Components {
		source, sourceKnown := component.Source.Value()
		ledgerValue, ledgerKnown := component.Ledger.Value()
		if !sourceKnown || !ledgerKnown {
			return nil, commands.Rejection{Code: "reconciliation_not_ready"}
		}
		if delta, changed := deltas[component.Name]; changed {
			var err error
			ledgerValue, err = ledgerValue.Add(delta)
			if err != nil {
				return nil, err
			}
		}
		remaining, err := source.Subtract(ledgerValue)
		if err != nil {
			return nil, err
		}
		if remaining.Sign() != 0 {
			return nil, commands.Rejection{Code: "component_not_adjustable"}
		}
		ledgerAmount, _ := reporting.KnownAmount(ledgerValue)
		difference, _ := reporting.KnownAmount(remaining)
		resolved = append(resolved, reconciliation.Component{Name: component.Name, Source: component.Source, Ledger: ledgerAmount, Difference: difference})
	}
	return resolved, nil
}

func (s *Service) DispatchReplay(ctx context.Context, principal household.Principal, id string) error {
	if s.scheduler == nil || s.transactions == nil {
		return reconciliation.ErrNotReady
	}
	var current reconciliation.Reconciliation
	err := s.transactions.WithinHousehold(ctx, principal, func(ctx context.Context) error {
		var readErr error
		current, readErr = s.repository.Reconciliation(ctx, principal, id)
		return readErr
	})
	if err != nil {
		return err
	}
	if current.Lifecycle != reconciliation.Open || current.Replay.Status != reconciliation.ReplayPending || current.Replay.JobID != "" {
		return reconciliation.ErrNotReady
	}
	job, err := s.scheduler.RequestReplay(ctx, principal, current.Replay.ConnectionID, current.Replay.Binding, current.Replay.AdmissionRevision, current.Replay.ConnectionGeneration, s.now().Time().Add(23*time.Hour), current.Replay.RequestID, current.Replay.From.Time(), current.Replay.To.Time())
	status, reason := reconciliation.ReplayPending, ""
	if errors.Is(err, connections.ErrProviderNotAdmitted) {
		status, reason, err = reconciliation.ReplayUnavailable, "provider_not_admitted", nil
	} else if errors.Is(err, connections.ErrSecretAccess) {
		status, reason, err = reconciliation.ReplayUnavailable, "source_reauth_required", nil
	}
	if err != nil {
		return err
	}
	return s.transactions.WithinHousehold(ctx, principal, func(ctx context.Context) error {
		return s.updateReplay(ctx, principal, current.Replay.RequestID, status, job.ID, reason)
	})
}

// RecordReplayOutcome runs inside the admitted job-attempt transaction. Completion
// additionally requires CommitPage to have fenced the page before invoking it.
func (s *Service) RecordReplayOutcome(ctx context.Context, principal household.Principal, id string, issued jobs.Job, status reconciliation.ReplayStatus, reason string) error {
	if issued.ID == "" || reason == "" || status != reconciliation.ReplayCompleted && status != reconciliation.ReplayFailed {
		return reconciliation.ErrInvalidReconciliation
	}
	current, err := s.repository.Reconciliation(ctx, principal, id)
	if err != nil {
		return err
	}
	if current.Lifecycle != reconciliation.Open || current.Replay.Status != reconciliation.ReplayPending || current.Replay.JobID != issued.ID {
		return reconciliation.ErrNotReady
	}
	if err = s.repository.FenceReplayOutcome(ctx, principal, current.Replay, issued, status); err != nil {
		return err
	}
	return s.updateReplay(ctx, principal, current.Replay.RequestID, status, issued.ID, reason)
}

func (s *Service) updateReplay(ctx context.Context, principal household.Principal, requestID string, status reconciliation.ReplayStatus, jobID, reason string) error {
	updated, changed, err := s.repository.UpdateReplay(ctx, principal, requestID, status, jobID, reason)
	if err != nil || !changed {
		return err
	}
	return s.repository.EmitEvent(ctx, "reconciliation", updated.ID, updated.Revision, "reconciliation.changed")
}

func (s *Service) reject(err error) error {
	switch {
	case errors.Is(err, reconciliation.ErrNotFound), errors.Is(err, account.ErrNotFound), errors.Is(err, ledger.ErrNotFound):
		return commands.Rejection{Code: "not_found"}
	case errors.Is(err, reconciliation.ErrNotReady):
		return commands.Rejection{Code: "reconciliation_not_ready"}
	case errors.Is(err, reconciliation.ErrComponentNotAdjustable):
		return commands.Rejection{Code: "component_not_adjustable"}
	case errors.Is(err, command.ErrVersionConflict):
		return commands.Rejection{Code: "version_conflict"}
	case errors.Is(err, household.ErrForbidden):
		return commands.Rejection{Code: "forbidden"}
	case errors.Is(err, money.ErrAssetMismatch):
		return commands.Rejection{Code: "asset_mismatch"}
	case errors.Is(err, money.ErrInvalidMoney), errors.Is(err, reconciliation.ErrInvalidReconciliation):
		return commands.Rejection{Code: "invalid_request"}
	}
	return err
}
