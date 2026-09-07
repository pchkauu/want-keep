//go:build integration

package storage_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func binding() connections.Binding {
	return connections.Binding{Provider: "raiffeisen", Environment: "test", AdapterBuildDigest: "sha256:" + strings.Repeat("a", 64), CollectorImageDigest: "sha256:" + strings.Repeat("b", 64), ContractVersion: "10", AllowlistRevision: "1", NonSecretConfigRevision: "1", OperatorPermissionRevision: "1"}
}
func (f *fixture) connection() string {
	f.t.Helper()
	id := uuid.NewString()
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		return f.store.CreateConnection(ctx, admission.Connection{HouseholdID: f.family.ID, ID: id, Provider: "raiffeisen", Owner: f.p.UserID(), Generation: 1, Authorized: true})
	}); err != nil {
		f.t.Fatal(err)
	}
	return id
}

func TestConnectionCarriesVerifiedHouseholdAndPreservesExternalOwner(t *testing.T) {
	f := newFixture(t)
	id := f.connection()
	c, err := f.store.Connection(testContext, f.q, id)
	if err != nil || c.HouseholdID != f.family.ID || c.Owner != f.p.UserID() {
		t.Fatal("connection identity lost", c, err)
	}
	owner := connections.ExternalOwnership{HouseholdID: c.HouseholdID, OwnerID: c.Owner}
	if owner.RequireManage(f.q) != nil || !errors.Is(owner.RequireAuthentication(f.q), household.ErrForbidden) {
		t.Fatal("partner impersonates external owner")
	}
	c.ID = uuid.NewString()
	c.HouseholdID = "foreign"
	if err = f.store.WithinHousehold(testContext, f.q, func(ctx context.Context) error { return f.store.CreateConnection(ctx, c) }); !errors.Is(err, household.ErrForbidden) {
		t.Fatal("foreign connection ownership accepted", err)
	}
}
func (f *fixture) admit(b connections.Binding) *admission.Service {
	f.t.Helper()
	service := admission.NewService(f.store, f.store)
	for _, kind := range []connections.CheckKind{connections.ProviderCheck, connections.HostCheck} {
		if _, err := service.RecordCheck(testContext, connections.Check{Kind: kind, Binding: b, Result: connections.CheckPassed, At: f.now}); err != nil {
			f.t.Fatal(err)
		}
	}
	return service
}
func (f *fixture) issued(service *admission.Service, id string, b connections.Binding) jobs.Job {
	f.t.Helper()
	j, err := service.RequestSync(testContext, f.p, id, b, time.Now().Add(time.Hour))
	if err != nil {
		f.t.Fatal(err)
	}
	entries, err := f.store.ClaimJobs(testContext, "sync", 100, time.Minute)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.ID == j.ID {
			return entry
		}
	}
	f.t.Fatal("sync not claimable")
	return jobs.Job{}
}
func TestAdmissionPersistenceConcurrentChecksAndBinding(t *testing.T) {
	f := newFixture(t)
	id := f.connection()
	b := binding()
	service := admission.NewService(f.store, f.store)
	if _, err := service.RequestSync(testContext, f.p, id, b, time.Now().Add(time.Hour)); !errors.Is(err, connections.ErrProviderNotAdmitted) || f.count("jobs") != 0 {
		t.Fatal("missing gate created job")
	}
	a, err := service.Rebind(testContext, b)
	if err != nil || a.Revision() != 1 {
		t.Fatal("initial admission revision")
	}
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, kind := range []connections.CheckKind{connections.ProviderCheck, connections.HostCheck} {
		wg.Add(1)
		go func(kind connections.CheckKind) {
			defer wg.Done()
			<-start
			_, e := service.RecordCheck(testContext, connections.Check{Kind: kind, Binding: b, Result: connections.CheckPassed, At: f.now})
			errs <- e
		}(kind)
	}
	close(start)
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	restarted := f.restart()
	service = admission.NewService(restarted, restarted)
	a, err = service.RecordCheck(testContext, connections.Check{Kind: connections.ProviderCheck, Binding: b, Result: connections.CheckPassed, At: f.now})
	if err != nil || a.Status() != connections.Admitted || a.Revision() != 3 || a.CheckedAt().String() != f.now.String() {
		t.Fatalf("restart changed evidence: %v", err)
	}
	originalJob, err := service.RequestSync(testContext, f.p, id, b, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	again, err := service.RequestSync(testContext, f.q, id, b, time.Now().Add(time.Hour))
	if err != nil || again.ID != originalJob.ID {
		t.Fatal("manual refresh duplicated job")
	}
	changed := b
	changed.AllowlistRevision = "2"
	a, err = service.Rebind(testContext, changed)
	if err != nil || a.Revision() != 4 || a.Status() != connections.Pending {
		t.Fatal("rebind failed")
	}
	old, err := f.store.Job(testContext, f.p, originalJob.ID)
	if err != nil || old.State != "canceled" {
		t.Fatal("unstarted job not invalidated")
	}
	a, err = service.Rebind(testContext, b)
	if err != nil || a.Revision() != 5 || a.Status() != connections.Pending {
		t.Fatal("A-B-A reset revision")
	}
	if _, err = service.RequestSync(testContext, f.p, id, b, time.Now().Add(time.Hour)); !errors.Is(err, connections.ErrProviderNotAdmitted) {
		t.Fatal("old evidence resurrected")
	}
	for _, kind := range []connections.CheckKind{connections.ProviderCheck, connections.HostCheck} {
		a, err = service.RecordCheck(testContext, connections.Check{Kind: kind, Binding: b, Result: connections.CheckPassed, At: instant("2026-09-07T12:00:01.123456789Z")})
		if err != nil {
			t.Fatal(err)
		}
	}
	if a.Revision() != 7 || a.Status() != connections.Admitted {
		t.Fatal("readmission revision")
	}
}
func TestRevocationBeforeAndAfterIOQuarantinesResult(t *testing.T) {
	f := newFixture(t)
	b := binding()
	service := f.admit(b)
	connection := f.connection()
	issued := f.issued(service, connection, b)
	if err := service.BeforeRead(testContext, f.p, issued); err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	finish := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(entered)
		<-finish
		_, err := service.CommitPage(testContext, f.p, issued, admission.Page{EvidenceRef: "synthetic:inflight", Coverage: "complete", Complete: true}, func(context.Context) error { t.Error("revoked result applied"); return nil })
		done <- err
	}()
	<-entered
	if _, err := service.RecordCheck(testContext, connections.Check{Kind: connections.HostCheck, Binding: b, Result: connections.CheckRevoked, At: instant("2026-09-07T12:00:01.123456789Z")}); err != nil {
		t.Fatal(err)
	}
	if err := service.BeforeRead(testContext, f.p, issued); !errors.Is(err, connections.ErrProviderNotAdmitted) {
		t.Fatal("IO allowed after revoke")
	}
	close(finish)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if f.count("quarantine") != 1 || f.count("source_records") != 0 || f.count("postings") != 0 {
		t.Fatal("stale financial effect")
	}
	j, err := f.store.Job(testContext, f.p, issued.ID)
	if err != nil || !j.CancelRequested || j.Cursor != "" {
		t.Fatal("cancellation or checkpoint")
	}
	if _, err = service.RecordCheck(testContext, connections.Check{Kind: connections.HostCheck, Binding: b, Result: connections.CheckPassed, At: instant("2026-09-07T12:00:02.123456789Z")}); err != nil {
		t.Fatal(err)
	}
	applied, err := service.CommitPage(testContext, f.p, issued, admission.Page{EvidenceRef: "synthetic:before-readmission", Coverage: "complete", Complete: true}, func(context.Context) error { t.Error("old revision accepted after readmission"); return nil })
	if err != nil || applied {
		t.Fatal("readmission bypass")
	}
}
func TestCommitFenceAndLeaseCannotBeBypassed(t *testing.T) {
	f := newFixture(t)
	service := f.admit(binding())
	id := f.connection()
	issued := f.issued(service, id, binding())
	account := f.account(money.RUB, "100")
	r := f.revision(uuid.NewString(), account, "-10", money.RUB, 1)
	applied, err := service.CommitPage(testContext, f.p, issued, admission.Page{EvidenceRef: "synthetic:valid", NextCursor: "p2", Coverage: "partial", Gaps: []string{"more_pages"}}, func(ctx context.Context) error { return f.writer.Append(ctx, f.p, r, 0) })
	if err != nil || !applied {
		t.Fatalf("valid result failed: %v", err)
	}
	if f.available(account) != "90" {
		t.Fatal("valid effect missing")
	}
	j, err := f.store.Job(testContext, f.p, issued.ID)
	if err != nil || j.Cursor != "p2" {
		t.Fatal("checkpoint not saved")
	}
	_, err = service.CommitPage(testContext, f.p, issued, admission.Page{EvidenceRef: "synthetic:repeat", Cursor: "", Coverage: "complete"}, func(context.Context) error { t.Error("repeated old page executed"); return nil })
	if err != nil || f.count("quarantine") != 1 {
		t.Fatal("old cursor not quarantined")
	}
	if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.store.Disconnect(ctx, id) }); err != nil {
		t.Fatal(err)
	}
	applied, err = service.CommitPage(testContext, f.p, issued, admission.Page{EvidenceRef: "synthetic:disconnected", Cursor: "p2", Coverage: "complete", Complete: true}, func(context.Context) error { t.Error("disconnected commit"); return nil })
	if err != nil || applied || f.count("quarantine") != 2 {
		t.Fatal("connection generation bypass")
	}
}

