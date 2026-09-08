//go:build integration

package storage_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (f *fixture) importSource(service *admission.Service, connection string, input ledger.SourceInput) (ledger.SourceOutcome, error) {
	f.t.Helper()
	issued := f.issued(service, connection, binding())
	input.ConnectionID = connection
	input.JobID = issued.ID
	input.FetchedAt = f.now
	sources := journal.NewSources(f.store, f.writer, nil)
	var outcome ledger.SourceOutcome
	applied, err := service.CommitPage(testContext, f.p, issued, admission.Page{EvidenceRef: input.EvidenceRef, Coverage: "complete", Complete: true}, func(ctx context.Context) error {
		var err error
		outcome, err = sources.Apply(ctx, f.p, input)
		return err
	})
	if err == nil && !applied {
		f.t.Error("valid import quarantined")
	}
	return outcome, err
}
func TestSourceIdentityReconnectCorrectionAndHumanOverride(t *testing.T) {
	f := newFixture(t)
	service := f.admit(binding())
	connection := f.connection()
	account := f.account(money.RUB, "1000")
	r := f.revision(uuid.NewString(), account, "-100", money.RUB, 1)
	input := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: "raiffeisen", ExternalAccountID: "stable-account-one", Product: "current", Log: "camt-detail", RecordID: "immutable-reference"}, PayloadHash: strings.Repeat("1", 64), EvidenceRef: "synthetic:first", Classification: "new", Operation: &r}
	result, err := f.importSource(service, connection, input)
	if err != nil {
		t.Fatal(err)
	}
	firstID := result.Record.ID
	if f.count("postings") != 1 || f.available(account) != "900" {
		t.Fatal("source effect missing")
	}
	reconnected := f.connection()
	result, err = f.importSource(service, reconnected, input)
	if err != nil || !result.Duplicate || result.Record.ID != firstID {
		t.Fatalf("reconnect duplicate: %v", err)
	}
	if f.count("source_records") != 1 || f.count("source_revisions") != 1 || f.count("postings") != 1 || f.count("source_provenance") != 2 {
		t.Fatal("reconnect identity lost")
	}
	correction := r
	correction.Revision = 2
	correction.Postings[0].Money = cash("-120", money.RUB)
	input.Operation = &correction
	input.Classification = "correction"
	input.ExpectedRevision = 1
	input.PayloadHash = strings.Repeat("2", 64)
	input.EvidenceRef = "synthetic:correction"
	result, err = f.importSource(service, reconnected, input)
	if err != nil || result.Record.Revision != 2 || f.available(account) != "880" {
		t.Fatalf("source correction: %v", err)
	}
	human := f.revision(r.OperationID, account, "-110", money.RUB, 3)
	human.HumanOverride = true
	if _, err = f.write(human, request()); err != nil {
		t.Fatal(err)
	}
	correction = f.revision(r.OperationID, account, "-130", money.RUB, 4)
	input.Operation = &correction
	input.ExpectedRevision = 2
	input.PayloadHash = strings.Repeat("3", 64)
	input.EvidenceRef = "synthetic:later-provider-change"
	result, err = f.importSource(service, reconnected, input)
	if err != nil || !result.PreservedOverride || result.Record.Revision != 3 || f.available(account) != "890" {
		t.Fatalf("override overwritten: %v", err)
	}
	if f.count("source_revisions") != 3 || f.count("operation_revisions") != 3 {
		t.Fatal("source or human evidence lost")
	}
	// The same record ID in another account is an independent provider fact.
	second := input
	second.Key.ExternalAccountID = "stable-account-two"
	second.Classification = "new"
	second.ExpectedRevision = 0
	next := f.revision(uuid.NewString(), account, "-10", money.RUB, 1)
	second.Operation = &next
	result, err = f.importSource(service, reconnected, second)
	if err != nil || result.Record.ID == firstID || f.count("source_records") != 2 {
		t.Fatal("external accounts merged")
	}
}
func TestSourceCollisionKeepsEvidenceWithoutNewPosting(t *testing.T) {
	f := newFixture(t)
	service := f.admit(binding())
	connection := f.connection()
	account := f.account(money.USD, "100")
	r := f.revision(uuid.NewString(), account, "-10", money.USD, 1)
	input := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: "raiffeisen", ExternalAccountID: "stable-usd", Product: "current", Log: "statement", RecordID: "entry-1"}, PayloadHash: strings.Repeat("a", 64), EvidenceRef: "synthetic:initial", Classification: "new", Operation: &r}
	first, err := f.importSource(service, connection, input)
	if err != nil {
		t.Fatal(err)
	}
	input.PayloadHash = strings.Repeat("b", 64)
	input.EvidenceRef = "synthetic:conflicting-fact"
	other := f.revision(uuid.NewString(), account, "-20", money.USD, 1)
	input.Operation = &other
	result, err := f.importSource(service, connection, input)
	if err != nil || !result.Record.Ambiguous || f.count("source_revisions") != 2 || f.count("postings") != 1 {
		t.Fatalf("collision posted: %v", err)
	}
	// Simulate an index digest collision; full identity comparison must reject merging it.
	collision := input
	collision.Key.RecordID = "different-entry"
	collision.EvidenceRef = "synthetic:digest-collision"
	digest := collision.Key.Digest()
	if _, err = f.admin.Exec(testContext, "UPDATE want_keep.source_records SET identity_digest=$1 WHERE id=$2", digest[:], first.Record.ID); err != nil {
		t.Fatal(err)
	}
	result, err = f.importSource(service, connection, collision)
	if err != nil || !result.Record.Ambiguous || f.count("quarantine") != 1 || f.count("postings") != 1 {
		t.Fatalf("digest used as identity: %v", err)
	}
}
func TestFailedPageDoesNotAdvanceCheckpoint(t *testing.T) {
	f := newFixture(t)
	service := f.admit(binding())
	connection := f.connection()
	issued := f.issued(service, connection, binding())
	input := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: "raiffeisen", ExternalAccountID: "stable", Product: "current", Log: "statement", RecordID: "entry"}, PayloadHash: strings.Repeat("a", 64), EvidenceRef: "synthetic:page", Classification: "new", ConnectionID: connection, JobID: issued.ID, FetchedAt: f.now}
	sources := journal.NewSources(f.store, f.writer, nil)
	applied, err := service.CommitPage(testContext, f.p, issued, admission.Page{EvidenceRef: input.EvidenceRef, NextCursor: "p2", Coverage: "partial", Gaps: []string{"more_pages"}}, func(ctx context.Context) error {
		if _, e := sources.Apply(ctx, f.p, input); e != nil {
			return e
		}
		return ledger.ErrInvalidSource
	})
	if err == nil || applied || f.count("source_records") != 0 {
		t.Fatal("failed page partially committed")
	}
	job, err := f.store.Job(testContext, f.p, issued.ID)
	if err != nil || job.Cursor != "" {
		t.Fatal("checkpoint skipped failed page")
	}
	if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { _, e := sources.Apply(ctx, f.p, input); return e }); err == nil {
		t.Fatal("import bypassed commit fence")
	}
}

