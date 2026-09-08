package domain

import (
	"errors"
	"testing"

	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func receiptMoney(t *testing.T, value string, asset money.Asset) money.Money {
	t.Helper()
	result, err := money.NewMoney(value, asset)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestReceiptItemsAllocateDiscountExactlyForEveryAsset(t *testing.T) {
	for _, asset := range []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH} {
		items, err := BuildReceiptItems([]ReceiptItemInput{
			{ID: "00000000-0000-4000-8000-000000000001", Name: "First", Quantity: "1", Gross: receiptMoney(t, "600", asset)},
			{ID: "00000000-0000-4000-8000-000000000002", Name: "Second", Quantity: "2.5", Gross: receiptMoney(t, "400", asset)},
		}, receiptMoney(t, "100", asset), receiptMoney(t, "900", asset))
		if err != nil || items[0].Discount.Amount() != "60" || items[1].Discount.Amount() != "40" {
			t.Fatalf("%s allocation: %+v %v", asset, items, err)
		}
	}
}

func TestReceiptItemsUseStableIDForEqualRemainders(t *testing.T) {
	items, err := BuildReceiptItems([]ReceiptItemInput{
		{ID: "b", Name: "B", Quantity: "1", Gross: receiptMoney(t, "1", money.RUB)},
		{ID: "a", Name: "A", Quantity: "1", Gross: receiptMoney(t, "1", money.RUB)},
		{ID: "c", Name: "C", Quantity: "1", Gross: receiptMoney(t, "1", money.RUB)},
	}, receiptMoney(t, "0.01", money.RUB), receiptMoney(t, "2.99", money.RUB))
	if err != nil || items[0].Discount.Amount() != "0.00" || items[1].Discount.Amount() != "0.01" || items[2].Discount.Amount() != "0.00" {
		t.Fatalf("unstable allocation: %+v %v", items, err)
	}
}

func TestReceiptItemsRejectAmbiguousAndInvalidInput(t *testing.T) {
	discount := receiptMoney(t, "10", money.RUB)
	_, err := BuildReceiptItems([]ReceiptItemInput{
		{ID: "a", Name: "A", Quantity: "1", Gross: receiptMoney(t, "60", money.RUB), Discount: &discount},
		{ID: "b", Name: "B", Quantity: "1", Gross: receiptMoney(t, "40", money.RUB)},
	}, discount, receiptMoney(t, "90", money.RUB))
	if !errors.Is(err, ErrClarificationRequired) {
		t.Fatalf("mixed discounts: %v", err)
	}
	for _, quantity := range []string{"0", "0.0", "-1", "1e2"} {
		item := ReceiptItem{ID: "a", Name: "A", Quantity: quantity, Gross: receiptMoney(t, "1", money.RUB), Discount: receiptMoney(t, "0", money.RUB)}
		if item.Validate() == nil {
			t.Fatalf("accepted quantity %q", quantity)
		}
	}
}

func TestClassificationProposalRejectsEmptyProposal(t *testing.T) {
	revision := Revision{Type: Expense}

	if err := (ClassificationProposal{}).ValidateFor(revision); !errors.Is(err, ErrInvalidAllocation) {
		t.Fatalf("ValidateFor() error = %v, want %v", err, ErrInvalidAllocation)
	}
}

func TestClassificationProposalRequiresMerchantForAlias(t *testing.T) {
	revision := Revision{Type: Expense}

	if err := (ClassificationProposal{MerchantAlias: "Dizengoff"}).ValidateFor(revision); !errors.Is(err, ErrInvalidAllocation) {
		t.Fatalf("ValidateFor() error = %v, want %v", err, ErrInvalidAllocation)
	}
}
