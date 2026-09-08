//go:build integration

package familyallocation_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	allocationapp "github.com/pchkauu/want-keep/backend/internal/allocation/application"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
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
	conflicted := createExpense(t, first, accountID, money.RUB, "100", merchant.Result.Id, unresolved())
	conflictedTransaction := readTransaction(t, second, conflicted.Result.Id)
	if conflictedTransaction.Allocation.State != "unresolved" || conflictedTransaction.Allocation.Reason != "rule_conflict" || len(conflictedTransaction.Allocation.Rules) != 2 {
		t.Fatalf("persisted rule conflict: %+v", conflictedTransaction.Allocation)
	}
	page := decode[generated.AllocationRulePage](t, first.call(http.MethodGet, "/allocation-rules?limit=1", "", nil, http.StatusOK))
	if len(page.Items) != 1 || page.NextCursor == nil {
		t.Fatalf("rule page: %+v", page)
	}
	second.call(http.MethodGet, "/allocation-rules?limit=1&cursor="+url.QueryEscape(*page.NextCursor), "", nil, http.StatusBadRequest)
}

func TestRuleCanBeArchivedAfterItsConditionIsArchived(t *testing.T) {
	for _, conditionKind := range []string{"merchant", "category"} {
		t.Run(conditionKind, func(t *testing.T) {
			fixture := newFixture(t)
			client := fixture.client(fixture.p)
			var condition map[string]any
			var archivePath string
			switch conditionKind {
			case "merchant":
				created := decode[generated.CommandSucceeded](t, client.call(http.MethodPost, "/merchants", uuid.NewString(), map[string]any{"name": "Archived merchant", "aliases": []string{}}, http.StatusAccepted))
				condition = map[string]any{"merchantId": created.Result.Id}
				archivePath = "/merchants/" + created.Result.Id
			case "category":
				created := decode[generated.CommandSucceeded](t, client.call(http.MethodPost, "/categories", uuid.NewString(), map[string]any{"name": "Archived category"}, http.StatusAccepted))
				condition = map[string]any{"categoryId": created.Result.Id}
				archivePath = "/categories/" + created.Result.Id
			}
			rule := decode[generated.CommandSucceeded](t, client.call(http.MethodPost, "/allocation-rules", uuid.NewString(), ruleInputForCondition(client, condition, "active", 10, "50", "50", 0), http.StatusAccepted))
			archivedCondition := decode[generated.CommandSucceeded](t, client.call(http.MethodPost, archivePath, uuid.NewString(), map[string]any{"expectedRevision": 1, "state": "archived"}, http.StatusAccepted))
			if archivedCondition.Status != "succeeded" {
				t.Fatal(archivedCondition)
			}
			archivedRule := decode[generated.CommandSucceeded](t, client.call(http.MethodPost, "/allocation-rules/"+rule.Result.Id, uuid.NewString(), ruleInputForCondition(client, condition, "archived", 10, "50", "50", 1), http.StatusAccepted))
			if archivedRule.Status != "succeeded" || archivedRule.Result.Revision != 2 {
				t.Fatalf("archived rule = %+v", archivedRule)
			}
			stored := decode[generated.AllocationRule](t, client.call(http.MethodGet, "/allocation-rules/"+rule.Result.Id, "", nil, http.StatusOK))
			if stored.State != "archived" {
				t.Fatalf("stored rule = %+v", stored)
			}
		})
	}
}

