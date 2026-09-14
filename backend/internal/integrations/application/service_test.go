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
	transaction "github.com/pchkauu/want-keep/backend/internal/transaction/domain"
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
				if batch.HouseholdID != string(principal().HouseholdID()) || batch.JobID != issued().ID || batch.Disposition != ingestion.EvidenceStaged {
					t.Fatal("evidence lost trusted ownership", batch.HouseholdID, batch.JobID)
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

func TestRejectedPageRetainsOwnedEvidenceDisposition(t *testing.T) {
	commitError := errors.New("invalid account projection")
	ctx, cancel := context.WithCancel(context.Background())
	retained, finalized := false, false
	gate := &gateFake{
		commit: func(context.Context, household.Principal, jobs.Job, admission.Page, func(context.Context) error) (bool, error) {
			cancel()
			return false, errors.Join(admission.ErrPageRejected, commitError)
		},
		reject: func(ctx context.Context, p household.Principal, job jobs.Job, evidence string) error {
			retained = ctx.Err() == nil && p.HouseholdID() == job.HouseholdID && job.ID == issued().ID && evidence != ""
			return nil
		},
	}
	evidence := evidenceFake{
		save: func(_ context.Context, batch ingestion.EvidenceBatch) error { return batch.Validate() },
		disposition: func(ctx context.Context, disposition ingestion.EvidenceDisposition) error {
			finalized = ctx.Err() == nil && disposition.State == ingestion.EvidenceRejected && disposition.Validate() == nil
			return nil
		},
	}
	service, _ := application.NewService(gate, evidence, accountFake{}, sourceFake{}, now, func() string { return "server-id" })
	job := issued()
	token, _ := application.TokenFromJob(job)
	result := page(token)
	applied, failure, err := service.Ingest(ctx, principal(), job, &gatewayFake{result: ingestion.Result{Page: &result}})
	if applied || failure != nil || !errors.Is(err, commitError) || !retained || !finalized || gate.rejectCalls != 1 {
		t.Fatal("rejected evidence did not receive a durable disposition", applied, failure, retained, finalized, gate.rejectCalls, err)
	}
}

func TestRejectedEvidenceStaysStagedWhenRetentionFails(t *testing.T) {
	commitError := errors.New("invalid account projection")
	retentionError := errors.New("rejection storage unavailable")
	finalized := false
	gate := &gateFake{
		commit: func(context.Context, household.Principal, jobs.Job, admission.Page, func(context.Context) error) (bool, error) {
			return false, errors.Join(ingestion.ErrResultRejected, commitError)
		},
		reject: func(context.Context, household.Principal, jobs.Job, string) error { return retentionError },
	}
	evidence := evidenceFake{
		save: func(_ context.Context, batch ingestion.EvidenceBatch) error { return batch.Validate() },
		disposition: func(context.Context, ingestion.EvidenceDisposition) error {
			finalized = true
			return nil
		},
	}
	service, _ := application.NewService(gate, evidence, accountFake{}, sourceFake{}, now, func() string { return "server-id" })
	job := issued()
	token, _ := application.TokenFromJob(job)
	result := page(token)
	applied, failure, err := service.Ingest(context.Background(), principal(), job, &gatewayFake{result: ingestion.Result{Page: &result}})
	if applied || failure != nil || !errors.Is(err, commitError) || !errors.Is(err, retentionError) || finalized {
		t.Fatal("unretained rejection left staged evidence", applied, failure, finalized, err)
	}
}

func TestStaleRetentionFailureNeverBecomesRejected(t *testing.T) {
	retentionError := errors.New("stale quarantine unavailable")
	for _, providerFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "page", true: "provider outcome"}[providerFailure], func(t *testing.T) {
			finalized := false
			staleError := errors.Join(jobs.ErrStaleAttempt, retentionError)
			gate := &gateFake{
				commit: func(context.Context, household.Principal, jobs.Job, admission.Page, func(context.Context) error) (bool, error) {
					return false, staleError
				},
				failure: func(context.Context, household.Principal, jobs.Job, string, jobs.State, jobs.Reason, time.Duration, func(context.Context) error) (bool, error) {
					return false, staleError
				},
			}
			evidence := evidenceFake{
				save: func(_ context.Context, batch ingestion.EvidenceBatch) error { return batch.Validate() },
				disposition: func(context.Context, ingestion.EvidenceDisposition) error {
					finalized = true
					return nil
				},
			}
			service, _ := application.NewService(gate, evidence, accountFake{}, sourceFake{}, now, func() string { return "server-id" })
			job := issued()
			token, _ := application.TokenFromJob(job)
			result := ingestion.Result{}
			if providerFailure {
				result.Failure = &ingestion.ProviderFailure{Token: token, Kind: ingestion.MFARequired, Evidence: []ingestion.Evidence{rawEvidence()}}
			} else {
				page := page(token)
				result.Page = &page
			}

			applied, _, err := service.Ingest(context.Background(), principal(), job, &gatewayFake{result: result})
			if applied || !errors.Is(err, jobs.ErrStaleAttempt) || !errors.Is(err, retentionError) || gate.rejectCalls != 0 || finalized {
				t.Fatal("stale evidence was misclassified as rejected", applied, gate.rejectCalls, finalized, err)
			}
		})
	}
}

