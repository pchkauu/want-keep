//go:build integration

package privacy_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	application "github.com/pchkauu/want-keep/backend/internal/attachments/application"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	"github.com/pchkauu/want-keep/backend/internal/attachments/files"
	"github.com/pchkauu/want-keep/backend/internal/attachments/processor"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	delivery "github.com/pchkauu/want-keep/backend/internal/delivery/attachments"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identityapp "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
	"github.com/pchkauu/want-keep/backend/internal/privacy/cryptobox"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

var testContext = context.Background()
var databaseURL *url.URL
var cluster *pgxpool.Pool
var documentProcessor *processor.Client

func TestMain(m *testing.M) {
	raw := os.Getenv("WANT_KEEP_TEST_DATABASE_URL")
	u, err := url.Parse(raw)
	if err != nil || raw == "" || u.Path != "/want_keep_test" || (u.Hostname() != "localhost" && (net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback())) {
		fmt.Fprintln(os.Stderr, "Privacy integration requires isolated loopback PostgreSQL want_keep_test.")
		os.Exit(2)
	}
	databaseURL = u
	documentProcessor, err = processor.NewClient(os.Getenv("WANT_KEEP_TEST_PROCESSOR_SOCKET"))
	if err == nil {
		for attempt := 0; attempt < 30 && !documentProcessor.Ready(testContext); attempt++ {
			time.Sleep(time.Second)
		}
	}
	if err != nil || !documentProcessor.Ready(testContext) {
		fmt.Fprintln(os.Stderr, "Privacy integration requires the pinned isolated document processor.")
		os.Exit(2)
	}
	cluster, err = pgxpool.New(testContext, raw)
	if err != nil || cluster.Ping(testContext) != nil {
		fmt.Fprintln(os.Stderr, "Privacy test PostgreSQL unavailable.")
		os.Exit(2)
	}
	_, err = cluster.Exec(testContext, `DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='want_keep_app') THEN CREATE ROLE want_keep_app LOGIN PASSWORD 'synthetic-app'; END IF; IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='want_keep_maintenance') THEN CREATE ROLE want_keep_maintenance LOGIN PASSWORD 'synthetic-maintenance'; END IF; END $$;`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot initialize privacy test roles.")
		os.Exit(2)
	}
	code := m.Run()
	cluster.Close()
	os.Exit(code)
}

