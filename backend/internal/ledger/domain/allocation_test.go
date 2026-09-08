package domain

import (
	"slices"
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestAllocationUsesExactLargestRemaindersForEveryAsset(t *testing.T) {
	members := []household.MembershipID{"member-b", "member-a", "member-c"}
	for _, asset := range []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH} {
		amount, err := money.NewMoney("0.00000000000000000001", asset)
		if err != nil {
			t.Fatal(err)
		}
		revision := expenseRevision(amount)
		input := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "33.333333333333333333"}, {MemberID: members[1], Share: "33.333333333333333334"}, {MemberID: members[2], Share: "33.333333333333333333"}}, Origin: AllocationExplicitPurchase}
		allocated, err := revision.WithAllocation(input, nil, members)
		if err != nil {
			t.Fatalf("%s: %v", asset, err)
		}
		if allocated.Allocation.State != AllocationResolved || len(allocated.Allocation.Members) != 1 || allocated.Allocation.Members[0].MemberID != "member-a" || allocated.Allocation.Members[0].Money.Amount() != amount.Amount() {
			t.Fatalf("%s allocation = %+v", asset, allocated.Allocation)
		}
	}
}

func TestMixedReceiptAllocationCountsHouseholdOnce(t *testing.T) {
	paid := mustAllocationMoney(t, "1000", money.RUB)
	revision := expenseRevision(paid)
	revision.ReceiptItems = []ReceiptItem{
		{ID: "shared", Name: "Shared", Quantity: "1", Gross: mustAllocationMoney(t, "600", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
		{ID: "personal-a", Name: "A", Quantity: "1", Gross: mustAllocationMoney(t, "100", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
		{ID: "personal-b", Name: "B", Quantity: "1", Gross: mustAllocationMoney(t, "300", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
	}
	members := []household.MembershipID{"member-a", "member-b"}
	shared := AllocationInput{Mode: AllocationEqual, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0]}, {MemberID: members[1]}}, Origin: AllocationExplicitItem}
	personalA := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationPersonal, Members: []AllocationMemberInput{{MemberID: members[0], Amount: allocationMoneyPtr(t, "100", money.RUB)}}, Origin: AllocationExplicitItem}
	personalB := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationPersonal, Members: []AllocationMemberInput{{MemberID: members[1], Amount: allocationMoneyPtr(t, "300", money.RUB)}}, Origin: AllocationExplicitItem}
	allocated, err := revision.WithAllocation(AllocationInput{Mode: AllocationUnknown, Reason: "item allocations"}, []ItemAllocationInput{{ItemID: "shared", Allocation: shared}, {ItemID: "personal-a", Allocation: personalA}, {ItemID: "personal-b", Allocation: personalB}}, members)
	if err != nil {
		t.Fatal(err)
	}
	if allocated.Allocation.State != AllocationResolved || allocated.Allocation.Mode != AllocationComposite || len(allocated.Allocation.Members) != 2 {
		t.Fatalf("allocation = %+v", allocated.Allocation)
	}
	if allocated.Allocation.Members[0].MemberID != members[0] || allocated.Allocation.Members[0].Money.Amount() != "400" || allocated.Allocation.Members[1].MemberID != members[1] || allocated.Allocation.Members[1].Money.Amount() != "600" {
		t.Fatalf("member totals = %+v", allocated.Allocation.Members)
	}
}

func TestAllocationRejectsInvalidMembersAndShareSumWithoutMutation(t *testing.T) {
	revision := expenseRevision(mustAllocationMoney(t, "1000", money.RUB))
	input := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: "member-a", Share: "60"}, {MemberID: "member-b", Share: "30"}}}
	updated, err := revision.WithAllocation(input, nil, []household.MembershipID{"member-a", "member-b"})
	if err != ErrInvalidAllocation {
		t.Fatalf("err = %v", err)
	}
	if updated.Allocation.State != "" || revision.Allocation.State != "" {
		t.Fatal("failed allocation mutated revision")
	}
	input.Members[1].Share = "40"
	input.Members[1].MemberID = "foreign"
	if _, err = revision.WithAllocation(input, nil, []household.MembershipID{"member-a", "member-b"}); err != ErrInvalidAllocation {
		t.Fatalf("foreign member err = %v", err)
	}
}

