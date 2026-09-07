package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reconciliationapp "github.com/pchkauu/want-keep/backend/internal/reconciliation/application"
	reconciliation "github.com/pchkauu/want-keep/backend/internal/reconciliation/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func (s *Store) SaveReconciliation(ctx context.Context, p household.Principal, value reconciliation.Reconciliation) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.principal != p {
		return household.ErrForbidden
	}
	entry, err := s.Account(ctx, p, value.AccountID)
	if err != nil {
		return err
	}
	if err = value.Validate(entry.Asset); err != nil {
		return err
	}
	var current uint64
	err = scope.tx.QueryRow(ctx, `SELECT current_revision FROM want_keep.reconciliations WHERE household_id=$1 AND id=$2 FOR UPDATE`, p.HouseholdID(), value.ID).Scan(&current)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if value.Revision != 1 {
			return command.ErrVersionConflict
		}
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.reconciliations(household_id,id,account_id,observation_id,current_revision,lifecycle) VALUES($1,$2,$3,$4,$5,$6)`, p.HouseholdID(), value.ID, value.AccountID, value.ObservationID, value.Revision, value.Lifecycle)
	case err != nil:
		return err
	default:
		if current >= command.MaxRevision || value.Revision != current+1 {
			return command.ErrVersionConflict
		}
		tag, updateErr := scope.tx.Exec(ctx, `UPDATE want_keep.reconciliations SET current_revision=$3,lifecycle=$4 WHERE household_id=$1 AND id=$2 AND current_revision=$5`, p.HouseholdID(), value.ID, value.Revision, value.Lifecycle, current)
		if updateErr != nil {
			return updateErr
		}
		if tag.RowsAffected() != 1 {
			return command.ErrVersionConflict
		}
	}
	if value.Replay.Status != reconciliation.ReplayNotRequired {
		if err = s.saveReplayRequest(ctx, p, value); err != nil {
			return err
		}
	}
	return s.insertReconciliationRevision(ctx, p, value, entry.Asset)
}

func (s *Store) SaveResolution(ctx context.Context, p household.Principal, value reconciliation.Reconciliation) error {
	if value.Resolution == nil || value.Lifecycle != reconciliation.Resolved {
		return reconciliation.ErrInvalidReconciliation
	}
	return s.SaveReconciliation(ctx, p, value)
}

func (s *Store) saveReplayRequest(ctx context.Context, p household.Principal, value reconciliation.Reconciliation) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	replay := value.Replay
	binding, err := json.Marshal(bindingFromDomain(replay.Binding))
	if err != nil {
		return err
	}
	from, fromNS := splitInstant(replay.From)
	to, toNS := splitInstant(replay.To)
	tag, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.reconciliation_replay_requests(household_id,id,reconciliation_id,connection_id,status,reason,job_id,range_from,range_from_ns,range_to,range_to_ns,binding,admission_revision,connection_generation) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')::uuid,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT(household_id,id) DO UPDATE SET status=EXCLUDED.status,reason=EXCLUDED.reason,job_id=EXCLUDED.job_id WHERE want_keep.reconciliation_replay_requests.reconciliation_id=EXCLUDED.reconciliation_id AND want_keep.reconciliation_replay_requests.connection_id=EXCLUDED.connection_id AND (want_keep.reconciliation_replay_requests.range_from,want_keep.reconciliation_replay_requests.range_from_ns,want_keep.reconciliation_replay_requests.range_to,want_keep.reconciliation_replay_requests.range_to_ns)=(EXCLUDED.range_from,EXCLUDED.range_from_ns,EXCLUDED.range_to,EXCLUDED.range_to_ns) AND want_keep.reconciliation_replay_requests.binding=EXCLUDED.binding AND want_keep.reconciliation_replay_requests.admission_revision=EXCLUDED.admission_revision AND want_keep.reconciliation_replay_requests.connection_generation=EXCLUDED.connection_generation`, p.HouseholdID(), replay.RequestID, value.ID, replay.ConnectionID, replay.Status, replay.Reason, replay.JobID, from, fromNS, to, toNS, binding, replay.AdmissionRevision, replay.ConnectionGeneration)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return command.ErrVersionConflict
	}
	return nil
}

