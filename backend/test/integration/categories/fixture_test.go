//go:build integration

package categories_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	catalog "github.com/pchkauu/want-keep/backend/internal/categories/application"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	categorydelivery "github.com/pchkauu/want-keep/backend/internal/delivery/categories"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	ledgerdelivery "github.com/pchkauu/want-keep/backend/internal/delivery/ledger"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identityapp "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
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
		fmt.Fprintln(os.Stderr, "Categories integration requires an isolated loopback PostgreSQL database named want_keep_test via WANT_KEEP_TEST_DATABASE_URL.")
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
	t        *testing.T
	store    *storage.Store
	admin    *pgxpool.Pool
	family   household.Household
	p, q     household.Principal
	members  []household.Membership
	now      calendar.Instant
	executor *commands.Executor
}

func newFixture(t *testing.T) *fixture {
	return newFixtureUsingMigrations(t, migrations.Files)
}

func newFixtureUsingMigrations(t *testing.T, files fs.FS) *fixture {
	t.Helper()
	name := "wk_categories_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	if err = storage.Migrate(testContext, admin, files); err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("want_keep_app", "synthetic-app")
	store, err := storage.Open(testContext, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 8})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	f := &fixture{t: t, store: store, admin: admin, family: household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Synthetic household"}, now: instant("2026-09-08T12:00:00.123456789Z")}
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
	f.executor = commands.NewExecutor(store, store, func() calendar.Instant { return f.now })
	return f
}

func instant(value string) calendar.Instant {
	result, err := calendar.ParseInstant(value)
	if err != nil {
		panic(err)
	}
	return result
}

func (f *fixture) account(asset money.Asset, balance string) string {
	f.t.Helper()
	id := uuid.NewString()
	ownership, _ := household.NewOwnership(f.family.ID, household.Shared, "")
	date, _ := calendar.ParseDate("2026-09-01")
	err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		if err := f.store.CreateAccount(ctx, account.Account{ID: id, Name: "Cash", Product: "cash", Ownership: ownership, Asset: asset, Revision: 1, OpeningDate: date}); err != nil {
			return err
		}
		coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
		for _, field := range []string{"owned", "available", "locked", "debt"} {
			value := balance
			if field == "locked" || field == "debt" {
				value = "0"
			}
			amount, _ := money.NewMoney(value, asset)
			known, _ := reporting.KnownAmount(amount)
			if err := f.store.RecordBalance(ctx, account.Balance{AccountID: id, Field: field, Amount: known, Coverage: coverage, Freshness: reporting.Fresh, ObservedAt: f.now}); err != nil {
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

func (f *fixture) ledgerService() *journal.Service {
	return journal.NewService(f.store, journal.NewWriter(f.store, f.store), func() calendar.Instant { return f.now }, uuid.NewString)
}

type client struct {
	f        *fixture
	handler  http.Handler
	token    identity.Token
	sessions *identityapp.Service
}

func (f *fixture) client(p household.Principal) *client {
	f.t.Helper()
	token, _ := identity.NewToken(32)
	var member household.Membership
	for _, candidate := range f.members {
		if candidate.UserID == p.UserID() {
			member = candidate
		}
	}
	handle := make([]byte, 32)
	_, _ = rand.Read(handle)
	credentialID := uuid.NewString()
	err := f.store.WithinIdentity(testContext, func(ctx context.Context) error {
		profile := identity.Profile{UserID: p.UserID(), HouseholdID: p.HouseholdID(), MembershipID: member.ID, Name: "Synthetic member", Locale: "en", ReportingAsset: "RUB", Handle: handle, Generation: 1}
		if err := f.store.CreateIdentityProfile(ctx, profile); err != nil {
			return err
		}
		credential := identity.Credential{ID: credentialID, UserID: p.UserID(), Name: "Synthetic credential", RPID: "localhost", RawID: handle, PublicKey: []byte{}, AAGUID: []byte{}, AttestationObject: []byte{}, AttestationClientData: []byte{}, AttestationClientHash: []byte{}, Transports: []string{}, UserVerified: true, CreatedAt: f.now.Time()}
		if err := f.store.SaveIdentityCredential(ctx, credential); err != nil {
			return err
		}
		return f.store.SaveIdentitySession(ctx, identity.Session{ID: uuid.NewString(), TokenHash: token.Hash(), CredentialID: credentialID, Name: "Synthetic browser", UserID: p.UserID(), CreatedAt: f.now.Time(), AuthenticatedAt: f.now.Time(), LastActivityAt: f.now.Time()})
	})
	if err != nil {
		f.t.Fatal(err)
	}
	verifier, _ := webauthn.New("localhost", "http://localhost")
	sessions, err := identityapp.NewService(f.store, f.store, verifier, "localhost", "http://localhost", 2, func() time.Time { return f.now.Time() })
	if err != nil {
		f.t.Fatal(err)
	}
	commandQueries := commands.NewQueries(f.store, f.store.AuthorizeCommandResult)
	config := security.Config{Environment: "test", Origin: "http://localhost"}
	categories, err := categorydelivery.New(catalog.NewService(f.store, uuid.NewString), f.executor, commandQueries, sessions, f.store, config, func() calendar.Instant { return f.now })
	if err != nil {
		f.t.Fatal(err)
	}
	ledger, err := ledgerdelivery.New(f.ledgerService(), journal.NewQueries(f.store), f.executor, commandQueries, sessions, f.store, config, func() calendar.Instant { return f.now })
	if err != nil {
		f.t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/v1/categories", categories)
	mux.Handle("/api/v1/categories/", categories)
	mux.Handle("/api/v1/merchants", categories)
	mux.Handle("/api/v1/merchants/", categories)
	mux.Handle("/api/v1/transactions", ledger)
	mux.Handle("/api/v1/transactions/", ledger)
	return &client{f: f, handler: mux, token: token, sessions: sessions}
}

func (c *client) call(method, path, key string, input any, expected int) *httptest.ResponseRecorder {
	c.f.t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		c.f.t.Fatal(err)
	}
	req := httptest.NewRequest(method, "http://localhost/api/v1"+path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost")
	req.Header.Set("X-CSRF-Token", c.token.CSRF())
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	req.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(c.token)})
	response := httptest.NewRecorder()
	c.handler.ServeHTTP(response, req)
	if response.Code != expected {
		c.f.t.Fatalf("%s %s => %d want %d: %s", method, path, response.Code, expected, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		c.f.t.Fatal("missing no-store")
	}
	return response
}

func decode[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}
