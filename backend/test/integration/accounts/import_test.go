//go:build integration

package accounts_test

import (
	"context"
	"strings"
	"testing"
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
func TestImportedIdentityAliasesDebtAndSnapshots(t *testing.T) {
	f := newFixture(t)
	s := f.admit()
	connection := f.connection(f.p)
	input := f.input(connection)
	input.Aliases = []account.CardAlias{{ID: uuid.NewString(), Label: "Debit", LastFour: "1001"}, {ID: uuid.NewString(), Label: "Spare", LastFour: "1002"}}
	result := f.importAccount(s, input)
	if result.Account == nil {
		t.Fatal(result.Reason)
	}
	id := result.Account.ID
	reconnect := input
	reconnect.ConnectionID = f.connection(f.p)
	reconnect.Observation.ID = uuid.NewString()
	again := f.importAccount(s, reconnect)
	if again.Account.ID != id || f.count("accounts") != 1 || f.count("card_aliases") != 2 || f.count("postings") != 0 {
		t.Fatal("reconnect duplicated money")
	}
	view, err := f.service().Read(testContext, f.q, id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Opening.Confirmed {
		t.Fatal("current snapshot confirmed historical opening")
	}
	m, _ := view.Source.Amounts.Owned.Value()
	d, _ := view.Source.Amounts.Debt.Value()
	if m.Amount() != "100" || d.Amount() != "300" {
		t.Fatal("credit limit affected balance")
	}
	if available, ok := view.FundingAvailability().Value(); !ok || available.Amount() != "100" {
		t.Fatal("verified own source availability was lost or included credit")
	}
	for _, product := range []string{"funding", "earn"} {
		x := input
		x.Product = product
		x.Aliases = nil
		x.Observation.ID = uuid.NewString()
		out := f.importAccount(s, x)
		if out.Account == nil || out.Account.ID == id {
			t.Fatal("products merged")
		}
	}
	other := input
	other.ConnectionID = f.connection(f.q)
	other.ExternalID = "other-stable-account"
	other.Aliases = nil
	other.Observation.ID = uuid.NewString()
	out := f.importAccount(s, other)
	if out.Account == nil || out.Account.ID == id || out.Account.Ownership.PersonalOwnerID() != f.q.UserID() {
		t.Fatal("external owner lost")
	}
	// A delayed observation does not replace the newer bank state.
	old := reconnect
	old.Aliases = nil
	old.Observation.ID = uuid.NewString()
	old.Observation.AsOf = instant("2026-09-06T00:00:00Z")
	old.Observation.Amounts, _ = account.CashAmounts(cash("999", "RUB"))
	f.importAccount(s, old)
	view, err = f.service().Read(testContext, f.p, id)
	if err != nil {
		t.Fatal(err)
	}
	m, _ = view.Source.Amounts.Owned.Value()
	if m.Amount() != "100" {
		t.Fatal("late snapshot replaced current")
	}
	conflict := reconnect
	conflict.Aliases = nil
	conflict.Observation.ID = uuid.NewString()
	conflict.Observation.Amounts, _ = account.CashAmounts(cash("101", "RUB"))
	f.importAccount(s, conflict)
	view, err = f.service().Read(testContext, f.p, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := view.Source.Amounts.Owned.Value(); ok || view.Source.Coverage.State() != reporting.Partial {
		t.Fatal("conflicting snapshot chosen silently")
	}
}
func TestUnsupportedIdentityCollisionAndAdmissionQuarantine(t *testing.T) {
	f := newFixture(t)
	s := f.admit()
	input := f.input(f.connection(f.p))
	input.ExternalAssetCode = "USDC.E"
	input.Asset = "USDC"
	out := f.importAccount(s, input)
	if out.Account != nil || out.Reason != "unsupported_asset" || f.count("accounts") != 0 || f.count("quarantine") != 1 {
		t.Fatal("unsupported asset mapped")
	}
	input = f.input(input.ConnectionID)
	first := f.importAccount(s, input)
	if first.Account == nil {
		t.Fatal(first.Reason)
	}
	input.ConnectionID = f.connection(f.q)
	input.Observation.ID = uuid.NewString()
	out = f.importAccount(s, input)
	if out.Account != nil || out.Reason != "source_ambiguous" || f.count("accounts") != 1 {
		t.Fatal("owner collision merged")
	}
	j := f.issued(s, input.ConnectionID)
	changed := binding()
	changed.AllowlistRevision = "2"
	if _, err := s.Rebind(testContext, changed); err != nil {
		t.Fatal(err)
	}
	called := false
	applied, err := s.CommitPage(testContext, f.p, j, admission.Page{EvidenceRef: "synthetic:stale", Coverage: "complete", Complete: true}, func(context.Context) error { called = true; return nil })
	if err != nil || applied || called {
		t.Fatal("stale result applied", err)
	}
}

func TestImportedOpeningDoesNotRewriteSourceOrDoubleSpend(t *testing.T) {
	f := newFixture(t)
	s := f.admit()
	input := f.input(f.connection(f.p))
	result := f.importAccount(s, input)
	id := result.Account.ID
	if c, err := f.correct(id, "2026-08-01", "100", f.q); err != nil || c.Status() != "succeeded" {
		t.Fatal(c.ErrorCode(), err)
	}
	r := f.revision(uuid.NewString(), id, "-20", "RUB", 1)
	r.OccurredAt = instant("2026-09-06T12:00:00Z")
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	view, err := f.service().Read(testContext, f.p, id)
	if err != nil {
		t.Fatal(err)
	}
	m, _ := view.Ledger.Owned.Value()
	source, _ := view.Source.Amounts.Owned.Value()
	if m.Amount() != "80" || source.Amount() != "100" || f.count("account_observations") != 1 || view.Coverage.State() != reporting.Partial {
		t.Fatal("source and ledger mixed")
	}
	r.OperationID = uuid.NewString()
	r.OccurredAt = instant(f.now.Time().Add(time.Nanosecond).Format(time.RFC3339Nano))
	if _, err := f.write(r, request()); err != nil {
		t.Fatal(err)
	}
	view, err = f.service().Read(testContext, f.p, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, known := view.FundingAvailability().Value(); known {
		t.Fatal("newer movement ignored when funding")
	}
	err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		funding, err := f.store.Funding(ctx, f.p, uuid.NewString(), []string{id})
		if err == nil {
			if _, known := funding[id].Available.Value(); known {
				t.Fatal("reserve funding disagrees with account")
			}
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"account_openings", "opening_amounts", "account_observations", "observation_amounts", "account_events"} {
		var allowed bool
		if err := f.admin.QueryRow(testContext, `SELECT has_table_privilege('want_keep_app',$1,'UPDATE') OR has_table_privilege('want_keep_app',$1,'DELETE')`, "want_keep."+table).Scan(&allowed); err != nil || allowed {
			t.Fatal("mutable account history", table, err)
		}
	}
}

func TestConfirmedRURMappingAndTransactionalOmissionCoverage(t *testing.T) {
	f := newFixture(t)
	s := f.admit()
	input := f.input(f.connection(f.p))
	input.ExternalAssetCode = "RUR"
	first := f.importAccount(s, input)
	if first.Account == nil || first.Account.ExternalAssetCode != "RUR" || first.Account.Asset != "RUB" {
		t.Fatal("RUR not preserved", first.Reason)
	}
	input.Observation.ID = uuid.NewString()
	input.ExternalAssetCode = "RUB"
	input.ConnectionID = f.connection(f.p)
	again := f.importAccount(s, input)
	if again.Account == nil || again.Account.ID != first.Account.ID {
		t.Fatal("source symbol became identity", again.Reason)
	}
	input.ExternalID = "unsupported-account"
	input.ExternalAssetCode = "USDC.E"
	input.Asset = "USDC"
	omitted := f.importAccount(s, input)
	if omitted.Reason != "unsupported_asset" {
		t.Fatal(omitted.Reason)
	}
	input = f.input(f.connection(f.q))
	ambiguous := f.importAccount(s, input)
	if ambiguous.Reason != "source_ambiguous" {
		t.Fatal(ambiguous.Reason)
	}
	var bad int
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.jobs j JOIN want_keep.quarantine q ON (q.household_id,q.job_id)=(j.household_id,j.id) WHERE q.reason IN ('unsupported_asset','source_ambiguous') AND (j.coverage!='partial' OR NOT(q.reason=ANY(j.gaps)))`).Scan(&bad); err != nil || bad != 0 {
		t.Fatal("omission committed as complete", bad, err)
	}
	if f.count("accounts") != 1 || f.count("quarantine") != 2 {
		t.Fatal("omission evidence or account identity lost")
	}
}

func TestOmissionSurvivesFollowingCompletePage(t *testing.T) {
	f := newFixture(t)
	s := f.admit()
	input := f.input(f.connection(f.p))
	input.ExternalAssetCode = "USDC.E"
	input.Asset = "USDC"
	job := f.issued(s, input.ConnectionID)
	input.JobID = job.ID
	page := admission.Page{EvidenceRef: input.EvidenceRef, Coverage: "complete", NextCursor: "next"}
	applied, err := s.CommitPage(testContext, f.p, job, page, func(ctx context.Context) error {
		_, err := f.service().Import(ctx, f.p, f.store, input)
		return err
	})
	if err != nil || !applied {
		t.Fatal("first page", err)
	}
	job, err = f.store.Job(testContext, f.p, job.ID)
	if err != nil || job.Cursor != "next" {
		t.Fatal("advanced job", job.Cursor, err)
	}
	page.Cursor, page.NextCursor, page.Complete = "next", "done", true
	applied, err = s.CommitPage(testContext, f.p, job, page, func(context.Context) error { return nil })
	if err != nil || !applied {
		t.Fatal("second page", err)
	}
	var coverage, cursor string
	var gaps []string
	if err := f.admin.QueryRow(testContext, `SELECT coverage,cursor,gaps FROM want_keep.jobs WHERE id=$1`, job.ID).Scan(&coverage, &cursor, &gaps); err != nil {
		t.Fatal(err)
	}
	if coverage != "partial" || cursor != "done" || len(gaps) != 1 || gaps[0] != "unsupported_asset" {
		t.Fatal("later page erased omission", coverage, cursor, gaps)
	}
}
