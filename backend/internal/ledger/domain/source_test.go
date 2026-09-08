package domain

import (
	"strings"
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
)

func TestSourceResolutionRequiresCurrentRevisionAndPreservesOriginal(t *testing.T) {
	at, _ := calendar.ParseInstant("2026-09-07T12:00:00Z")
	key := SourceKey{HouseholdID: "household", Provider: "raiffeisen", ExternalAccountID: "stable", Product: "current", Log: "statement", RecordID: "entry"}
	original := SourceRecord{Key: key, Revision: 2, PayloadHash: strings.Repeat("a", 64), Ambiguous: true}
	input := SourceInput{Key: key, PayloadHash: original.PayloadHash, EvidenceRef: "synthetic:resolution", ConnectionID: "connection", JobID: "job", FetchedAt: at, Classification: "correction", ExpectedRevision: 1}
	stale, _, err := original.Next(input)
	if err != nil || !stale.Ambiguous {
		t.Fatal("stale correction resolved ambiguity")
	}
	input.ExpectedRevision = 2
	resolved, duplicate, err := original.Next(input)
	if err != nil || duplicate || resolved.Ambiguous || resolved.Revision != 3 {
		t.Fatal("confirmed correction failed")
	}
	if !original.Ambiguous || original.Revision != 2 {
		t.Fatal("source mutated")
	}
	again, duplicate, err := resolved.Next(input)
	if err != nil || !duplicate || again != resolved {
		t.Fatal("same correction repeated")
	}
}

func TestSourceKeyUsesUnicodeCharacterLimitsAndRejectsNUL(t *testing.T) {
	key := SourceKey{HouseholdID: "household", Provider: "raiffeisen", ExternalAccountID: strings.Repeat("ё", 1500), Product: "current", Log: "statement", RecordID: "entry"}
	if err := key.Validate(); err != nil {
		t.Fatal("valid multibyte identity was measured as bytes", err)
	}
	key.ExternalAccountID = strings.Repeat("ё", 2001)
	if key.Validate() == nil {
		t.Fatal("identity beyond the contract character limit was accepted")
	}
	key.ExternalAccountID = "external\x00account"
	if key.Validate() == nil {
		t.Fatal("NUL identity was accepted")
	}
}