type fixture struct {
	t                         *testing.T
	store                     *storage.Store
	admin                     *pgxpool.Pool
	dsn                       string
	service                   *application.Service
	identity                  *identityapp.Service
	handler                   http.Handler
	blobs                     *files.Store
	ring                      *cryptobox.Keyring
	blobDir, keyPath          string
	a, b, foreign             household.Principal
	ta, tb, tf                identity.Token
	accountID, foreignAccount string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	name := "wk_privacy_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	f := &fixture{t: t, store: store, admin: admin, dsn: u.String(), blobDir: t.TempDir(), keyPath: filepath.Join(t.TempDir(), "keys")}
	if err = os.Chmod(f.blobDir, 0o700); err != nil {
		t.Fatal(err)
	}
	f.a, f.b = f.family()
	f.foreign, _ = f.family()
	f.ta = f.session(f.a)
	f.tb = f.session(f.b)
	f.tf = f.session(f.foreign)
	f.accountID = f.account(f.a)
	f.foreignAccount = f.account(f.foreign)
	v, err := webauthn.New("localhost", "http://localhost")
	if err != nil {
		t.Fatal(err)
	}
	f.identity, err = identityapp.NewService(store, store, v, "localhost", "http://localhost", 2, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if err = cryptobox.Generate(f.keyPath, "attachments"); err != nil {
		t.Fatal(err)
	}
	f.ring, err = cryptobox.Load(f.keyPath, "attachments")
	if err != nil {
		t.Fatal(err)
	}
	f.blobs, err = files.Open(f.blobDir, f.ring)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.blobs.Close() })
	f.service = application.NewService(store, f.blobs, documentProcessor)
	f.handler, err = delivery.New(f.service, f.identity, security.Config{Environment: "test", Origin: "http://localhost"}, func() bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func (f *fixture) family() (household.Principal, household.Principal) {
	f.t.Helper()
	h := household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Synthetic family"}
	users := []household.User{{ID: household.UserID(uuid.NewString()), Name: "Synthetic A"}, {ID: household.UserID(uuid.NewString()), Name: "Synthetic B"}}
	members := []household.Membership{{ID: household.MembershipID(uuid.NewString()), HouseholdID: h.ID, UserID: users[0].ID, Active: true}, {ID: household.MembershipID(uuid.NewString()), HouseholdID: h.ID, UserID: users[1].ID, Active: true}}
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	if err := f.store.InitializeHousehold(testContext, h, users, members, zone, 2); err != nil {
		f.t.Fatal(err)
	}
	a, _ := members[0].Principal()
	b, _ := members[1].Principal()
	return a, b
}
func (f *fixture) session(p household.Principal) identity.Token {
	f.t.Helper()
	member, err := f.store.Membership(testContext, p.HouseholdID(), p.UserID())
	if err != nil {
		f.t.Fatal(err)
	}
	handle := make([]byte, 32)
	if _, err = rand.Read(handle); err != nil {
		f.t.Fatal(err)
	}
	profile := identity.Profile{UserID: p.UserID(), HouseholdID: p.HouseholdID(), MembershipID: member.ID, Name: "Synthetic user", Locale: "ru", ReportingAsset: "RUB", Generation: 1, Handle: handle}
	token, err := identity.NewToken(32)
	if err != nil {
		f.t.Fatal(err)
	}
	now := time.Now().UTC()
	c := identity.Credential{ID: uuid.NewString(), UserID: p.UserID(), Name: "Synthetic credential", RPID: "localhost", RawID: handle, PublicKey: []byte{1}, AAGUID: []byte{}, AttestationObject: []byte{}, AttestationClientData: []byte{}, AttestationClientHash: []byte{}, Transports: []string{"internal"}, CreatedAt: now, UserVerified: true}
	s := identity.Session{ID: uuid.NewString(), UserID: p.UserID(), TokenHash: token.Hash(), CredentialID: c.ID, Name: "Synthetic session", CreatedAt: now, AuthenticatedAt: now, LastActivityAt: now}
	if err = f.store.WithinIdentity(testContext, func(ctx context.Context) error {
		if err := f.store.CreateIdentityProfile(ctx, profile); err != nil {
			return err
		}
		if err := f.store.SaveIdentityCredential(ctx, c); err != nil {
			return err
		}
		return f.store.SaveIdentitySession(ctx, s)
	}); err != nil {
		f.t.Fatal(err)
	}
	return token
}
func (f *fixture) account(p household.Principal) string {
	f.t.Helper()
	ownership, _ := household.NewOwnership(p.HouseholdID(), household.Personal, p.UserID())
	date, _ := calendar.ParseDate("2026-09-01")
	a := account.Account{ID: uuid.NewString(), Name: "Synthetic cash", Product: "cash", Asset: "RUB", Revision: 1, Ownership: ownership, OpeningDate: date}
	if err := f.store.WithinHousehold(testContext, p, func(ctx context.Context) error { return f.store.CreateAccount(ctx, a) }); err != nil {
		f.t.Fatal(err)
	}
	return a.ID
}
func (f *fixture) request(token identity.Token, method, path string, data []byte, contentType string) *httptest.ResponseRecorder {
	f.t.Helper()
	r := httptest.NewRequest(method, "http://localhost/api/v1"+path, bytes.NewReader(data))
	r.Header.Set("Origin", "http://localhost")
	r.Header.Set("Content-Type", contentType)
	if token != "" {
		r.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(token)})
		r.Header.Set("X-CSRF-Token", token.CSRF())
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w
}
func (f *fixture) upload(token identity.Token, id, account, media string, data []byte) *httptest.ResponseRecorder {
	f.t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if writer.WriteField("uploadId", id) != nil || writer.WriteField("accountId", account) != nil {
		f.t.Fatal("multipart fields")
	}
	header := textproto.MIMEHeader{"Content-Disposition": {`form-data; name="file"; filename="receipt"`}, "Content-Type": {media}}
	part, err := writer.CreatePart(header)
	if err != nil {
		f.t.Fatal(err)
	}
	if _, err = part.Write(data); err != nil {
		f.t.Fatal(err)
	}
	if writer.Close() != nil {
		f.t.Fatal("multipart close")
	}
	return f.request(token, http.MethodPost, "/attachments", body.Bytes(), writer.FormDataContentType())
}
func (f *fixture) metadata(token identity.Token, id string) domain.Attachment {
	f.t.Helper()
	p := f.a
	if token == f.tb {
		p = f.b
	} else if token == f.tf {
		p = f.foreign
	}
	a, err := f.service.Metadata(testContext, p, id)
	if err != nil {
		f.t.Fatal(err)
	}
	return a
}
func requireStatus(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status %d expected %d: %s", w.Code, status, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing no-store")
	}
}
func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var result T
	if json.Unmarshal(w.Body.Bytes(), &result) != nil {
		t.Fatal("invalid JSON response")
	}
	return result
}
