//go:build integration

package ledger_test

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func binding() connections.Binding {
	return connections.Binding{Provider: "raiffeisen", Environment: "test", AdapterBuildDigest: "sha256:" + strings.Repeat("a", 64), CollectorImageDigest: "sha256:" + strings.Repeat("b", 64), ContractVersion: "10", AllowlistRevision: "1", NonSecretConfigRevision: "1", OperatorPermissionRevision: "1"}
}
func (f *fixture) connection(owner household.Principal) string {
	f.t.Helper()
	id := uuid.NewString()
	if err := f.store.WithinHousehold(testContext, owner, func(ctx context.Context) error {
		return f.store.CreateConnection(ctx, admission.Connection{HouseholdID: f.family.ID, ID: id, Provider: "raiffeisen", Owner: owner.UserID(), Generation: 1, Authorized: true})
	}); err != nil {
		f.t.Fatal(err)
	}
	return id
}
func (f *fixture) admit() *admission.Service {
	f.t.Helper()
	s := admission.NewService(f.store, f.store)
	for _, kind := range []connections.CheckKind{connections.ProviderCheck, connections.HostCheck} {
		if _, err := s.RecordCheck(testContext, connections.Check{Kind: kind, Binding: binding(), Result: connections.CheckPassed, At: f.now}); err != nil {
			f.t.Fatal(err)
		}
	}
	return s
}
func (f *fixture) issued(s *admission.Service, id string) jobs.Job {
	f.t.Helper()
	j, err := s.RequestSync(testContext, f.p, id, binding(), time.Now().Add(time.Hour))
	if err != nil {
		f.t.Fatal(err)
	}
	claimed, err := f.store.ClaimJobs(testContext, "sync", 100, time.Minute)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, c := range claimed {
		if c.ID == j.ID {
			return c
		}
	}
	f.t.Fatal("unclaimed job")
	return jobs.Job{}
}
func (f *fixture) input(connection string) accounts.ImportInput {
	date, _ := calendar.ParseDate("2026-08-01")
	values, _ := account.CashAmounts(cash("100", "RUB"))
	debt, _ := reporting.KnownAmount(cash("300", "RUB"))
	values.Debt = debt
	credit, _ := reporting.KnownAmount(cash("1000", "RUB"))
	coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
	return accounts.ImportInput{ConnectionID: connection, Provider: "raiffeisen", ExternalID: "stable-external-account", Product: "current", ExternalAssetCode: "RUB", Name: "Bank", Asset: "RUB", OpeningDate: date, EvidenceRef: "synthetic:bank-balance", Origin: "historical_backfill", Observation: account.Observation{ID: uuid.NewString(), AsOf: f.now, FetchedAt: f.now, Amounts: values, CreditLimit: credit, OwnAvailable: true, Coverage: coverage, Freshness: reporting.Fresh}}
}
func (f *fixture) importAccount(s *admission.Service, input accounts.ImportInput) accounts.ImportResult {
	f.t.Helper()
	j := f.issued(s, input.ConnectionID)
	input.JobID = j.ID
	var out accounts.ImportResult
	page := admission.Page{EvidenceRef: input.EvidenceRef, Coverage: "complete", Complete: true}
	applied, err := s.CommitPage(testContext, f.p, j, page, func(ctx context.Context) error {
		var e error
		out, e = f.service().Import(ctx, f.p, f.store, input)
		return e
	})
	if err != nil || !applied {
		f.t.Fatal("import failed", err)
	}
	return out
}