func TestAllocationDistributesFeeInThirdAssetWithoutConversion(t *testing.T) {
	rub := mustAllocationMoney(t, "-9000", money.RUB)
	usdt := mustAllocationMoney(t, "100", money.USDT)
	ethFee := mustAllocationMoney(t, "-0.000000000000000003", money.ETH)
	revision := validAllocationRevision(Exchange, Posted, []Posting{{AccountID: "rub", Money: rub, Role: Principal, Funding: OwnFunds, Treatment: Movement}, {AccountID: "usdt", Money: usdt, Role: Principal, Funding: OwnFunds, Treatment: Movement}, {AccountID: "eth", Money: ethFee, Role: Fee, Funding: OwnFunds, Treatment: Movement}})
	members := []household.MembershipID{"member-a", "member-b"}
	input := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "60"}, {MemberID: members[1], Share: "40"}}}
	allocated, err := revision.WithAllocation(input, nil, members)
	if err != nil {
		t.Fatal(err)
	}
	if len(allocated.Allocation.Members) != 2 || allocated.Allocation.Members[0].Money.Asset() != money.ETH || allocated.Allocation.Members[0].Money.Amount() != "0.000000000000000002" || allocated.Allocation.Members[1].Money.Amount() != "0.000000000000000001" {
		t.Fatalf("third asset = %+v", allocated.Allocation.Members)
	}
}

func TestCorrectionRecalculatesSharesAndRejectsStaleAmounts(t *testing.T) {
	members := []household.MembershipID{"member-a", "member-b"}
	revision := expenseRevision(mustAllocationMoney(t, "100", money.RUB))
	shares := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "60"}, {MemberID: members[1], Share: "40"}}}
	allocated, err := revision.WithAllocation(shares, nil, members)
	if err != nil {
		t.Fatal(err)
	}
	changed := []Posting{{AccountID: "account", Money: mustAllocationMoney(t, "-200", money.RUB), Role: Principal, Funding: OwnFunds, Treatment: Movement}}
	corrected, fields, err := allocated.Correct(Correction{Principal: &changed})
	if err != nil || !slices.Contains(fields, AllocationField) || corrected.Allocation.Members[0].Money.Amount() != "120" || corrected.Allocation.Members[1].Money.Amount() != "80" {
		t.Fatalf("share correction = %+v %v %v", corrected.Allocation, fields, err)
	}
	amounts := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Amount: allocationMoneyPtr(t, "60", money.RUB)}, {MemberID: members[1], Amount: allocationMoneyPtr(t, "40", money.RUB)}}}
	allocated, err = revision.WithAllocation(amounts, nil, members)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = allocated.Correct(Correction{Principal: &changed}); err != ErrInvalidAllocation {
		t.Fatalf("amount correction error = %v", err)
	}
}

func TestSourceMergeCannotEraseOrDetachAllocation(t *testing.T) {
	members := []household.MembershipID{"member-a", "member-b"}
	revision := expenseRevision(mustAllocationMoney(t, "100", money.RUB))
	allocated, err := revision.WithAllocation(AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "60"}, {MemberID: members[1], Share: "40"}}}, nil, members)
	if err != nil {
		t.Fatal(err)
	}
	source := allocated.Clone()
	source.Allocation = AllocationSnapshot{}
	source.Merchant = "Updated by provider"
	merged, conflict, err := allocated.MergeSource(source)
	if err != nil || conflict || !allocated.FieldEqual(merged, AllocationField) || merged.Validate() != nil {
		t.Fatalf("safe source merge = %+v, conflict=%v, err=%v", merged.Allocation, conflict, err)
	}

	source.Postings[0].Money = mustAllocationMoney(t, "-200", money.RUB)
	merged, _, err = allocated.MergeSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := merged.Validate(); err != ErrInvalidAllocation {
		t.Fatalf("stale allocation accepted after source amount change: %v", err)
	}
}

