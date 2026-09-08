package contract_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	contract "github.com/pchkauu/want-keep/backend/internal/integrations/contract"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(file), "../../../../collector/contracts/v10/fixtures", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func token() ingestion.JobToken {
	digest := "sha256:" + strings.Repeat("0", 64)
	return ingestion.JobToken{JobID: "11111111-1111-4111-8111-111111111111", Attempt: 1, LeaseToken: "lease-token", ConnectionID: "22222222-2222-4222-8222-222222222222", ConnectionGeneration: 1, Binding: connections.Binding{Provider: "raiffeisen", Environment: "test", AdapterBuildDigest: digest, CollectorImageDigest: digest, ContractVersion: "10", AllowlistRevision: "allowlist-1", NonSecretConfigRevision: "config-1", OperatorPermissionRevision: "permission-1"}, AdmissionRevision: 1}
}

func goldenToken() ingestion.JobToken {
	t := token()
	t.LeaseToken = "lease-1"
	t.Binding.Provider = "bybit"
	t.Binding.CollectorImageDigest = "sha256:" + strings.Repeat("1", 64)
	t.AdmissionRevision = 3
	return t
}

func TestGoldenContractPreservesMeaningAndPrecision(t *testing.T) {
	manifest, err := contract.DecodeManifest(fixture(t, "manifest.json"), "bybit")
	if err != nil || manifest.Version != "10" || len(manifest.Logs) != 3 {
		t.Fatal(manifest, err)
	}
	result, err := contract.DecodeResult(fixture(t, "golden-page.json"), goldenToken())
	if err != nil {
		t.Fatal(err)
	}
	if result.Page == nil || len(result.Page.Records) != 17 || result.Page.Coverage.State() != "partial" {
		t.Fatal("golden page lost records or coverage")
	}
	amounts := map[string]string{}
	for _, record := range result.Page.Records {
		if record.Balance != nil {
			amounts[record.Balance.Reference.AssetCode] = record.Balance.Owned.Value
		}
	}
	for asset, expected := range map[string]string{"USDC": "0.123456789123456789", "BTC": "0.000000000000000001", "ETH": "1.000000000000000123", "USDC.E": "2.5"} {
		if amounts[asset] != expected {
			t.Fatalf("%s changed: %q", asset, amounts[asset])
		}
	}
	if _, err = contract.EncodeSyncRequest(goldenToken()); err != nil {
		t.Fatal(err)
	}
}

func TestDecoderRejectsUnsafeShapesAndEchoChanges(t *testing.T) {
	original := fixture(t, "golden-page.json")
	var value map[string]any
	if err := json.Unmarshal(original, &value); err != nil {
		t.Fatal(err)
	}
	page := value["page"].(map[string]any)
	page["householdId"] = "forged"
	unknown, _ := json.Marshal(value)
	if _, err := contract.DecodeResult(unknown, goldenToken()); err == nil {
		t.Fatal("unknown ownership field accepted")
	}
	if _, err := contract.DecodeResult(append(original, []byte(" {}")...), goldenToken()); err == nil {
		t.Fatal("trailing JSON accepted")
	}

	for name, mutate := range map[string]func(map[string]any){
		"numeric money": func(root map[string]any) {
			records := root["page"].(map[string]any)["records"].([]any)
			records[1].(map[string]any)["balanceSnapshot"].(map[string]any)["owned"].(map[string]any)["amount"] = 1
		},
		"invalid base64": func(root map[string]any) {
			root["page"].(map[string]any)["evidence"].([]any)[0].(map[string]any)["data"] = "%%%="
		},
		"wrong discriminator": func(root map[string]any) {
			record := root["page"].(map[string]any)["records"].([]any)[0].(map[string]any)
			record["recordType"] = "transaction"
		},
		"stale lease": func(root map[string]any) {
			root["page"].(map[string]any)["leaseToken"] = "other"
		},
		"invalid asset syntax": func(root map[string]any) {
			records := root["page"].(map[string]any)["records"].([]any)
			records[0].(map[string]any)["account"].(map[string]any)["assetCode"] = "USD C"
		},
		"invalid calendar date": func(root map[string]any) {
			records := root["page"].(map[string]any)["records"].([]any)
			records[0].(map[string]any)["account"].(map[string]any)["openingDate"] = "2026-02-30"
		},
		"missing account descriptor": func(root map[string]any) {
			page := root["page"].(map[string]any)
			records := page["records"].([]any)
			page["records"] = records[1:]
		},
	} {
		t.Run(name, func(t *testing.T) {
			var root map[string]any
			_ = json.Unmarshal(original, &root)
			mutate(root)
			data, _ := json.Marshal(root)
			if _, err := contract.DecodeResult(data, goldenToken()); err == nil {
				t.Fatal("invalid contract accepted")
			}
		})
	}
}
