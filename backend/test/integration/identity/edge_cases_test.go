//go:build integration

package identity_test

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	delivery "github.com/pchkauu/want-keep/backend/internal/delivery/identity"
	application "github.com/pchkauu/want-keep/backend/internal/identity/application"
	domain "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
)

func TestRecoveryWithExpiredCookieAndRollback(t *testing.T) {
	f := newFixture(t)
	b := f.browser()
	old, codes, _ := b.setup()
	f.advance(30 * time.Minute)
	b.csrf = ""
	grant := decode[generated.RecoveryAttempt](t, b.call("POST", "/auth/recovery", map[string]any{"recoveryCode": codes[0]}, 200))
	if b.cookies["want_keep_session"] != nil {
		t.Fatal("expired cookie retained")
	}
	opts := decode[generated.EnrollmentOptions](t, b.call("POST", "/auth/enrollment/options", map[string]any{"purpose": "recovery", "authorizationToken": grant.EnrollmentToken}, 200))
	key := newAuthenticator(t, opts.UserId, false)
	payload := key.registration(t, opts.Challenge, "http://localhost", "localhost", 5)
	raw, _ := json.Marshal(payload)
	var dto generated.RegistrationCredential
	if err := json.Unmarshal(raw, &dto); err != nil {
		t.Fatal(err)
	}
	input := application.Registration{ID: dto.Id, RawID: dto.RawId, ClientDataJSON: dto.ClientDataJSON, AttestationObject: dto.AttestationObject, Transports: []string{"internal"}}
	verifier, _ := webauthn.New("localhost", "http://localhost")
	faulty, err := application.NewService(failSessionRepository{f.store}, f.store, verifier, "localhost", "http://localhost", 2, f.clock)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = faulty.CompleteEnrollment(testContext, b.context(), opts.AttemptId, "New", input); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatal(err)
	}
	f.browser().login(old, "login")
	// Retry the same cryptographic completion after rollback, then deliberately lose its secret response.
	if _, err = f.service.CompleteEnrollment(testContext, b.context(), opts.AttemptId, "New", input); err != nil {
		t.Fatal(err)
	}
	f.restart()
	fresh := f.browser()
	fresh.login(key, "login")
	fresh.call("POST", "/security/recovery-codes", map[string]any{}, 200)
	if _, err = f.service.CompleteEnrollment(testContext, b.context(), opts.AttemptId, "New", input); !errors.Is(err, domain.ErrAttempt) {
		t.Fatal("secret replay", err)
	}
}

func TestBootstrapExpiryAndConcurrentCompletion(t *testing.T) {
	f := newFixture(t)
	b := f.browser()
	setup := domain.Setup{Name: "A", HouseholdName: "Family", Timezone: "UTC", Locale: "en", ReportingAsset: "USD"}
	b.call("POST", "/auth/login/options", map[string]any{}, 200)
	wrong, _ := domain.NewToken(32)
	if _, err := f.service.BeginBootstrap(testContext, b.context(), wrong, setup); !errors.Is(err, domain.ErrBootstrap) {
		t.Fatal(err)
	}
	f.advance(30 * time.Minute)
	if _, err := f.service.BeginBootstrap(testContext, b.context(), f.bootstrap, setup); !errors.Is(err, domain.ErrBootstrap) {
		t.Fatal(err)
	}
	f.advance(-30 * time.Minute)
	enrollments := make([]application.Enrollment, 2)
	inputs := make([]application.Registration, 2)
	for i := range 2 {
		var err error
		enrollments[i], err = f.service.BeginBootstrap(testContext, b.context(), f.bootstrap, setup)
		if err != nil {
			t.Fatal(err)
		}
		key := newAuthenticator(t, base64.RawURLEncoding.EncodeToString(enrollments[i].Profile.Handle), false)
		raw, _ := json.Marshal(key.registration(t, enrollments[i].Attempt.Challenge, "http://localhost", "localhost", 5))
		var dto generated.RegistrationCredential
		if err = json.Unmarshal(raw, &dto); err != nil {
			t.Fatal(err)
		}
		inputs[i] = application.Registration{ID: dto.Id, RawID: dto.RawId, ClientDataJSON: dto.ClientDataJSON, AttestationObject: dto.AttestationObject, Transports: []string{"internal"}}
	}
	start := make(chan struct{})
	out := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for i := range 2 {
		go func() {
			ready.Done()
			<-start
			_, err := f.service.CompleteEnrollment(context.Background(), b.context(), enrollments[i].Attempt.ID, "Key", inputs[i])
			out <- err
		}()
	}
	ready.Wait()
	close(start)
	success := 0
	for range 2 {
		err := <-out
		if err == nil {
			success++
		} else if !errors.Is(err, domain.ErrBootstrap) {
			t.Fatal(err)
		}
	}
	var families int
	if err := f.admin.QueryRow(testContext, "SELECT count(*) FROM want_keep.households").Scan(&families); err != nil {
		t.Fatal(err)
	}
	if success != 1 || families != 1 {
		t.Fatal("non-exclusive bootstrap", success, families)
	}
}

