//go:build integration

package ledger_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	attachment "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (f *fixture) attachment(accountID string, ready bool) string {
	f.t.Helper()
	id := uuid.NewString()
	err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		_, err := f.store.ReserveAttachment(ctx, f.p, attachment.Upload{ID: id, AccountID: accountID, Name: "Synthetic receipt", MediaType: "image/png", Size: 1, Hash: strings.Repeat("a", 64)})
		if err != nil {
			return err
		}
		if ready {
			return f.store.ReadyAttachment(ctx, f.p, id)
		}
		return nil
	})
	if err != nil {
		f.t.Fatal(err)
	}
	if ready {
		a, ok, err := f.store.ClaimAttachment(testContext)
		if err != nil || !ok {
			f.t.Fatal("claim", err)
		}
		if err = f.store.CompleteAttachment(testContext, a, []attachment.Page{{Number: 1, Width: 1, Height: 1, Size: 1, Hash: strings.Repeat("b", 64)}}, attachment.NoReason); err != nil {
			f.t.Fatal(err)
		}
	}
	return id
}

func TestHTTPAttachmentAccessAndReadiness(t *testing.T) {
	f := newFixture(t)
	a := f.create(money.RUB, "1000")
	b := f.create(money.RUB, "1000")
	c := f.client(f.q)
	ready := f.attachment(a, true)
	waiting := f.attachment(a, false)
	in := c.input(a, "RUB", "100")
	in["attachmentId"] = ready
	out := decode[generated.CommandSucceeded](t, c.call("POST", "/transactions", uuid.NewString(), in, 202))
	if out.Status != "succeeded" {
		t.Fatal(out)
	}
	r := decode[generated.Transaction](t, c.call("GET", "/transactions/"+out.Result.Id, "", nil, 200))
	if r.AttachmentId == nil || *r.AttachmentId != ready {
		t.Fatal("attachment link lost")
	}
	for _, test := range []struct{ account, id, code string }{{b, ready, "invalid_attachment"}, {a, waiting, "attachment_not_ready"}, {a, uuid.NewString(), "not_found"}} {
		in = c.input(test.account, "RUB", "1")
		in["attachmentId"] = test.id
		failed := decode[generated.CommandFailed](t, c.call("POST", "/transactions", uuid.NewString(), in, 202))
		if string(failed.Error.Code) != test.code {
			t.Fatal(failed)
		}
	}
	other := f.otherFamily()
	foreignAccount := other.account(money.RUB, "100")
	foreignAttachment := other.attachment(foreignAccount, false)
	in = c.input(a, "RUB", "1")
	in["attachmentId"] = foreignAttachment
	failed := decode[generated.CommandFailed](t, c.call("POST", "/transactions", uuid.NewString(), in, 202))
	if failed.Error.Code != "not_found" {
		t.Fatal("attachment existence leaked", failed)
	}
	if f.count("operation_revisions") != 3 {
		t.Fatal("rejected attachment produced money")
	}
}