func TestAdmissionLockAndIssuedMetadataAreProtected(t *testing.T) {
	f := newFixture(t)
	service := f.admit(binding())
	issued := f.issued(service, f.connection(), binding())
	err := f.store.WithinAdmission(testContext, "bybit", "test", func(ctx context.Context) error {
		return f.store.WithinHousehold(ctx, f.p, func(ctx context.Context) error { return f.store.FenceSyncResult(ctx, f.p, issued) })
	})
	if err == nil {
		t.Fatal("wrong gate lock accepted")
	}
	for _, column := range []string{"binding", "admission_revision", "connection_generation", "household_id", "actor_id", "deadline", "max_attempts"} {
		var allowed bool
		if err = f.admin.QueryRow(testContext, `SELECT has_column_privilege('want_keep_app','want_keep.jobs',$1,'UPDATE')`, column).Scan(&allowed); err != nil || allowed {
			t.Fatalf("job identity writable: %s: %v", column, err)
		}
	}
	for _, column := range []string{"id", "kind", "payload_hash", "actor_id", "household_id", "registered_at"} {
		var allowed bool
		if err = f.admin.QueryRow(testContext, `SELECT has_column_privilege('want_keep_app','want_keep.command_tombstones',$1,'UPDATE')`, column).Scan(&allowed); err != nil || allowed {
			t.Fatalf("command identity writable: %s: %v", column, err)
		}
	}
	restarted := f.restart()
	err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		return restarted.WithinHousehold(ctx, f.p, func(context.Context) error { t.Error("nested independent transaction executed"); return nil })
	})
	if err == nil {
		t.Fatal("cross-store transaction silently escaped")
	}
}