func (s *Store) insertReconciliationRevision(ctx context.Context, p household.Principal, value reconciliation.Reconciliation, asset money.Asset) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	sourceAt, sourceNS := splitInstant(value.SourceAsOf)
	evaluatedAt, evaluatedNS := splitInstant(value.EvaluatedAt)
	reasons := value.Coverage.Reasons()
	if reasons == nil {
		reasons = []string{}
	}
	var replayID any
	if value.Replay.Status != reconciliation.ReplayNotRequired {
		replayID = value.Replay.RequestID
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.reconciliation_revisions(household_id,reconciliation_id,revision,lifecycle,result,replay_status,replay_request_id,source_as_of,source_as_of_ns,evaluated_at,evaluated_at_ns,coverage,coverage_reasons,freshness) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')::uuid,$8,$9,$10,$11,$12,$13,$14)`, p.HouseholdID(), value.ID, value.Revision, value.Lifecycle, value.Result, value.Replay.Status, replayID, sourceAt, sourceNS, evaluatedAt, evaluatedNS, value.Coverage.State(), reasons, value.Freshness)
	if err != nil {
		return err
	}
	for _, component := range value.Components {
		columns := []any{p.HouseholdID(), value.ID, value.Revision, component.Name}
		for _, amount := range []reporting.Amount{component.Source, component.Ledger, component.Difference} {
			var decimal any
			if known, ok := amount.Value(); ok {
				decimal = known.Amount()
			}
			columns = append(columns, amount.Knowledge(), decimal, amount.Reason())
		}
		columns = append(columns, asset)
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.reconciliation_components(household_id,reconciliation_id,revision,component,source_knowledge,source_amount,source_reason,ledger_knowledge,ledger_amount,ledger_reason,difference_knowledge,difference_amount,difference_reason,asset) VALUES($1,$2,$3,$4,$5,$6::numeric,$7,$8,$9::numeric,$10,$11,$12::numeric,$13,$14)`, columns...)
		if err != nil {
			return err
		}
	}
	for position, explanation := range value.Explanations {
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.reconciliation_explanations(household_id,reconciliation_id,revision,position,code,message) VALUES($1,$2,$3,$4,$5,$6)`, p.HouseholdID(), value.ID, value.Revision, position, explanation.Code, explanation.Message); err != nil {
			return err
		}
	}
	for _, operationID := range value.RelatedOperationIDs {
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.reconciliation_operations(household_id,reconciliation_id,revision,operation_id) VALUES($1,$2,$3,$4)`, p.HouseholdID(), value.ID, value.Revision, operationID); err != nil {
			return err
		}
	}
	if value.Resolution != nil {
		at, ns := splitInstant(value.Resolution.At)
		components := make([]string, len(value.Resolution.Components))
		for index, component := range value.Resolution.Components {
			components[index] = string(component)
		}
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.reconciliation_resolutions(household_id,reconciliation_id,revision,actor_id,reason,adjustment_operation_id,components,recorded_at,recorded_at_ns) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, p.HouseholdID(), value.ID, value.Revision, value.Resolution.ActorID, value.Resolution.Reason, value.Resolution.AdjustmentTransactionID, components, at, ns)
	}
	return err
}

func (s *Store) ActiveReconciliation(ctx context.Context, p household.Principal, accountID string) (reconciliation.Reconciliation, bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return reconciliation.Reconciliation{}, false, err
	}
	var id string
	err = q.QueryRow(ctx, `SELECT id FROM want_keep.reconciliations WHERE household_id=$1 AND account_id=$2 AND lifecycle='open'`, p.HouseholdID(), accountID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return reconciliation.Reconciliation{}, false, nil
	}
	if err != nil {
		return reconciliation.Reconciliation{}, false, err
	}
	value, err := s.Reconciliation(ctx, p, id)
	return value, err == nil, err
}

func (s *Store) LastConfirmedReconciliation(ctx context.Context, p household.Principal, accountID string, before calendar.Instant) (calendar.Instant, bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return calendar.Instant{}, false, err
	}
	beforeAt, beforeNS := splitInstant(before)
	var at time.Time
	var ns int16
	err = q.QueryRow(ctx, `SELECT v.source_as_of,v.source_as_of_ns FROM want_keep.reconciliations r JOIN want_keep.reconciliation_revisions v ON (v.household_id,v.reconciliation_id,v.revision)=(r.household_id,r.id,r.current_revision) WHERE r.household_id=$1 AND r.account_id=$2 AND v.result='balanced' AND v.coverage='complete' AND (v.source_as_of,v.source_as_of_ns)<($3,$4) ORDER BY v.source_as_of DESC,v.source_as_of_ns DESC LIMIT 1`, p.HouseholdID(), accountID, beforeAt, beforeNS).Scan(&at, &ns)
	if errors.Is(err, pgx.ErrNoRows) {
		return calendar.Instant{}, false, nil
	}
	if err != nil {
		return calendar.Instant{}, false, err
	}
	result, err := restoreInstant(at, ns)
	return result, err == nil, err
}

func (s *Store) Reconciliation(ctx context.Context, p household.Principal, id string) (reconciliation.Reconciliation, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return reconciliation.Reconciliation{}, err
	}
	var revision uint64
	err = q.QueryRow(ctx, `SELECT current_revision FROM want_keep.reconciliations WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id).Scan(&revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return reconciliation.Reconciliation{}, reconciliation.ErrNotFound
	}
	if err != nil {
		return reconciliation.Reconciliation{}, err
	}
	return s.reconciliationRevision(ctx, p, id, revision)
}

