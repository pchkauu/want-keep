//go:build integration

package storage_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	app "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func TestConcurrentScheduleAndManualRefresh(t *testing.T) {
	f := newFixture(t)
	id := f.connection()
	b := binding()
	service := f.admit(b)
	scheduler := app.Scheduler{Repository: f.store, Admission: service, Bindings: []connections.Binding{b}}
	var wg sync.WaitGroup
	errs := make(chan error, 3)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- scheduler.Tick(testContext) }()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := service.RequestSync(testContext, f.q, id, b, time.Now().Add(time.Hour))
		errs <- err
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if f.count("jobs") != 1 {
		t.Fatal("duplicate sync")
	}
	if err := scheduler.Tick(testContext); err != nil || f.count("jobs") != 1 {
		t.Fatal("schedule repeated", err)
	}
	if _, err := f.admin.Exec(testContext, `UPDATE want_keep.sync_schedules SET next_due=clock_timestamp()-INTERVAL '8 hours'`); err != nil {
		t.Fatal(err)
	}
	if err := scheduler.Tick(testContext); err != nil || f.count("jobs") != 1 {
		t.Fatal("missed intervals expanded", err)
	}
}
func TestCheckpointSurvivesTerminalFailureAndReauth(t *testing.T) {
	f := newFixture(t)
	id := f.connection()
	b := binding()
	service := f.admit(b)
	issued := f.issued(service, id, b)
	apply := func(context.Context) error { return nil }
	if ok, err := service.CommitPage(testContext, f.p, issued, admission.Page{EvidenceRef: "synthetic:page1", Cursor: "", NextCursor: "page2", Coverage: "partial", Gaps: []string{"history_limited"}}, apply); err != nil || !ok {
		t.Fatal("first page", err)
	}
	if err := f.store.SetJobOutcome(testContext, f.p, issued, jobs.Failed, jobs.PermanentFailure, 0); err != nil {
		t.Fatal(err)
	}
	next := f.issued(service, id, b)
	if next.Cursor != "page2" || len(next.Gaps) != 1 {
		t.Fatal("checkpoint lost")
	}
	progress, err := f.store.SyncProgress(testContext, f.p, id)
	if err != nil || progress.LastSuccessAt != nil {
		t.Fatal("failure reported success", err)
	}
	if ok, err := service.CommitPage(testContext, f.p, next, admission.Page{EvidenceRef: "synthetic:page2", Cursor: "page2", Coverage: "complete", Complete: true}, apply); err != nil || !ok {
		t.Fatal("resume", err)
	}
	progress, err = f.store.SyncProgress(testContext, f.p, id)
	if err != nil || progress.LastSuccessAt == nil || progress.Coverage != "partial" {
		t.Fatal("coverage/success lost", err)
	}
	last := *progress.LastSuccessAt
	j := f.issued(service, id, b)
	if err = f.store.SetJobOutcome(testContext, f.p, j, jobs.Waiting, jobs.ReauthRequired, 0); err != nil {
		t.Fatal(err)
	}
	same, err := service.RequestSync(testContext, f.q, id, b, time.Now().Add(time.Hour))
	if err != nil || same.ID != j.ID {
		t.Fatal("waiting sync duplicated", err)
	}
	progress, _ = f.store.SyncProgress(testContext, f.p, id)
	if !progress.LastSuccessAt.Equal(last) {
		t.Fatal("reauth changed success time")
	}
}
func TestDisconnectedLeaseAndForgedActorCannotApply(t *testing.T) {
	f := newFixture(t)
	id := f.connection()
	b := binding()
	service := f.admit(b)
	j := f.issued(service, id, b)
	if err := f.store.Heartbeat(testContext, f.q, j, time.Minute); !errors.Is(err, household.ErrForbidden) {
		t.Fatal("partner acknowledged issued actor")
	}
	if err := f.store.WithinHousehold(testContext, f.q, func(ctx context.Context) error { return f.store.Disconnect(ctx, id) }); err != nil {
		t.Fatal(err)
	}
	applied := false
	ok, err := service.CommitPage(testContext, f.p, j, admission.Page{EvidenceRef: "synthetic:late", Coverage: "complete", Complete: true}, func(context.Context) error { applied = true; return nil })
	if err != nil || ok || applied || f.count("quarantine") != 1 {
		t.Fatal("stale disconnect result applied", err)
	}
}
