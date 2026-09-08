//go:build integration

package categories_test

import (
	"context"
	"io/fs"
	"net/url"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

func TestStarterCatalogHierarchyAndLocalizedRename(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	page := decode[generated.CategoryPage](t, client.call("GET", "/categories?limit=50", "", nil, 200))
	if len(page.Items) != 31 {
		t.Fatalf("starter catalog = %d", len(page.Items))
	}
	var food generated.Category
	for _, item := range page.Items {
		if item.StarterKey != nil && *item.StarterKey == "food" {
			food = item
		}
	}
	if food.Labels == nil || food.Labels.Ru != "Еда" || food.Labels.En != "Food" {
		t.Fatalf("localized starter missing: %+v", food)
	}
	english := decode[generated.CategoryPage](t, client.call("GET", "/categories?search=food", "", nil, 200))
	foundFood := false
	for _, item := range english.Items {
		foundFood = foundFood || item.Id == food.Id
	}
	if !foundFood {
		t.Fatalf("English starter search failed: %+v", english.Items)
	}
	conflict := decode[generated.CommandFailed](t, client.call("POST", "/categories", uuid.NewString(), map[string]any{"name": " Food "}, 202))
	if conflict.Error.Code != "invalid_request" {
		t.Fatalf("English starter name collision accepted: %+v", conflict)
	}
	result := decode[generated.CommandSucceeded](t, client.call("POST", "/categories/"+food.Id, uuid.NewString(), map[string]any{"expectedRevision": food.Revision, "nameAction": "set", "name": "Семейная еда"}, 202))
	if result.Status != "succeeded" {
		t.Fatal(result)
	}
	page = decode[generated.CategoryPage](t, client.call("GET", "/categories?search="+url.QueryEscape("  СЕМЕЙНАЯ   ЕДА "), "", nil, 200))
	if len(page.Items) != 1 || page.Items[0].Name != "Семейная еда" || page.Items[0].Labels.En != "Food" {
		t.Fatalf("rename or normalized search failed: %+v", page.Items)
	}
	customFood := createCategory(t, client, "Food", "")
	conflict = decode[generated.CommandFailed](t, client.call("POST", "/categories/"+food.Id, uuid.NewString(), map[string]any{"expectedRevision": 2, "nameAction": "restore_default"}, 202))
	if conflict.Error.Code != "invalid_request" {
		t.Fatalf("starter restored across English name collision: %+v", conflict)
	}
	archivedFood := decode[generated.CommandSucceeded](t, client.call("POST", "/categories/"+customFood, uuid.NewString(), map[string]any{"expectedRevision": 1, "state": "archived"}, 202))
	if archivedFood.Status != "succeeded" {
		t.Fatal(archivedFood)
	}
	restored := decode[generated.CommandSucceeded](t, client.call("POST", "/categories/"+food.Id, uuid.NewString(), map[string]any{"expectedRevision": 2, "nameAction": "restore_default"}, 202))
	if restored.Status != "succeeded" {
		t.Fatal(restored)
	}
	root := createCategory(t, client, "Custom root", "")
	child := createCategory(t, client, "Child", root)
	failed := decode[generated.CommandFailed](t, client.call("POST", "/categories", uuid.NewString(), map[string]any{"name": "Third level", "parentId": child}, 202))
	if failed.Error.Code != "invalid_request" {
		t.Fatalf("third level accepted: %+v", failed)
	}
	failed = decode[generated.CommandFailed](t, client.call("POST", "/categories/"+root, uuid.NewString(), map[string]any{"expectedRevision": 1, "state": "archived"}, 202))
	if failed.Error.Code != "category_has_active_children" {
		t.Fatalf("parent archive accepted: %+v", failed)
	}
}

func TestMerchantAliasesAndFamilyIsolation(t *testing.T) {
	f := newFixture(t)
	first := f.client(f.p)
	second := f.client(f.q)
	created := decode[generated.CommandSucceeded](t, first.call("POST", "/merchants", uuid.NewString(), map[string]any{"name": "Dizengoff", "aliases": []string{"DIZENGOFF TLV"}}, 202))
	if created.Status != "succeeded" {
		t.Fatal(created)
	}
	page := decode[generated.MerchantPage](t, second.call("GET", "/merchants?search=dizengoff", "", nil, 200))
	if len(page.Items) != 1 || len(page.Items[0].Aliases) != 2 {
		t.Fatalf("merchant missing: %+v", page.Items)
	}
	failed := decode[generated.CommandFailed](t, second.call("POST", "/merchants", uuid.NewString(), map[string]any{"name": "Other", "aliases": []string{" dizengoff   tlv "}}, 202))
	if failed.Error.Code != "merchant_alias_conflict" {
		t.Fatalf("alias collision accepted: %+v", failed)
	}
	archived := decode[generated.CommandSucceeded](t, first.call("POST", "/merchants/"+created.Result.Id, uuid.NewString(), map[string]any{"expectedRevision": 1, "state": "archived"}, 202))
	if archived.Status != "succeeded" {
		t.Fatal(archived)
	}
	replacement := decode[generated.CommandSucceeded](t, second.call("POST", "/merchants", uuid.NewString(), map[string]any{"name": "Replacement", "aliases": []string{"DIZENGOFF TLV"}}, 202))
	if replacement.Status != "succeeded" {
		t.Fatal(replacement)
	}
	failed = decode[generated.CommandFailed](t, first.call("POST", "/merchants/"+created.Result.Id, uuid.NewString(), map[string]any{"expectedRevision": 2, "state": "active"}, 202))
	if failed.Error.Code != "merchant_alias_conflict" {
		t.Fatalf("merchant restored across alias conflict: %+v", failed)
	}
}

func TestCategoryChangeRejectsNameWithoutAction(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	categoryID := createCategory(t, client, "Original name", "")

	client.call("POST", "/categories/"+categoryID, uuid.NewString(), map[string]any{"expectedRevision": 1, "name": "Ignored name", "state": "archived"}, 400)

	page := decode[generated.CategoryPage](t, client.call("GET", "/categories?search="+url.QueryEscape("Original name"), "", nil, 200))
	if len(page.Items) != 1 || page.Items[0].Id != categoryID || page.Items[0].Revision != 1 || page.Items[0].State != "active" {
		t.Fatalf("invalid category change mutated state: %+v", page.Items)
	}
	var commands int
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.command_tombstones`).Scan(&commands); err != nil || commands != 1 {
		t.Fatalf("commands after invalid category change = %d: %v", commands, err)
	}
}

func TestCategoryReparentRejectsArchivedDescendants(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	destinationID := createCategory(t, client, "Destination", "")
	rootID := createCategory(t, client, "Root with history", "")
	childID := createCategory(t, client, "Archived child", rootID)
	archived := decode[generated.CommandSucceeded](t, client.call("POST", "/categories/"+childID, uuid.NewString(), map[string]any{"expectedRevision": 1, "state": "archived"}, 202))
	if archived.Status != "succeeded" {
		t.Fatal(archived)
	}

	failed := decode[generated.CommandFailed](t, client.call("POST", "/categories/"+rootID, uuid.NewString(), map[string]any{"expectedRevision": 1, "parentAction": "set", "parentId": destinationID}, 202))
	if failed.Error.Code != "category_has_active_children" {
		t.Fatalf("root reparented over archived child: %+v", failed)
	}

	var revision, revisions int
	var parentID *string
	if err := f.admin.QueryRow(testContext, `SELECT revision,parent_id::text FROM want_keep.categories WHERE household_id=$1 AND id=$2`, f.family.ID, rootID).Scan(&revision, &parentID); err != nil {
		t.Fatal(err)
	}
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.category_revisions WHERE household_id=$1 AND id=$2`, f.family.ID, rootID).Scan(&revisions); err != nil {
		t.Fatal(err)
	}
	if revision != 1 || parentID != nil || revisions != 1 {
		t.Fatalf("failed reparent changed category: revision=%d parent=%v history=%d", revision, parentID, revisions)
	}
}

