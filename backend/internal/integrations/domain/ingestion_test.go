package domain_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func TestAmountDistinguishesKnownUnknownAndUnavailable(t *testing.T) {
	known := ingestion.Amount{State: reporting.Known, AssetCode: "USDC", Value: "0.123456789123456789"}
	converted, err := known.Reporting(money.USDC)
	value, present := converted.Value()
	if err != nil || !present || value.Amount() != known.Value {
		t.Fatal("known amount lost precision", converted, err)
	}
	for _, state := range []reporting.Knowledge{reporting.Unknown, reporting.Unavailable} {
		missing, err := (ingestion.Amount{State: state, AssetCode: "USDC", Reason: "provider_field_missing"}).Reporting(money.USDC)
		if _, present = missing.Value(); err != nil || present || missing.Knowledge() != state {
			t.Fatal("missing amount became a value", state, missing, err)
		}
	}
	if _, err = known.Reporting(money.USDT); !errors.Is(err, money.ErrAssetMismatch) {
		t.Fatal("asset substitution was accepted", err)
	}
}

func TestAmountPreservesExternalCodeWhileUsingCanonicalAsset(t *testing.T) {
	amount, err := (ingestion.Amount{State: reporting.Known, AssetCode: "RUR", Value: "5000"}).ReportingAs("RUR", money.RUB)
	value, known := amount.Value()
	if err != nil || !known || value.Asset() != money.RUB || value.Amount() != "5000" {
		t.Fatal("source amount did not keep its raw code and canonical value separate", amount, err)
	}
	if _, err = (ingestion.Amount{State: reporting.Known, AssetCode: "USD", Value: "5000"}).ReportingAs("RUR", money.RUB); !errors.Is(err, money.ErrAssetMismatch) {
		t.Fatal("mismatched raw asset code was accepted", err)
	}
}