func (s *Store) reconciliationRevision(ctx context.Context, p household.Principal, id string, revision uint64) (reconciliation.Reconciliation, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return reconciliation.Reconciliation{}, err
	}
	value := reconciliation.Reconciliation{ID: id, Revision: revision}
	var sourceAt, evaluatedAt time.Time
	var sourceNS, evaluatedNS int16
	var coverage, freshness string
	var reasons []string
	err = q.QueryRow(ctx, `SELECT r.account_id,r.observation_id,v.lifecycle,v.result,v.replay_status,v.source_as_of,v.source_as_of_ns,v.evaluated_at,v.evaluated_at_ns,v.coverage,v.coverage_reasons,v.freshness FROM want_keep.reconciliations r JOIN want_keep.reconciliation_revisions v ON (v.household_id,v.reconciliation_id,v.revision)=(r.household_id,r.id,$3) WHERE r.household_id=$1 AND r.id=$2`, p.HouseholdID(), id, revision).Scan(&value.AccountID, &value.ObservationID, &value.Lifecycle, &value.Result, &value.Replay.Status, &sourceAt, &sourceNS, &evaluatedAt, &evaluatedNS, &coverage, &reasons, &freshness)
	if errors.Is(err, pgx.ErrNoRows) {
		return value, reconciliation.ErrNotFound
	}
	if err != nil {
		return value, err
	}
	value.SourceAsOf, err = restoreInstant(sourceAt, sourceNS)
	if err != nil {
		return value, err
	}
	value.EvaluatedAt, err = restoreInstant(evaluatedAt, evaluatedNS)
	if err != nil {
		return value, err
	}
	value.Coverage, err = reporting.NewCoverage(reporting.CoverageState(coverage), reasons)
	if err != nil {
		return value, err
	}
	value.Freshness, err = reporting.ParseFreshness(freshness)
	if err != nil {
		return value, err
	}
	rows, err := q.Query(ctx, `SELECT component,source_knowledge,source_amount::text,source_reason,ledger_knowledge,ledger_amount::text,ledger_reason,difference_knowledge,difference_amount::text,difference_reason,asset FROM want_keep.reconciliation_components WHERE household_id=$1 AND reconciliation_id=$2 AND revision=$3 ORDER BY component`, p.HouseholdID(), id, revision)
	if err != nil {
		return value, err
	}
	for rows.Next() {
		var component reconciliation.Component
		var source, ledgerValue, difference accountAmountRow
		if err = rows.Scan(&component.Name, &source.knowledge, &source.amount, &source.reason, &ledgerValue.knowledge, &ledgerValue.amount, &ledgerValue.reason, &difference.knowledge, &difference.amount, &difference.reason, &source.asset); err != nil {
			rows.Close()
			return value, err
		}
		ledgerValue.asset, difference.asset = source.asset, source.asset
		component.Source, err = source.value()
		if err == nil {
			component.Ledger, err = ledgerValue.value()
		}
		if err == nil {
			component.Difference, err = difference.value()
		}
		if err != nil {
			rows.Close()
			return value, err
		}
		value.Components = append(value.Components, component)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return value, err
	}
	rows, err = q.Query(ctx, `SELECT code,message FROM want_keep.reconciliation_explanations WHERE household_id=$1 AND reconciliation_id=$2 AND revision=$3 ORDER BY position`, p.HouseholdID(), id, revision)
	if err != nil {
		return value, err
	}
	for rows.Next() {
		var explanation reconciliation.Explanation
		if err = rows.Scan(&explanation.Code, &explanation.Message); err != nil {
			rows.Close()
			return value, err
		}
		value.Explanations = append(value.Explanations, explanation)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return value, err
	}
	rows, err = q.Query(ctx, `SELECT operation_id FROM want_keep.reconciliation_operations WHERE household_id=$1 AND reconciliation_id=$2 AND revision=$3 ORDER BY operation_id`, p.HouseholdID(), id, revision)
	if err != nil {
		return value, err
	}
	for rows.Next() {
		var operationID string
		if err = rows.Scan(&operationID); err != nil {
			rows.Close()
			return value, err
		}
		value.RelatedOperationIDs = append(value.RelatedOperationIDs, operationID)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return value, err
	}
	if value.Replay.Status != reconciliation.ReplayNotRequired {
		if err = s.loadReplay(ctx, q, p, id, &value.Replay); err != nil {
			return value, err
		}
	}
	var resolution reconciliation.Resolution
	var at time.Time
	var ns int16
	var components []string
	err = q.QueryRow(ctx, `SELECT actor_id,reason,adjustment_operation_id,components,recorded_at,recorded_at_ns FROM want_keep.reconciliation_resolutions WHERE household_id=$1 AND reconciliation_id=$2 AND revision=$3`, p.HouseholdID(), id, revision).Scan(&resolution.ActorID, &resolution.Reason, &resolution.AdjustmentTransactionID, &components, &at, &ns)
	if err == nil {
		resolution.At, err = restoreInstant(at, ns)
		for _, component := range components {
			resolution.Components = append(resolution.Components, reconciliation.ComponentName(component))
		}
		value.Resolution = &resolution
	} else if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	if err != nil {
		return value, err
	}
	entry, err := s.Account(ctx, p, value.AccountID)
	if err != nil {
		return value, err
	}
	return value, value.Validate(entry.Asset)
}

