//go:build integration

package ingestion_test

import (
	"context"
	"errors"
	"os"
	"testing"

	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

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
