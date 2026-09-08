package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	application "github.com/pchkauu/want-keep/backend/internal/integrations/application"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func TestEvidenceIsDurableBeforeCommitAndFailureClosesTheBoundary(t *testing.T) {
	for _, test := range []struct {
		name          string
		evidenceError error
		wantCommit    bool
	}{
		{name: "saved before commit", wantCommit: true},
		{name: "storage failure", evidenceError: errors.New("unavailable")},
	} {
		t.Run(test.name, func(t *testing.T) {
			stored := false
			gate := &gateFake{commit: func(_ context.Context, _ household.Principal, _ jobs.Job, _ admission.Page, apply func(context.Context) error) (bool, error) {
				if !stored {
					t.Fatal("financial transaction opened before evidence was stored")
				}
				return true, apply(context.Background())
			}}
			evidence := evidenceFake{save: func(_ context.Context, batch ingestion.EvidenceBatch) error {
				if err := batch.Validate(); err != nil {
					return err
				}
				stored = true
				return test.evidenceError
			}}
			service, err := application.NewService(gate, evidence, accountFake{}, sourceFake{}, now, func() string { return "server-id" })
			if err != nil {
				t.Fatal(err)
			}
			job := issued()
			token, _ := application.TokenFromJob(job)
			page := page(token)
			applied, failure, err := service.Ingest(context.Background(), principal(), job, &gatewayFake{result: ingestion.Result{Page: &page}})
			if test.evidenceError != nil {
				if applied || failure != nil || !errors.Is(err, ingestion.ErrEvidence) || gate.commitCalls != 0 {
					t.Fatal("evidence failure did not close commit boundary", applied, failure, err)
				}
			} else if !applied || failure != nil || err != nil || gate.commitCalls != 1 {
				t.Fatal("valid page was not applied", applied, failure, err)
			}
		})
	}
}

func TestProviderFailureKeepsTypedReasonWithoutFinancialApply(t *testing.T) {
	gate := &gateFake{failure: func(_ context.Context, _ household.Principal, _ jobs.Job, _ string, apply func(context.Context) error) (bool, error) {
		return true, apply(context.Background())
	}}
	service, _ := application.NewService(gate, evidenceFake{save: func(context.Context, ingestion.EvidenceBatch) error { return nil }}, accountFake{}, sourceFake{}, now, func() string { return "server-id" })
	job := issued()
	token, _ := application.TokenFromJob(job)
	failure := ingestion.ProviderFailure{Token: token, Kind: ingestion.MFARequired, Evidence: []ingestion.Evidence{rawEvidence()}}
	applied, got, err := service.Ingest(context.Background(), principal(), job, &gatewayFake{result: ingestion.Result{Failure: &failure}})
	if err != nil || !applied || got == nil || got.Kind != ingestion.MFARequired || gate.commitCalls != 0 || gate.failureCalls != 1 {
		t.Fatal("typed provider failure crossed financial apply", applied, got, err)
	}
}

func TestReadFencePrecedesEveryGatewayCall(t *testing.T) {
	denied := errors.New("not admitted")
	gate := &gateFake{beforeRead: func(context.Context, household.Principal, jobs.Job) error { return denied }}
	gateway := &gatewayFake{}
	service, _ := application.NewService(gate, evidenceFake{save: func(context.Context, ingestion.EvidenceBatch) error { return nil }}, accountFake{}, sourceFake{}, now, func() string { return "server-id" })

	applied, failure, err := service.Ingest(context.Background(), principal(), issued(), gateway)
	if applied || failure != nil || !errors.Is(err, denied) || gateway.calls != 0 {
		t.Fatal("gateway called before read fence", applied, failure, err, gateway.calls)
	}
}

type gateFake struct {
	commitCalls, failureCalls int
	beforeRead                func(context.Context, household.Principal, jobs.Job) error
	commit                    func(context.Context, household.Principal, jobs.Job, admission.Page, func(context.Context) error) (bool, error)
	failure                   func(context.Context, household.Principal, jobs.Job, string, func(context.Context) error) (bool, error)
}

