//go:build integration

package matching_test

import (
	"context"
	"strings"

	"github.com/google/uuid"
	attachment "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
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