func TestCanonicalHashSeparatesBoundariesAndContractVersion(t *testing.T) {
	left, err := ingestion.CanonicalHash([]byte(`{"record":"ab"}`), strings.Repeat("c", 64))
	if err != nil {
		t.Fatal(err)
	}
	right, err := ingestion.CanonicalHash([]byte(`{"record":"a"}`), strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	if left == right {
		t.Fatal("different canonical records shared a payload hash")
	}
	if _, err = ingestion.CanonicalHash(nil, strings.Repeat("c", 64)); !errors.Is(err, ingestion.ErrInvalidContract) {
		t.Fatal("empty canonical record was accepted", err)
	}
}

func TestAccountReferenceKeyIsStructuralAndRejectsSeparators(t *testing.T) {
	valid := ingestion.AccountReference{ExternalAccountID: "external", Product: "wallet", Network: "ethereum", AssetCode: "USDT"}
	other := ingestion.AccountReference{ExternalAccountID: "external-ethereum", Product: "wallet", Network: "", AssetCode: "USDT"}
	if err := valid.Validate(); err != nil || valid.Key() == other.Key() {
		t.Fatal("account identity is not structural", err)
	}
	invalid := valid
	invalid.ExternalAccountID = "external\x00wallet"
	if !errors.Is(invalid.Validate(), ingestion.ErrInvalidContract) {
		t.Fatal("identity separator was accepted")
	}
}

func TestPageAndFailureEnforceCursorAndRetrySemantics(t *testing.T) {
	coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
	page := ingestion.Page{Token: token(), Complete: false, NextCursor: "page-2", Coverage: coverage, Evidence: []ingestion.Evidence{evidence()}}
	if err := page.Validate(); err != nil {
		t.Fatal(err)
	}
	page.NextCursor = ""
	if !errors.Is(page.Validate(), ingestion.ErrInvalidContract) {
		t.Fatal("incomplete page without cursor was accepted")
	}
	failure := ingestion.ProviderFailure{Token: token(), Kind: ingestion.RateLimited, Retryable: true, RetryAfterSeconds: 60, Evidence: []ingestion.Evidence{evidence()}}
	if err := failure.Validate(); err != nil {
		t.Fatal(err)
	}
	failure.Kind = ingestion.MFARequired
	if !errors.Is(failure.Validate(), ingestion.ErrInvalidContract) {
		t.Fatal("interactive failure was marked retryable")
	}
}

func TestProviderFailureEnforcesEvidenceIdentityAndAggregateSize(t *testing.T) {
	failure := ingestion.ProviderFailure{Token: token(), Kind: ingestion.MFARequired, Evidence: []ingestion.Evidence{evidence(), evidence()}}
	if !errors.Is(failure.Validate(), ingestion.ErrInvalidContract) {
		t.Fatal("duplicate failure evidence was accepted")
	}
	largeEvidence := func(id string, size int) ingestion.Evidence {
		data := make([]byte, size)
		digest := sha256.Sum256(data)
		return ingestion.Evidence{ID: id, MediaType: "application/json", Digest: hex.EncodeToString(digest[:]), Locator: "synthetic:" + id, Data: data}
	}
	failure.Evidence = []ingestion.Evidence{
		largeEvidence("first", ingestion.MaxEvidenceBytes/2+1),
		largeEvidence("second", ingestion.MaxEvidenceBytes/2),
	}
	if !errors.Is(failure.Validate(), ingestion.ErrInvalidContract) {
		t.Fatal("oversized aggregate failure evidence was accepted")
	}
}

func TestPageBoundsTotalPostings(t *testing.T) {
	coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
	reference := ingestion.AccountReference{ExternalAccountID: "external", Product: "current", AssetCode: "RUB"}
	postings := make([]ingestion.Posting, ingestion.MaxRecordsPerPage)
	for index := range postings {
		postings[index].Reference = reference
	}
	transaction := &ingestion.TransactionRecord{EvidenceID: "raw", Postings: postings}
	page := ingestion.Page{
		Token: token(), Complete: true, Coverage: coverage, Evidence: []ingestion.Evidence{evidence()},
		Records: []ingestion.Record{
			{Kind: ingestion.AccountRecordKind, Account: &ingestion.AccountRecord{Reference: reference, EvidenceID: "raw"}, CanonicalPayload: []byte("account")},
			{Kind: ingestion.TransactionRecordKind, Transaction: transaction, CanonicalPayload: []byte("first")},
			{Kind: ingestion.TransactionRecordKind, Transaction: &ingestion.TransactionRecord{EvidenceID: "raw", Postings: []ingestion.Posting{{Reference: reference}}}, CanonicalPayload: []byte("second")},
		},
	}
	if !errors.Is(page.Validate(), ingestion.ErrInvalidContract) {
		t.Fatal("page accepted more than the total posting limit")
	}
}

func TestEvidenceRequiresMatchingDigestAndBoundsTheBatch(t *testing.T) {
	invalid := evidence()
	invalid.Digest = strings.Repeat("0", 64)
	if !errors.Is(invalid.Validate(), ingestion.ErrInvalidContract) {
		t.Fatal("evidence with a mismatched digest was accepted")
	}
	invalid = evidence()
	invalid.Data = nil
	if !errors.Is(invalid.Validate(), ingestion.ErrInvalidContract) {
		t.Fatal("empty evidence was accepted")
	}

	data := make([]byte, 6*1024*1024)
	digest := sha256.Sum256(data)
	fetchedAt, _ := calendar.ParseInstant("2026-09-08T12:00:00Z")
	batch := ingestion.EvidenceBatch{
		HouseholdID:   "household",
		JobID:         "job",
		PageReference: "evidence:page:test",
		FetchedAt:     fetchedAt,
		Disposition:   ingestion.EvidenceStaged,
		Items: []ingestion.StoredEvidence{
			{Reference: "evidence:raw:one", Raw: ingestion.Evidence{ID: "one", MediaType: "application/json", Digest: hex.EncodeToString(digest[:]), Locator: "synthetic:one", Data: data}},
			{Reference: "evidence:raw:two", Raw: ingestion.Evidence{ID: "two", MediaType: "application/json", Digest: hex.EncodeToString(digest[:]), Locator: "synthetic:two", Data: data}},
		},
	}
	if !errors.Is(batch.Validate(), ingestion.ErrEvidence) {
		t.Fatal("evidence batch exceeded the aggregate byte limit")
	}
}

func token() ingestion.JobToken {
	return ingestion.JobToken{
		JobID: "job", Attempt: 1, LeaseToken: "lease", ConnectionID: "connection", ConnectionGeneration: 1,
		Binding: connections.Binding{
			Provider: "bybit", Environment: "test", AdapterBuildDigest: "sha256:" + strings.Repeat("a", 64),
			CollectorImageDigest: "sha256:" + strings.Repeat("b", 64), ContractVersion: "10",
			AllowlistRevision: "allowlist-1", NonSecretConfigRevision: "config-1", OperatorPermissionRevision: "permission-1",
		},
		AdmissionRevision: 1, ReplayFrom: time.Time{}, ReplayTo: time.Time{},
	}
}

func evidence() ingestion.Evidence {
	return ingestion.Evidence{ID: "raw", MediaType: "application/json", Digest: "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a", Locator: "synthetic:raw", Data: []byte("{}")}
}
