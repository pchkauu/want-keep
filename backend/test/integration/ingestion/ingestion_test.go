//go:build integration

package ingestion_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	application "github.com/pchkauu/want-keep/backend/internal/integrations/application"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func TestProviderFailuresPersistRecoverableJobOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		kind       ingestion.FailureKind
		retryable  bool
		retryAfter int
		wantState  string
		wantReason string
		wantDelay  bool
	}{
		{name: "reauthentication", kind: ingestion.MFARequired, wantState: "waiting", wantReason: "reauth_required"},
		{name: "rate limit", kind: ingestion.RateLimited, retryable: true, retryAfter: 60, wantState: "ready", wantReason: "temporary_failure", wantDelay: true},
		{name: "permanent", kind: ingestion.ContractViolation, wantState: "failed", wantReason: "permanent_failure"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			job := f.issued()
			if err := f.store.BeginExternal(testContext, f.p, job); err != nil {
				t.Fatal(err)
			}
			token, _ := application.TokenFromJob(job)
			gateway := f.gateway(job)
			gateway.result = ingestion.Result{Failure: &ingestion.ProviderFailure{
				Token: token, Kind: test.kind, Retryable: test.retryable, RetryAfterSeconds: test.retryAfter,
				Evidence: []ingestion.Evidence{gateway.result.Page.Evidence[0]},
			}}
			applied, failure, err := f.service.Ingest(testContext, f.p, job, gateway)
			if err != nil || !applied || failure == nil || failure.Kind != test.kind {
				t.Fatal("provider failure was not committed", applied, failure, err)
			}
			confirmed, receiptErr := f.gate.ResultReceipt(testContext, f.p, job, f.evidence.Last().PageReference, admission.ProviderOutcomeResult)
			if receiptErr != nil || !confirmed {
				t.Fatal("provider outcome receipt was not readable", confirmed, receiptErr)
			}
			var state, reason string
			var externalStarted bool
			var availableAt time.Time
			if err = f.admin.QueryRow(testContext, `SELECT state,reason,external_started,available_at FROM want_keep.jobs WHERE household_id=$1 AND id=$2`, f.family.ID, job.ID).Scan(&state, &reason, &externalStarted, &availableAt); err != nil {
				t.Fatal(err)
			}
			if state != test.wantState || reason != test.wantReason || externalStarted {
				t.Fatal("provider outcome lost its lifecycle", state, reason, externalStarted)
			}
			if test.wantDelay && time.Until(availableAt) < 50*time.Second {
				t.Fatal("provider retry delay was discarded", availableAt)
			}
			var evidenceRef, evidenceReason string
			if err = f.admin.QueryRow(testContext, `SELECT evidence_ref,reason FROM want_keep.quarantine WHERE household_id=$1 AND job_id=$2`, f.family.ID, job.ID).Scan(&evidenceRef, &evidenceReason); err != nil || evidenceRef == "" || evidenceReason != "provider_outcome" {
				t.Fatal("provider outcome evidence was not associated with the job", evidenceRef, evidenceReason, err)
			}
		})
	}
}

func TestProviderOutcomeWithoutExternalMarkerIsStale(t *testing.T) {
	f := newFixture(t)
	job := f.issued()
	token, _ := application.TokenFromJob(job)
	gateway := f.gateway(job)
	gateway.result = ingestion.Result{Failure: &ingestion.ProviderFailure{
		Token: token, Kind: ingestion.MFARequired,
		Evidence: []ingestion.Evidence{gateway.result.Page.Evidence[0]},
	}}

	applied, failure, err := f.service.Ingest(testContext, f.p, job, gateway)
	if err != nil || applied || failure == nil {
		t.Fatal("provider outcome without external marker was accepted", applied, failure, err)
	}
	providerOutcome, receiptErr := f.gate.ResultReceipt(testContext, f.p, job, f.evidence.Last().PageReference, admission.ProviderOutcomeResult)
	if receiptErr != nil || providerOutcome {
		t.Fatal("provider outcome receipt bypassed the external marker", providerOutcome, receiptErr)
	}
	stale, receiptErr := f.gate.ResultReceipt(testContext, f.p, job, f.evidence.Last().PageReference, admission.StaleResult)
	if receiptErr != nil || !stale {
		t.Fatal("unmarked provider outcome was not retained as stale", stale, receiptErr)
	}
	var state string
	if err = f.admin.QueryRow(testContext, `SELECT state FROM want_keep.jobs WHERE household_id=$1 AND id=$2`, f.family.ID, job.ID).Scan(&state); err != nil || state != "running" {
		t.Fatal("unmarked provider outcome changed the job", state, err)
	}
}