func TestAmountAllocationCoversMultipleComponentsAndAssetsExactly(t *testing.T) {
	members := []household.MembershipID{"member-a", "member-b"}
	revision := expenseRevision(mustAllocationMoney(t, "100", money.RUB))
	revision.Postings = append(revision.Postings, Posting{AccountID: "account", Money: mustAllocationMoney(t, "-10", money.RUB), Role: Fee, Funding: OwnFunds, Treatment: Movement})
	input := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Amount: allocationMoneyPtr(t, "70", money.RUB)}, {MemberID: members[1], Amount: allocationMoneyPtr(t, "40", money.RUB)}}}
	allocated, err := revision.WithAllocation(input, nil, members)
	if err != nil || allocationMemberTotal(allocated.Allocation, members[0], money.RUB) != "70" || allocationMemberTotal(allocated.Allocation, members[1], money.RUB) != "40" {
		t.Fatalf("multi-component amounts = %+v, err=%v", allocated.Allocation, err)
	}

	revision = validAllocationRevision(Transfer, Posted, []Posting{
		{AccountID: "from", Money: mustAllocationMoney(t, "-100", money.RUB), Role: Principal, Funding: OwnFunds, Treatment: Movement},
		{AccountID: "to", Money: mustAllocationMoney(t, "100", money.RUB), Role: Principal, Funding: OwnFunds, Treatment: Movement},
		{AccountID: "from", Money: mustAllocationMoney(t, "-5", money.RUB), Role: Fee, Funding: OwnFunds, Treatment: Movement},
		{AccountID: "eth", Money: mustAllocationMoney(t, "-0.000000000000000003", money.ETH), Role: Fee, Funding: OwnFunds, Treatment: Movement},
	})
	input.Members = []AllocationMemberInput{
		{MemberID: members[0], Amount: allocationMoneyPtr(t, "3", money.RUB)},
		{MemberID: members[1], Amount: allocationMoneyPtr(t, "2", money.RUB)},
		{MemberID: members[0], Amount: allocationMoneyPtr(t, "0.000000000000000002", money.ETH)},
		{MemberID: members[1], Amount: allocationMoneyPtr(t, "0.000000000000000001", money.ETH)},
	}
	allocated, err = revision.WithAllocation(input, nil, members)
	if err != nil || allocated.Validate() != nil || allocationMemberTotal(allocated.Allocation, members[0], money.ETH) != "0.000000000000000002" || allocationMemberTotal(allocated.Allocation, members[1], money.RUB) != "2" {
		t.Fatalf("multi-asset amounts = %+v, err=%v", allocated.Allocation, err)
	}
}

func TestCompositeAllocationPreservesSharedIntentAtOneQuantum(t *testing.T) {
	paid := mustAllocationMoney(t, "0.01", money.RUB)
	revision := expenseRevision(paid)
	revision.ReceiptItems = []ReceiptItem{{ID: "small", Name: "Small", Quantity: "1", Gross: paid, Discount: mustAllocationMoney(t, "0", money.RUB)}}
	members := []household.MembershipID{"member-a", "member-b"}
	shared := AllocationInput{Mode: AllocationEqual, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0]}, {MemberID: members[1]}}}
	allocated, err := revision.WithAllocation(AllocationInput{Mode: AllocationUnknown, Reason: "item allocation"}, []ItemAllocationInput{{ItemID: "small", Allocation: shared}}, members)
	if err != nil {
		t.Fatal(err)
	}
	if allocated.Allocation.Purpose != AllocationShared || allocated.Allocation.Members[0].MemberID != members[0] || allocated.Allocation.Members[0].Money.Amount() != "0.01" {
		t.Fatalf("shared one-quantum allocation = %+v", allocated.Allocation)
	}
}