func TestConfirmedCorrectionResolvesAmbiguousSourceExactlyOnce(t *testing.T) {
	for _, changed := range []bool{false, true} {
		name := "same_hash"
		if changed {
			name = "changed_hash"
		}
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			service := f.admit(binding())
			connection := f.connection()
			account := f.account(money.USD, "100")
			operation := f.revision(uuid.NewString(), account, "-20", money.USD, 1)
			input := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: "raiffeisen", ExternalAccountID: "stable", Product: "current", Log: "statement", RecordID: "ambiguous"}, PayloadHash: strings.Repeat("a", 64), EvidenceRef: "synthetic:ambiguous", Classification: "ambiguous", Operation: &operation}
			first, err := f.importSource(service, connection, input)
			if err != nil || !first.Record.Ambiguous || f.count("postings") != 0 {
				t.Fatalf("initial ambiguity: %v", err)
			}
			input.Classification = "correction"
			input.ExpectedRevision = first.Record.Revision
			input.EvidenceRef = "synthetic:resolved"
			if changed {
				input.PayloadHash = strings.Repeat("b", 64)
			}
			resolved, err := f.importSource(service, connection, input)
			if err != nil || resolved.Record.Ambiguous || resolved.Duplicate || resolved.Record.ID != first.Record.ID || resolved.Record.Revision != 2 || f.available(account) != "80" {
				t.Fatalf("resolution failed: %+v %v", resolved, err)
			}
			repeated, err := f.importSource(service, connection, input)
			if err != nil || !repeated.Duplicate || repeated.Record.Ambiguous || f.count("postings") != 1 || f.count("source_revisions") != 2 || f.available(account) != "80" {
				t.Fatalf("resolution replay: %+v %v", repeated, err)
			}
		})
	}
}