func TestStagedEvidenceCanBeFinalizedAfterRestart(t *testing.T) {
	for _, test := range []struct {
		kind admission.ResultKind
		want ingestion.EvidenceDispositionState
	}{
		{kind: admission.PageResult, want: ingestion.EvidenceApplied},
		{kind: admission.ProviderOutcomeResult, want: ingestion.EvidenceProviderOutcome},
		{kind: admission.RejectedResult, want: ingestion.EvidenceRejected},
		{kind: admission.StaleResult, want: ingestion.EvidenceStale},
	} {
		t.Run(string(test.kind), func(t *testing.T) {
			staged := ingestion.StagedEvidence{HouseholdID: string(principal().HouseholdID()), JobID: issued().ID, PageReference: "evidence:page:restart"}
			var got ingestion.EvidenceDisposition
			gate := &gateFake{evidenceResult: func(_ context.Context, p household.Principal, jobID, evidence string) (admission.ResultKind, bool, error) {
				if p.HouseholdID() != principal().HouseholdID() || jobID != staged.JobID || evidence != staged.PageReference {
					t.Fatal("staged evidence lost trusted scope")
				}
				return test.kind, true, nil
			}}
			evidence := evidenceFake{
				save: func(context.Context, ingestion.EvidenceBatch) error { return nil },
				staged: func(context.Context, string, int) ([]ingestion.StagedEvidence, error) {
					return []ingestion.StagedEvidence{staged}, nil
				},
				disposition: func(_ context.Context, value ingestion.EvidenceDisposition) error {
					got = value
					return nil
				},
			}
			service, _ := application.NewService(gate, evidence, accountFake{}, sourceFake{}, now, func() string { return "server-id" })
			completed, err := service.ReconcileStaged(context.Background(), principal(), 100)
			if err != nil || completed != 1 || got.State != test.want || got.PageReference != staged.PageReference {
				t.Fatal("staged evidence was not finalized", completed, got, err)
			}
		})
	}
}

func TestStagedEvidenceWithoutDurableResultRemainsStaged(t *testing.T) {
	staged := ingestion.StagedEvidence{HouseholdID: string(principal().HouseholdID()), JobID: issued().ID, PageReference: "evidence:page:unknown"}
	finalized := false
	evidence := evidenceFake{
		save: func(context.Context, ingestion.EvidenceBatch) error { return nil },
		staged: func(context.Context, string, int) ([]ingestion.StagedEvidence, error) {
			return []ingestion.StagedEvidence{staged}, nil
		},
		disposition: func(context.Context, ingestion.EvidenceDisposition) error {
			finalized = true
			return nil
		},
	}
	service, _ := application.NewService(&gateFake{}, evidence, accountFake{}, sourceFake{}, now, func() string { return "server-id" })
	completed, err := service.ReconcileStaged(context.Background(), principal(), 100)
	if err != nil || completed != 0 || finalized {
		t.Fatal("unknown result changed staged evidence", completed, finalized, err)
	}
}

func TestOperationalCommitErrorsLeaveEvidenceStaged(t *testing.T) {
	commitError := errors.New("provider outcome unavailable")
	for _, providerFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "page", true: "provider outcome"}[providerFailure], func(t *testing.T) {
			finalized := false
			gate := &gateFake{
				commit: func(context.Context, household.Principal, jobs.Job, admission.Page, func(context.Context) error) (bool, error) {
					return false, commitError
				},
				failure: func(context.Context, household.Principal, jobs.Job, string, jobs.State, jobs.Reason, time.Duration, func(context.Context) error) (bool, error) {
					return false, commitError
				},
			}
			evidence := evidenceFake{
				save: func(_ context.Context, batch ingestion.EvidenceBatch) error { return batch.Validate() },
				disposition: func(context.Context, ingestion.EvidenceDisposition) error {
					finalized = true
					return nil
				},
			}
			service, _ := application.NewService(gate, evidence, accountFake{}, sourceFake{}, now, func() string { return "server-id" })
			job := issued()
			token, _ := application.TokenFromJob(job)
			result := ingestion.Result{}
			if providerFailure {
				result.Failure = &ingestion.ProviderFailure{Token: token, Kind: ingestion.MFARequired, Evidence: []ingestion.Evidence{rawEvidence()}}
			} else {
				page := page(token)
				result.Page = &page
			}

			applied, got, err := service.Ingest(context.Background(), principal(), job, &gatewayFake{result: result})
			if applied || !errors.Is(err, commitError) || gate.rejectCalls != 0 || finalized || providerFailure != (got != nil) {
				t.Fatal("operational error changed staged evidence", applied, got, gate.rejectCalls, finalized, err)
			}
		})
	}
}