func TestCompositeShareAllocationRefreshesAndAmountBasisRejectsChanges(t *testing.T) {
	paid := mustAllocationMoney(t, "100", money.RUB)
	revision := expenseRevision(paid)
	revision.ReceiptItems = []ReceiptItem{
		{ID: "first", Name: "First", Quantity: "1", Gross: mustAllocationMoney(t, "40", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
		{ID: "second", Name: "Second", Quantity: "1", Gross: mustAllocationMoney(t, "60", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
	}
	members := []household.MembershipID{"member-a", "member-b"}
	fallback := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "50"}, {MemberID: members[1], Share: "50"}}}
	override := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "60"}, {MemberID: members[1], Share: "40"}}}
	allocated, err := revision.WithAllocation(fallback, []ItemAllocationInput{{ItemID: "first", Allocation: override}}, members)
	if err != nil || allocated.Allocation.Fallback == nil {
		t.Fatalf("composite allocation = %+v, err=%v", allocated.Allocation, err)
	}
	principal := []Posting{{AccountID: "account", Money: mustAllocationMoney(t, "-200", money.RUB), Role: Principal, Funding: OwnFunds, Treatment: Movement}}
	zero := mustAllocationMoney(t, "0", money.RUB)
	items := ReceiptItemsCorrection{Items: []ReceiptItemInput{
		{ID: "first", Name: "First", Quantity: "1", Gross: mustAllocationMoney(t, "80", money.RUB)},
		{ID: "second", Name: "Second", Quantity: "1", Gross: mustAllocationMoney(t, "120", money.RUB)},
	}, TotalDiscount: zero}
	corrected, fields, err := allocated.Correct(Correction{Principal: &principal, ReceiptItems: &items})
	if err != nil || !slices.Contains(fields, AllocationField) || allocationMemberTotal(corrected.Allocation, members[0], money.RUB) != "108" || allocationMemberTotal(corrected.Allocation, members[1], money.RUB) != "92" {
		t.Fatalf("refreshed allocation = %+v, fields=%v, err=%v", corrected.Allocation, fields, err)
	}

	amountOverride := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Amount: allocationMoneyPtr(t, "24", money.RUB)}, {MemberID: members[1], Amount: allocationMoneyPtr(t, "16", money.RUB)}}}
	allocated, err = revision.WithAllocation(fallback, []ItemAllocationInput{{ItemID: "first", Allocation: amountOverride}}, members)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = allocated.Correct(Correction{Principal: &principal, ReceiptItems: &items}); err != ErrInvalidAllocation {
		t.Fatalf("amount-based composite refresh error = %v", err)
	}
}

func TestAllocationInputOrderIsSemanticNoChange(t *testing.T) {
	revision := expenseRevision(mustAllocationMoney(t, "100", money.RUB))
	members := []household.MembershipID{"member-a", "member-b"}
	first := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "60"}, {MemberID: members[1], Share: "40"}}}
	allocated, err := revision.WithAllocation(first, nil, members)
	if err != nil {
		t.Fatal(err)
	}
	reordered := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[1], Share: "40"}, {MemberID: members[0], Share: "60"}}}
	if _, _, err = allocated.Correct(Correction{Allocation: &AllocationChange{Allocation: reordered, Members: members}}); err != ErrNoChange {
		t.Fatalf("reordered allocation error = %v", err)
	}
}

func TestAllocationExcludesMatchedInternalAndNonCarrierPrincipal(t *testing.T) {
	for _, tc := range []struct {
		name         string
		kind         ParticipationKind
		carrierID    string
		contribution ContributionRole
	}{
		{name: "internal transfer", kind: "transfer", carrierID: "operation", contribution: "outgoing"},
		{name: "payment evidence", kind: "payment", carrierID: "other", contribution: "payment"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			revision := expenseRevision(mustAllocationMoney(t, "1000", money.RUB))
			revision.Allocation, revision.Participation = AllocationSnapshot{}, Participation{GroupID: "group", Kind: tc.kind, State: "linked", Parts: []Contribution{{ComponentID: tc.carrierID + ":0", Position: 0, CarrierID: tc.carrierID, CarrierPosition: 0, Role: tc.contribution, State: Posted, At: revision.OccurredAt}}}
			refreshed, err := revision.RefreshAllocation()
			if err != nil || refreshed.Allocation.State != AllocationNotApplicable || refreshed.Validate() != nil {
				t.Fatalf("refreshed = %+v, err=%v", refreshed.Allocation, err)
			}
		})
	}
}