func (s *Store) loadReplay(ctx context.Context, q reader, p household.Principal, reconciliationID string, replay *reconciliation.Replay) error {
	var from, to time.Time
	var fromNS, toNS int16
	var binding []byte
	err := q.QueryRow(ctx, `SELECT id,connection_id,status,reason,COALESCE(job_id::text,''),range_from,range_from_ns,range_to,range_to_ns,binding,admission_revision,connection_generation FROM want_keep.reconciliation_replay_requests WHERE household_id=$1 AND reconciliation_id=$2`, p.HouseholdID(), reconciliationID).Scan(&replay.RequestID, &replay.ConnectionID, &replay.Status, &replay.Reason, &replay.JobID, &from, &fromNS, &to, &toNS, &binding, &replay.AdmissionRevision, &replay.ConnectionGeneration)
	if err != nil {
		return err
	}
	replay.From, err = restoreInstant(from, fromNS)
	if err == nil {
		replay.To, err = restoreInstant(to, toNS)
	}
	if err == nil {
		replay.Binding, err = decodeBinding(binding)
	}
	return err
}

func (s *Store) Reconciliations(ctx context.Context, p household.Principal, filter reconciliationapp.Filter, cursor reconciliationapp.Cursor, limit int) ([]reconciliation.Reconciliation, *reconciliationapp.Cursor, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, nil, err
	}
	var cursorAt any
	var cursorNS any
	var cursorID any
	if cursor.At.String() != "" {
		cursorAt, cursorNS = splitInstant(cursor.At)
		cursorID = cursor.ID
	}
	rows, err := q.Query(ctx, `SELECT r.id,v.evaluated_at,v.evaluated_at_ns FROM want_keep.reconciliations r JOIN want_keep.reconciliation_revisions v ON (v.household_id,v.reconciliation_id,v.revision)=(r.household_id,r.id,r.current_revision) WHERE r.household_id=$1 AND ($2='' OR r.account_id=NULLIF($2,'')::uuid) AND ($3='' OR r.lifecycle=$3) AND ($4='' OR v.result=$4) AND ($5::timestamptz IS NULL OR (v.evaluated_at,v.evaluated_at_ns,r.id)<($5,$6,$7::uuid)) ORDER BY v.evaluated_at DESC,v.evaluated_at_ns DESC,r.id DESC LIMIT $8`, p.HouseholdID(), filter.AccountID, filter.Lifecycle, filter.Result, cursorAt, cursorNS, cursorID, limit+1)
	if err != nil {
		return nil, nil, err
	}
	type reference struct {
		id string
		at time.Time
		ns int16
	}
	references := []reference{}
	for rows.Next() {
		var reference reference
		if err = rows.Scan(&reference.id, &reference.at, &reference.ns); err != nil {
			rows.Close()
			return nil, nil, err
		}
		references = append(references, reference)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, err
	}
	var next *reconciliationapp.Cursor
	if len(references) > limit {
		last := references[limit-1]
		instant, err := restoreInstant(last.at, last.ns)
		if err != nil {
			return nil, nil, err
		}
		next = &reconciliationapp.Cursor{At: instant, ID: last.id}
		references = references[:limit]
	}
	result := make([]reconciliation.Reconciliation, 0, len(references))
	for _, reference := range references {
		value, err := s.Reconciliation(ctx, p, reference.id)
		if err != nil {
			return nil, nil, err
		}
		result = append(result, value)
	}
	return result, next, nil
}