func TestCommitUnknownUsesReceiptWithoutGuessingEvidenceDisposition(t *testing.T) {
	commitError := errors.Join(transaction.ErrCommitOutcomeUnknown, errors.New("connection lost"))
	for _, test := range []struct {
		name      string
		confirmed bool
		readError error
		wantState ingestion.EvidenceDispositionState
	}{
		{name: "confirmed page", confirmed: true, wantState: ingestion.EvidenceApplied},
		{name: "missing receipt"},
		{name: "readback failed", readError: errors.New("read unavailable")},
	} {
		t.Run(test.name, func(t *testing.T) {
			var disposition ingestion.EvidenceDispositionState
			gate := &gateFake{
				commit: func(context.Context, household.Principal, jobs.Job, admission.Page, func(context.Context) error) (bool, error) {
					return false, commitError
				},
				receipt: func(context.Context, household.Principal, jobs.Job, string, admission.ResultKind) (bool, error) {
					return test.confirmed, test.readError
				},
			}
			evidence := evidenceFake{
				save: func(_ context.Context, batch ingestion.EvidenceBatch) error { return batch.Validate() },
				disposition: func(_ context.Context, value ingestion.EvidenceDisposition) error {
					disposition = value.State
					return nil
				},
			}
			service, _ := application.NewService(gate, evidence, accountFake{}, sourceFake{}, now, func() string { return "server-id" })
			job := issued()
			token, _ := application.TokenFromJob(job)
			result := page(token)
			applied, failure, err := service.Ingest(context.Background(), principal(), job, &gatewayFake{result: ingestion.Result{Page: &result}})
			if test.confirmed {
				if !applied || failure != nil || err != nil || disposition != test.wantState {
					t.Fatal("confirmed receipt did not recover commit", applied, failure, disposition, err)
				}
			} else if applied || failure != nil || !errors.Is(err, transaction.ErrCommitOutcomeUnknown) || disposition != "" {
				t.Fatal("unknown commit was guessed", applied, failure, disposition, err)
			}
			if gate.rejectCalls != 0 || gate.receiptCalls != 1 {
				t.Fatal("commit recovery used the rejection path", gate.rejectCalls, gate.receiptCalls)
			}
		})
	}
}

func TestProviderOutcomeCommitUnknownUsesReceipt(t *testing.T) {
	gate := &gateFake{
		failure: func(context.Context, household.Principal, jobs.Job, string, jobs.State, jobs.Reason, time.Duration, func(context.Context) error) (bool, error) {
			return false, transaction.ErrCommitOutcomeUnknown
		},
		receipt: func(_ context.Context, _ household.Principal, _ jobs.Job, _ string, kind admission.ResultKind) (bool, error) {
			return kind == admission.ProviderOutcomeResult, nil
		},
	}
	var disposition ingestion.EvidenceDispositionState
	evidence := evidenceFake{
		save: func(_ context.Context, batch ingestion.EvidenceBatch) error { return batch.Validate() },
		disposition: func(_ context.Context, value ingestion.EvidenceDisposition) error {
			disposition = value.State
			return nil
		},
	}
	service, _ := application.NewService(gate, evidence, accountFake{}, sourceFake{}, now, func() string { return "server-id" })
	job := issued()
	token, _ := application.TokenFromJob(job)
	failure := ingestion.ProviderFailure{Token: token, Kind: ingestion.MFARequired, Evidence: []ingestion.Evidence{rawEvidence()}}
	applied, got, err := service.Ingest(context.Background(), principal(), job, &gatewayFake{result: ingestion.Result{Failure: &failure}})
	if !applied || got == nil || err != nil || disposition != ingestion.EvidenceProviderOutcome || gate.rejectCalls != 0 {
		t.Fatal("provider outcome receipt did not recover commit", applied, got, disposition, err)
	}
}

