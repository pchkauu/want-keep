//go:build integration

package storage_test

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
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
		fmt.Fprintln(os.Stderr, "Storage integration requires an isolated loopback PostgreSQL database named want_keep_test via WANT_KEEP_TEST_DATABASE_URL.")
		os.Exit(2)
	}
	databaseURL = u
	cluster, err = pgxpool.New(testContext, u.String())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot configure test database")
		os.Exit(2)
	}
	if err = cluster.Ping(testContext); err != nil {
		fmt.Fprintln(os.Stderr, "Isolated PostgreSQL is unavailable")
		os.Exit(2)
	}
	// Fixed, public synthetic credentials exist only in this disposable test cluster.
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
	t        *testing.T
	store    *storage.Store
	admin    *pgxpool.Pool
	dsn      string
	family   household.Household
	p, q     household.Principal
	members  []household.Membership
	writer   *journal.Writer
	executor *commands.Executor
	now      calendar.Instant
}

func newDatabase(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	name := "wk_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := cluster.Exec(testContext, `CREATE DATABASE `+name); err != nil {
		t.Fatal(err)
	}
	u := *databaseURL
	u.Path = "/" + name
	admin, err := pgxpool.New(testContext, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	return admin, u.String()
}
func newFixture(t *testing.T) *fixture { return newFixtureWithMigrations(t, migrations.Files) }
func newFixtureWithMigrations(t *testing.T, files fs.FS) *fixture {
	t.Helper()
	admin, dsn := newDatabase(t)
	if err := storage.Migrate(testContext, admin, files); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(dsn)
	u.User = url.UserPassword("want_keep_app", "synthetic-app")
	s, err := storage.Open(testContext, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 8})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	f := &fixture{t: t, store: s, admin: admin, dsn: u.String(), family: household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Synthetic household"}, now: instant("2026-09-07T12:00:00.123456789Z")}
	users := []household.User{{ID: household.UserID(uuid.NewString()), Name: "Member A"}, {ID: household.UserID(uuid.NewString()), Name: "Member B"}}
	for _, user := range users {
		f.members = append(f.members, household.Membership{ID: household.MembershipID(uuid.NewString()), HouseholdID: f.family.ID, UserID: user.ID, Active: true})
	}
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	if err = s.InitializeHousehold(testContext, f.family, users, f.members, zone, 2); err != nil {
		t.Fatal(err)
	}
	f.p, _ = f.members[0].Principal()
	f.q, _ = f.members[1].Principal()
	f.writer = journal.NewWriter(s, s)
	f.executor = commands.NewExecutor(s, s, func() calendar.Instant { return f.now })
	return f
}
func instant(v string) calendar.Instant {
	i, err := calendar.ParseInstant(v)
	if err != nil {
		panic(err)
	}
	return i
}
func cash(v string, asset money.Asset) money.Money {
	m, err := money.NewMoney(v, asset)
	if err != nil {
		panic(err)
	}
	return m
}
func (f *fixture) account(asset money.Asset, balance string) string {
	f.t.Helper()
	id := uuid.NewString()
	o, _ := household.NewOwnership(f.family.ID, household.Shared, "")
	date, _ := calendar.ParseDate("2026-09-01")
	err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		if err := f.store.CreateAccount(ctx, account.Account{ID: id, Name: "Synthetic account", Product: "cash", Ownership: o, Asset: asset, Revision: 1, OpeningDate: date}); err != nil {
			return err
		}
		coverage, _ := reporting.NewCoverage(reporting.Complete, []string{})
		for _, field := range []string{"owned", "available", "locked", "debt"} {
			v := balance
			if field == "locked" || field == "debt" {
				v = "0"
			}
			amount, _ := reporting.KnownAmount(cash(v, asset))
			if err := f.store.RecordBalance(ctx, account.Balance{AccountID: id, Field: field, Amount: amount, Coverage: coverage, Freshness: reporting.Fresh, ObservedAt: f.now}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return id
}
func (f *fixture) revision(id, accountID, amount string, asset money.Asset, n uint64) ledger.Revision {
	date, _ := calendar.ParseDate("2026-09-07")
	month, _ := calendar.ParseMonth("2026-09")
	return ledger.Revision{OperationID: id, Revision: n, ActorID: f.p.UserID(), Reason: "Synthetic entry", Type: "expense", State: "posted", OccurredAt: f.now, CashDate: date, ExpenseMonth: month, PayerState: "known", PayerMemberID: f.members[0].ID, Postings: []ledger.Posting{{AccountID: accountID, Money: cash(amount, asset), Role: "principal"}}}
}
func request() commands.Request {
	return commands.Request{ID: uuid.NewString(), Kind: "transaction.create", PayloadHash: strings.Repeat("a", 64)}
}
func (f *fixture) write(r ledger.Revision, key commands.Request) (command.Command, error) {
	return f.executor.Execute(testContext, f.p, key, func(ctx context.Context) (command.Result, error) {
		err := f.writer.Append(ctx, f.p, r, r.Revision-1)
		return command.Result{ResourceType: "transaction", ResourceID: r.OperationID, Revision: r.Revision}, err
	})
}
func (f *fixture) count(table string) int {
	f.t.Helper()
	var n int
	if err := f.admin.QueryRow(testContext, "SELECT count(*) FROM want_keep."+table).Scan(&n); err != nil {
		f.t.Fatal(err)
	}
	return n
}
func (f *fixture) available(id string) string {
	f.t.Helper()
	b, err := f.store.Balance(testContext, f.p, id, "available")
	if err != nil {
		f.t.Fatal(err)
	}
	v, known := b.Amount.Value()
	if !known {
		f.t.Fatal("unknown available")
	}
	return v.Amount()
}
func (f *fixture) restart() *storage.Store {
	f.t.Helper()
	s, err := storage.Open(testContext, storage.Config{DSN: f.dsn, Environment: "test", MaxConnections: 8})
	if err != nil {
		f.t.Fatal(err)
	}
	f.t.Cleanup(s.Close)
	return s
}
