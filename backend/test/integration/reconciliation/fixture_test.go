//go:build integration

package reconciliation_test

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobsapp "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	ledgerapp "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reconciliationapp "github.com/pchkauu/want-keep/backend/internal/reconciliation/application"
	reconciliation "github.com/pchkauu/want-keep/backend/internal/reconciliation/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

var databaseURL *url.URL
var cluster *pgxpool.Pool
var testContext = context.Background()

func TestMain(m *testing.M) {
	raw := os.Getenv("WANT_KEEP_TEST_DATABASE_URL")
	u, err := url.Parse(raw)
	if err != nil || raw == "" || u.Path != "/want_keep_test" || (u.Hostname() != "localhost" && (net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback())) {
		fmt.Fprintln(os.Stderr, "Reconciliation integration requires an isolated loopback PostgreSQL database named want_keep_test via WANT_KEEP_TEST_DATABASE_URL.")
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
	family     household.Household
	p, q       household.Principal
	members    []household.Membership
	now        calendar.Instant
	admission  *admission.Service
	accounts   *accounts.Service
	reconciler *reconciliationapp.Service
	writer     *ledgerapp.Writer
	ledger     *ledgerapp.Service
	executor   *commands.Executor
}

func newFixture(t *testing.T) *fixture {
	return newFixtureUsingMigrations(t, migrations.Files)
}

func newFixtureUsingMigrations(t *testing.T, files fs.FS) *fixture {
	t.Helper()
	name := "wk_reconciliation_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := cluster.Exec(testContext, `CREATE DATABASE `+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := cluster.Exec(testContext, `DROP DATABASE `+name+` WITH (FORCE)`); err != nil {
			t.Errorf("drop synthetic reconciliation database: %v", err)
		}
	})
	u := *databaseURL
	u.Path = "/" + name
	admin, err := pgxpool.New(testContext, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	if err = storage.Migrate(testContext, admin, files); err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("want_keep_app", "synthetic-app")
	store, err := storage.Open(testContext, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 12})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	now := instant(time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano))
	f := &fixture{t: t, store: store, admin: admin, family: household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Synthetic household"}, now: now}
	users := []household.User{{ID: household.UserID(uuid.NewString()), Name: "Member A"}, {ID: household.UserID(uuid.NewString()), Name: "Member B"}}
	for _, user := range users {
		f.members = append(f.members, household.Membership{ID: household.MembershipID(uuid.NewString()), HouseholdID: f.family.ID, UserID: user.ID, Active: true})
	}
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	if err = store.InitializeHousehold(testContext, f.family, users, f.members, zone, 2); err != nil {
		t.Fatal(err)
	}
	f.p, _ = f.members[0].Principal()
	f.q, _ = f.members[1].Principal()
	f.admission = admission.NewService(store, store)
	baseWriter := ledgerapp.NewWriter(store, store)
	f.reconciler = reconciliationapp.NewService(store, store, baseWriter, f.admission, func() calendar.Instant { return f.now }, uuid.NewString)
	f.writer = ledgerapp.NewWriterWithReconciliation(store, store, f.reconciler)
	f.ledger = ledgerapp.NewService(store, f.writer, func() calendar.Instant { return f.now }, uuid.NewString)
	f.accounts = accounts.NewServiceWithReconciliation(store, store, f.reconciler, func() calendar.Instant { return f.now }, uuid.NewString)
	f.executor = commands.NewExecutor(store, store, func() calendar.Instant { return f.now })
	f.admit()
	return f
}

func instant(value string) calendar.Instant {
	result, err := calendar.ParseInstant(value)
	if err != nil {
		panic(err)
	}
	return result
}

func cash(value string, asset money.Asset) money.Money {
	result, err := money.NewMoney(value, asset)
	if err != nil {
		panic(err)
	}
	return result
}

func known(value string, asset money.Asset) reporting.Amount {
	result, err := reporting.KnownAmount(cash(value, asset))
	if err != nil {
		panic(err)
	}
	return result
}

