//go:build integration

package ingestion_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	application "github.com/pchkauu/want-keep/backend/internal/integrations/application"
	contract "github.com/pchkauu/want-keep/backend/internal/integrations/contract"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

var (
	databaseURL *url.URL
	cluster     *pgxpool.Pool
	testContext = context.Background()
)

func TestMain(m *testing.M) {
	raw := os.Getenv("WANT_KEEP_TEST_DATABASE_URL")
	u, err := url.Parse(raw)
	if err != nil || raw == "" || u.Path != "/want_keep_test" || (u.Hostname() != "localhost" && (net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback())) {
		fmt.Fprintln(os.Stderr, "Ingestion integration requires an isolated loopback PostgreSQL database named want_keep_test via WANT_KEEP_TEST_DATABASE_URL.")
		os.Exit(2)
	}
	databaseURL = u
	cluster, err = pgxpool.New(testContext, u.String())
	if err != nil || cluster.Ping(testContext) != nil {
		fmt.Fprintln(os.Stderr, "Isolated PostgreSQL is unavailable")
		os.Exit(2)
	}
	_, err = cluster.Exec(testContext, `DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='want_keep_app') THEN CREATE ROLE want_keep_app LOGIN PASSWORD 'synthetic-app'; END IF; IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='want_keep_maintenance') THEN CREATE ROLE want_keep_maintenance LOGIN PASSWORD 'synthetic-maintenance'; END IF; END $$;`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot prepare isolated test roles")
		os.Exit(2)
	}
	code := m.Run()
	cluster.Close()
	os.Exit(code)
}

type fixture struct {
	t          *testing.T
	store      *storage.Store
	admin      *pgxpool.Pool
	p          household.Principal
	family     household.Household
	now        calendar.Instant
	gate       *admission.Service
	service    *application.Service
	evidence   *fileEvidenceStore
	connection string
	binding    connections.Binding
}

func newFixture(t *testing.T) *fixture {
	return newFixtureForProvider(t, "bybit")
}

