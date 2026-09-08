//go:build integration

package familyallocation_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestFamilyAllocationHTTPRulesAndExactAssets(t *testing.T) {
	fixture := newFixture(t)
	first := fixture.client(fixture.p)
	second := fixture.client(fixture.q)
	merchant := decode[generated.CommandSucceeded](t, first.call(http.MethodPost, "/merchants", uuid.NewString(), map[string]any{"name": "Synthetic market", "aliases": []string{}}, http.StatusAccepted))
	if merchant.Status != "succeeded" {
		t.Fatal(merchant)
	}
	rule := createRule(t, first, merchant.Result.Id, 20, "50", "50")
	read := decode[generated.AllocationRule](t, second.call(http.MethodGet, "/allocation-rules/"+rule.Result.Id, "", nil, http.StatusOK))
	if read.Revision != 1 || len(read.Shares) != 2 {
		t.Fatalf("rule round trip: %+v", read)
	}
	preview := decode[generated.AllocationRulePreview](t, second.call(http.MethodPost, "/allocation-rules/preview", "", map[string]any{"merchantId": merchant.Result.Id}, http.StatusOK))
	if preview.State != "resolved" || preview.Reason != "matched" || len(preview.Rules) != 1 {
		t.Fatalf("preview: %+v", preview)
	}

	for _, asset := range []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH} {
		accountID := fixture.account(asset, "10")
		created := createExpense(t, first, accountID, asset, "0.0000000000000000012345", merchant.Result.Id, unresolved())
		transaction := readTransaction(t, second, created.Result.Id)
		if transaction.Allocation.State != "resolved" || transaction.Allocation.Origin != "rule" || len(transaction.Allocation.Members) != 2 || len(transaction.Allocation.Rules) != 1 {
			t.Fatalf("%s allocation: %+v", asset, transaction.Allocation)
		}
		total := "0.0000000000000000012345"
		if transaction.Allocation.Members[0].Amount.Amount == total || transaction.Allocation.Members[1].Amount.Amount == total {
			t.Fatalf("%s duplicated family fact: %+v", asset, transaction.Allocation.Members)
		}
	}

	changed := decode[generated.CommandSucceeded](t, second.call(http.MethodPost, "/allocation-rules/"+rule.Result.Id, uuid.NewString(), ruleInput(second, merchant.Result.Id, 20, "60", "40", int64(1)), http.StatusAccepted))
	if changed.Status != "succeeded" || changed.Result.Revision != 2 {
		t.Fatal(changed)
	}
	accountID := fixture.account(money.RUB, "1000")
	created := createExpense(t, first, accountID, money.RUB, "100", merchant.Result.Id, unresolved())
	transaction := readTransaction(t, first, created.Result.Id)
	assertMemberAmounts(t, fixture, transaction.Allocation, "60", "40")

	createRule(t, first, merchant.Result.Id, 20, "40", "60")
	preview = decode[generated.AllocationRulePreview](t, first.call(http.MethodPost, "/allocation-rules/preview", "", map[string]any{"merchantId": merchant.Result.Id}, http.StatusOK))
	if preview.State != "unresolved" || preview.Reason != "rule_conflict" || len(preview.Rules) != 2 {
		t.Fatalf("tie conflict: %+v", preview)
	}
	page := decode[generated.AllocationRulePage](t, first.call(http.MethodGet, "/allocation-rules?limit=1", "", nil, http.StatusOK))
	if len(page.Items) != 1 || page.NextCursor == nil {
		t.Fatalf("rule page: %+v", page)
	}
	second.call(http.MethodGet, "/allocation-rules?limit=1&cursor="+url.QueryEscape(*page.NextCursor), "", nil, http.StatusBadRequest)
}

