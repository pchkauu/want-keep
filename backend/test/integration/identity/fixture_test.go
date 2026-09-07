//go:build integration

package identity_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	delivery "github.com/pchkauu/want-keep/backend/internal/delivery/identity"
	application "github.com/pchkauu/want-keep/backend/internal/identity/application"
	domain "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
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
		fmt.Fprintln(os.Stderr, "Identity integration requires an isolated loopback PostgreSQL database named want_keep_test via WANT_KEEP_TEST_DATABASE_URL.")
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
	t         *testing.T
	store     *storage.Store
	admin     *pgxpool.Pool
	service   *application.Service
	handler   http.Handler
	now       atomic.Int64
	bootstrap domain.Token
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	name := "wk_identity_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := cluster.Exec(testContext, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
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
	f := &fixture{t: t, store: store, admin: admin}
	f.now.Store(time.Now().UTC().Truncate(time.Second).UnixMicro())
	f.bootstrap, err = domain.NewToken(32)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = admin.Exec(testContext, "UPDATE want_keep.identity_bootstrap SET token_hash=$1,expires_at=$2", f.bootstrap.Hash(), f.clock().Add(domain.BootstrapLifetime)); err != nil {
		t.Fatal(err)
	}
	f.restart()
	return f
}
func (f *fixture) clock() time.Time        { return time.UnixMicro(f.now.Load()).UTC() }
func (f *fixture) advance(d time.Duration) { f.now.Add(d.Microseconds()) }
func (f *fixture) restart() {
	v, err := webauthn.New("localhost", "http://localhost")
	if err != nil {
		f.t.Fatal(err)
	}
	f.service, err = application.NewService(f.store, f.store, v, "localhost", "http://localhost", 2, f.clock)
	if err != nil {
		f.t.Fatal(err)
	}
	f.handler, err = delivery.New(f.service, delivery.Config{Environment: "test", Origin: "http://localhost"})
	if err != nil {
		f.t.Fatal(err)
	}
}

type browser struct {
	f       *fixture
	cookies map[string]*http.Cookie
	csrf    string
	expires map[string]time.Time
}

func (f *fixture) browser() *browser {
	return &browser{f: f, cookies: map[string]*http.Cookie{}, expires: map[string]time.Time{}}
}
func (b *browser) call(method, path string, input any, expected int) *httptest.ResponseRecorder {
	b.f.t.Helper()
	data, err := json.Marshal(input)
	if err != nil {
		b.f.t.Fatal(err)
	}
	req := httptest.NewRequest(method, "http://localhost/api/v1"+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost")
	req.Header.Set("X-CSRF-Token", b.csrf)
	for name, cookie := range b.cookies {
		if expiry, ok := b.expires[name]; ok && !b.f.clock().Before(expiry) {
			delete(b.cookies, name)
			delete(b.expires, name)
			continue
		}
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	b.f.handler.ServeHTTP(rr, req)
	if rr.Code != expected {
		b.f.t.Fatalf("%s %s: got %d want %d: %s", method, path, rr.Code, expected, rr.Body.String())
	}
	for _, cookie := range rr.Result().Cookies() {
		if cookie.MaxAge < 0 {
			delete(b.cookies, cookie.Name)
			delete(b.expires, cookie.Name)
		} else {
			b.cookies[cookie.Name] = cookie
			if cookie.MaxAge > 0 {
				b.expires[cookie.Name] = b.f.clock().Add(time.Duration(cookie.MaxAge) * time.Second)
			}
		}
	}
	if rr.Header().Get("Cache-Control") != "no-store" {
		b.f.t.Fatal("missing no-store")
	}
	return rr
}
func decode[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()
	var result T
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}
func (b *browser) setup() (*authenticator, []string, generated.Me) {
	options := decode[generated.EnrollmentOptions](b.f.t, b.call("POST", "/household/bootstrap", map[string]any{"name": "Synthetic A", "householdName": "Synthetic family", "timezone": "Europe/Moscow", "locale": "ru", "reportingAsset": "RUB", "bootstrapToken": string(b.f.bootstrap)}, 200))
	key := newAuthenticator(b.f.t, options.UserId, false)
	result := b.enroll(options, key, "bootstrap")
	return key, result.RecoveryCodes.Codes, result.Me
}
func (b *browser) enroll(options generated.EnrollmentOptions, key *authenticator, purpose string) generated.InitialEnrollmentResult {
	rr := b.call("POST", "/auth/enrollment/verify", map[string]any{"attemptId": options.AttemptId, "name": "Synthetic passkey", "credential": key.registration(b.f.t, options.Challenge, "http://localhost", "localhost", 5)}, 200)
	result := decode[generated.InitialEnrollmentResult](b.f.t, rr)
	b.csrf = result.Me.CsrfToken
	return result
}
func (b *browser) login(key *authenticator, purpose string) generated.Me {
	options := decode[generated.LoginOptions](b.f.t, b.call("POST", "/auth/login/options", map[string]any{"purpose": purpose}, 200))
	result := decode[generated.Me](b.f.t, b.call("POST", "/auth/login/verify", map[string]any{"attemptId": options.AttemptId, "credential": key.assertion(b.f.t, options.Challenge, "http://localhost", "localhost", 5)}, 200))
	b.csrf = result.CsrfToken
	return result
}