func TestProviderFailureMapsToRecoverableJobOutcome(t *testing.T) {
	tests := []struct {
		name       string
		failure    ingestion.ProviderFailure
		wantState  jobs.State
		wantReason jobs.Reason
		wantDelay  time.Duration
	}{
		{name: "reauthentication", failure: ingestion.ProviderFailure{Kind: ingestion.ReauthenticationRequired}, wantState: jobs.Waiting, wantReason: jobs.ReauthRequired},
		{name: "mfa", failure: ingestion.ProviderFailure{Kind: ingestion.MFARequired}, wantState: jobs.Waiting, wantReason: jobs.ReauthRequired},
		{name: "captcha", failure: ingestion.ProviderFailure{Kind: ingestion.CaptchaRequired}, wantState: jobs.Waiting, wantReason: jobs.ReauthRequired},
		{name: "rate limit", failure: ingestion.ProviderFailure{Kind: ingestion.RateLimited, Retryable: true, RetryAfterSeconds: 3600}, wantState: jobs.Ready, wantReason: jobs.TemporaryFailure, wantDelay: time.Hour},
		{name: "temporary default", failure: ingestion.ProviderFailure{Kind: ingestion.TemporaryFailure, Retryable: true}, wantState: jobs.Ready, wantReason: jobs.TemporaryFailure, wantDelay: 5 * time.Second},
		{name: "temporary requested", failure: ingestion.ProviderFailure{Kind: ingestion.TemporaryFailure, Retryable: true, RetryAfterSeconds: 86400}, wantState: jobs.Ready, wantReason: jobs.TemporaryFailure, wantDelay: 24 * time.Hour},
		{name: "permanent", failure: ingestion.ProviderFailure{Kind: ingestion.PermanentFailure}, wantState: jobs.Failed, wantReason: jobs.PermanentFailure},
		{name: "unsupported", failure: ingestion.ProviderFailure{Kind: ingestion.UnsupportedCapability}, wantState: jobs.Failed, wantReason: jobs.PermanentFailure},
		{name: "contract", failure: ingestion.ProviderFailure{Kind: ingestion.ContractViolation}, wantState: jobs.Failed, wantReason: jobs.PermanentFailure},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gate := &gateFake{failure: func(_ context.Context, _ household.Principal, _ jobs.Job, _ string, _ jobs.State, _ jobs.Reason, _ time.Duration, apply func(context.Context) error) (bool, error) {
				return true, apply(context.Background())
			}}
			service, _ := application.NewService(gate, evidenceFake{save: func(context.Context, ingestion.EvidenceBatch) error { return nil }}, accountFake{}, sourceFake{}, now, func() string { return "server-id" })
			job := issued()
			token, _ := application.TokenFromJob(job)
			test.failure.Token = token
			test.failure.Evidence = []ingestion.Evidence{rawEvidence()}
			applied, got, err := service.Ingest(context.Background(), principal(), job, &gatewayFake{result: ingestion.Result{Failure: &test.failure}})
			if err != nil || !applied || got == nil || got.Kind != test.failure.Kind || gate.commitCalls != 0 || gate.failureCalls != 1 || gate.outcomeState != test.wantState || gate.outcomeReason != test.wantReason || gate.outcomeDelay != test.wantDelay {
				t.Fatal("typed provider failure mapped incorrectly", applied, got, gate.outcomeState, gate.outcomeReason, gate.outcomeDelay, err)
			}
		})
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

func TestGatewayBindingMatchesTheAdmittedBuildBeforeIO(t *testing.T) {
	job := issued()
	other := job.Binding
	other.AdapterBuildDigest = "sha256:" + strings.Repeat("c", 64)
	gateway := &gatewayFake{bindingValue: other}
	service, _ := application.NewService(&gateFake{}, evidenceFake{save: func(context.Context, ingestion.EvidenceBatch) error { return nil }}, accountFake{}, sourceFake{}, now, func() string { return "server-id" })

	applied, failure, err := service.Ingest(context.Background(), principal(), job, gateway)
	if applied || failure != nil || !errors.Is(err, connections.ErrProviderNotAdmitted) || gateway.calls != 0 {
		t.Fatal("misbound gateway reached provider IO", applied, failure, err, gateway.calls)
	}
}

