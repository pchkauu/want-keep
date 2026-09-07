package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func (s *Store) Admission(ctx context.Context, provider, environment string) (connections.Admission, bool, error) {
	scope, err := s.scope(ctx)
	if err != nil {
		return connections.Admission{}, false, err
	}
	var state connections.Snapshot
	var binding []byte
	var providerResult, hostResult *string
	var providerAt, hostAt *time.Time
	var providerNS, hostNS *int16
	err = scope.tx.QueryRow(ctx, `SELECT binding,revision,provider_result,provider_at,provider_ns,host_result,host_at,host_ns FROM want_keep.deployment_admissions WHERE provider=$1 AND environment=$2`, provider, environment).Scan(&binding, &state.Revision, &providerResult, &providerAt, &providerNS, &hostResult, &hostAt, &hostNS)
	if errors.Is(err, pgx.ErrNoRows) {
		return connections.Admission{}, false, nil
	}
	if err != nil {
		return connections.Admission{}, false, err
	}
	state.Binding, err = decodeBinding(binding)
	if err != nil {
		return connections.Admission{}, false, err
	}
	if state.Binding.Provider != provider || state.Binding.Environment != environment {
		return connections.Admission{}, false, connections.ErrInvalidAdmission
	}
	for _, entry := range []struct {
		kind   connections.CheckKind
		result *string
		at     *time.Time
		ns     *int16
		target *connections.Check
	}{{connections.ProviderCheck, providerResult, providerAt, providerNS, &state.Provider}, {connections.HostCheck, hostResult, hostAt, hostNS, &state.Host}} {
		if entry.result == nil {
			continue
		}
		if entry.at == nil || entry.ns == nil {
			return connections.Admission{}, false, connections.ErrInvalidAdmission
		}
		at, err := restoreInstant(*entry.at, *entry.ns)
		if err != nil {
			return connections.Admission{}, false, err
		}
		*entry.target = connections.Check{Kind: entry.kind, Binding: state.Binding, Result: connections.CheckResult(*entry.result), At: at}
	}
	a, err := connections.Restore(state)
	return a, err == nil, err
}
func (s *Store) SaveAdmission(ctx context.Context, a connections.Admission) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if !scope.holdsAdmission(a.Binding().Provider, a.Binding().Environment) || scope.householdLocked {
		return ErrTransactionRequired
	}
	state := a.Snapshot()
	if _, err = connections.Restore(state); err != nil {
		return err
	}
	binding, err := json.Marshal(bindingFromDomain(state.Binding))
	if err != nil {
		return err
	}
	var pResult, pAt, pNS, hResult, hAt, hNS any
	if state.Provider != (connections.Check{}) {
		at, ns := splitInstant(state.Provider.At)
		pResult, pAt, pNS = state.Provider.Result, at, ns
	}
	if state.Host != (connections.Check{}) {
		at, ns := splitInstant(state.Host.At)
		hResult, hAt, hNS = state.Host.Result, at, ns
	}
	tag, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.deployment_admissions(provider,environment,binding,revision,provider_result,provider_at,provider_ns,host_result,host_at,host_ns) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(provider,environment) DO UPDATE SET binding=EXCLUDED.binding,revision=EXCLUDED.revision,provider_result=EXCLUDED.provider_result,provider_at=EXCLUDED.provider_at,provider_ns=EXCLUDED.provider_ns,host_result=EXCLUDED.host_result,host_at=EXCLUDED.host_at,host_ns=EXCLUDED.host_ns WHERE want_keep.deployment_admissions.revision=EXCLUDED.revision-1`, state.Binding.Provider, state.Binding.Environment, binding, state.Revision, pResult, pAt, pNS, hResult, hAt, hNS)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return connections.ErrInvalidAdmission
	}
	// Explicit audit DTO preserves timestamps; domain Instant deliberately has no JSON representation.
	audit := struct {
		Binding                                        bindingRow
		Revision                                       int64
		ProviderResult, ProviderAt, HostResult, HostAt string
	}{bindingFromDomain(state.Binding), state.Revision, string(state.Provider.Result), state.Provider.At.String(), string(state.Host.Result), state.Host.At.String()}
	data, err := json.Marshal(audit)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.admission_events(provider,environment,revision,snapshot) VALUES($1,$2,$3,$4)`, state.Binding.Provider, state.Binding.Environment, state.Revision, data)
	return err
}
func (s *Store) InvalidateJobs(ctx context.Context, a connections.Admission) error {
	scope, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if !scope.holdsAdmission(a.Binding().Provider, a.Binding().Environment) {
		return ErrTransactionRequired
	}
	b := a.Binding()
	binding, err := json.Marshal(bindingFromDomain(b))
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `UPDATE want_keep.jobs SET state=CASE WHEN external_started THEN 'unresolved' ELSE 'canceled' END,reason=CASE WHEN external_started THEN 'external_unknown' ELSE 'canceled' END,cancel_requested=true WHERE kind='sync' AND state IN ('ready','running','waiting','unresolved') AND binding->>'provider'=$1 AND binding->>'environment'=$2 AND (binding!=$3::jsonb OR admission_revision!=$4 OR $5!='admitted')`, b.Provider, b.Environment, binding, a.Revision(), a.Status())
	return err
}
func (s *Store) CreateConnection(ctx context.Context, c admission.Connection) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if c.HouseholdID != scope.principal.HouseholdID() {
		return household.ErrForbidden
	}
	if c.ID == "" || c.Generation != 1 || c.Owner == "" {
		return jobs.ErrInvalidJob
	}
	if c.SecretPurpose == "" {
		c.SecretPurpose = connections.APICredentials
	}
	if !c.SecretPurpose.Valid() {
		return jobs.ErrInvalidJob
	}
	switch c.Provider {
	case "alfa", "raiffeisen", "ozon", "bybit", "aifory", "emcd":
	default:
		return jobs.ErrInvalidJob
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.connections(household_id,id,provider,external_owner_id,generation,authorized,secret_purpose) VALUES($1,$2,$3,$4,$5,$6,$7)`, scope.principal.HouseholdID(), c.ID, c.Provider, c.Owner, c.Generation, c.Authorized, c.SecretPurpose)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.sync_schedules(household_id,connection_id,next_due) VALUES($1,$2,clock_timestamp())`, c.HouseholdID, c.ID)
	return err
}
func (s *Store) Connection(ctx context.Context, p household.Principal, id string) (admission.Connection, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return admission.Connection{}, err
	}
	c := admission.Connection{ID: id}
	err = q.QueryRow(ctx, `SELECT household_id,provider,external_owner_id,generation,authorized,secret_purpose FROM want_keep.connections WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id).Scan(&c.HouseholdID, &c.Provider, &c.Owner, &c.Generation, &c.Authorized, &c.SecretPurpose)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}
func (s *Store) Disconnect(ctx context.Context, id string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.connections SET authorized=false,generation=generation+1 WHERE household_id=$1 AND id=$2 AND generation<9007199254740991`, scope.principal.HouseholdID(), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return jobs.ErrInvalidJob
	}
	_, err = scope.tx.Exec(ctx, `UPDATE want_keep.jobs SET state=CASE WHEN external_started THEN 'unresolved' ELSE 'canceled' END,reason=CASE WHEN external_started THEN 'external_unknown' ELSE 'canceled' END,cancel_requested=true WHERE household_id=$1 AND connection_id=$2 AND state IN ('ready','running','waiting','unresolved')`, scope.principal.HouseholdID(), id)
	if err != nil {
		return err
	}
	return s.RevokeConnectionSecrets(ctx, id)
}
func (s *Store) CreateSyncJob(ctx context.Context, c admission.Connection, a connections.Admission, deadline time.Time) (jobs.Job, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return jobs.Job{}, err
	}
	if !scope.holdsAdmission(a.Binding().Provider, a.Binding().Environment) {
		return jobs.Job{}, ErrTransactionRequired
	}
	if err = a.RequireSync(a.Binding()); err != nil {
		return jobs.Job{}, err
	}
	if err = (connections.ExternalOwnership{HouseholdID: c.HouseholdID, OwnerID: c.Owner}).RequireManage(scope.principal); err != nil {
		return jobs.Job{}, err
	}
	if !c.Authorized || c.Provider != a.Binding().Provider {
		return jobs.Job{}, connections.ErrProviderNotAdmitted
	}
	now, err := s.DatabaseTime(ctx)
	if err != nil {
		return jobs.Job{}, err
	}
	if !deadline.After(now) || deadline.After(now.Add(24*time.Hour)) {
		return jobs.Job{}, jobs.ErrInvalidJob
	}
	existing, err := scanJob(scope.tx.QueryRow(ctx, `SELECT `+jobColumns+` FROM want_keep.jobs WHERE household_id=$1 AND connection_id=$2 AND kind='sync' AND state IN ('ready','running','waiting','unresolved') ORDER BY (state='unresolved') DESC,id LIMIT 1 `, scope.principal.HouseholdID(), c.ID))
	if err == nil {
		if existing.Binding != a.Binding() || existing.AdmissionRevision != a.Revision() || existing.ConnectionGeneration != c.Generation || existing.SecretPurpose != c.SecretPurpose {
			return jobs.Job{}, connections.ErrProviderNotAdmitted
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return jobs.Job{}, err
	}
	binding, err := json.Marshal(bindingFromDomain(a.Binding()))
	if err != nil {
		return jobs.Job{}, err
	}
	id := newID()
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.jobs(household_id,id,actor_id,kind,connection_id,connection_generation,binding,admission_revision,state,max_attempts,available_at,deadline,secret_purpose) VALUES($1,$2,$3,'sync',$4,$5,$6,$7,'ready',5,clock_timestamp(),$8,$9)`, scope.principal.HouseholdID(), id, scope.principal.UserID(), c.ID, c.Generation, binding, a.Revision(), deadline, c.SecretPurpose)
	if err != nil {
		return jobs.Job{}, err
	}
	_, err = scope.tx.Exec(ctx, `UPDATE want_keep.jobs j SET cursor=p.cursor,coverage=p.coverage,gaps=p.gaps FROM want_keep.sync_progress p WHERE j.household_id=$1 AND j.id=$2 AND p.household_id=j.household_id AND p.connection_id=j.connection_id AND p.generation=j.connection_generation AND NOT p.completed`, scope.principal.HouseholdID(), id)
	if err != nil {
		return jobs.Job{}, err
	}
	return s.Job(ctx, scope.principal, id)
}
func (s *Store) Quarantine(ctx context.Context, j jobs.Job, evidence, reason string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if j.HouseholdID != scope.principal.HouseholdID() {
		return household.ErrForbidden
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.quarantine(household_id,id,job_id,evidence_ref,reason) VALUES($1,$2,$3,$4,$5) ON CONFLICT(household_id,job_id,evidence_ref,reason) DO NOTHING`, j.HouseholdID, newID(), j.ID, evidence, reason)
	return err
}
func (s *Store) SaveCheckpoint(ctx context.Context, j jobs.Job, cursor, coverage string, gaps []string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if j.HouseholdID != scope.principal.HouseholdID() {
		return household.ErrForbidden
	}
	if gaps == nil {
		gaps = []string{}
	}
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.jobs SET cursor=$5,coverage=$6,gaps=$7 WHERE household_id=$1 AND id=$2 AND lease_token=$3 AND attempt=$4 AND state='running' AND NOT cancel_requested AND lease_until>clock_timestamp() AND COALESCE(run_deadline,deadline)>clock_timestamp()`, j.HouseholdID, j.ID, j.LeaseToken, j.Attempt, cursor, coverage, gaps)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return jobs.ErrStaleAttempt
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.sync_progress(household_id,connection_id,generation,cursor,coverage,gaps,last_job_id,completed) VALUES($1,$2,$3,$4,$5,$6,$7,false) ON CONFLICT(household_id,connection_id) DO UPDATE SET generation=EXCLUDED.generation,cursor=EXCLUDED.cursor,coverage=EXCLUDED.coverage,gaps=EXCLUDED.gaps,last_job_id=EXCLUDED.last_job_id,completed=false`, j.HouseholdID, j.ConnectionID, j.ConnectionGeneration, cursor, coverage, gaps, j.ID)
	return err
}

func (s *Store) FenceSyncResult(ctx context.Context, p household.Principal, issued jobs.Job) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if !scope.holdsAdmission(issued.Binding.Provider, issued.Binding.Environment) || scope.principal != p {
		return ErrTransactionRequired
	}
	a, _, err := s.Admission(ctx, issued.Binding.Provider, issued.Binding.Environment)
	if err != nil {
		return err
	}
	if err = a.RequireResult(issued.Binding, issued.AdmissionRevision); err != nil {
		return err
	}
	current, err := s.Job(ctx, p, issued.ID)
	if err != nil {
		return err
	}
	now, err := s.DatabaseTime(ctx)
	if err != nil {
		return err
	}
	if err = current.RequireAttempt(issued, now); err != nil {
		return err
	}
	c, err := s.Connection(ctx, p, current.ConnectionID)
	if err != nil {
		return err
	}
	if !c.Authorized || c.Generation != issued.ConnectionGeneration {
		return connections.ErrProviderNotAdmitted
	}
	scope.syncJobID = current.ID
	scope.syncConnectionID = current.ConnectionID
	return nil
}
