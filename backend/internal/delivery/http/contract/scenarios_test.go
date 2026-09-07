package contract_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func TestPayerAndBudgetPreviewShapes(t *testing.T) {
	b, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	const id = `10000000-0000-4000-8000-000000000001`
	const line = `"line":{"ownership":{"scope":"household"},"kind":"obligation","planned":{"amount":"12000","asset":"RUB"},"allocation":{"mode":"unresolved","reason":"clarify"}}`
	const budget = `"budgetId":"` + id + `","expectedRevision":2`
	for _, tc := range []struct {
		name, data string
		valid      bool
	}{
		{"Payer", `{"state":"known","memberId":"` + id + `"}`, true},
		{"Payer", `{"state":"unknown"}`, true},
		{"Payer", `{"state":"not_applicable"}`, true},
		{"Payer", `{"state":"known"}`, false},
		{"Payer", `{"state":"unknown","memberId":"` + id + `"}`, false},
		{"TransactionCorrection", `{"expectedRevision":1,"reason":"payer corrected","payer":{"state":"known","memberId":"` + id + `"}}`, true},
		{"TransactionCorrection", `{"expectedRevision":1,"reason":"payer corrected","payer":{"state":"known","memberId":"` + id + `"},"actorId":"` + id + `"}`, false},
		{"TransactionCreate", `{"type":"expense","accountId":"` + id + `","occurredAt":"2026-09-07T00:00:00Z","amount":{"amount":"10","asset":"RUB"},"allocation":{"mode":"unresolved","reason":"clarify"},"payer":{"state":"known","memberId":"` + id + `"}}`, true},
		{"BudgetPreview", `{"mode":"create",` + budget + `,` + line + `}`, true},
		{"BudgetPreview", `{"mode":"update",` + budget + `,"lineId":"` + id + `",` + line + `}`, true},
		{"BudgetPreview", `{"mode":"delete",` + budget + `,"lineId":"` + id + `","reason":"cancelled"}`, true},
		{"BudgetPreview", `{` + budget + `,` + line + `}`, false},
		{"BudgetPreview", `{"mode":"update",` + budget + `,` + line + `}`, false},
		{"BudgetPreview", `{"mode":"create",` + budget + `,"lineId":"` + id + `",` + line + `}`, false},
		{"BudgetPreview", `{"mode":"delete",` + budget + `,"lineId":"` + id + `","reason":"cancelled",` + line + `}`, false},
	} {
		t.Run(tc.name+tc.data, func(t *testing.T) {
			var out json.RawMessage
			err := b.Decode(tc.name, []byte(tc.data), &out)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
		})
	}
}

func TestTransferReferenceRevisionSurvivesBoundary(t *testing.T) {
	b, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	const data = `{"fromAccountId":"10000000-0000-4000-8000-000000000001","toAccountId":"10000000-0000-4000-8000-000000000002","occurredAt":"2026-09-07T00:00:00Z","sent":{"amount":"1","asset":"USD"},"received":{"amount":"1","asset":"USD"},"fees":[],"existingTransactions":[{"transactionId":"10000000-0000-4000-8000-000000000003","expectedRevision":1}]}`
	var dto generated.TransferCreate
	if err := b.Decode("TransferCreate", []byte(data), &dto); err != nil {
		t.Fatal(err)
	}
	principal, _ := (household.Membership{ID: "membership", UserID: "actor", HouseholdID: "family", Active: true}).Principal()
	cmd, err := command.NewCommand("10000000-0000-4000-8000-000000000004", "transfers.create", strings.Repeat("a", 64), principal)
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(cmd.RequireRevision(uint64(dto.ExistingTransactions[0].ExpectedRevision), 2), command.ErrVersionConflict) {
		t.Fatal("stale linked revision accepted")
	}
	var out json.RawMessage
	if err := b.Decode("TransferCreate", []byte(strings.Replace(data, `,"expectedRevision":1`, "", 1)), &out); err == nil {
		t.Fatal("missing linked revision accepted")
	}
}

func TestDimensionlessReturnValues(t *testing.T) {
	b, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{`{"state":"known","ratio":"0.123456789123456789"}`, `{"state":"known","ratio":"-0.05"}`, `{"state":"known","ratio":"0"}`, `{"state":"unavailable","reason":"no_unique_root"}`} {
		var dto generated.ReturnValue
		if err := b.Decode("ReturnValue", []byte(value), &dto); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(dto)
		if err != nil {
			t.Fatal(err)
		}
		var in, out any
		if err := json.Unmarshal([]byte(value), &in); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(encoded, &out); err != nil {
			t.Fatal(err)
		}
		left, _ := json.Marshal(in)
		right, _ := json.Marshal(out)
		if string(left) != string(right) {
			t.Fatal("return precision/state lost")
		}
	}
	for _, value := range []string{`{"state":"known","ratio":0.12}`, `{"state":"known","ratio":"1e-2"}`, `{"state":"unavailable","reason":"no_unique_root","ratio":"0"}`, `{"state":"unavailable"}`} {
		var out json.RawMessage
		if err := b.Decode("ReturnValue", []byte(value), &out); err == nil {
			t.Fatal("invalid return accepted")
		}
	}
}