func TestNewImportedFactUsesConfirmedMerchantAliasRule(t *testing.T) {
	fixture := newFixture(t)
	first := fixture.client(fixture.p)
	merchant := decode[generated.CommandSucceeded](t, first.call(http.MethodPost, "/merchants", uuid.NewString(), map[string]any{"name": "Synthetic market", "aliases": []string{"Market statement alias"}}, http.StatusAccepted))
	createRule(t, first, merchant.Result.Id, 20, "60", "40")
	accountID := fixture.account(money.RUB, "1000")
	amount, _ := money.NewMoney("-100", money.RUB)
	revision := ledger.Revision{OperationID: uuid.NewString(), Revision: 1, Reason: "Imported expense", Type: ledger.Expense, State: ledger.Posted, OccurredAt: fixture.now, FeeKnowledge: ledger.KnownFees, Merchant: "  MARKET statement   alias ", PayerState: "unknown", Postings: []ledger.Posting{{AccountID: accountID, Money: amount, Role: ledger.Principal, Funding: ledger.OwnFunds, Treatment: ledger.Movement}}}
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	revision, err := revision.InTimezone(zone)
	if err != nil {
		t.Fatal(err)
	}
	untrustedAmount, _ := money.NewMoney("100", money.RUB)
	untrustedAllocation := ledger.AllocationInput{Mode: ledger.AllocationByAmounts, Purpose: ledger.AllocationPersonal, Members: []ledger.AllocationMemberInput{{MemberID: fixture.members[1].ID, Amount: &untrustedAmount}}}
	revision, err = revision.WithAllocation(untrustedAllocation, nil, []household.MembershipID{fixture.members[0].ID, fixture.members[1].ID})
	if err != nil {
		t.Fatal(err)
	}
	gate := fixture.admittedImport()
	connectionID := fixture.importConnection()
	issued := fixture.issuedImport(gate, connectionID)
	input := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: fixture.family.ID, Provider: "raiffeisen", ExternalAccountID: "synthetic", Product: "current", Log: "transactions", RecordID: uuid.NewString()}, PayloadHash: strings.Repeat("a", 64), EvidenceRef: "synthetic-evidence", ConnectionID: connectionID, JobID: issued.ID, FetchedAt: fixture.now, Classification: "new", Operation: &revision}
	sources := journal.NewSources(fixture.store, journal.NewWriter(fixture.store, fixture.store), allocationapp.NewService(fixture.store, uuid.NewString))
	applied, err := gate.CommitPage(testContext, fixture.p, issued, admission.Page{EvidenceRef: input.EvidenceRef, Coverage: "complete", Complete: true}, func(ctx context.Context) error {
		_, err := sources.Apply(ctx, fixture.p, input)
		return err
	})
	if err != nil || !applied {
		t.Fatal(err)
	}
	stored, found, err := fixture.store.CurrentLedgerRevision(testContext, fixture.p, revision.OperationID)
	if err != nil || !found || stored.Allocation.Origin != ledger.AllocationRule {
		t.Fatalf("source rule allocation: found=%v allocation=%+v err=%v", found, stored.Allocation, err)
	}
	if storedMemberTotal(stored.Allocation, fixture.members[0].ID, money.RUB) != "60" || storedMemberTotal(stored.Allocation, fixture.members[1].ID, money.RUB) != "40" {
		t.Fatalf("source-supplied allocation bypassed trusted rule = %+v", stored.Allocation.Members)
	}

	createRule(t, first, merchant.Result.Id, 20, "40", "60")
	conflictedRevision := revision.Clone()
	conflictedRevision.OperationID = uuid.NewString()
	conflictedInput := input
	conflictedInput.Key.RecordID = uuid.NewString()
	conflictedInput.Operation = &conflictedRevision
	conflictedIssued := fixture.issuedImport(gate, connectionID)
	conflictedInput.JobID = conflictedIssued.ID
	conflictedInput.PayloadHash = strings.Repeat("b", 64)
	applied, err = gate.CommitPage(testContext, fixture.p, conflictedIssued, admission.Page{EvidenceRef: conflictedInput.EvidenceRef, Coverage: "complete", Complete: true}, func(ctx context.Context) error {
		_, applyErr := sources.Apply(ctx, fixture.p, conflictedInput)
		return applyErr
	})
	if err != nil || !applied {
		t.Fatal(err)
	}
	stored, found, err = fixture.store.CurrentLedgerRevision(testContext, fixture.p, conflictedRevision.OperationID)
	if err != nil || !found || stored.Allocation.State != ledger.AllocationUnresolved || stored.Allocation.Reason != "rule_conflict" || len(stored.Allocation.RuleRefs) != 2 {
		t.Fatalf("source rule conflict: found=%v allocation=%+v err=%v", found, stored.Allocation, err)
	}
}

