//go:build integration

package matching_test

import (
	"testing"

	"github.com/google/uuid"
	category "github.com/pchkauu/want-keep/backend/internal/categories/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestMatchingUndoPreservesLaterClassification(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "5000")
	a, b := uuid.NewString(), uuid.NewString()
	for _, id := range []string{a, b} {
		if _, err := f.write(f.revision(id, account, "-500", money.RUB, 1), request()); err != nil {
			t.Fatal(err)
		}
	}
	f.link(matching.Payment, a, a, b)
	linkDecision := f.current(a).DecisionID
	c := f.client(f.p)
	categoryID := category.StarterCategories(f.family.ID)[0].ID
	c.result("/transactions/"+a+"/corrections", map[string]any{"expectedRevision": f.current(a).Revision, "category": map[string]any{"action": "set", "id": categoryID}, "reason": "Classify confirmed purchase"})
	classificationDecision := f.current(a).DecisionID
	c.result("/transactions/"+a+"/undo", map[string]any{"decisionId": linkDecision, "expectedRevisions": f.versions(a, b), "reason": "Restore unresolved evidence"})
	r := f.current(a)
	if r.CategoryID != categoryID || r.Protections[ledger.CategoryField].DecisionID != classificationDecision {
		t.Fatal("link undo lost independent classification", r)
	}
	f.balance(account, "owned", "4500")
	c.result("/transactions/"+a+"/undo", map[string]any{"decisionId": classificationDecision, "expectedRevisions": f.versions(a), "reason": "Undo classification"})
	if f.current(a).CategoryID != "" {
		t.Fatal("classification undo did not restore the previous value")
	}
}

func TestMatchingRejectsConflictingProtectedCategories(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "5000")
	a, b := uuid.NewString(), uuid.NewString()
	c := f.client(f.p)
	categories := category.StarterCategories(f.family.ID)
	for i, id := range []string{a, b} {
		if _, err := f.write(f.revision(id, account, "-500", money.RUB, 1), request()); err != nil {
			t.Fatal(err)
		}
		c.result("/transactions/"+id+"/corrections", map[string]any{"expectedRevision": f.current(id).Revision, "category": map[string]any{"action": "set", "id": categories[i].ID}, "reason": "Confirmed classification"})
	}
	beforeA, beforeB := f.current(a), f.current(b)
	out := decode[generated.CommandFailed](t, c.call("POST", "/transactions/"+a+"/links", uuid.NewString(), map[string]any{"kind": "receipt_match", "expectedRevisions": f.versions(a, b), "reason": "Try conflicting classifications"}, 202))
	if out.Error.Code != "matching_conflict" || f.current(a).Revision != beforeA.Revision || f.current(b).Revision != beforeB.Revision {
		t.Fatal("conflicting classification changed matching", out)
	}
	f.balance(account, "owned", "4500")
}

func TestMatchingReceiptItemsAndCompoundAmountCorrection(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "5000")
	a, b := uuid.NewString(), uuid.NewString()
	for _, id := range []string{a, b} {
		if _, err := f.write(f.revision(id, account, "-500", money.RUB, 1), request()); err != nil {
			t.Fatal(err)
		}
	}
	f.link(matching.Payment, a, a, b)
	c := f.client(f.p)
	itemID := uuid.NewString()
	items := map[string]any{"action": "replace", "totalDiscount": map[string]string{"amount": "0", "asset": "RUB"}, "items": []map[string]any{{"id": itemID, "name": "Synthetic groceries", "quantity": "1", "gross": map[string]string{"amount": "500", "asset": "RUB"}}}}
	c.result("/transactions/"+a+"/corrections", map[string]any{"expectedRevision": f.current(a).Revision, "receiptItems": items, "reason": "Record receipt items"})
	c.result("/transactions/"+b+"/corrections", map[string]any{"expectedRevision": f.current(b).Revision, "receiptItems": items, "reason": "Confirm matching receipt items"})
	got := c.transaction(a)
	if len(got.ReceiptItems) != 1 || got.ReceiptItems[0].Id != itemID || got.ReceiptItems[0].Net.Amount != "500" {
		t.Fatal("receipt items did not survive matching transport", got.ReceiptItems)
	}
	principal := c.transaction(a).Postings
	principal[0].Money.Amount = "-700"
	body := map[string]any{"expectedRevision": f.current(a).Revision, "principal": principal, "relatedChanges": []map[string]any{{"transactionId": b, "expectedRevision": f.current(b).Revision, "principal": principal}}, "reason": "Correct payment and receipt together"}
	failed := decode[generated.CommandFailed](t, c.call("POST", "/transactions/"+a+"/corrections", uuid.NewString(), body, 202))
	if failed.Error.Code != "invalid_allocation" {
		t.Fatal("inconsistent receipt allowed", failed)
	}
	f.balance(account, "owned", "4500")
	items["items"].([]map[string]any)[0]["gross"] = map[string]string{"amount": "700", "asset": "RUB"}
	body["receiptItems"] = items
	body["relatedChanges"].([]map[string]any)[0]["receiptItems"] = items
	c.result("/transactions/"+a+"/corrections", body)
	f.balance(account, "owned", "4300")
	if f.current(a).ReceiptItems[0].Gross.Amount() != "700" || f.current(b).ReceiptItems[0].Gross.Amount() != "700" || f.current(b).Postings[0].Money.Amount() != "-700" {
		t.Fatal("compound correction did not retain the matching and receipt invariant")
	}
}
