package domain

import (
	"errors"
	"slices"
	"sort"
	"time"
	"unicode/utf8"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

var (
	ErrInvalidReconciliation  = errors.New("invalid reconciliation")
	ErrNotFound               = errors.New("reconciliation not found")
	ErrNotReady               = errors.New("reconciliation is not ready for resolution")
	ErrComponentNotAdjustable = errors.New("reconciliation component is not adjustable")
)

type ComponentName string

const (
	Owned     ComponentName = "owned"
	Available ComponentName = "available"
	Locked    ComponentName = "locked"
	Debt      ComponentName = "debt"
)

func (n ComponentName) Valid() bool {
	switch n {
	case Owned, Available, Locked, Debt:
		return true
	}
	return false
}

func (n ComponentName) Adjustable() bool { return n == Owned || n == Debt }

type Lifecycle string

const (
	Open       Lifecycle = "open"
	Resolved   Lifecycle = "resolved"
	Superseded Lifecycle = "superseded"
)

func (v Lifecycle) Valid() bool { return v == Open || v == Resolved || v == Superseded }

type Result string

const (
	Balanced   Result = "balanced"
	Discrepant Result = "discrepant"
	Incomplete Result = "incomplete"
)

func (v Result) Valid() bool { return v == Balanced || v == Discrepant || v == Incomplete }

type ReplayStatus string

const (
	ReplayNotRequired ReplayStatus = "not_required"
	ReplayPending     ReplayStatus = "pending"
	ReplayCompleted   ReplayStatus = "completed"
	ReplayFailed      ReplayStatus = "failed"
	ReplayUnavailable ReplayStatus = "unavailable"
)

func (v ReplayStatus) Valid() bool {
	switch v {
	case ReplayNotRequired, ReplayPending, ReplayCompleted, ReplayFailed, ReplayUnavailable:
		return true
	}
	return false
}

type Component struct {
	Name                       ComponentName
	Source, Ledger, Difference reporting.Amount
}

func (c Component) Validate(asset money.Asset) error {
	if !c.Name.Valid() {
		return ErrInvalidReconciliation
	}
	for _, value := range []reporting.Amount{c.Source, c.Ledger, c.Difference} {
		if err := value.Validate(); err != nil {
			return err
		}
		if amount, known := value.Value(); known && amount.Asset() != asset {
			return money.ErrAssetMismatch
		}
	}
	if source, sourceKnown := c.Source.Value(); sourceKnown {
		if ledger, ledgerKnown := c.Ledger.Value(); ledgerKnown {
			difference, differenceKnown := c.Difference.Value()
			if !differenceKnown {
				return ErrInvalidReconciliation
			}
			expected, err := source.Subtract(ledger)
			if err != nil {
				return err
			}
			comparison, err := expected.Compare(difference)
			if err != nil || comparison != 0 {
				return ErrInvalidReconciliation
			}
		}
	}
	return nil
}

type Explanation struct {
	Code, Message string
}

func (e Explanation) Validate() error {
	if utf8.RuneCountInString(e.Code) < 1 || utf8.RuneCountInString(e.Code) > 100 || utf8.RuneCountInString(e.Message) < 1 || utf8.RuneCountInString(e.Message) > 2000 {
		return ErrInvalidReconciliation
	}
	return nil
}

type Replay struct {
	RequestID, ConnectionID, JobID, Reason string
	Status                                 ReplayStatus
	From, To                               calendar.Instant
	Binding                                connections.Binding
	AdmissionRevision                      int64
	ConnectionGeneration                   uint64
}

func (r Replay) Validate() error {
	if !r.Status.Valid() {
		return ErrInvalidReconciliation
	}
	if r.Status == ReplayNotRequired {
		if r.RequestID != "" || r.ConnectionID != "" || r.JobID != "" || r.Reason != "" || r.From.String() != "" || r.To.String() != "" || r.Binding != (connections.Binding{}) || r.AdmissionRevision != 0 || r.ConnectionGeneration != 0 {
			return ErrInvalidReconciliation
		}
		return nil
	}
	if r.RequestID == "" || r.ConnectionID == "" || r.From.String() == "" || r.To.String() == "" || !r.From.Time().Before(r.To.Time()) || r.To.Time().Sub(r.From.Time()) > 90*24*time.Hour {
		return ErrInvalidReconciliation
	}
	if r.Status == ReplayPending && r.Reason != "" || r.Status != ReplayPending && r.Reason == "" {
		return ErrInvalidReconciliation
	}
	if r.Status == ReplayUnavailable && r.JobID != "" || (r.Status == ReplayCompleted || r.Status == ReplayFailed) && r.JobID == "" {
		return ErrInvalidReconciliation
	}
	if r.Binding.Validate() != nil || r.AdmissionRevision < 1 || r.ConnectionGeneration < 1 {
		return ErrInvalidReconciliation
	}
	return nil
}

func (r Replay) Transition(status ReplayStatus, jobID, reason string) (Replay, bool, error) {
	next := r
	next.Status, next.JobID, next.Reason = status, jobID, reason
	if sameReplay(r, next) {
		return r, false, nil
	}
	if r.Status != ReplayPending || r.JobID != "" && r.JobID != jobID || next.Validate() != nil {
		return r, false, ErrNotReady
	}
	return next, true, nil
}

type Resolution struct {
	ActorID, Reason, AdjustmentTransactionID string
	At                                       calendar.Instant
	Components                               []ComponentName
}

func (r Resolution) Validate() error {
	if r.ActorID == "" || utf8.RuneCountInString(r.Reason) < 1 || utf8.RuneCountInString(r.Reason) > 2000 || r.AdjustmentTransactionID == "" || r.At.String() == "" || len(r.Components) < 1 || len(r.Components) > 2 {
		return ErrInvalidReconciliation
	}
	seen := map[ComponentName]bool{}
	for _, component := range r.Components {
		if !component.Adjustable() || seen[component] {
			return ErrComponentNotAdjustable
		}
		seen[component] = true
	}
	return nil
}

type Reconciliation struct {
	ID, AccountID, ObservationID string
	Revision                     uint64
	Lifecycle                    Lifecycle
	Result                       Result
	SourceAsOf, EvaluatedAt      calendar.Instant
	Coverage                     reporting.Coverage
	Freshness                    reporting.Freshness
	Components                   []Component
	Explanations                 []Explanation
	RelatedOperationIDs          []string
	Replay                       Replay
	Resolution                   *Resolution
}

type Evaluation struct {
	ID, AccountID, ObservationID string
	Revision                     uint64
	SourceAsOf, EvaluatedAt      calendar.Instant
	Asset                        money.Asset
	Source, Ledger               account.Amounts
	Coverage                     reporting.Coverage
	Freshness                    reporting.Freshness
	Replay                       Replay
	RelatedOperationIDs          []string
}

func Evaluate(input Evaluation) (Reconciliation, error) {
	result := Reconciliation{
		ID: input.ID, AccountID: input.AccountID, ObservationID: input.ObservationID,
		Revision: input.Revision, Lifecycle: Open, SourceAsOf: input.SourceAsOf,
		EvaluatedAt: input.EvaluatedAt, Coverage: input.Coverage, Freshness: input.Freshness,
		Replay: input.Replay, RelatedOperationIDs: uniqueSorted(input.RelatedOperationIDs),
	}
	fields := []struct {
		name           ComponentName
		source, ledger reporting.Amount
	}{
		{Owned, input.Source.Owned, input.Ledger.Owned},
		{Available, input.Source.Available, input.Ledger.Available},
		{Locked, input.Source.Locked, input.Ledger.Locked},
		{Debt, input.Source.Debt, input.Ledger.Debt},
	}
	allKnown := true
	hasDifference := false
	for _, field := range fields {
		difference, err := subtract(field.source, field.ledger)
		if err != nil {
			return Reconciliation{}, err
		}
		component := Component{Name: field.name, Source: field.source, Ledger: field.ledger, Difference: difference}
		if err = component.Validate(input.Asset); err != nil {
			return Reconciliation{}, err
		}
		if value, known := difference.Value(); !known {
			allKnown = false
		} else if value.Sign() != 0 {
			hasDifference = true
		}
		result.Components = append(result.Components, component)
	}
	switch {
	case input.Coverage.State() != reporting.Complete || !allKnown:
		result.Result = Incomplete
		result.Explanations = append(result.Explanations, Explanation{Code: "history_incomplete", Message: "The ledger or source does not cover every required balance component at this instant."})
	case hasDifference:
		result.Result = Discrepant
		result.Explanations = append(result.Explanations, Explanation{Code: "balance_difference", Message: "The source and ledger differ at the same source timestamp."})
	default:
		result.Result = Balanced
	}
	if input.Freshness == reporting.Stale {
		result.Explanations = append(result.Explanations, Explanation{Code: "source_stale", Message: "The source observation is stale; equality applies only to its recorded timestamp."})
	}
	if err := result.Validate(input.Asset); err != nil {
		return Reconciliation{}, err
	}
	return result, nil
}

func subtract(source, ledger reporting.Amount) (reporting.Amount, error) {
	left, leftKnown := source.Value()
	right, rightKnown := ledger.Value()
	if !leftKnown {
		return reporting.MissingAmount(source.Knowledge(), source.Reason())
	}
	if !rightKnown {
		return reporting.MissingAmount(ledger.Knowledge(), ledger.Reason())
	}
	value, err := left.Subtract(right)
	if err != nil {
		return reporting.Amount{}, err
	}
	return reporting.KnownAmount(value)
}

func (r Reconciliation) Validate(asset money.Asset) error {
	if r.ID == "" || r.AccountID == "" || r.ObservationID == "" || r.Revision < 1 || r.Revision > 9007199254740991 || !r.Lifecycle.Valid() || !r.Result.Valid() || r.SourceAsOf.String() == "" || r.EvaluatedAt.String() == "" || r.SourceAsOf.Time().After(r.EvaluatedAt.Time()) || len(r.Components) != 4 {
		return ErrInvalidReconciliation
	}
	if _, err := reporting.NewCoverage(r.Coverage.State(), r.Coverage.Reasons()); err != nil {
		return err
	}
	if _, err := reporting.ParseFreshness(string(r.Freshness)); err != nil {
		return err
	}
	seen := map[ComponentName]bool{}
	for _, component := range r.Components {
		if seen[component.Name] || component.Validate(asset) != nil {
			return ErrInvalidReconciliation
		}
		seen[component.Name] = true
	}
	for _, explanation := range r.Explanations {
		if err := explanation.Validate(); err != nil {
			return err
		}
	}
	if err := r.Replay.Validate(); err != nil {
		return err
	}
	if r.Lifecycle == Resolved {
		if r.Resolution == nil || r.Result != Balanced {
			return ErrInvalidReconciliation
		}
		if err := r.Resolution.Validate(); err != nil {
			return err
		}
	} else if r.Resolution != nil {
		return ErrInvalidReconciliation
	}
	return nil
}

func (r Reconciliation) Component(name ComponentName) (Component, bool) {
	for _, component := range r.Components {
		if component.Name == name {
			return component, true
		}
	}
	return Component{}, false
}

func (r Reconciliation) SameEvaluation(other Reconciliation) bool {
	if r.ObservationID != other.ObservationID || r.Lifecycle != other.Lifecycle || r.Result != other.Result || r.Coverage.State() != other.Coverage.State() || !slices.Equal(r.Coverage.Reasons(), other.Coverage.Reasons()) || r.Freshness != other.Freshness || !sameReplay(r.Replay, other.Replay) || !slices.Equal(r.RelatedOperationIDs, other.RelatedOperationIDs) || len(r.Explanations) != len(other.Explanations) || len(r.Components) != len(other.Components) {
		return false
	}
	for index := range r.Explanations {
		if r.Explanations[index] != other.Explanations[index] {
			return false
		}
	}
	for _, component := range r.Components {
		right, ok := other.Component(component.Name)
		if !ok || !sameAmount(component.Source, right.Source) || !sameAmount(component.Ledger, right.Ledger) || !sameAmount(component.Difference, right.Difference) {
			return false
		}
	}
	return true
}

func sameReplay(left, right Replay) bool {
	return left.RequestID == right.RequestID && left.ConnectionID == right.ConnectionID && left.JobID == right.JobID && left.Reason == right.Reason && left.Status == right.Status && left.From.String() == right.From.String() && left.To.String() == right.To.String() && left.Binding == right.Binding && left.AdmissionRevision == right.AdmissionRevision && left.ConnectionGeneration == right.ConnectionGeneration
}

func sameAmount(left, right reporting.Amount) bool {
	if left.Knowledge() != right.Knowledge() || left.Reason() != right.Reason() {
		return false
	}
	a, known := left.Value()
	if !known {
		return true
	}
	b, ok := right.Value()
	if !ok {
		return false
	}
	comparison, err := a.Compare(b)
	return err == nil && comparison == 0
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		if value != "" {
			seen[value] = true
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
