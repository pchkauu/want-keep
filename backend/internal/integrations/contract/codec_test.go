package contract_test

import (
	"context"
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
	if err != nil || manifest.Version != "10" || len(manifest.Logs) != 5 {
		t.Fatal(manifest, err)
	}
	result, err := contract.DecodeResult(fixture(t, "golden-page.json"), goldenToken())
	if err != nil {
		t.Fatal(err)
	}
	if result.Page == nil || len(result.Page.Records) != 17 || result.Page.Coverage.State() != "partial" {
		t.Fatal("golden page lost records or coverage")
	}
	if err = manifest.RequirePage(*result.Page); err != nil {
		t.Fatal("golden page exceeded its manifest", err)
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

func TestGatewayCarriesTrustedDeploymentBinding(t *testing.T) {
	binding := goldenToken().Binding
	raw := &rawGateway{manifest: fixture(t, "manifest.json"), result: fixture(t, "golden-page.json")}
	gateway, err := contract.NewGateway(binding, raw)
	if err != nil || gateway.Binding() != binding {
		t.Fatal("gateway lost its trusted binding", err)
	}
	changed := goldenToken()
	changed.Binding.AllowlistRevision = "allowlist-2"
	if _, err = gateway.Read(context.Background(), changed); err == nil || raw.reads != 0 {
		t.Fatal("misbound request reached the raw gateway", err, raw.reads)
	}
}

func TestManifestDistinguishesOmittedLookbackFromExplicitZero(t *testing.T) {
	var value map[string]any
	if err := json.Unmarshal(fixture(t, "manifest.json"), &value); err != nil {
		t.Fatal(err)
	}
	history := value["history"].(map[string]any)
	history["maximumLookbackDays"] = 0
	invalid, _ := json.Marshal(value)
	if _, err := contract.DecodeManifest(invalid, "bybit"); err == nil {
		t.Fatal("explicit zero lookback was accepted")
	}
	history["maximumLookbackDays"] = nil
	nullLimit, _ := json.Marshal(value)
	if _, err := contract.DecodeManifest(nullLimit, "bybit"); err == nil {
		t.Fatal("explicit null lookback was accepted")
	}
	delete(history, "maximumLookbackDays")
	withoutLimit, _ := json.Marshal(value)
	manifest, err := contract.DecodeManifest(withoutLimit, "bybit")
	if err != nil || manifest.MaximumLookbackDays != nil {
		t.Fatal("omitted lookback did not remain unlimited", err)
	}
}

type rawGateway struct {
	manifest, result []byte
	reads            int
}

func (g *rawGateway) CapabilityManifest(context.Context) ([]byte, error) {
	return g.manifest, nil
}

func (g *rawGateway) Read(context.Context, []byte) ([]byte, error) {
	g.reads++
	return g.result, nil
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
		"line-broken base64": func(root map[string]any) {
			evidence := root["page"].(map[string]any)["evidence"].([]any)[0].(map[string]any)
			value := evidence["data"].(string)
			evidence["data"] = value[:4] + "\n" + value[4:]
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
		"zero calendar year": func(root map[string]any) {
			records := root["page"].(map[string]any)["records"].([]any)
			records[0].(map[string]any)["account"].(map[string]any)["openingDate"] = "0000-01-01"
		},
		"non-canonical UTC offset": func(root map[string]any) {
			records := root["page"].(map[string]any)["records"].([]any)
			records[1].(map[string]any)["balanceSnapshot"].(map[string]any)["sourceAsOf"] = "2026-09-08T09:00:00+00:00"
		},
		"zero instant year": func(root map[string]any) {
			records := root["page"].(map[string]any)["records"].([]any)
			records[1].(map[string]any)["balanceSnapshot"].(map[string]any)["sourceAsOf"] = "0000-01-01T00:00:00Z"
		},
		"card label with separated PAN": func(root map[string]any) {
			records := root["page"].(map[string]any)["records"].([]any)
			aliases := records[0].(map[string]any)["account"].(map[string]any)["aliases"].([]any)
			aliases[0].(map[string]any)["label"] = "4242.4242.4242.4242"
		},
		"card label with Unicode digits": func(root map[string]any) {
			records := root["page"].(map[string]any)["records"].([]any)
			aliases := records[0].(map[string]any)["account"].(map[string]any)["aliases"].([]any)
			aliases[0].(map[string]any)["label"] = "٤٢٤٢"
		},
		"missing account descriptor": func(root map[string]any) {
			page := root["page"].(map[string]any)
			records := page["records"].([]any)
			page["records"] = records[1:]
		},
		"duplicate coverage gap": func(root map[string]any) {
			root["page"].(map[string]any)["coverage"].(map[string]any)["gaps"] = []any{"gap", "gap"}
		},
		"too many coverage gaps": func(root map[string]any) {
			gaps := make([]any, 101)
			for index := range gaps {
				gaps[index] = "gap-" + strings.Repeat("x", index+1)
			}
			root["page"].(map[string]any)["coverage"].(map[string]any)["gaps"] = gaps
		},
		"coverage gap with NUL": func(root map[string]any) {
			root["page"].(map[string]any)["coverage"].(map[string]any)["gaps"] = []any{"invalid\x00gap"}
		},
		"coverage gap over Unicode limit": func(root map[string]any) {
			root["page"].(map[string]any)["coverage"].(map[string]any)["gaps"] = []any{strings.Repeat("ё", 2001)}
		},
		"unsupported fee identifier": func(root map[string]any) {
			records := root["page"].(map[string]any)["records"].([]any)
			for _, item := range records {
				transaction, ok := item.(map[string]any)["transaction"].(map[string]any)
				if ok && len(transaction["postings"].([]any)) > 0 {
					transaction["postings"].([]any)[0].(map[string]any)["feeId"] = "fee-1"
					return
				}
			}
		},
		"known amount with forbidden reason": func(root map[string]any) {
			records := root["page"].(map[string]any)["records"].([]any)
			records[1].(map[string]any)["balanceSnapshot"].(map[string]any)["owned"].(map[string]any)["reason"] = ""
		},
		"unknown amount with forbidden value": func(root map[string]any) {
			records := root["page"].(map[string]any)["records"].([]any)
			for _, raw := range records {
				balance, ok := raw.(map[string]any)["balanceSnapshot"].(map[string]any)
				if ok && balance["available"].(map[string]any)["state"] == "unknown" {
					balance["available"].(map[string]any)["amount"] = ""
					return
				}
			}
		},
		"empty optional enum": func(root map[string]any) {
			firstTransaction(root)["pnlBasis"] = ""
		},
		"empty posting enum": func(root map[string]any) {
			firstTransaction(root)["postings"].([]any)[0].(map[string]any)["funding"] = ""
		},
		"empty posting treatment": func(root map[string]any) {
			firstTransaction(root)["postings"].([]any)[0].(map[string]any)["treatment"] = ""
		},
		"null required cursor": func(root map[string]any) {
			root["page"].(map[string]any)["cursor"] = nil
		},
		"null required boolean": func(root map[string]any) {
			root["page"].(map[string]any)["complete"] = nil
		},
		"null required object": func(root map[string]any) {
			root["page"].(map[string]any)["coverage"] = nil
		},
		"null required array": func(root map[string]any) {
			root["page"].(map[string]any)["records"] = nil
		},
		"null optional field": func(root map[string]any) {
			root["page"].(map[string]any)["nextCursor"] = nil
		},
		"empty next cursor": func(root map[string]any) {
			root["page"].(map[string]any)["nextCursor"] = ""
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

	invalidSurrogate := strings.Replace(string(original), `"name": "Synthetic RUB"`, `"name": "\ud800"`, 1)
	if _, err := contract.DecodeResult([]byte(invalidSurrogate), goldenToken()); err == nil {
		t.Fatal("unpaired JSON surrogate was accepted")
	}
}

func TestDecoderRejectsMissingRequiredFields(t *testing.T) {
	original := fixture(t, "golden-page.json")
	for name, mutate := range map[string]func(map[string]any){
		"result outcome": func(root map[string]any) { delete(root, "outcome") },
		"page records":   func(root map[string]any) { delete(root["page"].(map[string]any), "records") },
		"page complete":  func(root map[string]any) { delete(root["page"].(map[string]any), "complete") },
		"binding provider": func(root map[string]any) {
			delete(root["page"].(map[string]any)["binding"].(map[string]any), "provider")
		},
		"coverage gaps": func(root map[string]any) {
			delete(root["page"].(map[string]any)["coverage"].(map[string]any), "gaps")
		},
		"evidence locator": func(root map[string]any) {
			delete(root["page"].(map[string]any)["evidence"].([]any)[0].(map[string]any), "locator")
		},
		"account name": func(root map[string]any) {
			delete(root["page"].(map[string]any)["records"].([]any)[0].(map[string]any)["account"].(map[string]any), "name")
		},
		"card alias label": func(root map[string]any) {
			account := root["page"].(map[string]any)["records"].([]any)[0].(map[string]any)["account"].(map[string]any)
			delete(account["aliases"].([]any)[0].(map[string]any), "label")
		},
		"balance ownAvailable": func(root map[string]any) {
			delete(root["page"].(map[string]any)["records"].([]any)[1].(map[string]any)["balanceSnapshot"].(map[string]any), "ownAvailable")
		},
		"known amount": func(root map[string]any) {
			balance := root["page"].(map[string]any)["records"].([]any)[1].(map[string]any)["balanceSnapshot"].(map[string]any)
			delete(balance["owned"].(map[string]any), "amount")
		},
		"transaction fee knowledge": func(root map[string]any) {
			delete(firstTransaction(root), "feeKnowledge")
		},
		"posting role": func(root map[string]any) {
			delete(firstTransaction(root)["postings"].([]any)[0].(map[string]any), "role")
		},
	} {
		t.Run(name, func(t *testing.T) {
			var root map[string]any
			_ = json.Unmarshal(original, &root)
			mutate(root)
			data, _ := json.Marshal(root)
			if _, err := contract.DecodeResult(data, goldenToken()); err == nil {
				t.Fatal("missing required field was accepted")
			}
		})
	}

	manifestRaw := fixture(t, "manifest.json")
	for name, mutate := range map[string]func(map[string]any){
		"manifest actions":  func(root map[string]any) { delete(root, "actions") },
		"history paginated": func(root map[string]any) { delete(root["history"].(map[string]any), "paginated") },
		"log record kinds":  func(root map[string]any) { delete(root["logs"].([]any)[0].(map[string]any), "recordKinds") },
		"blank log namespace": func(root map[string]any) {
			root["logs"].([]any)[0].(map[string]any)["namespace"] = " \t"
		},
	} {
		t.Run(name, func(t *testing.T) {
			var root map[string]any
			_ = json.Unmarshal(manifestRaw, &root)
			mutate(root)
			data, _ := json.Marshal(root)
			if _, err := contract.DecodeManifest(data, "bybit"); err == nil {
				t.Fatal("missing manifest field was accepted")
			}
		})
	}
}

func TestCanonicalRecordIgnoresPageLocalEvidenceIdentity(t *testing.T) {
	original, err := contract.DecodeResult(fixture(t, "golden-page.json"), goldenToken())
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	_ = json.Unmarshal(fixture(t, "golden-page.json"), &root)
	page := root["page"].(map[string]any)
	page["evidence"].([]any)[0].(map[string]any)["id"] = "replayed-evidence"
	page["evidence"].([]any)[0].(map[string]any)["locator"] = "synthetic:replayed"
	for _, raw := range page["records"].([]any) {
		record := raw.(map[string]any)
		for _, field := range []string{"account", "balanceSnapshot", "transaction"} {
			if payload, ok := record[field].(map[string]any); ok {
				payload["evidenceId"] = "replayed-evidence"
				if payload["network"] == "" {
					delete(payload, "network")
				}
				if field == "account" {
					if _, present := payload["aliases"]; !present {
						payload["aliases"] = []any{}
					}
				}
				if field == "transaction" {
					for _, optional := range []string{"merchant", "note"} {
						if _, present := payload[optional]; !present {
							payload[optional] = ""
						}
					}
					for _, posting := range payload["postings"].([]any) {
						value := posting.(map[string]any)
						if value["network"] == "" {
							delete(value, "network")
						}
					}
				}
			}
		}
	}
	encoded, _ := json.Marshal(root)
	replayed, err := contract.DecodeResult(encoded, goldenToken())
	if err != nil {
		t.Fatal(err)
	}
	for index := range original.Page.Records {
		if string(original.Page.Records[index].CanonicalPayload) != string(replayed.Page.Records[index].CanonicalPayload) {
			t.Fatalf("record %d canonical payload depends on evidence identity", index)
		}
	}
}

func firstTransaction(root map[string]any) map[string]any {
	for _, raw := range root["page"].(map[string]any)["records"].([]any) {
		if transaction, ok := raw.(map[string]any)["transaction"].(map[string]any); ok {
			return transaction
		}
	}
	panic("fixture has no transaction")
}

func TestProviderFailureRequiresExactCursorEcho(t *testing.T) {
	var golden map[string]any
	if err := json.Unmarshal(fixture(t, "golden-page.json"), &golden); err != nil {
		t.Fatal(err)
	}
	page := golden["page"].(map[string]any)
	failure := map[string]any{
		"outcome": "failure",
		"failure": map[string]any{
			"jobId": page["jobId"], "attempt": page["attempt"], "leaseToken": page["leaseToken"],
			"connectionGeneration": page["connectionGeneration"], "binding": page["binding"],
			"admissionRevision": page["admissionRevision"], "cursor": "", "kind": "mfa_required",
			"retryable": false, "evidence": page["evidence"],
		},
	}
	encoded, _ := json.Marshal(failure)
	if _, err := contract.DecodeResult(encoded, goldenToken()); err != nil {
		t.Fatal("valid provider failure was rejected", err)
	}
	payload := failure["failure"].(map[string]any)
	payload["safeMessage"] = ""
	encoded, _ = json.Marshal(failure)
	if _, err := contract.DecodeResult(encoded, goldenToken()); err != nil {
		t.Fatal("empty optional safe message was rejected", err)
	}
	delete(payload, "safeMessage")
	payload["retryable"] = nil
	encoded, _ = json.Marshal(failure)
	if _, err := contract.DecodeResult(encoded, goldenToken()); err == nil {
		t.Fatal("null provider failure retryable flag was accepted")
	}
	payload["retryable"] = false
	payload["cursor"] = "stale-cursor"
	encoded, _ = json.Marshal(failure)
	if _, err := contract.DecodeResult(encoded, goldenToken()); err == nil {
		t.Fatal("stale provider failure cursor was accepted")
	}
	delete(payload, "cursor")
	encoded, _ = json.Marshal(failure)
	if _, err := contract.DecodeResult(encoded, goldenToken()); err == nil {
		t.Fatal("missing provider failure cursor was accepted")
	}
	payload["cursor"] = ""
	delete(payload, "retryable")
	encoded, _ = json.Marshal(failure)
	if _, err := contract.DecodeResult(encoded, goldenToken()); err == nil {
		t.Fatal("missing provider failure retryable flag was accepted")
	}
	payload["retryable"] = true
	payload["kind"] = "temporary_failure"
	payload["retryAfterSeconds"] = 0
	encoded, _ = json.Marshal(failure)
	if _, err := contract.DecodeResult(encoded, goldenToken()); err == nil {
		t.Fatal("explicit zero retry delay was accepted")
	}
	payload["retryable"] = false
	payload["kind"] = "mfa_required"
	encoded, _ = json.Marshal(failure)
	if _, err := contract.DecodeResult(encoded, goldenToken()); err == nil {
		t.Fatal("explicit zero retry delay bypassed absent-only semantics")
	}
}

func TestDecoderUsesOpenAPIUnicodeCharacterLimits(t *testing.T) {
	var value map[string]any
	if err := json.Unmarshal(fixture(t, "golden-page.json"), &value); err != nil {
		t.Fatal(err)
	}
	records := value["page"].(map[string]any)["records"].([]any)
	records[0].(map[string]any)["account"].(map[string]any)["name"] = strings.Repeat("ё", 1500)
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = contract.DecodeResult(encoded, goldenToken()); err != nil {
		t.Fatal("valid Unicode text was measured as bytes", err)
	}
	records[0].(map[string]any)["account"].(map[string]any)["name"] = strings.Repeat("ё", 2001)
	encoded, _ = json.Marshal(value)
	if _, err = contract.DecodeResult(encoded, goldenToken()); err == nil {
		t.Fatal("text beyond the OpenAPI character limit was accepted")
	}
}