func (g *gateFake) BeforeRead(ctx context.Context, p household.Principal, job jobs.Job) error {
	if g.beforeRead != nil {
		return g.beforeRead(ctx, p, job)
	}
	return nil
}
func (g *gateFake) CommitPage(ctx context.Context, p household.Principal, job jobs.Job, page admission.Page, apply func(context.Context) error) (bool, error) {
	g.commitCalls++
	if g.commit == nil {
		return false, errors.New("unexpected commit")
	}
	return g.commit(ctx, p, job, page, apply)
}
func (g *gateFake) CommitFailure(ctx context.Context, p household.Principal, job jobs.Job, evidence string, apply func(context.Context) error) (bool, error) {
	g.failureCalls++
	if g.failure == nil {
		return false, errors.New("unexpected failure commit")
	}
	return g.failure(ctx, p, job, evidence, apply)
}

type evidenceFake struct {
	save func(context.Context, ingestion.EvidenceBatch) error
}

func (e evidenceFake) Save(ctx context.Context, batch ingestion.EvidenceBatch) error {
	return e.save(ctx, batch)
}

type accountFake struct{}

func (accountFake) Resolve(context.Context, household.Principal, accounts.ImportInput) (accounts.ImportResult, error) {
	return accounts.ImportResult{}, nil
}

func (accountFake) Import(context.Context, household.Principal, accounts.ImportInput) (accounts.ImportResult, error) {
	return accounts.ImportResult{}, nil
}

type sourceFake struct{}

func (sourceFake) AccountTimezone(context.Context, household.Principal) (calendar.Timezone, error) {
	return calendar.ParseTimezone("UTC")
}
func (sourceFake) Source(context.Context, household.Principal, ledger.SourceKey) (ledger.SourceRecord, bool, error) {
	return ledger.SourceRecord{}, false, nil
}
func (sourceFake) Apply(context.Context, household.Principal, ledger.SourceInput) (ledger.SourceOutcome, error) {
	return ledger.SourceOutcome{}, nil
}

type gatewayFake struct {
	result ingestion.Result
	calls  int
}

func (g *gatewayFake) Manifest(context.Context) (ingestion.Manifest, error) {
	g.calls++
	return ingestion.Manifest{Provider: "raiffeisen", Version: "10", Actions: []ingestion.ReadAction{ingestion.ReadAccounts}, Products: []string{"current"}, Logs: []ingestion.CapabilityLog{{Product: "current", Namespace: "accounts", RecordKinds: []ingestion.RecordKind{ingestion.AccountRecordKind}}}}, nil
}
func (g *gatewayFake) Read(context.Context, ingestion.JobToken) (ingestion.Result, error) {
	g.calls++
	return g.result, nil
}

func issued() jobs.Job {
	t := time.Now().Add(time.Hour)
	return jobs.Job{ID: "11111111-1111-4111-8111-111111111111", HouseholdID: "household", ActorID: "user", Kind: "sync", ConnectionID: "22222222-2222-4222-8222-222222222222", ConnectionGeneration: 1, Binding: binding(), AdmissionRevision: 1, State: jobs.Running, Attempt: 1, LeaseToken: "lease", LeaseUntil: t, Deadline: t}
}

func binding() connections.Binding {
	return connections.Binding{Provider: "raiffeisen", Environment: "test", AdapterBuildDigest: "sha256:" + strings.Repeat("a", 64), CollectorImageDigest: "sha256:" + strings.Repeat("b", 64), ContractVersion: "10", AllowlistRevision: "1", NonSecretConfigRevision: "1", OperatorPermissionRevision: "1"}
}

func principal() household.Principal {
	p, _ := (household.Membership{ID: "member", UserID: "user", HouseholdID: "household", Active: true}).Principal()
	return p
}

func page(token ingestion.JobToken) ingestion.Page {
	coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
	return ingestion.Page{Token: token, Complete: true, Coverage: coverage, Evidence: []ingestion.Evidence{rawEvidence()}}
}

func rawEvidence() ingestion.Evidence {
	return ingestion.Evidence{ID: "raw", MediaType: "application/json", Digest: "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a", Locator: "synthetic:raw", Data: []byte("{}")}
}

func now() calendar.Instant {
	value, _ := calendar.ParseInstant("2026-09-08T12:00:00.123456789Z")
	return value
}
