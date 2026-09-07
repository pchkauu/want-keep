package domain_test

import (
	"errors"
	"reflect"
	"testing"

	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestCorrectionOwnsCopiesAndKeepsAccountingIndependent(t *testing.T) {
	e := examples{}
	r := e.expense()
	r.Protections = map[ledger.Field]ledger.Protection{ledger.MerchantField: {DecisionID: "previous", Revision: 1}}
	r.FieldVersions = map[ledger.Field]uint64{ledger.MerchantField: 1}
	before := r.Clone()
	posts := append([]ledger.Posting(nil), r.Postings...)
	posts[0].Money = e.money("-700", money.RUB)
	next, fields, err := r.Correct(ledger.Correction{Principal: &posts})
	if err != nil {
		t.Fatal(err)
	}
	d := ledger.Decision{ID: "correction", Kind: "correction", ActorID: "b", Reason: "Fixed amount", At: r.PostedAt}
	next = next.WithDecision(d, fields)
	next.Protections[ledger.MerchantField] = ledger.Protection{DecisionID: "changed", Revision: 2}
	if !reflect.DeepEqual(r, before) || posts[0].Money.Amount() != "-700" {
		t.Fatal("input changed")
	}
	next.AccountingState = ledger.ExcludedFromAccounting
	effects, err := next.BalanceEffects()
	if err != nil || len(effects) != 0 || next.State != ledger.Posted || len(next.Postings) != 1 {
		t.Fatal("exclusion destroyed facts", err)
	}
	next.AccountingState = "invalid"
	if next.Validate() == nil {
		t.Fatal("invalid disposition")
	}
}

func TestMoneyNoOpAndInvalidPrincipalIdentity(t *testing.T) {
	e := examples{}
	r := e.expense()
	posts := append([]ledger.Posting(nil), r.Postings...)
	posts[0].Money = e.money("-500.000", money.RUB)
	if _, _, err := r.Correct(ledger.Correction{Principal: &posts}); !errors.Is(err, ledger.ErrNoChange) {
		t.Fatal("scale became a correction", err)
	}
	posts[0].AccountID = "different"
	if _, _, err := r.Correct(ledger.Correction{Principal: &posts}); err == nil {
		t.Fatal("source identity replaced")
	}
	posts[0] = r.Postings[0]
	posts[0].Money = e.money("-700", money.USD)
	if _, _, err := r.Correct(ledger.Correction{Principal: &posts}); err == nil {
		t.Fatal("asset replaced")
	}
	r.Revision = 9007199254740991
	updated := r.WithDecision(ledger.Decision{ID: "overflow", ActorID: "b", Reason: "overflow", At: r.PostedAt}, []ledger.Field{ledger.NoteField})
	if updated.Validate() == nil || r.Revision != 9007199254740991 {
		t.Fatal("overflow or mutation")
	}
}

func TestSelectiveUndoChecksFieldOriginNotOnlyValue(t *testing.T) {
	e := examples{}
	initial := e.expense()
	name := "Market"
	changed, fields, err := initial.Correct(ledger.Correction{Merchant: &name})
	if err != nil {
		t.Fatal(err)
	}
	d := ledger.Decision{ID: "one", Kind: "correction", ActorID: "b", Reason: "merchant", At: initial.PostedAt}
	changed = changed.WithDecision(d, fields)
	entry := ledger.DecisionEntry{OperationID: initial.OperationID, Before: 1, After: 2, Fields: fields}
	later := changed.Clone()
	later.Revision = 4
	later.FieldVersions[ledger.MerchantField] = 4
	if _, err = later.UndoFields(entry, initial, nil); !errors.Is(err, ledger.ErrDecisionConflict) {
		t.Fatal("ABA accepted", err)
	}
	later = changed.Clone()
	later.Revision = 3
	later.Note = "Partner note"
	later.FieldVersions[ledger.NoteField] = 3
	undone, err := later.UndoFields(entry, initial, nil)
	if err != nil || undone.Merchant != "" || undone.Note != "Partner note" {
		t.Fatal("selective undo", err)
	}
	if later.Merchant != "Market" {
		t.Fatal("undo mutated input")
	}
}
