package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"

	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type ProviderGateway interface {
	Binding() connections.Binding
	Manifest(context.Context) (ingestion.Manifest, error)
	Read(context.Context, ingestion.JobToken) (ingestion.Result, error)
}

type EvidenceStore interface {
	// Save durably binds the raw bytes and references to the server-derived household and job.
	Save(context.Context, ingestion.EvidenceBatch) error
}

type Gate interface {
	BeforeRead(context.Context, household.Principal, jobs.Job) error
	CommitPage(context.Context, household.Principal, jobs.Job, admission.Page, func(context.Context) error) (bool, error)
	CommitProviderOutcome(context.Context, household.Principal, jobs.Job, string, jobs.State, jobs.Reason, time.Duration, func(context.Context) error) (bool, error)
	RetainRejectedResult(context.Context, household.Principal, jobs.Job, string) error
}

type AccountImporter interface {
	Resolve(context.Context, household.Principal, accounts.ImportInput) (accounts.ImportResult, error)
	Import(context.Context, household.Principal, accounts.ImportInput) (accounts.ImportResult, error)
}

type SourceWriter interface {
	AccountTimezone(context.Context, household.Principal) (calendar.Timezone, error)
	Source(context.Context, household.Principal, ledger.SourceKey) (ledger.SourceRecord, bool, error)
	Apply(context.Context, household.Principal, ledger.SourceInput) (ledger.SourceOutcome, error)
}

type Service struct {
	gate     Gate
	evidence EvidenceStore
	accounts AccountImporter
	sources  SourceWriter
	now      func() calendar.Instant
	newID    func() string
}

func NewService(gate Gate, evidence EvidenceStore, accountImporter AccountImporter, sources SourceWriter, now func() calendar.Instant, newID func() string) (*Service, error) {
	if gate == nil || evidence == nil || accountImporter == nil || sources == nil || now == nil || newID == nil {
		return nil, ingestion.ErrInvalidContract
	}
	return &Service{gate: gate, evidence: evidence, accounts: accountImporter, sources: sources, now: now, newID: newID}, nil
}

// Ingest performs provider IO only after the read fence, durably stages raw evidence,
// and delegates every financial write to the existing admission transaction.
func (s *Service) Ingest(ctx context.Context, p household.Principal, issued jobs.Job, gateway ProviderGateway) (bool, *ingestion.ProviderFailure, error) {
	if gateway == nil {
		return false, nil, ingestion.ErrInvalidContract
	}
	token, err := TokenFromJob(issued)
	if err != nil {
		return false, nil, err
	}
	if gateway.Binding() != issued.Binding {
		return false, nil, connections.ErrProviderNotAdmitted
	}
	if err = s.gate.BeforeRead(ctx, p, issued); err != nil {
		return false, nil, err
	}
	manifest, err := gateway.Manifest(ctx)
	if err != nil {
		return false, nil, err
	}
	if err = manifest.Validate(issued.Binding.Provider); err != nil {
		return false, nil, err
	}
	result, err := gateway.Read(ctx, token)
	if err != nil {
		return false, nil, err
	}
	if err = result.Validate(); err != nil {
		return false, nil, err
	}
	if result.Page != nil {
		if !ingestion.SameToken(token, result.Page.Token) {
			return false, nil, ingestion.ErrInvalidContract
		}
		if err = manifest.RequirePage(*result.Page); err != nil {
			return false, nil, err
		}
		batch, references, err := s.stage(ctx, p, issued, result.Page.Evidence)
		if err != nil {
			return false, nil, err
		}
		page := admission.Page{EvidenceRef: batch.PageReference, Cursor: result.Page.Token.Cursor, NextCursor: result.Page.NextCursor, Coverage: string(result.Page.Coverage.State()), Gaps: result.Page.Coverage.Reasons(), Complete: result.Page.Complete}
		applied, err := s.gate.CommitPage(ctx, p, issued, page, func(tx context.Context) error {
			return s.applyPage(tx, p, issued, *result.Page, batch.FetchedAt, references)
		})
		if err != nil {
			if retainErr := s.gate.RetainRejectedResult(ctx, p, issued, batch.PageReference); retainErr != nil {
				return false, nil, errors.Join(err, errors.Join(ingestion.ErrEvidence, retainErr))
			}
		}
		return applied, nil, err
	}
	if !ingestion.SameToken(token, result.Failure.Token) {
		return false, nil, ingestion.ErrInvalidContract
	}
	batch, _, err := s.stage(ctx, p, issued, result.Failure.Evidence)
	if err != nil {
		return false, nil, err
	}
	state, reason, delay := providerFailureOutcome(*result.Failure, issued.Attempt)
	applied, err := s.gate.CommitProviderOutcome(ctx, p, issued, batch.PageReference, state, reason, delay, func(context.Context) error { return nil })
	return applied, result.Failure, err
}