func TestMixedReceiptDirectAllocationAndValidation(t *testing.T) {
	fixture := newFixture(t)
	first := fixture.client(fixture.p)
	second := fixture.client(fixture.q)
	accountID := fixture.account(money.RUB, "5000")
	created := createExpense(t, first, accountID, money.RUB, "1000", "", unresolved())
	corrected := decode[generated.CommandSucceeded](t, first.call(http.MethodPost, "/transactions/"+created.Result.Id+"/corrections", uuid.NewString(), map[string]any{
		"expectedRevision": 1,
		"reason":           "Attach synthetic receipt items",
		"receiptItems": map[string]any{
			"action":        "replace",
			"totalDiscount": map[string]any{"amount": "0", "asset": "RUB"},
			"items": []any{
				map[string]any{"id": uuid.NewString(), "name": "Shared", "quantity": "1", "gross": map[string]any{"amount": "600", "asset": "RUB"}},
				map[string]any{"id": uuid.NewString(), "name": "Personal A", "quantity": "1", "gross": map[string]any{"amount": "100", "asset": "RUB"}},
				map[string]any{"id": uuid.NewString(), "name": "Personal B", "quantity": "1", "gross": map[string]any{"amount": "300", "asset": "RUB"}},
			},
		},
	}, http.StatusAccepted))
	if corrected.Status != "succeeded" {
		t.Fatal(corrected)
	}
	revision := readTransaction(t, first, created.Result.Id)
	items := []any{
		map[string]any{"itemId": revision.ReceiptItems[0].Id, "allocation": equalShared()},
		map[string]any{"itemId": revision.ReceiptItems[1].Id, "allocation": personal(string(fixture.members[0].ID), "100")},
		map[string]any{"itemId": revision.ReceiptItems[2].Id, "allocation": personal(string(fixture.members[1].ID), "300")},
	}
	allocated := decode[generated.CommandSucceeded](t, second.call(http.MethodPost, "/transactions/"+created.Result.Id+"/allocations", uuid.NewString(), map[string]any{"expectedRevision": 2, "reason": "Split mixed receipt", "allocation": unresolved(), "items": items}, http.StatusAccepted))
	if allocated.Status != "succeeded" {
		t.Fatal(allocated)
	}
	revision = readTransaction(t, first, created.Result.Id)
	if revision.Allocation.State != "resolved" || revision.Allocation.Mode == nil || *revision.Allocation.Mode != "composite" || len(revision.ReceiptItems) != 3 {
		t.Fatalf("mixed allocation: %+v", revision.Allocation)
	}
	assertMemberAmounts(t, fixture, revision.Allocation, "400", "600")
	if revision.Postings[0].Money.Amount != "-1000" {
		t.Fatal("analytical allocation changed family posting")
	}

	invalid := shareAllocation("70", "20", fixture)
	failed := decode[generated.CommandFailed](t, first.call(http.MethodPost, "/transactions/"+created.Result.Id+"/allocations", uuid.NewString(), map[string]any{"expectedRevision": 3, "reason": "Invalid split", "allocation": invalid}, http.StatusAccepted))
	if failed.Error.Code != "invalid_allocation" {
		t.Fatalf("invalid split accepted: %+v", failed)
	}
	foreign := shareAllocation("50", "50", fixture)
	foreign["members"].([]any)[1].(map[string]any)["memberId"] = uuid.NewString()
	failed = decode[generated.CommandFailed](t, first.call(http.MethodPost, "/transactions/"+created.Result.Id+"/allocations", uuid.NewString(), map[string]any{"expectedRevision": 3, "reason": "Foreign member", "allocation": foreign}, http.StatusAccepted))
	if failed.Error.Code != "invalid_allocation" {
		t.Fatalf("foreign member accepted: %+v", failed)
	}
	unchanged := readTransaction(t, first, created.Result.Id)
	if unchanged.Revision != 3 || unchanged.Postings[0].Money.Amount != "-1000" {
		t.Fatal("failed allocation left partial effect")
	}
}

func TestAmountAllocationAcrossAssetsRoundTripsPostgreSQL(t *testing.T) {
	fixture := newFixture(t)
	rubAccount := fixture.account(money.RUB, "100")
	ethAccount := fixture.account(money.ETH, "1")
	rubPrincipal, _ := money.NewMoney("-10", money.RUB)
	ethFee, _ := money.NewMoney("-0.000000000000000003", money.ETH)
	rubA, rubB := allocationAmount(t, "6", money.RUB), allocationAmount(t, "4", money.RUB)
	ethA, ethB := allocationAmount(t, "0.000000000000000002", money.ETH), allocationAmount(t, "0.000000000000000001", money.ETH)
	revision := ledger.Revision{
		OperationID: uuid.NewString(), Revision: 1, ActorID: fixture.p.UserID(), Reason: "Synthetic multi-asset expense",
		Type: ledger.Expense, State: ledger.Posted, OccurredAt: fixture.now, FeeKnowledge: ledger.KnownFees,
		PayerState: "known", PayerMemberID: fixture.members[0].ID,
		Postings: []ledger.Posting{
			{AccountID: rubAccount, Money: rubPrincipal, Role: ledger.Principal, Funding: ledger.OwnFunds, Treatment: ledger.Movement},
			{AccountID: ethAccount, Money: ethFee, Role: ledger.Fee, Funding: ledger.OwnFunds, Treatment: ledger.Movement},
		},
	}
	input := ledger.AllocationInput{Mode: ledger.AllocationByAmounts, Purpose: ledger.AllocationShared, Members: []ledger.AllocationMemberInput{
		{MemberID: fixture.members[0].ID, Amount: rubA}, {MemberID: fixture.members[1].ID, Amount: rubB},
		{MemberID: fixture.members[0].ID, Amount: ethA}, {MemberID: fixture.members[1].ID, Amount: ethB},
	}}
	allocated, err := revision.WithAllocation(input, nil, []household.MembershipID{fixture.members[0].ID, fixture.members[1].ID})
	if err != nil {
		t.Fatal(err)
	}
	writer := journal.NewWriter(fixture.store, fixture.store)
	if err = fixture.store.WithinHousehold(testContext, fixture.p, func(ctx context.Context) error {
		return writer.Append(ctx, fixture.p, allocated, 0)
	}); err != nil {
		t.Fatal(err)
	}
	stored, found, err := fixture.store.CurrentLedgerRevision(testContext, fixture.p, revision.OperationID)
	if err != nil || !found || len(stored.Allocation.Inputs) != 4 || len(stored.Allocation.Members) != 4 {
		t.Fatalf("multi-asset allocation round trip: found=%v inputs=%d members=%d err=%v", found, len(stored.Allocation.Inputs), len(stored.Allocation.Members), err)
	}
}