func TestCatalogHistoryOutlivesCommandRetention(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	createCategory(t, client, "Long-lived category", "")
	merchant := decode[generated.CommandSucceeded](t, client.call("POST", "/merchants", uuid.NewString(), map[string]any{"name": "Long-lived merchant", "aliases": []string{}}, 202))
	if merchant.Status != "succeeded" {
		t.Fatal(merchant)
	}

	u, err := url.Parse(f.admin.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("want_keep_maintenance", "synthetic-maintenance")
	maintenance, err := storage.Open(testContext, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer maintenance.Close()
	future := instant("2028-01-01T00:00:00Z")
	if _, err = maintenance.CleanupCommandDetails(testContext, future, 1000); err != nil {
		t.Fatal(err)
	}
	if _, err = maintenance.CleanupCommandTombstones(testContext, future, 1000); err != nil {
		t.Fatal(err)
	}

	for table, expected := range map[string]int{"command_tombstones": 0, "category_revisions": 32, "merchant_revisions": 1} {
		var actual int
		if err = f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.`+table).Scan(&actual); err != nil || actual != expected {
			t.Fatalf("%s after retention = %d, want %d: %v", table, actual, expected, err)
		}
	}
	for _, table := range []string{"category_revisions", "merchant_revisions"} {
		var retained int
		if err = f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.`+table+` WHERE command_id IS NOT NULL`).Scan(&retained); err != nil || retained != 1 {
			t.Fatalf("%s command audit references = %d: %v", table, retained, err)
		}
	}
}

func TestMigrationSeedsExistingHouseholdAndProtectsHistory(t *testing.T) {
	files := fstest.MapFS{}
	names, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if name >= "010_" {
			continue
		}
		data, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			t.Fatal(err)
		}
		files[name] = &fstest.MapFile{Data: data}
	}
	f := newFixtureUsingMigrations(t, files)
	if err = storage.Migrate(testContext, f.admin, migrations.Files); err != nil {
		t.Fatal(err)
	}
	var categories int
	if err = f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.categories WHERE household_id=$1`, f.family.ID).Scan(&categories); err != nil || categories != 31 {
		t.Fatalf("existing household catalog = %d: %v", categories, err)
	}
	for _, table := range []string{"category_revisions", "merchant_revisions", "merchant_alias_revisions", "receipt_items", "ledger_classification_proposals"} {
		var mutable bool
		if err = f.admin.QueryRow(testContext, `SELECT has_table_privilege('want_keep_app',$1,'UPDATE') OR has_table_privilege('want_keep_app',$1,'DELETE')`, "want_keep."+table).Scan(&mutable); err != nil || mutable {
			t.Fatalf("mutable history %s: %v", table, err)
		}
	}
	for table, columns := range map[string]struct {
		allowed   []string
		forbidden []string
	}{
		"categories":       {allowed: []string{"revision", "parent_id", "custom_name", "normalized_name", "state"}, forbidden: []string{"household_id", "id", "key", "name_ru", "name_en", "origin"}},
		"merchants":        {allowed: []string{"revision", "name", "normalized_name", "state"}, forbidden: []string{"household_id", "id"}},
		"merchant_aliases": {allowed: []string{"state", "merchant_active"}, forbidden: []string{"household_id", "merchant_id", "id", "name", "normalized_name", "origin"}},
	} {
		var tableUpdate bool
		if err = f.admin.QueryRow(testContext, `SELECT has_table_privilege('want_keep_app',$1,'UPDATE')`, "want_keep."+table).Scan(&tableUpdate); err != nil || tableUpdate {
			t.Fatalf("table-wide update privilege for %s: %v", table, err)
		}
		for _, column := range columns.allowed {
			var allowed bool
			if err = f.admin.QueryRow(testContext, `SELECT has_column_privilege('want_keep_app',$1,$2,'UPDATE')`, "want_keep."+table, column).Scan(&allowed); err != nil || !allowed {
				t.Fatalf("missing update privilege for %s.%s: %v", table, column, err)
			}
		}
		for _, column := range columns.forbidden {
			var allowed bool
			if err = f.admin.QueryRow(testContext, `SELECT has_column_privilege('want_keep_app',$1,$2,'UPDATE')`, "want_keep."+table, column).Scan(&allowed); err != nil || allowed {
				t.Fatalf("excessive update privilege for %s.%s: %v", table, column, err)
			}
		}
	}
	var claimsMutable bool
	if err = f.admin.QueryRow(testContext, `SELECT has_table_privilege('want_keep_app','want_keep.category_name_claims','INSERT,UPDATE,DELETE')`).Scan(&claimsMutable); err != nil || claimsMutable {
		t.Fatalf("category name claims are directly mutable: %v", err)
	}
}

func TestReceiptItemsCorrectionUndoAndFilters(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	accountID := f.account(money.RUB, "5000")
	categoryID := createCategory(t, client, "Groceries", "")
	merchant := decode[generated.CommandSucceeded](t, client.call("POST", "/merchants", uuid.NewString(), map[string]any{"name": "Market", "aliases": []string{}}, 202))
	created := decode[generated.CommandSucceeded](t, client.call("POST", "/transactions", uuid.NewString(), map[string]any{
		"type": "expense", "accountId": accountID, "amount": map[string]any{"amount": "900", "asset": "RUB"}, "occurredAt": "2026-09-08T10:00:00Z", "categoryId": categoryID, "merchantId": merchant.Result.Id,
		"allocation": map[string]any{"mode": "unresolved", "reason": "Synthetic allocation"}, "payer": map[string]any{"state": "known", "memberId": f.members[0].ID},
	}, 202))
	transaction := decode[generated.Transaction](t, client.call("GET", "/transactions/"+created.Result.Id, "", nil, 200))
	corrected := decode[generated.CommandSucceeded](t, client.call("POST", "/transactions/"+transaction.Id+"/corrections", uuid.NewString(), map[string]any{
		"expectedRevision": transaction.Revision, "reason": "Split receipt", "category": map[string]any{"action": "clear"},
		"receiptItems": map[string]any{"action": "replace", "totalDiscount": map[string]any{"amount": "100", "asset": "RUB"}, "items": []any{
			map[string]any{"id": uuid.NewString(), "name": "Milk", "quantity": "1", "gross": map[string]any{"amount": "600", "asset": "RUB"}, "categoryId": categoryID},
			map[string]any{"id": uuid.NewString(), "name": "Bread", "quantity": "2", "gross": map[string]any{"amount": "400", "asset": "RUB"}, "categoryId": categoryID},
		}},
	}, 202))
	if corrected.Status != "succeeded" {
		t.Fatal(corrected)
	}
	transaction = decode[generated.Transaction](t, client.call("GET", "/transactions/"+transaction.Id, "", nil, 200))
	if transaction.CategoryId != nil || len(transaction.ReceiptItems) != 2 || transaction.ReceiptItems[0].Discount.Amount != "60" || transaction.ReceiptItems[1].Discount.Amount != "40" {
		t.Fatalf("receipt classification invalid: %+v", transaction)
	}
	for _, query := range []string{"categoryId=" + categoryID, "merchantId=" + merchant.Result.Id, "itemSearch=milk"} {
		page := decode[generated.TransactionPage](t, client.call("GET", "/transactions?"+query, "", nil, 200))
		if len(page.Items) != 1 {
			t.Fatalf("filter %s returned %d", query, len(page.Items))
		}
	}
	undo := decode[generated.CommandSucceeded](t, client.call("POST", "/transactions/"+transaction.Id+"/undo", uuid.NewString(), map[string]any{"decisionId": *transaction.DecisionId, "reason": "Undo split", "expectedRevisions": []any{map[string]any{"transactionId": transaction.Id, "expectedRevision": transaction.Revision}}}, 202))
	if undo.Status != "succeeded" {
		t.Fatal(undo)
	}
	transaction = decode[generated.Transaction](t, client.call("GET", "/transactions/"+transaction.Id, "", nil, 200))
	if transaction.CategoryId == nil || *transaction.CategoryId != categoryID || len(transaction.ReceiptItems) != 0 {
		t.Fatalf("classification undo failed: %+v", transaction)
	}
	proposalCategory := createCategory(t, client, "AI proposal", "")
	err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		return f.ledgerService().CompleteReview(ctx, f.p, journal.ReviewInput{OperationID: transaction.Id, Revision: uint64(transaction.Revision), State: "reviewed", Rationale: "Synthetic classification suggestion", Proposal: &ledger.ClassificationProposal{CategoryID: proposalCategory, MerchantID: merchant.Result.Id, MerchantAlias: "Dizengoff Center"}})
	})
	if err != nil {
		t.Fatal(err)
	}
	transaction = decode[generated.Transaction](t, client.call("GET", "/transactions/"+transaction.Id, "", nil, 200))
	if transaction.CategoryId == nil || *transaction.CategoryId != categoryID || transaction.Review == nil || transaction.Review.ClassificationProposal == nil || *transaction.Review.ClassificationProposal.CategoryId != proposalCategory || transaction.Review.ClassificationProposal.MerchantAlias == nil || *transaction.Review.ClassificationProposal.MerchantAlias != "Dizengoff Center" {
		t.Fatalf("proposal was applied or not exposed: %+v", transaction)
	}
}

func createCategory(t *testing.T, client *client, name, parent string) string {
	t.Helper()
	input := map[string]any{"name": name}
	if parent != "" {
		input["parentId"] = parent
	}
	result := decode[generated.CommandSucceeded](t, client.call("POST", "/categories", uuid.NewString(), input, 202))
	if result.Status != "succeeded" {
		t.Fatal(result)
	}
	return result.Result.Id
}