func TestZeroCounterBrowserBindingAndAttemptExpiry(t *testing.T) {
	f := newFixture(t)
	key, _, _ := f.browser().setup()
	for range 2 {
		key.count = math.MaxUint32
		f.browser().login(key, "login")
	}
	b := f.browser()
	o := decode[generated.LoginOptions](t, b.call("POST", "/auth/login/options", map[string]any{}, 200))
	c := key.assertion(t, o.Challenge, "http://localhost", "localhost", 5)
	other := f.browser()
	other.call("POST", "/auth/login/options", map[string]any{}, 200)
	other.call("POST", "/auth/login/verify", map[string]any{"attemptId": o.AttemptId, "credential": c}, 400)
	f.advance(5 * time.Minute)
	b.call("POST", "/auth/login/verify", map[string]any{"attemptId": o.AttemptId, "credential": c}, 400)
}

func TestHTTPOriginProxyAndRequestBoundaries(t *testing.T) {
	f := newFixture(t)
	for _, change := range []string{"origin", "host", "cross-site", "body", "unknown"} {
		req := httptest.NewRequest("POST", "http://localhost/api/v1/auth/login/options", strings.NewReader("{}"))
		req.Header.Set("Origin", "http://localhost")
		req.Header.Set("Content-Type", "application/json")
		switch change {
		case "origin":
			req.Header.Set("Origin", "https://attacker.invalid")
		case "host":
			req.Host = "attacker.invalid"
		case "cross-site":
			req.Header.Set("Sec-Fetch-Site", "cross-site")
		case "body":
			req = httptest.NewRequest("POST", "http://localhost/api/v1/auth/login/options", strings.NewReader(strings.Repeat(" ", 128*1024)+"{}"))
			req.Header.Set("Origin", "http://localhost")
			req.Header.Set("Content-Type", "application/json")
		case "unknown":
			req.URL.Path = "/api/v1/accounts"
		}
		rr := httptest.NewRecorder()
		f.handler.ServeHTTP(rr, req)
		if rr.Code < 400 {
			t.Fatal("accepted", change)
		}
	}
	proxy, err := delivery.New(f.service, delivery.Config{Environment: "production", Origin: "https://want-keep.tech", TrustedProxies: []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")}})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		peer    string
		tls     bool
		forward string
		status  int
	}{{"198.51.100.1:80", false, "https", 401}, {"192.0.2.1:80", false, "http", 401}, {"192.0.2.1:80", false, "https", 200}, {"198.51.100.1:80", true, "", 200}} {
		req := httptest.NewRequest("POST", "https://want-keep.tech/api/v1/auth/login/options", strings.NewReader("{}"))
		req.TLS = nil
		if test.tls {
			req.TLS = &tls.ConnectionState{}
		}
		req.RemoteAddr = test.peer
		req.Header.Set("Origin", "https://want-keep.tech")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-Proto", test.forward)
		rr := httptest.NewRecorder()
		proxy.ServeHTTP(rr, req)
		if rr.Code != test.status {
			t.Fatal("proxy", rr.Code, test.status)
		}
	}
}
