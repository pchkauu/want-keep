package application

import (
	"context"
	"slices"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

type decisionRestorer struct {
	repository DecisionRepository
}

func (u decisionRestorer) restore(ctx context.Context, p household.Principal, current ledger.Revision, entry ledger.DecisionEntry, before ledger.Revision, source *ledger.Revision) (ledger.Revision, error) {
	next, err := current.UndoFields(entry, before, source)
	if err != nil || source == nil {
		return next, err
	}
	independent := []ledger.Field{}
	for _, field := range []ledger.Field{ledger.PrincipalField, ledger.FeesField, ledger.DateField, ledger.PayerField, ledger.MerchantField, ledger.NoteField} {
		version := current.FieldVersions[field]
		if version <= entry.After || slices.Contains(entry.Fields, field) {
			continue
		}
		changed, err := u.repository.LedgerRevision(ctx, p, current.OperationID, version)
		if err != nil {
			return next, err
		}
		if changed.DecisionID != "" {
			independent = append(independent, field)
		}
	}
	return next.ReapplySource(*source, independent)
}