func TestLeaseExpiresDuringPageCommitQuarantinesAfterRollback(t *testing.T) {
	for _, nested := range []bool{false, true} {
		name := "outermost"
		if nested {
			name = "nested_admission"
		}
		t.Run(name, func(t *testing.T) { testLeaseExpiryRollback(t, nested) })
	}
}
func testLeaseExpiryRollback(t *testing.T, nested bool) {
	f := newFixture(t)
	service := f.admit(binding())
	issued := f.issued(service, f.connection(), binding())
	account := f.account(money.RUB, "100")
	r := f.revision(uuid.NewString(), account, "-10", money.RUB, 1)
	appliedWrites := make(chan struct{})
	continueCommit := make(chan struct{})
	type result struct {
		applied bool
		err     error
	}
	done := make(chan result, 1)
	go func() {
		var applied bool
		commit := func(ctx context.Context) error {
			var err error
			applied, err = service.CommitPage(ctx, f.p, issued, admission.Page{EvidenceRef: "synthetic:lease-expired-during-commit", NextCursor: "p2", Coverage: "complete", Complete: true}, func(ctx context.Context) error {
				if err := f.writer.Append(ctx, f.p, r, 0); err != nil {
					return err
				}
				close(appliedWrites)
				<-continueCommit
				return nil
			})
			return err
		}
		var err error
		if nested {
			err = f.store.WithinAdmission(testContext, binding().Provider, binding().Environment, commit)
		} else {
			err = commit(testContext)
		}
		done <- result{applied, err}
	}()
	select {
	case <-appliedWrites:
	case early := <-done:
		t.Fatalf("page failed before barrier: %v", early.err)
	}
	_, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET lease_until=clock_timestamp()-INTERVAL '1 second' WHERE household_id=$1 AND id=$2`, f.family.ID, issued.ID)
	close(continueCommit)
	if err != nil {
		t.Fatal(err)
	}
	got := <-done
	if got.err != nil || got.applied || f.count("postings") != 0 || f.count("operation_revisions") != 0 || f.count("outbox") != 0 || f.count("quarantine") != 1 || f.available(account) != "100" {
		t.Fatalf("late expiry was not quarantined atomically: %+v", got)
	}
	current, err := f.store.Job(testContext, f.p, issued.ID)
	if err != nil || current.Cursor != "" {
		t.Fatal("expired attempt advanced checkpoint")
	}
}
