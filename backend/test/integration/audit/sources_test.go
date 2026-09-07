//go:build integration

package audit_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestSourceProgressionKeepsOnlyProtectedFieldsAndUndoUsesLatestFact(t *testing.T) {
	f := newFixture(t)
	gate := f.admit()
	connection := f.connection(f.p)
	account := f.create(money.RUB, "5000")
	client := f.client(f.q)
	r := f.revision(uuid.NewString(), account, "-500", money.RUB, 1)
	r.State = ledger.Pending
	r.Merchant = "Bank merchant"
	r.FeeKnowledge = ledger.KnownFees
	in := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: "raiffeisen", ExternalAccountID: "synthetic-business", Product: "current", Log: "statement", RecordID: "synthetic-purchase"}, PayloadHash: strings.Repeat("a", 64), EvidenceRef: "synthetic:pending", Classification: "new", Operation: &r}
	if _, err := f.importSource(gate, connection, in); err != nil {
		t.Fatal(err)
	}
	view := client.transaction(r.OperationID)
	view = client.correct(view, map[string]any{"merchant": "My merchant"})
	r.State = ledger.Posted
	r.PostedAt = f.now
	r.Revision = 2
	in.Classification = "correction"
	in.ExpectedRevision = 1
	in.PayloadHash = strings.Repeat("b", 64)
	if _, err := f.importSource(gate, connection, in); err != nil {
		t.Fatal(err)
	}
	view = client.transaction(r.OperationID)
	if view.State != "posted" || view.Revision != 3 || view.Merchant == nil || *view.Merchant != "My merchant" {
		t.Fatal("status stopped or protected merchant lost", view)
	}
	f.balance(account, "owned", "4500")
	f.balance(account, "locked", "0")
	principal := append([]generated.Posting(nil), view.Postings...)
	principal[0].Money.Amount = "-700"
	view = client.correct(view, map[string]any{"principal": principal})
	decision := *view.DecisionId
	r.Revision = 3
	r.Postings[0].Money = cash("-600", money.RUB)
	in.ExpectedRevision = 2
	in.PayloadHash = strings.Repeat("c", 64)
	if _, err := f.importSource(gate, connection, in); err != nil {
		t.Fatal(err)
	}
	view = client.transaction(r.OperationID)
	f.balance(account, "owned", "4300")
	if !view.SourceConflict {
		t.Fatal("conflict hidden")
	}
	if len(view.SourceFacts) != 1 || view.SourceFacts[0].Postings[0].Money.Amount != "-600" || view.SourceFacts[0].Merchant != "Bank merchant" {
		t.Fatal("source comparison missing", view.SourceFacts)
	}
	history := decode[generated.TransactionHistoryPage](t, client.call("GET", "/transactions/"+view.Id+"/history", "", nil, 200))
	if len(history.Items[0].Evidence) == 0 {
		t.Fatal("decision evidence missing")
	}
	view = client.undo(view, decision)
	f.balance(account, "owned", "4400")
	if *view.Merchant != "My merchant" {
		t.Fatal("unrelated override lost")
	}
	old := decode[generated.Transaction](t, client.call("GET", "/transactions/"+view.Id+"/revisions/1", "", nil, 200))
	for _, source := range old.Sources {
		if source.Revision != "1" {
			t.Fatal("historical evidence drifted", source)
		}
	}
}

func TestExcludedPendingFollowsBankLifecycleWithoutResurrection(t *testing.T) {
	f := newFixture(t)
	gate := f.admit()
	connection := f.connection(f.p)
	account := f.create(money.RUB, "5000")
	c := f.client(f.p)
	r := f.revision(uuid.NewString(), account, "-500", money.RUB, 1)
	r.State = ledger.Pending
	in := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: f.family.ID, Provider: "raiffeisen", ExternalAccountID: "synthetic-business", Product: "current", Log: "statement", RecordID: "excluded-purchase"}, PayloadHash: strings.Repeat("a", 64), EvidenceRef: "synthetic:pending", Classification: "new", Operation: &r}
	if _, err := f.importSource(gate, connection, in); err != nil {
		t.Fatal(err)
	}
	c.call("POST", "/transactions/"+r.OperationID+"/exclude", uuid.NewString(), map[string]any{"expectedRevision": 1, "reason": "Duplicate accounting"}, 202)
	view := c.transaction(r.OperationID)
	decision := *view.DecisionId
	f.balance(account, "available", "5000")
	r.State = ledger.Posted
	r.PostedAt = f.now
	in.Classification = "correction"
	in.ExpectedRevision = 1
	in.PayloadHash = strings.Repeat("b", 64)
	if _, err := f.importSource(gate, connection, in); err != nil {
		t.Fatal(err)
	}
	view = c.transaction(r.OperationID)
	if view.AccountingState != "excluded" || view.State != "posted" {
		t.Fatal("excluded lifecycle", view)
	}
	f.balance(account, "owned", "5000")
	r.State = ledger.Reversed
	in.ExpectedRevision = 2
	in.PayloadHash = strings.Repeat("c", 64)
	if _, err := f.importSource(gate, connection, in); err != nil {
		t.Fatal(err)
	}
	view = c.undo(c.transaction(r.OperationID), decision)
	if view.State != "reversed" || view.AccountingState != "included" {
		t.Fatal("bank status resurrected")
	}
	f.balance(account, "owned", "5000")
}