func binding() connections.Binding {
	return connections.Binding{
		Provider: "raiffeisen", Environment: "test",
		AdapterBuildDigest: "sha256:" + strings.Repeat("a", 64), CollectorImageDigest: "sha256:" + strings.Repeat("b", 64),
		ContractVersion: "10", AllowlistRevision: "1", NonSecretConfigRevision: "1", OperatorPermissionRevision: "1",
	}
}

func (f *fixture) admit() {
	f.t.Helper()
	for _, kind := range []connections.CheckKind{connections.ProviderCheck, connections.HostCheck} {
		if _, err := f.admission.RecordCheck(testContext, connections.Check{Kind: kind, Binding: binding(), Result: connections.CheckPassed, At: f.now}); err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *fixture) connection(owner household.Principal, authorized bool) string {
	f.t.Helper()
	id := uuid.NewString()
	if err := f.store.WithinHousehold(testContext, owner, func(ctx context.Context) error {
		return f.store.CreateConnection(ctx, admission.Connection{HouseholdID: f.family.ID, ID: id, Provider: "raiffeisen", Owner: owner.UserID(), Generation: 1, Authorized: authorized})
	}); err != nil {
		f.t.Fatal(err)
	}
	return id
}

func (f *fixture) claim(jobID string) jobs.Job {
	f.t.Helper()
	claimed, err := f.store.ClaimJobs(testContext, "sync", 100, time.Minute)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, job := range claimed {
		if job.ID == jobID {
			return job
		}
	}
	f.t.Fatalf("job %s was not claimable", jobID)
	return jobs.Job{}
}

func (f *fixture) importAccount(asset money.Asset, product string, amounts account.Amounts, coverage reporting.Coverage, freshness reporting.Freshness) string {
	f.t.Helper()
	connectionID := f.connection(f.p, true)
	issued, err := f.admission.RequestSync(testContext, f.p, connectionID, binding(), time.Now().Add(time.Hour))
	if err != nil {
		f.t.Fatal(err)
	}
	issued = f.claim(issued.ID)
	date, _ := calendar.ParseDate(f.now.Time().AddDate(0, 0, -7).Format(time.DateOnly))
	zero := known("0", asset)
	input := accounts.ImportInput{
		ConnectionID: connectionID, JobID: issued.ID, Provider: "raiffeisen", ExternalID: uuid.NewString(), Product: product,
		ExternalAssetCode: string(asset), Name: "Synthetic source account", Asset: asset, OpeningDate: date,
		EvidenceRef: "synthetic:source-balance", Origin: "historical_backfill",
		Observation: account.Observation{ID: uuid.NewString(), AsOf: f.now, FetchedAt: f.now, Amounts: amounts, CreditLimit: zero, OwnAvailable: true, Coverage: coverage, Freshness: freshness},
	}
	var result accounts.ImportResult
	applied, err := f.admission.CommitPage(testContext, f.p, issued, admission.Page{EvidenceRef: input.EvidenceRef, Coverage: string(coverage.State()), Gaps: coverage.Reasons(), Complete: true}, func(ctx context.Context) error {
		var importErr error
		result, importErr = f.accounts.Import(ctx, f.p, f.store, input)
		return importErr
	})
	if err != nil || !applied || result.Account == nil {
		f.t.Fatalf("import failed: applied=%v result=%#v err=%v", applied, result, err)
	}
	return result.Account.ID
}

func (f *fixture) observe(accountID string, amounts account.Amounts, coverage reporting.Coverage, freshness reporting.Freshness) {
	f.t.Helper()
	current := f.active(accountID)
	connectionID := current.Replay.ConnectionID
	if connectionID == "" {
		if err := f.admin.QueryRow(testContext, `SELECT connection_id FROM want_keep.account_observations WHERE household_id=$1 AND account_id=$2 ORDER BY as_of DESC,as_of_ns DESC LIMIT 1`, f.family.ID, accountID).Scan(&connectionID); err != nil {
			f.t.Fatal(err)
		}
	}
	issued, err := f.admission.RequestSync(testContext, f.p, connectionID, binding(), time.Now().Add(time.Hour))
	if err != nil {
		f.t.Fatal(err)
	}
	issued = f.claim(issued.ID)
	entry, err := f.store.Account(testContext, f.p, accountID)
	if err != nil {
		f.t.Fatal(err)
	}
	observation := account.Observation{
		ID: uuid.NewString(), AccountID: accountID, ConnectionID: connectionID, JobID: issued.ID, EvidenceRef: "synthetic:new-source-balance",
		AsOf: f.now, FetchedAt: f.now, Amounts: amounts, CreditLimit: known("0", entry.Asset), OwnAvailable: true, Coverage: coverage, Freshness: freshness,
	}
	applied, err := f.admission.CommitPage(testContext, f.p, issued, admission.Page{EvidenceRef: observation.EvidenceRef, Coverage: string(coverage.State()), Gaps: coverage.Reasons(), Complete: true}, func(ctx context.Context) error {
		if recordErr := f.store.RecordObservation(ctx, f.p, observation); recordErr != nil {
			return recordErr
		}
		return f.reconciler.ReconcileAccount(ctx, f.p, accountID)
	})
	if err != nil || !applied {
		f.t.Fatalf("observation failed: applied=%v err=%v", applied, err)
	}
}

func (f *fixture) correctOpening(accountID string, values account.Amounts) {
	f.t.Helper()
	entry, err := f.store.Account(testContext, f.p, accountID)
	if err != nil {
		f.t.Fatal(err)
	}
	date, _ := calendar.ParseDate(f.now.Time().AddDate(0, 0, -7).Format(time.DateOnly))
	request := commands.Request{ID: uuid.NewString(), Kind: "account.opening.correct", PayloadHash: strings.Repeat("a", 64)}
	result, err := f.executor.Execute(testContext, f.p, request, func(ctx context.Context) (command.Result, error) {
		return f.accounts.CorrectOpening(ctx, f.p, accountID, accounts.Correction{ExpectedRevision: entry.Revision, Date: date, Amounts: values, Reason: "Confirmed synthetic opening"})
	})
	if err != nil || result.Status() != command.Succeeded {
		f.t.Fatalf("opening correction failed: %s %v", result.ErrorCode(), err)
	}
}

func (f *fixture) expense(accountID string, asset money.Asset, amount string) string {
	return f.expenseWithFee(accountID, asset, amount, "")
}

func (f *fixture) expenseWithFee(accountID string, asset money.Asset, amount, fee string) string {
	f.t.Helper()
	id := uuid.NewString()
	date, _ := calendar.ParseDate(f.now.Time().Format(time.DateOnly))
	month, _ := calendar.ParseMonth(f.now.Time().Format("2006-01"))
	postings := []ledger.Posting{{AccountID: accountID, Money: cash("-"+amount, asset), Role: ledger.Principal, Funding: ledger.OwnFunds, Treatment: ledger.Movement}}
	if fee != "" {
		postings = append(postings, ledger.Posting{AccountID: accountID, Money: cash("-"+fee, asset), Role: ledger.Fee, Funding: ledger.OwnFunds, Treatment: ledger.Movement})
	}
	revision := ledger.Revision{
		OperationID: id, Revision: 1, ActorID: f.p.UserID(), Reason: "Synthetic expense", Type: ledger.Expense, State: ledger.Posted,
		OccurredAt: f.now, PostedAt: f.now, RecordedAt: f.now, CashDate: date, ExpenseMonth: month, Timezone: timezone(),
		PayerState: "known", PayerMemberID: f.members[0].ID, FeeKnowledge: ledger.KnownFees, AllocationReason: "unresolved",
		Postings: postings,
	}
	request := commands.Request{ID: uuid.NewString(), Kind: "transaction.create", PayloadHash: strings.Repeat("b", 64)}
	result, err := f.executor.Execute(testContext, f.p, request, func(ctx context.Context) (command.Result, error) {
		err := f.writer.Append(ctx, f.p, revision, 0)
		return command.Result{ResourceType: "transaction", ResourceID: id, Revision: 1}, err
	})
	if err != nil || result.Status() != command.Succeeded {
		f.t.Fatalf("expense failed: %s %v", result.ErrorCode(), err)
	}
	return id
}

func timezone() calendar.Timezone {
	result, _ := calendar.ParseTimezone("Europe/Moscow")
	return result
}

func (f *fixture) active(accountID string) reconciliation.Reconciliation {
	f.t.Helper()
	var id string
	if err := f.admin.QueryRow(testContext, `SELECT id FROM want_keep.reconciliations WHERE household_id=$1 AND account_id=$2 AND lifecycle='open'`, f.family.ID, accountID).Scan(&id); err != nil {
		f.t.Fatal(err)
	}
	return f.read(id)
}

func (f *fixture) read(id string) reconciliation.Reconciliation {
	f.t.Helper()
	var value reconciliation.Reconciliation
	if err := f.store.WithinFinancialRead(testContext, f.p, func(ctx context.Context) error {
		var readErr error
		value, readErr = f.reconciler.Read(ctx, f.p, id)
		return readErr
	}); err != nil {
		f.t.Fatal(err)
	}
	return value
}

func (f *fixture) completeReplay(accountID string) reconciliation.Reconciliation {
	f.t.Helper()
	current := f.active(accountID)
	if current.Replay.Status == reconciliation.ReplayPending && current.Replay.JobID == "" {
		f.dispatchReplayEvent(current.ID)
		current = f.active(accountID)
	}
	issued := f.claim(current.Replay.JobID)
	applied, err := f.admission.CommitPage(testContext, f.p, issued, admission.Page{EvidenceRef: "synthetic:history-replay", Coverage: "complete", Complete: true}, func(ctx context.Context) error {
		return f.reconciler.RecordReplayOutcome(ctx, f.p, current.ID, issued, reconciliation.ReplayCompleted, "history_replayed")
	})
	if err != nil || !applied {
		f.t.Fatalf("replay page failed: %v", err)
	}
	return f.active(accountID)
}

func (f *fixture) dispatchReplayEvent(reconciliationID string) {
	f.t.Helper()
	worker := jobsapp.Worker{
		Repository: f.store,
		Handler:    jobsapp.OutboxHandler{Repository: f.store, Reconciliation: f.reconciler},
		Config:     jobsapp.DefaultWorkerConfig(jobs.Outbox),
	}
	for range 20 {
		current := f.read(reconciliationID)
		if current.Replay.Status != reconciliation.ReplayPending || current.Replay.JobID != "" {
			return
		}
		if err := worker.Step(testContext); err != nil {
			f.t.Fatal(err)
		}
	}
	f.t.Fatal("durable reconciliation event did not dispatch replay")
}

func (f *fixture) resolve(value reconciliation.Reconciliation, components ...reconciliation.ComponentName) command.Command {
	f.t.Helper()
	request := commands.Request{ID: uuid.NewString(), Kind: "reconciliations.resolve", PayloadHash: strings.Repeat("c", 64)}
	result, err := f.executor.Execute(testContext, f.p, request, func(ctx context.Context) (command.Result, error) {
		return f.reconciler.Resolve(ctx, f.p, value.ID, reconciliationapp.ResolutionInput{ExpectedRevision: value.Revision, Reason: "Confirmed source adjustment", Components: components})
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return result
}

func (f *fixture) count(table string) int {
	f.t.Helper()
	var count int
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.`+table).Scan(&count); err != nil {
		f.t.Fatal(err)
	}
	return count
}