func TestIncomeAllocationIsNotApplicableEvenWithFee(t *testing.T) {
	revision := validAllocationRevision(Income, Posted, []Posting{
		{AccountID: "account", Money: mustAllocationMoney(t, "100", money.RUB), Role: Principal, Funding: OwnFunds, Treatment: Movement},
		{AccountID: "account", Money: mustAllocationMoney(t, "-5", money.RUB), Role: Fee, Funding: OwnFunds, Treatment: Movement},
	})
	allocated, err := revision.WithAllocation(AllocationInput{}, nil, nil)
	if err != nil || allocated.Allocation.State != AllocationNotApplicable || allocated.Validate() != nil {
		t.Fatalf("income allocation = %+v, err=%v", allocated.Allocation, err)
	}
}

func TestPurchaseAmountFallbackProducesExactItemSnapshots(t *testing.T) {
	members := []household.MembershipID{"member-a", "member-b"}
	revision := expenseRevision(mustAllocationMoney(t, "100", money.RUB))
	revision.ReceiptItems = []ReceiptItem{
		{ID: "first", Name: "First", Quantity: "1", Gross: mustAllocationMoney(t, "33", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
		{ID: "second", Name: "Second", Quantity: "1", Gross: mustAllocationMoney(t, "67", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
	}
	input := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Amount: allocationMoneyPtr(t, "40", money.RUB)}, {MemberID: members[1], Amount: allocationMoneyPtr(t, "60", money.RUB)}}}
	allocated, err := revision.WithAllocation(input, nil, members)
	if err != nil || allocated.Validate() != nil || allocationMemberTotal(allocated.Allocation, members[0], money.RUB) != "40" || allocationMemberTotal(allocated.Allocation, members[1], money.RUB) != "60" {
		t.Fatalf("purchase amount fallback = %+v, err=%v", allocated.Allocation, err)
	}
	for _, item := range allocated.ReceiptItems {
		if item.Allocation.State != AllocationResolved {
			t.Fatalf("item allocation = %+v", item.Allocation)
		}
	}
}

func allocationMemberTotal(snapshot AllocationSnapshot, memberID household.MembershipID, asset money.Asset) string {
	for _, member := range snapshot.Members {
		if member.MemberID == memberID && member.Money.Asset() == asset {
			return member.Money.Amount()
		}
	}
	return "0"
}

func expenseRevision(amount money.Money) Revision {
	zero, _ := money.NewMoney("0", amount.Asset())
	negative, _ := zero.Subtract(amount)
	return validAllocationRevision(Expense, Posted, []Posting{{AccountID: "account", Money: negative, Role: Principal, Funding: OwnFunds, Treatment: Movement}})
}

func validAllocationRevision(kind Type, state State, postings []Posting) Revision {
	at, _ := calendar.ParseInstant("2026-09-08T10:00:00Z")
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	date, _ := at.DateIn(zone)
	month, _ := calendar.ParseMonth(date.String()[:7])
	return Revision{OperationID: "operation", Revision: 1, ActorID: "actor", Reason: "synthetic", Type: kind, State: state, OccurredAt: at, PostedAt: at, Timezone: zone, CashDate: date, ExpenseMonth: month, Origin: "manual", FeeKnowledge: KnownFees, PayerState: "unknown", Postings: postings}
}

func mustAllocationMoney(t *testing.T, amount string, asset money.Asset) money.Money {
	t.Helper()
	value, err := money.NewMoney(amount, asset)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func allocationMoneyPtr(t *testing.T, amount string, asset money.Asset) *money.Money {
	value := mustAllocationMoney(t, amount, asset)
	return &value
}