func TestSourceRulesUseExpenseComponentsAndDoNotBlockIncome(t *testing.T) {
	fixture := newFixture(t)
	client := fixture.client(fixture.p)
	merchant := decode[generated.CommandSucceeded](t, client.call(http.MethodPost, "/merchants", uuid.NewString(), map[string]any{"name": "Income and transfer merchant", "aliases": []string{"Matched source merchant"}}, http.StatusAccepted))
	createRule(t, client, merchant.Result.Id, 20, "60", "40")
	from, to := fixture.account(money.RUB, "1000"), fixture.account(money.RUB, "1000")
	gate := fixture.admittedImport()
	connectionID := fixture.importConnection()
	sources := journal.NewSources(fixture.store, journal.NewWriter(fixture.store, fixture.store), allocationapp.NewService(fixture.store, uuid.NewString))
	zone, _ := calendar.ParseTimezone("Europe/Moscow")

	apply := func(revision ledger.Revision, hash, evidence string) ledger.Revision {
		t.Helper()
		var err error
		revision, err = revision.InTimezone(zone)
		if err != nil {
			t.Fatal(err)
		}
		issued := fixture.issuedImport(gate, connectionID)
		input := ledger.SourceInput{Key: ledger.SourceKey{HouseholdID: fixture.family.ID, Provider: "raiffeisen", ExternalAccountID: "synthetic", Product: "current", Log: "transactions", RecordID: uuid.NewString()}, PayloadHash: strings.Repeat(hash, 64), EvidenceRef: evidence, ConnectionID: connectionID, JobID: issued.ID, FetchedAt: fixture.now, Classification: "new", Operation: &revision}
		applied, commitErr := gate.CommitPage(testContext, fixture.p, issued, admission.Page{EvidenceRef: evidence, Coverage: "complete", Complete: true}, func(ctx context.Context) error {
			_, applyErr := sources.Apply(ctx, fixture.p, input)
			return applyErr
		})
		if commitErr != nil || !applied {
			t.Fatalf("source page applied=%v err=%v", applied, commitErr)
		}
		stored, found, readErr := fixture.store.CurrentLedgerRevision(testContext, fixture.p, revision.OperationID)
		if readErr != nil || !found {
			t.Fatalf("stored source operation found=%v err=%v", found, readErr)
		}
		return stored
	}

	incomeAmount, _ := money.NewMoney("100", money.RUB)
	income := ledger.Revision{OperationID: uuid.NewString(), Revision: 1, Reason: "Imported income", Type: ledger.Income, State: ledger.Posted, OccurredAt: fixture.now, FeeKnowledge: ledger.KnownFees, Merchant: "Matched source merchant", PayerState: "not_applicable", Postings: []ledger.Posting{{AccountID: from, Money: incomeAmount, Role: ledger.Principal, Funding: ledger.OwnFunds, Treatment: ledger.Movement}}}
	storedIncome := apply(income, "c", "income-evidence")
	if storedIncome.Allocation.State != ledger.AllocationNotApplicable {
		t.Fatalf("income allocation = %+v", storedIncome.Allocation)
	}

	sent, _ := money.NewMoney("-100", money.RUB)
	received, _ := money.NewMoney("100", money.RUB)
	fee, _ := money.NewMoney("-10", money.RUB)
	transfer := ledger.Revision{OperationID: uuid.NewString(), Revision: 1, Reason: "Imported transfer", Type: ledger.Transfer, State: ledger.Posted, OccurredAt: fixture.now, FeeKnowledge: ledger.KnownFees, Merchant: "Matched source merchant", PayerState: "not_applicable", Postings: []ledger.Posting{{AccountID: from, Money: sent, Role: ledger.Principal, Funding: ledger.OwnFunds, Treatment: ledger.Movement}, {AccountID: to, Money: received, Role: ledger.Principal, Funding: ledger.OwnFunds, Treatment: ledger.Movement}, {AccountID: from, Money: fee, Role: ledger.Fee, Funding: ledger.OwnFunds, Treatment: ledger.Movement}}}
	storedTransfer := apply(transfer, "d", "transfer-evidence")
	if storedTransfer.Allocation.Origin != ledger.AllocationRule || storedMemberTotal(storedTransfer.Allocation, fixture.members[0].ID, money.RUB) != "6" || storedMemberTotal(storedTransfer.Allocation, fixture.members[1].ID, money.RUB) != "4" {
		t.Fatalf("transfer fee allocation = %+v", storedTransfer.Allocation)
	}
}

