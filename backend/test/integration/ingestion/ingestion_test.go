//go:build integration

package ingestion_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
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
			var state, reason string
			var availableAt time.Time
			if err = f.admin.QueryRow(testContext, `SELECT state,reason,available_at FROM want_keep.jobs WHERE household_id=$1 AND id=$2`, f.family.ID, job.ID).Scan(&state, &reason, &availableAt); err != nil {
				t.Fatal(err)
			}
			if state != test.wantState || reason != test.wantReason {
				t.Fatal("provider outcome lost its lifecycle", state, reason)
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
	for range 2 {
		job := f.issued()
		applied, failure, err := f.service.Ingest(testContext, f.p, job, f.gateway(job))
		if err != nil || !applied || failure != nil {
			t.Fatal(applied, failure, err)
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