func TestDelayedProviderFailureCannotOverrideAdvancedCursor(t *testing.T) {
	f := newFixture(t)
	job := f.issued()
	token, _ := application.TokenFromJob(job)
	delayed := f.gateway(job)
	delayed.result = ingestion.Result{Failure: &ingestion.ProviderFailure{
		Token: token, Kind: ingestion.TemporaryFailure, Retryable: true,
		Evidence: []ingestion.Evidence{delayed.result.Page.Evidence[0]},
	}}
	started, release := make(chan struct{}), make(chan struct{})
	delayed.read = func() {
		close(started)
		<-release
	}
	type outcome struct {
		applied bool
		failure *ingestion.ProviderFailure
		err     error
	}
	result := make(chan outcome, 1)
	go func() {
		applied, failure, err := f.service.Ingest(context.Background(), f.p, job, delayed)
		result <- outcome{applied: applied, failure: failure, err: err}
	}()
	<-started
	page := f.gatewayWithMutation(job, func(root map[string]any) {
		payload := root["page"].(map[string]any)
		payload["complete"] = false
		payload["nextCursor"] = "page-2"
	})
	if applied, _, err := f.service.Ingest(testContext, f.p, job, page); err != nil || !applied {
		t.Fatal("page did not advance the cursor", applied, err)
	}
	close(release)
	delayedResult := <-result
	if delayedResult.applied || delayedResult.failure == nil || delayedResult.err != nil {
		t.Fatal("delayed failure changed the current job", delayedResult)
	}
	var cursor, state string
	if err := f.admin.QueryRow(testContext, `SELECT cursor,state FROM want_keep.jobs WHERE household_id=$1 AND id=$2`, f.family.ID, job.ID).Scan(&cursor, &state); err != nil || cursor != "page-2" || state != "running" {
		t.Fatal("delayed failure overwrote job progress", cursor, state, err)
	}
	var count int
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.quarantine WHERE household_id=$1 AND job_id=$2 AND reason='stale_result'`, f.family.ID, job.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("delayed failure evidence was not quarantined", count, err)
	}
}

func TestRaiffeisenRURUsesCanonicalRUBWithoutLosingRawCode(t *testing.T) {
	f := newFixtureForProvider(t, "raiffeisen")
	job := f.issued()
	gateway := f.gatewayWithMutation(job, func(root map[string]any) {
		for _, item := range root["page"].(map[string]any)["records"].([]any) {
			record := item.(map[string]any)
			for _, name := range []string{"account", "balanceSnapshot"} {
				value, ok := record[name].(map[string]any)
				if !ok || value["externalAccountId"] != "acct-rub" {
					continue
				}
				value["assetCode"] = "RUR"
				if name == "balanceSnapshot" {
					for _, field := range []string{"owned", "available", "locked", "debt", "creditLimit"} {
						value[field].(map[string]any)["assetCode"] = "RUR"
					}
				}
			}
			transaction, ok := record["transaction"].(map[string]any)
			if !ok {
				continue
			}
			for _, raw := range transaction["postings"].([]any) {
				posting := raw.(map[string]any)
				if posting["externalAccountId"] == "acct-rub" {
					posting["assetCode"] = "RUR"
				}
			}
		}
	})
	applied, failure, err := f.service.Ingest(testContext, f.p, job, gateway)
	if err != nil || !applied || failure != nil {
		t.Fatal("confirmed RUR mapping was not applied", applied, failure, err)
	}
	var asset, externalCode string
	if err = f.admin.QueryRow(testContext, `SELECT asset,external_asset_code FROM want_keep.accounts WHERE household_id=$1 AND name='Synthetic RUB'`, f.family.ID).Scan(&asset, &externalCode); err != nil || asset != "RUB" || externalCode != "RUR" {
		t.Fatal("canonical and external assets were not kept separately", asset, externalCode, err)
	}
	var amountAsset string
	if err = f.admin.QueryRow(testContext, `SELECT asset FROM want_keep.observation_amounts WHERE household_id=$1 AND amount=5000`, f.family.ID).Scan(&amountAsset); err != nil || amountAsset != "RUB" {
		t.Fatal("RUR balance did not become canonical RUB", amountAsset, err)
	}
}

func TestAmbiguousAccountCommitsTheRestOfThePageAsPartial(t *testing.T) {
	f := newFixture(t)
	partner := uuid.NewString()
	membership := uuid.NewString()
	external := uuid.NewString()
	digest := sha256.Sum256([]byte("acct-rub"))
	tx, err := f.admin.Begin(testContext)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(testContext)
	if _, err = tx.Exec(testContext, `INSERT INTO want_keep.users(id,name) VALUES($1,'Synthetic partner')`, partner); err == nil {
		_, err = tx.Exec(testContext, `INSERT INTO want_keep.memberships(household_id,id,user_id,active) VALUES($1,$2,$3,true)`, f.family.ID, membership, partner)
	}
	if err == nil {
		_, err = tx.Exec(testContext, `INSERT INTO want_keep.external_accounts(household_id,id,provider,stable_id,identity_digest,external_owner_id) VALUES($1,$2,'bybit','acct-rub',$3,$4)`, f.family.ID, external, digest[:], partner)
	}
	if err == nil {
		err = tx.Commit(testContext)
	}
	if err != nil {
		t.Fatal("prepare ambiguous external account", err)
	}
	job := f.issued()
	applied, failure, err := f.service.Ingest(testContext, f.p, job, f.gateway(job))
	if err != nil || !applied || failure != nil {
		t.Fatal("mixed ambiguous page did not commit", applied, failure, err)
	}
	accounts, sources, postings := f.count("accounts"), f.count("source_records"), f.count("postings")
	if accounts != 5 || sources != 2 || postings != 4 {
		t.Fatalf("valid records were lost with the ambiguous account: accounts=%d sources=%d postings=%d", accounts, sources, postings)
	}
	var gaps []string
	if err = f.admin.QueryRow(testContext, `SELECT gaps FROM want_keep.jobs WHERE household_id=$1 AND id=$2`, f.family.ID, job.ID).Scan(&gaps); err != nil || !contains(gaps, "source_ambiguous") || !contains(gaps, "transaction_unresolved") {
		t.Fatal("ambiguity omissions were not retained", gaps, err)
	}
}

func TestDeclaredAmbiguityCannotCompleteTheCheckpoint(t *testing.T) {
	f := newFixture(t)
	job := f.issued()
	gateway := f.gatewayWithMutation(job, func(root map[string]any) {
		records := root["page"].(map[string]any)["records"].([]any)
		for _, item := range records {
			transaction, ok := item.(map[string]any)["transaction"].(map[string]any)
			if ok {
				transaction["classification"] = "ambiguous"
				return
			}
		}
	})
	applied, failure, err := f.service.Ingest(testContext, f.p, job, gateway)
	if err != nil || !applied || failure != nil {
		t.Fatal("declared ambiguity did not commit safely", applied, failure, err)
	}
	var coverage string
	var gaps []string
	if err = f.admin.QueryRow(testContext, `SELECT coverage,gaps FROM want_keep.jobs WHERE household_id=$1 AND id=$2`, f.family.ID, job.ID).Scan(&coverage, &gaps); err != nil || coverage != "partial" || !contains(gaps, "source_ambiguous") {
		t.Fatal("declared ambiguity escaped checkpoint coverage", coverage, gaps, err)
	}
}

func TestSamePageSourceIdentityIsPreflightedBeforeFinancialApply(t *testing.T) {
	for _, test := range []struct {
		name         string
		changeAmount bool
		wantPostings int
		wantPartial  bool
	}{
		{name: "identical duplicate", wantPostings: 5},
		{name: "conflicting fact", changeAmount: true, wantPostings: 4, wantPartial: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			job := f.issued()
			gateway := f.gatewayWithMutation(job, func(root map[string]any) {
				page := root["page"].(map[string]any)
				records := page["records"].([]any)
				for _, raw := range records {
					record := raw.(map[string]any)
					transaction, ok := record["transaction"].(map[string]any)
					if !ok || transaction["providerRecordId"] != "expense-1" {
						continue
					}
					encoded, _ := json.Marshal(record)
					var duplicate map[string]any
					_ = json.Unmarshal(encoded, &duplicate)
					if test.changeAmount {
						duplicate["transaction"].(map[string]any)["postings"].([]any)[0].(map[string]any)["money"] = "-700"
					}
					page["records"] = append(records, duplicate)
					return
				}
			})
			applied, failure, err := f.service.Ingest(testContext, f.p, job, gateway)
			if err != nil || !applied || failure != nil {
				t.Fatal("same-page source group did not commit safely", applied, failure, err)
			}
			if got := f.count("postings"); got != test.wantPostings {
				t.Fatalf("unexpected postings after source preflight: got=%d want=%d", got, test.wantPostings)
			}
			var coverage string
			var gaps []string
			if err = f.admin.QueryRow(testContext, `SELECT coverage,gaps FROM want_keep.jobs WHERE household_id=$1 AND id=$2`, f.family.ID, job.ID).Scan(&coverage, &gaps); err != nil {
				t.Fatal(err)
			}
			if test.wantPartial != (coverage == "partial" && contains(gaps, "source_ambiguous")) {
				t.Fatal("same-page ambiguity coverage mismatch", coverage, gaps)
			}
		})
	}
}

func TestConflictingSamePageSourceReplayDoesNotAdvanceHistory(t *testing.T) {
	f := newFixture(t)
	for range 2 {
		job := f.issued()
		gateway := f.gatewayWithMutation(job, func(root map[string]any) {
			page := root["page"].(map[string]any)
			records := page["records"].([]any)
			for _, raw := range records {
				record := raw.(map[string]any)
				transaction, ok := record["transaction"].(map[string]any)
				if !ok || transaction["providerRecordId"] != "expense-1" {
					continue
				}
				encoded, _ := json.Marshal(record)
				var conflicting map[string]any
				_ = json.Unmarshal(encoded, &conflicting)
				conflicting["transaction"].(map[string]any)["postings"].([]any)[0].(map[string]any)["money"] = "-700"
				page["records"] = append(records, conflicting)
				return
			}
		})
		applied, failure, err := f.service.Ingest(testContext, f.p, job, gateway)
		if err != nil || !applied || failure != nil {
			t.Fatal("conflicting source group replay failed", applied, failure, err)
		}
	}
	if revisions := f.count("source_revisions"); revisions != 4 {
		t.Fatalf("conflicting replay changed source history: revisions=%d", revisions)
	}
	if postings := f.count("postings"); postings != 4 {
		t.Fatalf("conflicting replay changed financial effect: postings=%d", postings)
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestGoldenPageRoundTripsThroughPostgreSQLWithoutDuplicateEffects(t *testing.T) {
	f := newFixture(t)
	for attempt := range 2 {
		job := f.issued()
		gateway := f.gateway(job)
		if attempt == 1 {
			gateway = f.gatewayWithMutation(job, func(root map[string]any) {
				page := root["page"].(map[string]any)
				page["evidence"].([]any)[0].(map[string]any)["id"] = "replayed-evidence"
				page["evidence"].([]any)[0].(map[string]any)["locator"] = "synthetic:replayed"
				for _, raw := range page["records"].([]any) {
					record := raw.(map[string]any)
					for _, field := range []string{"account", "balanceSnapshot", "transaction"} {
						if payload, ok := record[field].(map[string]any); ok {
							payload["evidenceId"] = "replayed-evidence"
						}
					}
				}
			})
		}
		applied, failure, err := f.service.Ingest(testContext, f.p, job, gateway)
		if err != nil || !applied || failure != nil {
			t.Fatal(applied, failure, err)
		}
		confirmed, err := f.gate.ResultReceipt(testContext, f.p, job, f.evidence.Last().PageReference, admission.PageResult)
		if err != nil || !confirmed {
			t.Fatal("committed page receipt was not readable", confirmed, err)
		}
		wrongKind, err := f.gate.ResultReceipt(testContext, f.p, job, f.evidence.Last().PageReference, admission.ProviderOutcomeResult)
		if err != nil || wrongKind {
			t.Fatal("receipt matched a different outcome kind", wrongKind, err)
		}
	}
	if f.count("accounts") != 6 || f.count("account_observations") != 6 || f.count("card_aliases") != 1 {
		t.Fatal("account identity or snapshot replay duplicated records")
	}
	sources, sourceRevisions, postings := f.count("source_records"), f.count("source_revisions"), f.count("postings")
	if sources != 3 || sourceRevisions != 3 || postings != 5 {
		t.Fatalf("source replay changed the financial effect: sources=%d revisions=%d postings=%d", sources, sourceRevisions, postings)
	}
	rows, err := f.admin.Query(testContext, `SELECT asset,amount::text FROM want_keep.observation_amounts WHERE field='owned' ORDER BY asset`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := map[string]string{}
	for rows.Next() {
		var asset, amount string
		if err = rows.Scan(&asset, &amount); err != nil {
			t.Fatal(err)
		}
		got[asset] = amount
	}
	want := map[string]string{"RUB": "5000", "USD": "12.000000000123", "USDT": "10.000000000001", "USDC": "0.123456789123456789", "BTC": "0.000000000000000001", "ETH": "1.000000000000000123"}
	for asset, amount := range want {
		if got[asset] != amount {
			t.Fatalf("%s amount changed: %q", asset, got[asset])
		}
	}
	key := ledger.SourceKey{HouseholdID: f.family.ID, Provider: "bybit", ExternalAccountID: "acct-rub", Product: "current", Log: "payments", RecordID: "expense-1"}
	source, found, err := f.store.Source(testContext, f.p, key)
	if err != nil || !found || source.Revision != 1 {
		t.Fatal(source, found, err)
	}
	revision, found, err := f.store.CurrentLedgerRevision(testContext, f.p, source.OperationID)
	if err != nil || !found || revision.Postings[0].Money.Amount() != "-500" || revision.ActorID != f.p.UserID() {
		t.Fatal("trusted source mapping changed", revision, err)
	}
	if files, _ := os.ReadDir(f.evidence.path); len(files) != 2 {
		t.Fatalf("raw evidence was not retained for both deliveries: %d", len(files))
	}
}

func TestConfirmedCorrectionReplayDoesNotCreateAnotherSourceRevision(t *testing.T) {
	f := newFixture(t)
	for attempt := range 3 {
		job := f.issued()
		gateway := f.gatewayWithMutation(job, func(root map[string]any) {
			classification := "correction"
			if attempt == 0 {
				classification = "ambiguous"
			}
			page := root["page"].(map[string]any)
			for _, raw := range page["records"].([]any) {
				record := raw.(map[string]any)
				if transaction, ok := record["transaction"].(map[string]any); ok {
					transaction["classification"] = classification
				}
			}
		})
		applied, failure, err := f.service.Ingest(testContext, f.p, job, gateway)
		if err != nil || !applied || failure != nil {
			t.Fatal(applied, failure, err)
		}
	}
	if revisions := f.count("source_revisions"); revisions != 6 {
		t.Fatalf("replayed corrections changed source history: revisions=%d", revisions)
	}
}

func TestResultReceiptIsImmutableForApplicationRole(t *testing.T) {
	f := newFixture(t)
	job := f.issued()
	if applied, failure, err := f.service.Ingest(testContext, f.p, job, f.gateway(job)); err != nil || !applied || failure != nil {
		t.Fatal("page was not committed", applied, failure, err)
	}
	connection, err := f.admin.Acquire(testContext)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Release()
	if _, err = connection.Exec(testContext, "SET ROLE want_keep_app"); err != nil {
		t.Fatal(err)
	}
	defer connection.Exec(testContext, "RESET ROLE")
	if _, err = connection.Exec(testContext, `UPDATE want_keep.ingestion_result_receipts SET attempt=attempt+1 WHERE household_id=$1 AND job_id=$2`, f.family.ID, job.ID); err == nil {
		t.Fatal("application role changed immutable ingestion receipt")
	}
}

func TestStagedEvidenceFinalizesFromDurableReceipt(t *testing.T) {
	f := newFixture(t)
	f.evidence.failDisposition = true
	job := f.issued()
	applied, failure, err := f.service.Ingest(testContext, f.p, job, f.gateway(job))
	if !applied || failure != nil || !errors.Is(err, ingestion.ErrEvidence) || f.count("source_records") != 3 {
		t.Fatal("commit did not survive disposition failure", applied, failure, err)
	}
	f.evidence.failDisposition = false
	completed, err := f.service.ReconcileStaged(testContext, f.p, 100)
	if err != nil || completed != 1 || f.count("source_records") != 3 {
		t.Fatal("durable receipt did not finalize staged evidence", completed, err)
	}
	entries, err := os.ReadDir(f.evidence.path)
	if err != nil || len(entries) != 1 || !strings.Contains(entries[0].Name(), string(ingestion.EvidenceApplied)) {
		t.Fatal("evidence did not reach applied disposition", entries, err)
	}
}

func TestUnicodeContractLimitRoundTripsThroughPostgreSQL(t *testing.T) {
	f := newFixture(t)
	job := f.issued()
	externalID := strings.Repeat("ё", 1500)
	name := strings.Repeat("с", 1500)
	gateway := f.gatewayWithMutation(job, func(root map[string]any) {
		records := root["page"].(map[string]any)["records"].([]any)
		for _, item := range records {
			record := item.(map[string]any)
			for _, key := range []string{"account", "balanceSnapshot"} {
				value, ok := record[key].(map[string]any)
				if !ok || value["externalAccountId"] != "acct-usd" {
					continue
				}
				value["externalAccountId"] = externalID
				if key == "account" {
					value["name"] = name
				}
			}
		}
	})
	applied, failure, err := f.service.Ingest(testContext, f.p, job, gateway)
	if err != nil || !applied || failure != nil {
		t.Fatal("valid Unicode page did not commit", applied, failure, err)
	}
	var storedID, storedName string
	if err = f.admin.QueryRow(testContext, `SELECT e.stable_id,a.name FROM want_keep.external_accounts e JOIN want_keep.accounts a ON (a.household_id,a.external_account_id)=(e.household_id,e.id) WHERE e.household_id=$1 AND e.stable_id=$2`, f.family.ID, externalID).Scan(&storedID, &storedName); err != nil {
		t.Fatal(err)
	}
	if storedID != externalID || storedName != name {
		t.Fatal("Unicode values changed across PostgreSQL", len([]rune(storedID)), len([]rune(storedName)))
	}
}

func TestAccountDescriptorDoesNotInventBalanceObservation(t *testing.T) {
	f := newFixture(t)
	job := f.issued()
	gateway := f.gateway(job)
	for _, record := range gateway.result.Page.Records {
		if record.Account != nil && record.Account.Reference.AssetCode == "RUB" {
			gateway.result.Page.Records = []ingestion.Record{record}
			break
		}
	}
	if len(gateway.result.Page.Records) != 1 {
		t.Fatal("synthetic account descriptor was not found")
	}
	applied, failure, err := f.service.Ingest(testContext, f.p, job, gateway)
	if err != nil || !applied || failure != nil {
		t.Fatal(applied, failure, err)
	}
	if f.count("accounts") != 1 || f.count("account_openings") != 1 || f.count("account_observations") != 0 {
		t.Fatal("account-only page invented a source balance")
	}
}

func TestEvidenceFailureAndStaleAdmissionCannotCrossCommitFence(t *testing.T) {
	t.Run("evidence failure", func(t *testing.T) {
		f := newFixture(t)
		f.evidence.fail = true
		job := f.issued()
		applied, _, err := f.service.Ingest(testContext, f.p, job, f.gateway(job))
		if applied || !errors.Is(err, ingestion.ErrEvidence) || f.count("accounts") != 0 || f.count("sync_progress") != 0 {
			t.Fatal("financial commit survived evidence failure", applied, err)
		}
	})

	t.Run("rejected page", func(t *testing.T) {
		f := newFixture(t)
		job := f.issued()
		gateway := f.gatewayWithMutation(job, func(root map[string]any) {
			records := root["page"].(map[string]any)["records"].([]any)
			for _, item := range records {
				balance, ok := item.(map[string]any)["balanceSnapshot"].(map[string]any)
				if ok && balance["externalAccountId"] == "acct-rub" {
					balance["available"].(map[string]any)["amount"] = "-1"
					return
				}
			}
		})
		applied, _, err := f.service.Ingest(testContext, f.p, job, gateway)
		if applied || !errors.Is(err, ingestion.ErrInvalidContract) || f.count("accounts") != 0 || f.count("source_records") != 0 || f.count("sync_progress") != 0 {
			t.Fatal("invalid page left a partial financial effect", applied, err)
		}
		var evidenceRef string
		if queryErr := f.admin.QueryRow(testContext, `SELECT evidence_ref FROM want_keep.quarantine WHERE household_id=$1 AND job_id=$2 AND reason='rejected_result'`, f.family.ID, job.ID).Scan(&evidenceRef); queryErr != nil || evidenceRef == "" {
			t.Fatal("rejected evidence was not associated with its household and job", evidenceRef, queryErr)
		}
		entries, _ := os.ReadDir(f.evidence.path)
		if len(entries) != 1 || !strings.Contains(entries[0].Name(), string(f.family.ID)) || !strings.Contains(entries[0].Name(), job.ID) || !strings.Contains(entries[0].Name(), string(ingestion.EvidenceRejected)) {
			t.Fatal("raw evidence was not stored under trusted ownership", entries)
		}
		kind, found, receiptErr := f.gate.EvidenceResult(testContext, f.p, job.ID, evidenceRef)
		if receiptErr != nil || !found || kind != admission.RejectedResult {
			t.Fatal("rejected evidence finalization was not durable", kind, found, receiptErr)
		}
	})

	t.Run("stale admission", func(t *testing.T) {
		f := newFixture(t)
		job := f.issued()
		gateway := f.gateway(job)
		gateway.read = func() {
			changed := binding()
			changed.AllowlistRevision = "allowlist-2"
			if _, err := f.gate.Rebind(testContext, changed); err != nil {
				t.Fatal(err)
			}
		}
		applied, _, err := f.service.Ingest(testContext, f.p, job, gateway)
		if applied || err != nil || f.count("accounts") != 0 || f.count("source_records") != 0 || f.count("sync_progress") != 0 || f.count("quarantine") != 1 {
			t.Fatal("stale result crossed commit fence", applied, err)
		}
		if files, _ := os.ReadDir(f.evidence.path); len(files) != 1 {
			t.Fatal("stale raw evidence was not retained")
		}
		kind, found, receiptErr := f.gate.EvidenceResult(testContext, f.p, job.ID, f.evidence.Last().PageReference)
		if receiptErr != nil || !found || kind != admission.StaleResult {
			t.Fatal("stale evidence finalization was not durable", kind, found, receiptErr)
		}
	})
}

func TestConcurrentDeliveryAppliesOneFinancialEffect(t *testing.T) {
	f := newFixture(t)
	job := f.issued()
	gateway := f.gateway(job)
	start := make(chan struct{})
	type result struct {
		applied bool
		err     error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			<-start
			applied, _, err := f.service.Ingest(context.Background(), f.p, job, gateway)
			results <- result{applied: applied, err: err}
		}()
	}
	close(start)
	var successes int
	for range 2 {
		result := <-results
		if result.err == nil && result.applied {
			successes++
		}
	}
	if successes != 1 || f.count("source_records") != 3 || f.count("postings") != 5 {
		t.Fatal("concurrent delivery was not fenced", successes)
	}
}
