package contract_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func TestSharedMoneyFixtures(t *testing.T) {
	boundary, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test location unavailable")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "../../../../../api/fixtures/money.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []json.RawMessage
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	converter := contract.MoneyConverter{}
	for _, fixture := range fixtures {
		var dto generated.Money
		if err := boundary.Decode("Money", fixture, &dto); err != nil {
			t.Fatal(err)
		}
		domain, err := converter.FromDTO(dto)
		if err != nil {
			t.Fatal(err)
		}
		out, err := converter.ToDTO(domain)
		if err != nil || dto != out {
			t.Fatalf("roundtrip: %v %v %v", dto, out, err)
		}
		encoded, err := json.Marshal(out)
		if err != nil {
			t.Fatal(err)
		}
		var restored generated.Money
		if err := boundary.Decode("Money", encoded, &restored); err != nil || restored != dto {
			t.Fatal("JSON changed exact amount")
		}
	}
}

func TestSchemaRejectsInvalidShapesAndCommandStates(t *testing.T) {
	boundary, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, data string }{
		{"Money", `{"amount":0.1,"asset":"USD"}`},
		{"Money", `{"amount":null,"asset":"USD"}`},
		{"Money", `{"amount":"1e8","asset":"USD"}`},
		{"Money", `{"amount":"1","asset":"USDC.E"}`},
		{"Money", `{"amount":"1","asset":"USD","actorId":"forged"}`},
		{"Money", `{"amount":"1","asset":"USD"} {}`},
		{"PositiveMoney", `{"amount":"0.00","asset":"USD"}`},
		{"AmountValue", `{"knowledge":"known"}`},
		{"AmountValue", `{"knowledge":"unknown","value":{"amount":"0","asset":"RUB"},"reason":"gap"}`},
		{"AmountValue", `{"knowledge":"unknown"}`},
		{"Coverage", `{"state":"complete","reasons":["gap"]}`},
		{"Coverage", `{"state":"partial","reasons":[]}`},
		{"CommandStatus", `{"id":"10000000-0000-4000-8000-000000000001","type":"transactions.create","status":"unknown","registeredAt":"2026-09-07T00:00:00Z"}`},
		{"CommandStatus", `{"id":"10000000-0000-4000-8000-000000000001","type":"transactions.create","status":"succeeded","registeredAt":"2026-09-07T00:00:00Z"}`},
		{"Revision", `9007199254740992`},
		{"Ownership", `{"householdId":"10000000-0000-4000-8000-000000000001","scope":"household","personalOwnerId":"10000000-0000-4000-8000-000000000002"}`},
		{"TransactionCorrection", `{"expectedRevision":1,"reason":"none"}`},
		{"MessageCreate", `{"text":"","attachmentIds":[]}`},
		{"MessageCreate", `{"text":"receipt","attachmentIds":["10000000-0000-4000-8000-000000000001"]}`},
	} {
		t.Run(tc.name+tc.data, func(t *testing.T) {
			var result json.RawMessage
			if err := boundary.Decode(tc.name, []byte(tc.data), &result); err == nil {
				t.Fatal("invalid shape accepted")
			}
		})
	}
}

func TestAvailabilityAndOwnershipMapping(t *testing.T) {
	b, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	zero, _ := money.NewMoney("0", money.USD)
	known, _ := reporting.KnownAmount(zero)
	missing, _ := reporting.MissingAmount(reporting.Unknown, "missing_balance")
	for _, value := range []reporting.Amount{known, missing} {
		dto, err := b.AmountToDTO(value)
		if err != nil {
			t.Fatal(err)
		}
		out, err := b.AmountFromDTO(dto)
		if err != nil || out.Knowledge() != value.Knowledge() || out.Reason() != value.Reason() {
			t.Fatal("availability lost")
		}
	}
	for _, state := range []reporting.CoverageState{reporting.Complete, reporting.Partial, reporting.NoCoverage} {
		reasons := []string{}
		if state != reporting.Complete {
			reasons = []string{"history_gap"}
		}
		coverage, _ := reporting.NewCoverage(state, reasons)
		dto, err := b.CoverageToDTO(coverage)
		if err != nil {
			t.Fatal(err)
		}
		out, err := b.CoverageFromDTO(dto)
		if err != nil || out.State() != state {
			t.Fatal("coverage lost")
		}
	}
	for _, scope := range []household.Scope{household.Personal, household.Shared} {
		owner := household.UserID("")
		if scope == household.Personal {
			owner = "20000000-0000-4000-8000-000000000001"
		}
		ownership, _ := household.NewOwnership("10000000-0000-4000-8000-000000000001", scope, owner)
		dto, err := b.OwnershipToDTO(ownership)
		if err != nil {
			t.Fatal(err)
		}
		out, err := b.OwnershipFromDTO(dto)
		if err != nil || out != ownership {
			t.Fatal("ownership fields conflated")
		}
	}
}

func TestContractHasCompleteGroupsAndGuards(t *testing.T) {
	schema, err := generated.GetSwagger()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/accounts", "/transactions", "/commands/recent", "/commands/{commandId}", "/reports/dashboard", "/goals/preview", "/household", "/connections/raiffeisen/callback", "/auth/login/verify", "/threads/{threadId}/messages"} {
		if schema.Paths.Find(path) == nil {
			t.Errorf("missing path %s", path)
		}
	}
	for path, item := range schema.Paths.Map() {
		for method, operation := range item.Operations() {
			if method == "GET" {
				if _, ok := operation.Extensions["x-command-type"]; ok {
					t.Errorf("mutating GET %s", path)
				}
				continue
			}
			if _, ok := operation.Extensions["x-command-type"]; !ok {
				continue
			}
			headers := map[string]bool{}
			for _, parameter := range operation.Parameters {
				if parameter.Value.In == "header" && parameter.Value.Required {
					headers[parameter.Value.Name] = true
				}
			}
			if !headers["Idempotency-Key"] || !headers["X-CSRF-Token"] {
				t.Errorf("missing command guard %s", path)
			}
			if operation.Security != nil && len(*operation.Security) == 0 {
				t.Errorf("public financial command %s", path)
			}
		}
	}
}

func TestErrorsAreSafeAndVersioned(t *testing.T) {
	converter := contract.ErrorConverter{}
	privateCause := errors.New("private provider token=synthetic-test-only")
	response := converter.ToResponse(privateCause, "10000000-0000-4000-8000-000000000001")
	data, _ := json.Marshal(response.Body)
	if strings.Contains(string(data), "token=") || response.Body.Version != "1" || response.Body.Retryable || response.Status != 500 {
		t.Fatal("unsafe unknown failure")
	}
	response = converter.ToResponse(money.ErrUnsupportedAsset, "10000000-0000-4000-8000-000000000001")
	if response.Status != 422 || response.Body.Code != "unsupported_asset" {
		t.Fatal("typed error lost")
	}
	boundary, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	var dto generated.Money
	err = boundary.Decode("Money", []byte(`{"amount":4,"asset":"RUB"}`), &dto)
	response = converter.ToResponse(err, "10000000-0000-4000-8000-000000000001")
	if len(response.Body.Violations) != 1 || response.Body.Violations[0].Field != "/amount" {
		t.Fatal("field violation lost")
	}
}