func TestAllocationConditionRepositoryFailureIsNotAStoredBusinessRejection(t *testing.T) {
	fixture := newFixture(t)
	first := fixture.client(fixture.p)
	merchant := decode[generated.CommandSucceeded](t, first.call(http.MethodPost, "/merchants", uuid.NewString(), map[string]any{"name": "Repository failure merchant", "aliases": []string{}}, http.StatusAccepted))
	err := fixture.store.WithinHousehold(testContext, fixture.p, func(transactionContext context.Context) error {
		ctx, cancel := context.WithCancel(transactionContext)
		cancel()
		_, previewErr := allocationapp.NewService(fixture.store, uuid.NewString).Preview(ctx, fixture.p, merchant.Result.Id, "")
		if !errors.Is(previewErr, context.Canceled) {
			t.Fatalf("repository failure classified as %v", previewErr)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func storedMemberTotal(snapshot ledger.AllocationSnapshot, memberID household.MembershipID, asset money.Asset) string {
	for _, member := range snapshot.Members {
		if member.MemberID == memberID && member.Money.Asset() == asset {
			return member.Money.Amount()
		}
	}
	return "0"
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
	stored, found, err := fixture.store.CurrentLedgerRevision(testContext, fixture.p, created.Result.Id)
	if err != nil || !found || stored.Allocation.Fallback == nil {
		t.Fatalf("composite fallback round trip: found=%v fallback=%+v err=%v", found, stored.Allocation.Fallback, err)
	}
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
	unresolvedResult := decode[generated.CommandSucceeded](t, second.call(http.MethodPost, "/transactions/"+created.Result.Id+"/allocations", uuid.NewString(), map[string]any{
		"expectedRevision": 3,
		"reason":           "Preserve item ambiguity",
		"allocation":       unresolved(),
		"items":            []any{map[string]any{"itemId": revision.ReceiptItems[0].Id, "allocation": map[string]any{"mode": "unresolved", "reason": "Ambiguous shared item"}}},
	}, http.StatusAccepted))
	if unresolvedResult.Status != "succeeded" {
		t.Fatal(unresolvedResult)
	}
	unresolvedTransaction := readTransaction(t, first, created.Result.Id)
	if unresolvedTransaction.Allocation.State != "unresolved" || unresolvedTransaction.ReceiptItems[0].Allocation.Reason != "Ambiguous shared item" {
		t.Fatalf("unresolved item allocation = %+v", unresolvedTransaction)
	}
	feeResult := decode[generated.CommandSucceeded](t, first.call(http.MethodPost, "/transactions/"+created.Result.Id+"/corrections", uuid.NewString(), map[string]any{
		"expectedRevision": 4,
		"reason":           "Add confirmed fee",
		"fees":             []any{map[string]any{"accountId": accountID, "role": "fee", "money": map[string]any{"asset": "RUB", "amount": "-10"}}},
	}, http.StatusAccepted))
	if feeResult.Status != "succeeded" {
		t.Fatal(feeResult)
	}
	refreshed := readTransaction(t, first, created.Result.Id)
	if refreshed.ReceiptItems[0].Allocation.Origin != "explicit_item" || refreshed.ReceiptItems[0].Allocation.Reason != "Ambiguous shared item" {
		t.Fatalf("correction changed explicit unresolved item = %+v", refreshed.ReceiptItems[0].Allocation)
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
	return ruleInputForCondition(client, map[string]any{"merchantId": merchantID}, "active", priority, first, second, expected)
}

func ruleInputForCondition(client *client, condition map[string]any, state string, priority int, first, second string, expected int64) map[string]any {
	input := map[string]any{"priority": priority, "state": state, "condition": condition, "shares": []any{map[string]any{"memberId": string(client.f.members[0].ID), "share": first}, map[string]any{"memberId": string(client.f.members[1].ID), "share": second}}}
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