func TestManifestCapabilitiesFencePageBeforeEvidence(t *testing.T) {
	token, _ := application.TokenFromJob(issued())
	coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
	page := ingestion.Page{
		Token: token, Complete: true, Coverage: coverage, Evidence: []ingestion.Evidence{rawEvidence()},
		Records: []ingestion.Record{{
			Kind: ingestion.AccountRecordKind,
			Account: &ingestion.AccountRecord{
				Reference:    ingestion.AccountReference{ExternalAccountID: "external", Product: "current", AssetCode: "RUB"},
				LogNamespace: "undeclared",
				EvidenceID:   "raw",
			},
			CanonicalPayload: []byte(`{"recordType":"account"}`),
		}},
	}
	stored := false
	service, _ := application.NewService(&gateFake{}, evidenceFake{save: func(context.Context, ingestion.EvidenceBatch) error { stored = true; return nil }}, accountFake{}, sourceFake{}, now, func() string { return "server-id" })
	applied, failure, err := service.Ingest(context.Background(), principal(), issued(), &gatewayFake{result: ingestion.Result{Page: &page}})
	if applied || failure != nil || !errors.Is(err, ingestion.ErrInvalidContract) || stored {
		t.Fatal("out-of-manifest record crossed the evidence boundary", applied, failure, stored, err)
	}
}

type gateFake struct {
	commitCalls, failureCalls int
	rejectCalls, receiptCalls int
	outcomeState              jobs.State
	outcomeReason             jobs.Reason
	outcomeDelay              time.Duration
	beforeRead                func(context.Context, household.Principal, jobs.Job) error
	commit                    func(context.Context, household.Principal, jobs.Job, admission.Page, func(context.Context) error) (bool, error)
	failure                   func(context.Context, household.Principal, jobs.Job, string, jobs.State, jobs.Reason, time.Duration, func(context.Context) error) (bool, error)
	receipt                   func(context.Context, household.Principal, jobs.Job, string, admission.ResultKind) (bool, error)
	evidenceResult            func(context.Context, household.Principal, string, string) (admission.ResultKind, bool, error)
	reject                    func(context.Context, household.Principal, jobs.Job, string) error
}

func (g *gateFake) EvidenceResult(ctx context.Context, p household.Principal, jobID, evidence string) (admission.ResultKind, bool, error) {
	if g.evidenceResult == nil {
		return "", false, nil
	}
	return g.evidenceResult(ctx, p, jobID, evidence)
}

func (g *gateFake) ResultReceipt(ctx context.Context, p household.Principal, job jobs.Job, evidence string, kind admission.ResultKind) (bool, error) {
	g.receiptCalls++
	if g.receipt == nil {
		return false, nil
	}
	return g.receipt(ctx, p, job, evidence, kind)
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
func (g *gateFake) CommitProviderOutcome(ctx context.Context, p household.Principal, job jobs.Job, evidence string, state jobs.State, reason jobs.Reason, delay time.Duration, apply func(context.Context) error) (bool, error) {
	g.failureCalls++
	g.outcomeState, g.outcomeReason, g.outcomeDelay = state, reason, delay
	if g.failure == nil {
		return false, errors.New("unexpected failure commit")
	}
	return g.failure(ctx, p, job, evidence, state, reason, delay, apply)
}
func (g *gateFake) RetainRejectedResult(ctx context.Context, p household.Principal, job jobs.Job, evidence string) error {
	g.rejectCalls++
	if g.reject == nil {
		return nil
	}
	return g.reject(ctx, p, job, evidence)
}

type evidenceFake struct {
	save        func(context.Context, ingestion.EvidenceBatch) error
	disposition func(context.Context, ingestion.EvidenceDisposition) error
	staged      func(context.Context, string, int) ([]ingestion.StagedEvidence, error)
}

func (e evidenceFake) Save(ctx context.Context, batch ingestion.EvidenceBatch) error {
	return e.save(ctx, batch)
}

func (e evidenceFake) SetDisposition(ctx context.Context, disposition ingestion.EvidenceDisposition) error {
	if e.disposition == nil {
		return nil
	}
	return e.disposition(ctx, disposition)
}

func (e evidenceFake) Staged(ctx context.Context, householdID string, limit int) ([]ingestion.StagedEvidence, error) {
	if e.staged == nil {
		return nil, nil
	}
	return e.staged(ctx, householdID, limit)
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
	result       ingestion.Result
	manifest     *ingestion.Manifest
	bindingValue connections.Binding
	calls        int
}

func (g *gatewayFake) Binding() connections.Binding {
	if g.bindingValue.Provider != "" {
		return g.bindingValue
	}
	return binding()
}

func (g *gatewayFake) Manifest(context.Context) (ingestion.Manifest, error) {
	g.calls++
	if g.manifest != nil {
		return *g.manifest, nil
	}
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