func newFixtureForProvider(t *testing.T, provider string) *fixture {
	t.Helper()
	name := "wk_ingestion_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := cluster.Exec(testContext, `CREATE DATABASE `+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := cluster.Exec(testContext, `DROP DATABASE `+name+` WITH (FORCE)`); err != nil {
			t.Errorf("drop synthetic ingestion database: %v", err)
		}
	})
	u := *databaseURL
	u.Path = "/" + name
	admin, err := pgxpool.New(testContext, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	if err = storage.Migrate(testContext, admin, migrations.Files); err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("want_keep_app", "synthetic-app")
	store, err := storage.Open(testContext, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 8})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	now, _ := calendar.ParseInstant("2026-09-08T12:00:00.123456789Z")
	family := household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Synthetic household"}
	user := household.User{ID: household.UserID(uuid.NewString()), Name: "Synthetic member"}
	membership := household.Membership{ID: household.MembershipID(uuid.NewString()), HouseholdID: family.ID, UserID: user.ID, Active: true}
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	if err = store.InitializeHousehold(testContext, family, []household.User{user}, []household.Membership{membership}, zone, 2); err != nil {
		t.Fatal(err)
	}
	p, _ := membership.Principal()
	writer := journal.NewWriter(store, store)
	accountService := accounts.NewService(store, store, func() calendar.Instant { return now }, uuid.NewString)
	accountImporter, err := application.NewAccountImporter(accountService, store)
	if err != nil {
		t.Fatal(err)
	}
	sources := journal.NewSources(store, writer)
	sourceWriter, err := application.NewSourceWriter(sources, store)
	if err != nil {
		t.Fatal(err)
	}
	gate := admission.NewService(store, store)
	evidence := &fileEvidenceStore{path: t.TempDir()}
	service, err := application.NewService(gate, evidence, accountImporter, sourceWriter, func() calendar.Instant { return now }, uuid.NewString)
	if err != nil {
		t.Fatal(err)
	}
	providerBinding := bindingFor(provider)
	f := &fixture{t: t, store: store, admin: admin, p: p, family: family, now: now, gate: gate, service: service, evidence: evidence, connection: uuid.NewString(), binding: providerBinding}
	if err = store.WithinHousehold(testContext, p, func(ctx context.Context) error {
		return store.CreateConnection(ctx, admission.Connection{HouseholdID: family.ID, ID: f.connection, Provider: provider, Owner: p.UserID(), Generation: 1, Authorized: true})
	}); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []connections.CheckKind{connections.ProviderCheck, connections.HostCheck} {
		if _, err = gate.RecordCheck(testContext, connections.Check{Kind: kind, Binding: providerBinding, Result: connections.CheckPassed, At: now}); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func binding() connections.Binding {
	return bindingFor("bybit")
}

func bindingFor(provider string) connections.Binding {
	return connections.Binding{Provider: provider, Environment: "test", AdapterBuildDigest: "sha256:" + strings.Repeat("0", 64), CollectorImageDigest: "sha256:" + strings.Repeat("1", 64), ContractVersion: "10", AllowlistRevision: "allowlist-1", NonSecretConfigRevision: "config-1", OperatorPermissionRevision: "permission-1"}
}

func (f *fixture) issued() jobs.Job {
	f.t.Helper()
	requested, err := f.gate.RequestSync(testContext, f.p, f.connection, f.binding, time.Now().Add(time.Hour))
	if err != nil {
		f.t.Fatal(err)
	}
	claimed, err := f.store.ClaimJobs(testContext, "sync", 100, time.Minute)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, job := range claimed {
		if job.ID == requested.ID {
			return job
		}
	}
	f.t.Fatal("sync job was not claimed")
	return jobs.Job{}
}

func (f *fixture) gateway(job jobs.Job) *fixtureGateway {
	return f.gatewayWithMutation(job, nil)
}

func (f *fixture) gatewayWithMutation(job jobs.Job, mutate func(map[string]any)) *fixtureGateway {
	f.t.Helper()
	manifest, err := contract.DecodeManifest(readFixture(f.t, "manifest.json"), "bybit")
	if err != nil {
		f.t.Fatal(err)
	}
	manifest.Provider = job.Binding.Provider
	var raw map[string]any
	if err = json.Unmarshal(readFixture(f.t, "golden-page.json"), &raw); err != nil {
		f.t.Fatal(err)
	}
	page := raw["page"].(map[string]any)
	page["jobId"], page["attempt"], page["leaseToken"] = job.ID, job.Attempt, job.LeaseToken
	page["connectionGeneration"], page["admissionRevision"], page["cursor"] = job.ConnectionGeneration, job.AdmissionRevision, job.Cursor
	page["binding"] = map[string]any{"provider": job.Binding.Provider, "environment": job.Binding.Environment, "adapterBuildDigest": job.Binding.AdapterBuildDigest, "collectorImageDigest": job.Binding.CollectorImageDigest, "contractVersion": job.Binding.ContractVersion, "allowlistRevision": job.Binding.AllowlistRevision, "nonSecretConfigRevision": job.Binding.NonSecretConfigRevision, "operatorPermissionRevision": job.Binding.OperatorPermissionRevision}
	if mutate != nil {
		mutate(raw)
	}
	data, _ := json.Marshal(raw)
	token, err := application.TokenFromJob(job)
	if err != nil {
		f.t.Fatal(err)
	}
	result, err := contract.DecodeResult(data, token)
	if err != nil {
		f.t.Fatal(err)
	}
	return &fixtureGateway{bindingValue: job.Binding, manifestValue: manifest, result: result}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("../../../../collector/contracts/v10/fixtures", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type fixtureGateway struct {
	bindingValue  connections.Binding
	manifestValue ingestion.Manifest
	result        ingestion.Result
	read          func()
}

func (g *fixtureGateway) Binding() connections.Binding { return g.bindingValue }

func (g *fixtureGateway) Manifest(context.Context) (ingestion.Manifest, error) {
	return g.manifestValue, nil
}
func (g *fixtureGateway) Read(context.Context, ingestion.JobToken) (ingestion.Result, error) {
	if g.read != nil {
		g.read()
	}
	return g.result, nil
}

type fileEvidenceStore struct {
	path            string
	fail            bool
	failDisposition bool
	mu              sync.Mutex
	last            ingestion.EvidenceBatch
}

func (s *fileEvidenceStore) Save(_ context.Context, batch ingestion.EvidenceBatch) error {
	if s.fail {
		return fmt.Errorf("synthetic storage failure")
	}
	if err := batch.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	s.last = batch
	s.mu.Unlock()
	prefix := s.batchPrefix(batch.HouseholdID, batch.JobID, batch.PageReference)
	for _, item := range batch.Items {
		name := prefix + string(batch.Disposition) + "_" + s.safeName(item.Reference)
		temporary := filepath.Join(s.path, name+".tmp")
		if err := os.WriteFile(temporary, item.Raw.Data, 0o600); err != nil {
			return err
		}
		if err := os.Rename(temporary, filepath.Join(s.path, name)); err != nil {
			return err
		}
	}
	return nil
}

func (s *fileEvidenceStore) Last() ingestion.EvidenceBatch {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last
}

func (s *fileEvidenceStore) SetDisposition(_ context.Context, disposition ingestion.EvidenceDisposition) error {
	if s.failDisposition {
		return fmt.Errorf("synthetic disposition failure")
	}
	if err := disposition.Validate(); err != nil {
		return err
	}
	entries, err := os.ReadDir(s.path)
	if err != nil {
		return err
	}
	prefix := s.batchPrefix(disposition.HouseholdID, disposition.JobID, disposition.PageReference)
	stagedPrefix := prefix + string(ingestion.EvidenceStaged) + "_"
	terminalPrefix := prefix + string(disposition.State) + "_"
	found := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), terminalPrefix) {
			found = true
			continue
		}
		if !strings.HasPrefix(entry.Name(), stagedPrefix) {
			continue
		}
		found = true
		target := terminalPrefix + strings.TrimPrefix(entry.Name(), stagedPrefix)
		if err = os.Rename(filepath.Join(s.path, entry.Name()), filepath.Join(s.path, target)); err != nil {
			return err
		}
	}
	if !found {
		return fmt.Errorf("staged evidence batch not found")
	}
	return nil
}