func providerFailureOutcome(failure ingestion.ProviderFailure, attempt int) (jobs.State, jobs.Reason, time.Duration) {
	switch failure.Kind {
	case ingestion.ReauthenticationRequired, ingestion.MFARequired, ingestion.CaptchaRequired:
		return jobs.Waiting, jobs.ReauthRequired, 0
	case ingestion.RateLimited:
		return jobs.Ready, jobs.TemporaryFailure, time.Duration(failure.RetryAfterSeconds) * time.Second
	case ingestion.TemporaryFailure:
		if failure.RetryAfterSeconds > 0 {
			return jobs.Ready, jobs.TemporaryFailure, time.Duration(failure.RetryAfterSeconds) * time.Second
		}
		return jobs.Ready, jobs.TemporaryFailure, jobs.DefaultRetryPolicy().Delay(attempt, 1)
	default:
		return jobs.Failed, jobs.PermanentFailure, 0
	}
}

func TokenFromJob(job jobs.Job) (ingestion.JobToken, error) {
	token := ingestion.JobToken{JobID: job.ID, LeaseToken: job.LeaseToken, ConnectionID: job.ConnectionID, Attempt: job.Attempt, ConnectionGeneration: job.ConnectionGeneration, Binding: job.Binding, AdmissionRevision: job.AdmissionRevision, Cursor: job.Cursor, ReplayFrom: job.RangeFrom, ReplayTo: job.RangeTo}
	if err := token.Validate(); err != nil {
		return ingestion.JobToken{}, err
	}
	return token, nil
}

func (s *Service) stage(ctx context.Context, p household.Principal, issued jobs.Job, raw []ingestion.Evidence) (ingestion.EvidenceBatch, map[string]ingestion.StoredEvidence, error) {
	if issued.HouseholdID != p.HouseholdID() {
		return ingestion.EvidenceBatch{}, nil, household.ErrForbidden
	}
	batch := ingestion.EvidenceBatch{HouseholdID: string(p.HouseholdID()), JobID: issued.ID, PageReference: "evidence:page:" + s.newID(), FetchedAt: s.now()}
	references := make(map[string]ingestion.StoredEvidence, len(raw))
	for _, evidence := range raw {
		item := ingestion.StoredEvidence{Reference: "evidence:raw:" + s.newID(), Raw: evidence}
		batch.Items = append(batch.Items, item)
		references[evidence.ID] = item
	}
	if err := batch.Validate(); err != nil {
		return ingestion.EvidenceBatch{}, nil, err
	}
	if err := s.evidence.Save(ctx, batch); err != nil {
		return ingestion.EvidenceBatch{}, nil, errors.Join(ingestion.ErrEvidence, err)
	}
	return batch, references, nil
}