func createRule(t *testing.T, client *client, merchantID string, priority int, first, second string) generated.CommandSucceeded {
	t.Helper()
	result := decode[generated.CommandSucceeded](t, client.call(http.MethodPost, "/allocation-rules", uuid.NewString(), ruleInput(client, merchantID, priority, first, second, 0), http.StatusAccepted))
	if result.Status != "succeeded" {
		t.Fatal(result)
	}
	return result
}

func ruleInput(client *client, merchantID string, priority int, first, second string, expected int64) map[string]any {
	input := map[string]any{"priority": priority, "state": "active", "condition": map[string]any{"merchantId": merchantID}, "shares": []any{map[string]any{"memberId": string(client.f.members[0].ID), "share": first}, map[string]any{"memberId": string(client.f.members[1].ID), "share": second}}}
	if expected > 0 {
		input["expectedRevision"] = expected
	}
	return input
}

func createExpense(t *testing.T, client *client, accountID string, asset money.Asset, amount, merchantID string, allocation map[string]any) generated.CommandSucceeded {
	t.Helper()
	input := map[string]any{"type": "expense", "accountId": accountID, "amount": map[string]any{"amount": amount, "asset": string(asset)}, "occurredAt": "2026-09-08T10:00:00Z", "payer": map[string]any{"state": "known", "memberId": string(client.f.members[0].ID)}, "allocation": allocation}
	if merchantID != "" {
		input["merchantId"] = merchantID
	}
	result := decode[generated.CommandSucceeded](t, client.call(http.MethodPost, "/transactions", uuid.NewString(), input, http.StatusAccepted))
	if result.Status != "succeeded" {
		t.Fatal(result)
	}
	return result
}

func readTransaction(t *testing.T, client *client, id string) generated.Transaction {
	t.Helper()
	return decode[generated.Transaction](t, client.call(http.MethodGet, "/transactions/"+id, "", nil, http.StatusOK))
}

func unresolved() map[string]any {
	return map[string]any{"mode": "unresolved", "reason": "Needs explicit family allocation"}
}
func equalShared() map[string]any { return map[string]any{"mode": "equal", "purpose": "shared"} }
func personal(memberID, amount string) map[string]any {
	return map[string]any{"mode": "amounts", "purpose": "personal", "members": []any{map[string]any{"memberId": memberID, "amount": map[string]any{"amount": amount, "asset": "RUB"}}}}
}
func shareAllocation(first, second string, fixture *fixture) map[string]any {
	return map[string]any{"mode": "shares", "purpose": "shared", "members": []any{map[string]any{"memberId": string(fixture.members[0].ID), "share": first}, map[string]any{"memberId": string(fixture.members[1].ID), "share": second}}}
}
func assertMemberAmounts(t *testing.T, fixture *fixture, allocation generated.AllocationSnapshot, first, second string) {
	t.Helper()
	amounts := map[string]string{}
	for _, member := range allocation.Members {
		amounts[member.MemberId] = member.Amount.Amount
	}
	if len(amounts) != 2 || amounts[string(fixture.members[0].ID)] != first || amounts[string(fixture.members[1].ID)] != second {
		t.Fatalf("member amounts = %v, want %s/%s", amounts, first, second)
	}
}

func allocationAmount(t *testing.T, amount string, asset money.Asset) *money.Money {
	t.Helper()
	value, err := money.NewMoney(amount, asset)
	if err != nil {
		t.Fatal(err)
	}
	return &value
}