func (s *fileEvidenceStore) Staged(_ context.Context, householdID string, limit int) ([]ingestion.StagedEvidence, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit < 1 || s.last.HouseholdID != householdID || s.last.Disposition != ingestion.EvidenceStaged {
		return nil, nil
	}
	entries, err := os.ReadDir(s.path)
	if err != nil {
		return nil, err
	}
	prefix := s.batchPrefix(s.last.HouseholdID, s.last.JobID, s.last.PageReference) + string(ingestion.EvidenceStaged) + "_"
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			return []ingestion.StagedEvidence{{HouseholdID: s.last.HouseholdID, JobID: s.last.JobID, PageReference: s.last.PageReference}}, nil
		}
	}
	return nil, nil
}

func (s *fileEvidenceStore) batchPrefix(householdID, jobID, pageReference string) string {
	return s.safeName(householdID + "_" + jobID + "_" + pageReference + "_")
}

func (s *fileEvidenceStore) safeName(value string) string {
	return strings.NewReplacer(":", "_", "/", "_").Replace(value)
}

func (f *fixture) count(table string) int {
	f.t.Helper()
	var count int
	if err := f.admin.QueryRow(testContext, "SELECT count(*) FROM want_keep."+table).Scan(&count); err != nil {
		f.t.Fatal(err)
	}
	return count
}