func (s *Service) applyPage(ctx context.Context, p household.Principal, issued jobs.Job, page ingestion.Page, fetchedAt calendar.Instant, evidence map[string]ingestion.StoredEvidence) error {
	accountsByKey := map[string]ingestion.AccountRecord{}
	balancesByKey := map[string]ingestion.BalanceSnapshot{}
	for _, record := range page.Records {
		if record.Account != nil {
			key := record.Account.Reference.Key()
			if _, exists := accountsByKey[key]; exists {
				return ingestion.ErrInvalidContract
			}
			accountsByKey[key] = *record.Account
		}
		if record.Balance != nil {
			key := record.Balance.Reference.Key()
			if _, exists := balancesByKey[key]; exists {
				return ingestion.ErrInvalidContract
			}
			balancesByKey[key] = *record.Balance
		}
	}
	keys := make([]string, 0, len(accountsByKey))
	for key := range accountsByKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	resolved := map[string]string{}
	unresolved := map[string]string{}
	for _, key := range keys {
		record := accountsByKey[key]
		input, err := s.accountDescriptorInput(issued, record, evidence)
		if err != nil {
			return err
		}
		outcome, err := s.accounts.Resolve(ctx, p, input)
		if err != nil {
			return err
		}
		if outcome.Account != nil {
			resolved[key] = outcome.Account.ID
			continue
		}
		switch outcome.Reason {
		case "source_ambiguous":
			unresolved[key] = outcome.Reason
		case "unsupported_asset":
			if !hasGap(page.Coverage, outcome.Reason) {
				return ingestion.ErrInvalidContract
			}
			unresolved[key] = outcome.Reason
		default:
			return ingestion.ErrInvalidContract
		}
	}
	balanceKeys := make([]string, 0, len(balancesByKey))
	for key := range balancesByKey {
		balanceKeys = append(balanceKeys, key)
	}
	sort.Strings(balanceKeys)
	for _, key := range balanceKeys {
		record, exists := accountsByKey[key]
		if !exists {
			return ingestion.ErrInvalidContract
		}
		if _, supported := resolved[key]; !supported {
			continue
		}
		input, err := s.accountInput(issued, record, balancesByKey[key], fetchedAt, evidence)
		if err != nil {
			return err
		}
		outcome, err := s.accounts.Import(ctx, p, input)
		if err != nil || outcome.Account == nil {
			if err != nil {
				return err
			}
			return ingestion.ErrInvalidContract
		}
		resolved[key] = outcome.Account.ID
	}
	for _, record := range page.Records {
		if record.Transaction == nil {
			continue
		}
		if err := s.applyTransaction(ctx, p, issued, *record.Transaction, record.CanonicalPayload, fetchedAt, evidence, resolved, unresolved); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) accountInput(issued jobs.Job, record ingestion.AccountRecord, balance ingestion.BalanceSnapshot, fetchedAt calendar.Instant, evidence map[string]ingestion.StoredEvidence) (accounts.ImportInput, error) {
	if record.Reference != balance.Reference {
		return accounts.ImportInput{}, ingestion.ErrInvalidContract
	}
	stored, found := evidence[balance.EvidenceID]
	if !found {
		return accounts.ImportInput{}, ingestion.ErrInvalidContract
	}
	asset := money.Asset(record.Reference.AssetCode)
	if canonical, canonicalErr := accounts.CanonicalSourceAsset(issued.Binding.Provider, record.Reference.AssetCode); canonicalErr == nil {
		asset = canonical
	}
	values := []*ingestion.Amount{&balance.Owned, &balance.Available, &balance.Locked, &balance.Debt, &balance.CreditLimit}
	converted := make([]reporting.Amount, 0, len(values))
	for _, value := range values {
		amount, convertErr := value.ReportingAs(record.Reference.AssetCode, asset)
		if convertErr != nil {
			return accounts.ImportInput{}, convertErr
		}
		converted = append(converted, amount)
	}
	if !nonNegative(converted[1], converted[2], converted[3], converted[4]) {
		return accounts.ImportInput{}, ingestion.ErrInvalidContract
	}
	input, err := s.accountDescriptorInput(issued, record, evidence)
	if err != nil {
		return accounts.ImportInput{}, err
	}
	input.Asset = asset
	input.EvidenceRef = stored.Reference
	input.Observation = account.Observation{
		ID:           stableID("observation", issued.Binding.Provider, record.Reference.Key(), balance.SourceAsOf.String(), stored.Raw.Digest),
		ConnectionID: issued.ConnectionID,
		JobID:        issued.ID,
		EvidenceRef:  stored.Reference,
		AsOf:         balance.SourceAsOf,
		FetchedAt:    fetchedAt,
		Amounts:      account.Amounts{Owned: converted[0], Available: converted[1], Locked: converted[2], Debt: converted[3]},
		CreditLimit:  converted[4],
		OwnAvailable: balance.OwnAvailable,
		Coverage:     balance.Coverage,
		Freshness:    balance.Freshness,
	}
	return input, nil
}

func (s *Service) accountDescriptorInput(issued jobs.Job, record ingestion.AccountRecord, evidence map[string]ingestion.StoredEvidence) (accounts.ImportInput, error) {
	stored, found := evidence[record.EvidenceID]
	if !found {
		return accounts.ImportInput{}, ingestion.ErrInvalidContract
	}
	aliases := make([]account.CardAlias, 0, len(record.Aliases))
	for _, alias := range record.Aliases {
		aliases = append(aliases, account.CardAlias{ID: stableID("alias", issued.Binding.Provider, record.Reference.Key(), alias.ID), Label: alias.Label, LastFour: alias.LastFour})
	}
	origin := "live_sync"
	if !issued.RangeFrom.IsZero() {
		origin = "historical_backfill"
	}
	asset := money.Asset(record.Reference.AssetCode)
	if canonical, canonicalErr := accounts.CanonicalSourceAsset(issued.Binding.Provider, record.Reference.AssetCode); canonicalErr == nil {
		asset = canonical
	}
	return accounts.ImportInput{
		ConnectionID:      issued.ConnectionID,
		JobID:             issued.ID,
		Provider:          issued.Binding.Provider,
		ExternalID:        record.Reference.ExternalAccountID,
		Product:           record.Reference.Product,
		Network:           record.Reference.Network,
		ExternalAssetCode: record.Reference.AssetCode,
		Name:              record.Name,
		EvidenceRef:       stored.Reference,
		Asset:             asset,
		OpeningDate:       record.OpeningDate,
		Aliases:           aliases,
		Origin:            origin,
	}, nil
}

func (s *Service) applyTransaction(ctx context.Context, p household.Principal, issued jobs.Job, record ingestion.TransactionRecord, canonical []byte, fetchedAt calendar.Instant, evidence map[string]ingestion.StoredEvidence, resolved, unresolved map[string]string) error {
	stored, found := evidence[record.EvidenceID]
	if !found {
		return ingestion.ErrInvalidContract
	}
	payloadHash, err := sourceHash(canonical, stored.Raw.Digest)
	if err != nil {
		return err
	}
	key := ledger.SourceKey{HouseholdID: p.HouseholdID(), Provider: issued.Binding.Provider, ExternalAccountID: record.ExternalAccountID, Product: record.Product, Log: record.LogNamespace, RecordID: record.ProviderRecordID}
	current, exists, err := s.sources.Source(ctx, p, key)
	if err != nil && !errors.Is(err, ledger.ErrSourceAmbiguous) {
		return err
	}
	operationID := s.newID()
	expectedSourceRevision := uint64(0)
	if exists {
		expectedSourceRevision = current.Revision
		if current.OperationID != "" {
			operationID = current.OperationID
		}
	}
	input := ledger.SourceInput{Key: key, PayloadHash: payloadHash, EvidenceRef: stored.Reference, ConnectionID: issued.ConnectionID, JobID: issued.ID, FetchedAt: fetchedAt, Classification: record.Classification, ExpectedRevision: expectedSourceRevision}
	if record.ProviderState == "unknown" {
		input.UnresolvedReason = "provider_state_unknown"
		_, err = s.sources.Apply(ctx, p, input)
		return err
	}
	for _, posting := range record.Postings {
		key := posting.Reference.Key()
		if _, missing := unresolved[key]; missing {
			input.UnresolvedReason = "transaction_unresolved"
			_, err = s.sources.Apply(ctx, p, input)
			return err
		}
		if _, ok := resolved[key]; !ok {
			return ingestion.ErrInvalidContract
		}
	}
	zone, err := s.sources.AccountTimezone(ctx, p)
	if err != nil {
		return err
	}
	cashDate, err := record.OccurredAt.DateIn(zone)
	if err != nil {
		return err
	}
	expenseMonth, err := calendar.ParseMonth(cashDate.String()[:7])
	if err != nil {
		return err
	}
	revision := ledger.Revision{OperationID: operationID, Revision: 1, ActorID: p.UserID(), Reason: "provider_import", Type: ledger.Type(record.EconomicType), State: ledger.State(record.ProviderState), OccurredAt: record.OccurredAt, PostedAt: record.PostedAt, Timezone: zone, CashDate: cashDate, ExpenseMonth: expenseMonth, Merchant: record.Merchant, Note: record.Note, Origin: "source", FeeKnowledge: ledger.FeeKnowledge(record.FeeKnowledge), PnLBasis: ledger.PnLBasis(record.PnLBasis), PayerState: "unknown"}
	for _, posting := range record.Postings {
		asset, parseErr := accounts.CanonicalSourceAsset(issued.Binding.Provider, posting.Reference.AssetCode)
		if parseErr != nil {
			input.UnresolvedReason = "unsupported_asset"
			input.Operation = nil
			_, err = s.sources.Apply(ctx, p, input)
			return err
		}
		amount, parseErr := money.NewMoney(posting.Amount, asset)
		if parseErr != nil {
			return parseErr
		}
		accountID, ok := resolved[posting.Reference.Key()]
		if !ok {
			return ingestion.ErrInvalidContract
		}
		revision.Postings = append(revision.Postings, ledger.Posting{AccountID: accountID, Money: amount, Role: ledger.Role(posting.Role), Funding: ledger.FundingKind(posting.Funding), Treatment: ledger.Treatment(posting.Treatment)})
	}
	input.Operation = &revision
	_, err = s.sources.Apply(ctx, p, input)
	return err
}

type AccountsAdapter struct {
	Service    *accounts.Service
	Repository accounts.ImportRepository
}

func NewAccountImporter(service *accounts.Service, repository accounts.ImportRepository) (AccountImporter, error) {
	adapter := AccountsAdapter{Service: service, Repository: repository}
	if service == nil || repository == nil {
		return nil, ingestion.ErrInvalidContract
	}
	return adapter, nil
}

func (a AccountsAdapter) Import(ctx context.Context, p household.Principal, input accounts.ImportInput) (accounts.ImportResult, error) {
	if a.Service == nil || a.Repository == nil {
		return accounts.ImportResult{}, ingestion.ErrInvalidContract
	}
	return a.Service.Import(ctx, p, a.Repository, input)
}

func (a AccountsAdapter) Resolve(ctx context.Context, p household.Principal, input accounts.ImportInput) (accounts.ImportResult, error) {
	if a.Service == nil || a.Repository == nil {
		return accounts.ImportResult{}, ingestion.ErrInvalidContract
	}
	return a.Service.ResolveImported(ctx, p, a.Repository, input)
}

type SourcesAdapter struct {
	Service    *journal.Sources
	Repository journal.SourceRepository
}

func NewSourceWriter(service *journal.Sources, repository journal.SourceRepository) (SourceWriter, error) {
	adapter := SourcesAdapter{Service: service, Repository: repository}
	if service == nil || repository == nil {
		return nil, ingestion.ErrInvalidContract
	}
	return adapter, nil
}

func (a SourcesAdapter) Source(ctx context.Context, p household.Principal, key ledger.SourceKey) (ledger.SourceRecord, bool, error) {
	if a.Repository == nil {
		return ledger.SourceRecord{}, false, ingestion.ErrInvalidContract
	}
	return a.Repository.Source(ctx, p, key)
}

func (a SourcesAdapter) AccountTimezone(ctx context.Context, p household.Principal) (calendar.Timezone, error) {
	if a.Repository == nil {
		return calendar.Timezone{}, ingestion.ErrInvalidContract
	}
	return a.Repository.AccountTimezone(ctx, p)
}

func (a SourcesAdapter) Apply(ctx context.Context, p household.Principal, input ledger.SourceInput) (ledger.SourceOutcome, error) {
	if a.Service == nil {
		return ledger.SourceOutcome{}, ingestion.ErrInvalidContract
	}
	return a.Service.Apply(ctx, p, input)
}

func hashString(value [32]byte) string { return hex.EncodeToString(value[:]) }

func nonNegative(values ...reporting.Amount) bool {
	for _, amount := range values {
		if value, known := amount.Value(); known && value.Sign() < 0 {
			return false
		}
	}
	return true
}

func sourceHash(canonical []byte, digest string) (string, error) {
	value, err := ingestion.CanonicalHash(canonical, digest)
	if err != nil {
		return "", err
	}
	return hashString(value), nil
}

func hasGap(coverage reporting.Coverage, expected string) bool {
	for _, gap := range coverage.Reasons() {
		if gap == expected {
			return true
		}
	}
	return false
}

func stableID(parts ...string) string {
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	digest[6] = digest[6]&0x0f | 0x80
	digest[8] = digest[8]&0x3f | 0x80
	encoded := hex.EncodeToString(digest[:16])
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:]
}
