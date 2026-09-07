//go:build integration

package privacy_test

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	"github.com/pchkauu/want-keep/backend/internal/connections/access"
	"github.com/pchkauu/want-keep/backend/internal/connections/admission"
	"github.com/pchkauu/want-keep/backend/internal/connections/credentials"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	"github.com/pchkauu/want-keep/backend/internal/privacy/cryptobox"
)

type secretFixture struct {
	*fixture
	access     *access.Service
	admission  *admission.Service
	vault      *credentials.Vault
	connection string
	purpose    connections.SecretPurpose
	binding    connections.Binding
}

func newSecretFixture(t *testing.T, purpose connections.SecretPurpose) *secretFixture {
	f := newFixture(t)
	sf := &secretFixture{fixture: f, connection: uuid.NewString(), purpose: purpose, binding: connections.Binding{Provider: "raiffeisen", Environment: "test", AdapterBuildDigest: "sha256:" + strings.Repeat("a", 64), CollectorImageDigest: "sha256:" + strings.Repeat("b", 64), ContractVersion: "10", AllowlistRevision: "1", NonSecretConfigRevision: "1", OperatorPermissionRevision: "1"}}
	if err := f.store.WithinHousehold(testContext, f.a, func(ctx context.Context) error {
		return f.store.CreateConnection(ctx, admission.Connection{ID: sf.connection, Provider: "raiffeisen", Owner: f.a.UserID(), Generation: 1, Authorized: true, SecretPurpose: purpose})
	}); err != nil {
		t.Fatal(err)
	}
	sf.admission = admission.NewService(f.store, f.store)
	at, _ := calendar.ParseInstant(time.Now().UTC().Format(time.RFC3339Nano))
	for _, kind := range []connections.CheckKind{connections.ProviderCheck, connections.HostCheck} {
		if _, err := sf.admission.RecordCheck(testContext, connections.Check{Kind: kind, Binding: sf.binding, Result: connections.CheckPassed, At: at}); err != nil {
			t.Fatal(err)
		}
	}
	sf.access = access.NewService(f.identity, f.store, sf.admission)
	path := filepath.Join(t.TempDir(), "connection-keys")
	if err := cryptobox.Generate(path, "connections"); err != nil {
		t.Fatal(err)
	}
	ring, err := cryptobox.Load(path, "connections")
	if err != nil {
		t.Fatal(err)
	}
	sf.vault = credentials.New(sf.access, f.store, ring)
	return sf
}
func (f *secretFixture) save(data []byte) connections.SecretGrant {
	f.t.Helper()
	grant, err := f.access.Begin(testContext, f.ta, f.connection, f.purpose)
	if err != nil {
		f.t.Fatal(err)
	}
	if err = f.vault.Save(testContext, f.ta, grant.ID, f.purpose, data); err != nil {
		f.t.Fatal(err)
	}
	return grant
}
func (f *secretFixture) job() jobs.Job {
	f.t.Helper()
	j, err := f.admission.RequestSync(testContext, f.a, f.connection, f.binding, time.Now().Add(time.Hour))
	if err != nil {
		f.t.Fatal(err)
	}
	batch, err := f.store.ClaimJobs(testContext, "sync", 10, time.Minute)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, job := range batch {
		if job.ID == j.ID {
			return job
		}
	}
	f.t.Fatal("sync job unavailable")
	return jobs.Job{}
}
func TestSecretPurposeOwnerAndPersistedJobFences(t *testing.T) {
	for _, purpose := range []connections.SecretPurpose{connections.APICredentials, connections.OAuthTokens, connections.BrowserSession} {
		t.Run(string(purpose), func(t *testing.T) {
			f := newSecretFixture(t, purpose)
			plain := []byte("synthetic-private-credential-never-log")
			grant := f.save(plain)
			job := f.job()
			if _, err := f.access.Begin(testContext, f.tb, f.connection, purpose); !errors.Is(err, connections.ErrSecretAccess) {
				t.Fatal("partner authorization", err)
			}
			if err := f.vault.Save(testContext, f.tb, grant.ID, purpose, plain); err == nil {
				t.Fatal("partner saved owner secret")
			}
			if err := f.vault.Save(testContext, f.ta, grant.ID, purpose, plain); err == nil {
				t.Fatal("consumed grant replay")
			}
			var lent []byte
			if err := f.vault.WithJobSecret(testContext, f.a, job, purpose, func(data []byte) error {
				if !bytes.Equal(data, plain) {
					t.Error("secret round trip")
				}
				lent = data
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(lent, make([]byte, len(lent))) {
				t.Fatal("borrowed secret not cleared")
			}
			reject := func(j jobs.Job) {
				t.Helper()
				called := false
				err := f.vault.WithJobSecret(testContext, f.a, j, purpose, func([]byte) error { called = true; return nil })
				if err == nil || called {
					t.Fatal("invalid job read secret")
				}
			}
			changed := job
			changed.ConnectionGeneration++
			reject(changed)
			changed = job
			changed.AdmissionRevision++
			reject(changed)
			changed = job
			changed.LeaseToken = uuid.NewString()
			reject(changed)
			changed = job
			changed.ConnectionID = uuid.NewString()
			reject(changed)
			changed = job
			changed.Kind = "outbox"
			reject(changed)
			changed = job
			changed.Binding.AllowlistRevision = "2"
			reject(changed)
			changed = job
			changed.SecretPurpose = "unknown"
			reject(changed)
			changed = job
			changed.ActorID = f.b.UserID()
			if err := f.vault.WithJobSecret(testContext, f.b, changed, purpose, func([]byte) error { t.Error("forged actor reached consumer"); return nil }); err == nil {
				t.Fatal("forged actor accepted")
			}
			var cipher []byte
			if err := f.admin.QueryRow(testContext, `SELECT ciphertext FROM want_keep.connection_secrets WHERE household_id=$1 AND connection_id=$2`, f.a.HouseholdID(), f.connection).Scan(&cipher); err != nil || bytes.Contains(cipher, plain) {
				t.Fatal("plaintext database", err)
			}
			if _, err := f.admin.Exec(testContext, `UPDATE want_keep.connection_secrets SET ciphertext=$3 WHERE household_id=$1 AND connection_id=$2`, f.a.HouseholdID(), f.connection, append([]byte{}, cipher[:len(cipher)-1]...)); err != nil {
				t.Fatal(err)
			}
			reject(job)
		})
	}
}
func TestDisconnectDuringIORevokesAndQuarantines(t *testing.T) {
	f := newSecretFixture(t, connections.BrowserSession)
	f.save([]byte("synthetic browser state"))
	job := f.job()
	grant, err := f.access.Begin(testContext, f.ta, f.connection, f.purpose)
	if err != nil {
		t.Fatal(err)
	}
	entered, finish := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- f.vault.WithJobSecret(testContext, f.a, job, f.purpose, func([]byte) error {
			close(entered)
			<-finish
			applied, err := f.admission.CommitPage(testContext, f.a, job, admission.Page{EvidenceRef: "synthetic:late-private-io", Coverage: "complete", Complete: true}, func(context.Context) error { t.Error("stale page reached financial storage"); return nil })
			if applied {
				t.Error("stale page accepted")
			}
			return err
		})
	}()
	select {
	case <-entered:
	case err := <-done:
		t.Fatal("read failed before barrier", err)
	}
	if err = f.access.Disconnect(testContext, f.tb, f.connection, 1); err != nil {
		close(finish)
		t.Fatal(err)
	}
	close(finish)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if err = f.vault.Save(testContext, f.ta, grant.ID, f.purpose, []byte("late")); err == nil {
		t.Fatal("revoked grant saved secret")
	}
	if err = f.vault.WithJobSecret(testContext, f.a, job, f.purpose, func([]byte) error { t.Error("IO after disconnect"); return nil }); err == nil {
		t.Fatal("revoked job read")
	}
	var secretRevoked, grantRevoked bool
	var quarantine, postings int
	if err = f.admin.QueryRow(testContext, `SELECT (SELECT bool_and(revoked) FROM want_keep.connection_secrets),(SELECT bool_and(revoked) FROM want_keep.connection_secret_grants),(SELECT count(*) FROM want_keep.quarantine),(SELECT count(*) FROM want_keep.postings)`).Scan(&secretRevoked, &grantRevoked, &quarantine, &postings); err != nil || !secretRevoked || !grantRevoked || quarantine != 1 || postings != 0 {
		t.Fatal("disconnect not atomic", err)
	}
}
func TestSecretGrantConcurrentConsumptionAndSessionRevocation(t *testing.T) {
	f := newSecretFixture(t, connections.APICredentials)
	grant, err := f.access.Begin(testContext, f.ta, f.connection, f.purpose)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- f.vault.Save(testContext, f.ta, grant.ID, f.purpose, []byte("synthetic"))
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatal("grant not single use", succeeded)
	}
	grant, err = f.access.Begin(testContext, f.ta, f.connection, f.purpose)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.identity.Logout(testContext, f.ta); err != nil {
		t.Fatal(err)
	}
	if err = f.vault.Save(testContext, f.ta, grant.ID, f.purpose, []byte("after logout")); err == nil {
		t.Fatal("expired session saved")
	}
	if _, err = f.identity.Me(testContext, f.tb); err != nil {
		t.Fatal("partner session revoked", err)
	}
}
