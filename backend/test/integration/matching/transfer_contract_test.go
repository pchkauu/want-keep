//go:build integration

package matching_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestHTTPExistingTransferRequiresExactPrincipalAndFees(t *testing.T) {
	for _, tc := range []struct {
		name, sent, received, fee, remaining string
		asset                                money.Asset
	}{
		{"RUB", "1000", "1000", "10", "8990", money.RUB},
		{"exchange", "9000", "100", "50", "950", money.USDT},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			from, to := f.create(money.RUB, "10000"), f.create(tc.asset, "0")
			a, b := f.revision(uuid.NewString(), from, "-"+tc.sent, money.RUB, 1), f.revision(uuid.NewString(), to, tc.received, tc.asset, 1)
			b.Type = ledger.Income
			a.Postings = append(a.Postings, ledger.Posting{AccountID: from, Money: cash("-"+tc.fee, money.RUB), Role: ledger.Fee, Funding: ledger.OwnFunds})
			for _, r := range []ledger.Revision{a, b} {
				if _, err := f.write(r, request()); err != nil {
					t.Fatal(err)
				}
			}
			c := f.client(f.p)
			input := map[string]any{"fromAccountId": from, "toAccountId": to, "occurredAt": f.now.String(), "sent": map[string]string{"asset": "RUB", "amount": tc.sent}, "received": map[string]string{"asset": string(tc.asset), "amount": tc.received}, "fees": []any{}, "existingTransactions": f.versions(a.OperationID, b.OperationID)}
			out := decode[generated.CommandStatus](t, c.call("POST", "/transfers", uuid.NewString(), input, 202))
			failed, err := out.AsCommandFailed()
			if err != nil || failed.Status != "failed" {
				t.Fatal("missing fee accepted", out)
			}
			if f.current(a.OperationID).Revision != 1 || f.current(b.OperationID).Revision != 1 {
				t.Fatal("partial link persisted")
			}
			input["fees"] = []map[string]any{{"accountId": from, "amount": map[string]string{"asset": "RUB", "amount": tc.fee}}}
			key := uuid.NewString()
			for range 2 {
				out = decode[generated.CommandStatus](t, c.call("POST", "/transfers", key, input, 202))
				success, err := out.AsCommandSucceeded()
				if err != nil || success.Status != "succeeded" {
					t.Fatal("existing transfer failed", out)
				}
			}
			f.balance(from, "owned", tc.remaining)
			f.balance(to, "owned", tc.received)
			if f.current(a.OperationID).Revision != 2 || f.current(b.OperationID).Revision != 2 {
				t.Fatal("replay created revisions")
			}
			if f.count("operations") != 4 {
				t.Fatal("link created an operation")
			}
		})
	}
}
