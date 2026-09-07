//go:build integration

package ledger_test

import (
	"context"

	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func (f *fixture) importSource(service *admission.Service, connection string, input ledger.SourceInput) (ledger.SourceOutcome, error) {
	f.t.Helper()
	issued := f.issued(service, connection)
	input.ConnectionID = connection
	input.JobID = issued.ID
	input.FetchedAt = f.now
	sources := journal.NewSources(f.store, f.writer)
	var outcome ledger.SourceOutcome
	applied, err := service.CommitPage(testContext, f.p, issued, admission.Page{EvidenceRef: input.EvidenceRef, Coverage: "complete", Complete: true}, func(ctx context.Context) error {
		var err error
		outcome, err = sources.Apply(ctx, f.p, input)
		return err
	})
	if err == nil && !applied {
		f.t.Error("valid import quarantined")
	}
	return outcome, err
}
