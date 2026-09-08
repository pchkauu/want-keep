//go:build integration

package matching_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestExplicitPaymentLinkRejectsDistinctVerifiedMovements(t *testing.T) {
	for _, movement := range []bool{false, true} {
		t.Run(map[bool]string{false: "payment", true: "blockchain_movement"}[movement], func(t *testing.T) {
			f := newFixture(t)
			c := f.client(f.p)
			account := f.create(money.RUB, "5000")
			a, b := f.revision(uuid.NewString(), account, "-300", money.RUB, 1), f.revision(uuid.NewString(), account, "-300", money.RUB, 1)
			for _, r := range []*ledger.Revision{&a, &b} {
				r.Correspondence = &ledger.Correspondence{Kind: "payment", Namespace: "synthetic:payment", Reference: r.OperationID}
				if movement {
					r.Correspondence.Reference = "same_tx"
					r.Correspondence.Network = "synthetic"
					r.Correspondence.Movement = r.OperationID
				}
				if _, err := f.write(*r, request()); err != nil {
					t.Fatal(err)
				}
			}
			input := map[string]any{"kind": "receipt_match", "expectedRevisions": f.versions(a.OperationID, b.OperationID), "reason": "Claim same payment"}
			response := decode[generated.CommandStatus](t, c.call("POST", "/transactions/"+a.OperationID+"/links", uuid.NewString(), input, 202))
			failed, err := response.AsCommandFailed()
			if err != nil || failed.Error.Code != "matching_conflict" {
				t.Fatal(response, err)
			}
			f.balance(account, "owned", "4400")
		})
	}
}

func TestPaymentLinkReasonUsesUnicodeCharacterLimit(t *testing.T) {
	f := newFixture(t)
	c := f.client(f.p)
	account := f.create(money.RUB, "5000")
	a, b := uuid.NewString(), uuid.NewString()
	for _, id := range []string{a, b} {
		if _, err := f.write(f.revision(id, account, "-300", money.RUB, 1), request()); err != nil {
			t.Fatal(err)
		}
	}
	input := map[string]any{"kind": "receipt_match", "expectedRevisions": f.versions(a, b), "reason": strings.Repeat("я", 2001)}
	c.call("POST", "/transactions/"+a+"/links", uuid.NewString(), input, 400)
	input["reason"] = strings.Repeat("я", 2000)
	c.result("/transactions/"+a+"/links", input)
	f.balance(account, "owned", "4700")
}