func (s *Store) UpdateReplay(ctx context.Context, p household.Principal, requestID string, status reconciliation.ReplayStatus, jobID, reason string) (reconciliation.Reconciliation, bool, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return reconciliation.Reconciliation{}, false, err
	}
	var reconciliationID string
	err = scope.tx.QueryRow(ctx, `SELECT reconciliation_id FROM want_keep.reconciliation_replay_requests WHERE household_id=$1 AND id=$2 FOR UPDATE`, p.HouseholdID(), requestID).Scan(&reconciliationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return reconciliation.Reconciliation{}, false, reconciliation.ErrNotFound
	}
	if err != nil {
		return reconciliation.Reconciliation{}, false, err
	}
	value, err := s.Reconciliation(ctx, p, reconciliationID)
	if err != nil {
		return reconciliation.Reconciliation{}, false, err
	}
	next, changed, err := value.Replay.Transition(status, jobID, reason)
	if err != nil || !changed {
		return value, false, err
	}
	value.Revision++
	value.EvaluatedAt, err = databaseInstant(ctx, scope.tx)
	if err != nil {
		return reconciliation.Reconciliation{}, false, err
	}
	value.Replay = next
	err = s.SaveReconciliation(ctx, p, value)
	return value, err == nil, err
}

func databaseInstant(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) (calendar.Instant, error) {
	var now time.Time
	if err := q.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		return calendar.Instant{}, err
	}
	return calendar.ParseInstant(now.UTC().Format(time.RFC3339Nano))
}
