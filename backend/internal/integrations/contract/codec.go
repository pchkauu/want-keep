package contract

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	generated "github.com/pchkauu/want-keep/backend/internal/integrations/contract/generated"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

const MaxEncodedResultBytes = 14 * 1024 * 1024

var (
	decimalSyntax = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)
	assetSyntax   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)
)

func DecodeManifest(data []byte, provider string) (ingestion.Manifest, error) {
	var source generated.CapabilityManifest
	if err := decodeStrict(data, &source, validManifestShape); err != nil {
		return ingestion.Manifest{}, err
	}
	result := ingestion.Manifest{Provider: string(source.Provider), Version: string(source.ContractVersion), Paginated: source.History.Paginated}
	if source.History.MaximumLookbackDays != nil {
		value := *source.History.MaximumLookbackDays
		if value < 1 || value > 36500 {
			return ingestion.Manifest{}, ingestion.ErrInvalidContract
		}
		result.MaximumLookbackDays = &value
	}
	for _, action := range source.Actions {
		result.Actions = append(result.Actions, ingestion.ReadAction(action))
	}
	for _, product := range source.Products {
		result.Products = append(result.Products, string(product))
	}
	for _, log := range source.Logs {
		entry := ingestion.CapabilityLog{Product: string(log.Product), Namespace: log.Namespace}
		for _, kind := range log.RecordKinds {
			entry.RecordKinds = append(entry.RecordKinds, ingestion.RecordKind(kind))
		}
		result.Logs = append(result.Logs, entry)
	}
	if err := result.Validate(provider); err != nil {
		return ingestion.Manifest{}, err
	}
	return result, nil
}

func EncodeSyncRequest(token ingestion.JobToken) ([]byte, error) {
	if err := token.Validate(); err != nil {
		return nil, err
	}
	jobID, err := parseUUID(token.JobID)
	if err != nil {
		return nil, err
	}
	connectionID, err := parseUUID(token.ConnectionID)
	if err != nil {
		return nil, err
	}
	request := generated.SyncRequest{
		JobId: jobID, Attempt: token.Attempt, LeaseToken: token.LeaseToken,
		ConnectionId: connectionID, ConnectionGeneration: int64(token.ConnectionGeneration),
		Binding: fromBinding(token.Binding), AdmissionRevision: token.AdmissionRevision,
	}
	if token.Cursor != "" {
		request.Cursor = &token.Cursor
	}
	if !token.ReplayFrom.IsZero() {
		request.ReplayRange = &generated.ReplayRange{
			From: token.ReplayFrom.UTC().Format(time.RFC3339Nano),
			To:   token.ReplayTo.UTC().Format(time.RFC3339Nano),
		}
	}
	return json.Marshal(request)
}

func DecodeResult(data []byte, expected ingestion.JobToken) (ingestion.Result, error) {
	if err := expected.Validate(); err != nil {
		return ingestion.Result{}, err
	}
	var source generated.SyncResult
	if err := decodeStrict(data, &source, validResultShape); err != nil {
		return ingestion.Result{}, err
	}
	if !source.Outcome.Valid() || (source.Page == nil) == (source.Failure == nil) {
		return ingestion.Result{}, ingestion.ErrInvalidContract
	}
	var result ingestion.Result
	var err error
	if source.Outcome == generated.Page && source.Page != nil {
		var page ingestion.Page
		page, err = pageFromGenerated(*source.Page, expected)
		result.Page = &page
	} else if source.Outcome == generated.Failure && source.Failure != nil {
		var failure ingestion.ProviderFailure
		failure, err = failureFromGenerated(*source.Failure, expected)
		result.Failure = &failure
	} else {
		err = ingestion.ErrInvalidContract
	}
	if err != nil || result.Validate() != nil {
		return ingestion.Result{}, ingestion.ErrInvalidContract
	}
	return result, nil
}

func pageFromGenerated(source generated.SyncPage, expected ingestion.JobToken) (ingestion.Page, error) {
	token, err := tokenFromPage(source, expected)
	if err != nil || !ingestion.SameToken(expected, token) {
		return ingestion.Page{}, ingestion.ErrInvalidContract
	}
	coverage, err := coverageFromGenerated(source.Coverage)
	if err != nil {
		return ingestion.Page{}, err
	}
	page := ingestion.Page{Token: token, Complete: source.Complete, Coverage: coverage}
	if source.NextCursor != nil {
		if *source.NextCursor == "" {
			return ingestion.Page{}, ingestion.ErrInvalidContract
		}
		page.NextCursor = *source.NextCursor
	}
	page.Evidence, err = evidenceFromGenerated(source.Evidence)
	if err != nil {
		return ingestion.Page{}, err
	}
	for _, record := range source.Records {
		converted, convertErr := recordFromGenerated(record)
		if convertErr != nil {
			return ingestion.Page{}, convertErr
		}
		page.Records = append(page.Records, converted)
	}
	return page, page.Validate()
}

func failureFromGenerated(source generated.ProviderFailure, expected ingestion.JobToken) (ingestion.ProviderFailure, error) {
	token, err := tokenFromFailure(source, expected)
	if err != nil || !ingestion.SameToken(expected, token) {
		return ingestion.ProviderFailure{}, ingestion.ErrInvalidContract
	}
	evidence, err := evidenceFromGenerated(source.Evidence)
	if err != nil {
		return ingestion.ProviderFailure{}, err
	}
	result := ingestion.ProviderFailure{Token: token, Kind: ingestion.FailureKind(source.Kind), Retryable: source.Retryable, Evidence: evidence}
	if source.RetryAfterSeconds != nil {
		if *source.RetryAfterSeconds < 1 || *source.RetryAfterSeconds > 86400 {
			return ingestion.ProviderFailure{}, ingestion.ErrInvalidContract
		}
		result.RetryAfterSeconds = *source.RetryAfterSeconds
	}
	if source.SafeMessage != nil {
		result.SafeMessage = *source.SafeMessage
	}
	return result, result.Validate()
}

func tokenFromPage(source generated.SyncPage, expected ingestion.JobToken) (ingestion.JobToken, error) {
	binding, err := bindingFromGenerated(source.Binding)
	if err != nil {
		return ingestion.JobToken{}, err
	}
	return ingestion.JobToken{JobID: source.JobId.String(), Attempt: source.Attempt, LeaseToken: source.LeaseToken, ConnectionID: expected.ConnectionID, ConnectionGeneration: uint64(source.ConnectionGeneration), Binding: binding, AdmissionRevision: source.AdmissionRevision, Cursor: source.Cursor, ReplayFrom: expected.ReplayFrom, ReplayTo: expected.ReplayTo}, nil
}

func tokenFromFailure(source generated.ProviderFailure, expected ingestion.JobToken) (ingestion.JobToken, error) {
	binding, err := bindingFromGenerated(source.Binding)
	if err != nil {
		return ingestion.JobToken{}, err
	}
	return ingestion.JobToken{JobID: source.JobId.String(), Attempt: source.Attempt, LeaseToken: source.LeaseToken, ConnectionID: expected.ConnectionID, ConnectionGeneration: uint64(source.ConnectionGeneration), Binding: binding, AdmissionRevision: source.AdmissionRevision, Cursor: source.Cursor, ReplayFrom: expected.ReplayFrom, ReplayTo: expected.ReplayTo}, nil
}

func bindingFromGenerated(source generated.DeploymentBinding) (connections.Binding, error) {
	result := connections.Binding{Provider: string(source.Provider), Environment: source.Environment, AdapterBuildDigest: source.AdapterBuildDigest, CollectorImageDigest: source.CollectorImageDigest, ContractVersion: string(source.ContractVersion), AllowlistRevision: source.AllowlistRevision, NonSecretConfigRevision: source.NonSecretConfigRevision, OperatorPermissionRevision: source.OperatorPermissionRevision}
	if err := result.Validate(); err != nil || result.ContractVersion != ingestion.ContractVersion {
		return connections.Binding{}, ingestion.ErrInvalidContract
	}
	return result, nil
}

func fromBinding(source connections.Binding) generated.DeploymentBinding {
	return generated.DeploymentBinding{Provider: generated.Provider(source.Provider), Environment: source.Environment, AdapterBuildDigest: source.AdapterBuildDigest, CollectorImageDigest: source.CollectorImageDigest, ContractVersion: generated.DeploymentBindingContractVersion(source.ContractVersion), AllowlistRevision: source.AllowlistRevision, NonSecretConfigRevision: source.NonSecretConfigRevision, OperatorPermissionRevision: source.OperatorPermissionRevision}
}

func evidenceFromGenerated(source []generated.EvidenceBlob) ([]ingestion.Evidence, error) {
	result := make([]ingestion.Evidence, 0, len(source))
	for _, raw := range source {
		data, err := base64.StdEncoding.Strict().DecodeString(raw.Data)
		if err != nil || len(data) == 0 || base64.StdEncoding.EncodeToString(data) != raw.Data {
			return nil, ingestion.ErrInvalidContract
		}
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != raw.Sha256 {
			return nil, ingestion.ErrInvalidContract
		}
		entry := ingestion.Evidence{ID: raw.Id, MediaType: string(raw.MediaType), Data: data, Digest: raw.Sha256, Locator: raw.Locator}
		if err = entry.Validate(); err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, nil
}

func coverageFromGenerated(source generated.Coverage) (reporting.Coverage, error) {
	if !ingestion.ValidGapReasons(source.Gaps) {
		return reporting.Coverage{}, ingestion.ErrInvalidContract
	}
	return reporting.NewCoverage(reporting.CoverageState(source.State), source.Gaps)
}

func recordFromGenerated(source generated.IngestionRecord) (ingestion.Record, error) {
	result := ingestion.Record{Kind: ingestion.RecordKind(source.RecordType)}
	count := 0
	if source.Account != nil {
		count++
		accountRecord, convertErr := accountFromGenerated(*source.Account)
		if convertErr != nil {
			return result, convertErr
		}
		result.Account = &accountRecord
	}
	if source.BalanceSnapshot != nil {
		count++
		balance, convertErr := balanceFromGenerated(*source.BalanceSnapshot)
		if convertErr != nil {
			return result, convertErr
		}
		result.Balance = &balance
	}
	if source.Transaction != nil {
		count++
		transaction, convertErr := transactionFromGenerated(*source.Transaction)
		if convertErr != nil {
			return result, convertErr
		}
		result.Transaction = &transaction
	}
	if count != 1 || string(result.Kind) == "" || source.RecordType != generated.RecordKind(result.Kind) {
		return ingestion.Record{}, ingestion.ErrInvalidContract
	}
	switch result.Kind {
	case ingestion.AccountRecordKind:
		if result.Account == nil {
			return ingestion.Record{}, ingestion.ErrInvalidContract
		}
	case ingestion.BalanceRecordKind:
		if result.Balance == nil {
			return ingestion.Record{}, ingestion.ErrInvalidContract
		}
	case ingestion.TransactionRecordKind:
		if result.Transaction == nil {
			return ingestion.Record{}, ingestion.ErrInvalidContract
		}
	default:
		return ingestion.Record{}, ingestion.ErrInvalidContract
	}
	canonical, err := canonicalRecord(source)
	if err != nil {
		return ingestion.Record{}, err
	}
	result.CanonicalPayload = canonical
	return result, nil
}

func canonicalRecord(source generated.IngestionRecord) ([]byte, error) {
	source = normalizeCanonicalRecord(source)
	encoded, err := json.Marshal(source)
	if err != nil {
		return nil, ingestion.ErrInvalidContract
	}
	var envelope map[string]json.RawMessage
	if json.Unmarshal(encoded, &envelope) != nil {
		return nil, ingestion.ErrInvalidContract
	}
	for _, field := range []string{"account", "balanceSnapshot", "transaction"} {
		raw, ok := envelope[field]
		if !ok {
			continue
		}
		var payload map[string]json.RawMessage
		if json.Unmarshal(raw, &payload) != nil {
			return nil, ingestion.ErrInvalidContract
		}
		delete(payload, "evidenceId")
		envelope[field], err = json.Marshal(payload)
		if err != nil {
			return nil, ingestion.ErrInvalidContract
		}
	}
	encoded, err = json.Marshal(envelope)
	if err != nil {
		return nil, ingestion.ErrInvalidContract
	}
	return encoded, nil
}

func normalizeCanonicalRecord(source generated.IngestionRecord) generated.IngestionRecord {
	if source.Account != nil {
		value := *source.Account
		value.Network = nilIfEmpty(value.Network)
		if value.Aliases != nil && len(*value.Aliases) == 0 {
			value.Aliases = nil
		}
		source.Account = &value
	}
	if source.BalanceSnapshot != nil {
		value := *source.BalanceSnapshot
		value.Network = nilIfEmpty(value.Network)
		source.BalanceSnapshot = &value
	}
	if source.Transaction != nil {
		value := *source.Transaction
		value.Merchant = nilIfEmpty(value.Merchant)
		value.Note = nilIfEmpty(value.Note)
		value.Postings = append([]generated.TransactionPosting(nil), value.Postings...)
		for index := range value.Postings {
			value.Postings[index].Network = nilIfEmpty(value.Postings[index].Network)
		}
		source.Transaction = &value
	}
	return source
}

func nilIfEmpty(value *string) *string {
	if value != nil && *value == "" {
		return nil
	}
	return value
}

func accountFromGenerated(source generated.AccountRecord) (ingestion.AccountRecord, error) {
	reference, err := reference(string(source.ExternalAccountId), string(source.Product), optional(source.Network), source.AssetCode)
	if err != nil || !validText(source.LogNamespace) || !validText(source.Name) || !validEvidenceID(source.EvidenceId) {
		return ingestion.AccountRecord{}, ingestion.ErrInvalidContract
	}
	date, err := calendar.ParseDate(source.OpeningDate)
	if err != nil {
		return ingestion.AccountRecord{}, err
	}
	result := ingestion.AccountRecord{Reference: reference, LogNamespace: source.LogNamespace, Name: source.Name, OpeningDate: date, EvidenceID: source.EvidenceId}
	if source.Aliases != nil {
		if len(*source.Aliases) > 32 {
			return result, ingestion.ErrInvalidContract
		}
		for _, alias := range *source.Aliases {
			if !validText(alias.Id) || !validTextLimit(alias.Label, 100) || strings.TrimSpace(alias.Label) == "" || !account.SafeCardAliasLabel(alias.Label, alias.LastFour) {
				return result, ingestion.ErrInvalidContract
			}
			result.Aliases = append(result.Aliases, ingestion.CardAlias{ID: alias.Id, Label: alias.Label, LastFour: alias.LastFour})
		}
	}
	return result, nil
}

func balanceFromGenerated(source generated.BalanceSnapshotRecord) (ingestion.BalanceSnapshot, error) {
	reference, err := reference(source.ExternalAccountId, string(source.Product), optional(source.Network), source.AssetCode)
	if err != nil || !validText(source.LogNamespace) || !validEvidenceID(source.EvidenceId) || source.SourceAsOf == "" {
		return ingestion.BalanceSnapshot{}, ingestion.ErrInvalidContract
	}
	asOf, err := instant(source.SourceAsOf)
	if err != nil {
		return ingestion.BalanceSnapshot{}, err
	}
	coverage, err := coverageFromGenerated(source.Coverage)
	if err != nil {
		return ingestion.BalanceSnapshot{}, err
	}
	freshness, err := reporting.ParseFreshness(string(source.Freshness))
	if err != nil {
		return ingestion.BalanceSnapshot{}, err
	}
	amounts := []generated.SourceAmount{source.Owned, source.Available, source.Locked, source.Debt, source.CreditLimit}
	converted := make([]ingestion.Amount, 0, len(amounts))
	for _, amount := range amounts {
		value, convertErr := amountFromGenerated(amount, source.AssetCode)
		if convertErr != nil {
			return ingestion.BalanceSnapshot{}, convertErr
		}
		converted = append(converted, value)
	}
	return ingestion.BalanceSnapshot{Reference: reference, LogNamespace: source.LogNamespace, SourceAsOf: asOf, Owned: converted[0], Available: converted[1], Locked: converted[2], Debt: converted[3], CreditLimit: converted[4], OwnAvailable: source.OwnAvailable, Coverage: coverage, Freshness: freshness, EvidenceID: source.EvidenceId}, nil
}

func amountFromGenerated(source generated.SourceAmount, assetCode string) (ingestion.Amount, error) {
	if source.AssetCode != assetCode {
		return ingestion.Amount{}, ingestion.ErrInvalidContract
	}
	result := ingestion.Amount{State: reporting.Knowledge(source.State), AssetCode: source.AssetCode}
	if source.Amount != nil {
		result.Value = *source.Amount
	}
	if source.Reason != nil {
		result.Reason = *source.Reason
	}
	if result.State == reporting.Known {
		if !validDecimal(result.Value) || result.Reason != "" {
			return ingestion.Amount{}, ingestion.ErrInvalidContract
		}
	} else if (result.State != reporting.Unknown && result.State != reporting.Unavailable) || result.Value != "" || !validText(result.Reason) {
		return ingestion.Amount{}, ingestion.ErrInvalidContract
	}
	return result, nil
}

func transactionFromGenerated(source generated.TransactionRecord) (ingestion.TransactionRecord, error) {
	if !validText(source.ExternalAccountId) || !ingestion.ValidProduct(string(source.Product)) || !validText(source.LogNamespace) || !validText(source.ProviderRecordId) || !validEvidenceID(source.EvidenceId) || len(source.Postings) > ingestion.MaxRecordsPerPage || source.OccurredAt == "" {
		return ingestion.TransactionRecord{}, ingestion.ErrInvalidContract
	}
	occurredAt, err := instant(source.OccurredAt)
	if err != nil {
		return ingestion.TransactionRecord{}, err
	}
	result := ingestion.TransactionRecord{ExternalAccountID: source.ExternalAccountId, Product: string(source.Product), LogNamespace: source.LogNamespace, ProviderRecordID: source.ProviderRecordId, Classification: string(source.Classification), ProviderState: string(source.ProviderState), EconomicType: string(source.EconomicType), OccurredAt: occurredAt, Merchant: optional(source.Merchant), Note: optional(source.Note), FeeKnowledge: string(source.FeeKnowledge), PnLBasis: optionalEnum(source.PnlBasis), EvidenceID: source.EvidenceId}
	if source.PostedAt != nil {
		result.PostedAt, err = instant(*source.PostedAt)
		if err != nil {
			return ingestion.TransactionRecord{}, err
		}
	}
	if result.Classification != "new" && result.Classification != "correction" && result.Classification != "ambiguous" || !validProviderState(result.ProviderState) || !validEconomicType(result.EconomicType) || (result.FeeKnowledge != "known" && result.FeeKnowledge != "unknown") || !validOptionalText(result.Merchant, ingestion.MaxTextLength) || !validOptionalText(result.Note, ingestion.MaxTextLength) || source.PnlBasis != nil && !source.PnlBasis.Valid() || result.EconomicType != "trade_result" && source.PnlBasis != nil {
		return ingestion.TransactionRecord{}, ingestion.ErrInvalidContract
	}
	for _, posting := range source.Postings {
		ref, convertErr := reference(posting.ExternalAccountId, string(posting.Product), optional(posting.Network), posting.AssetCode)
		if convertErr != nil || !validDecimal(posting.Money) {
			return ingestion.TransactionRecord{}, ingestion.ErrInvalidContract
		}
		p := ingestion.Posting{Reference: ref, Amount: posting.Money, Role: string(posting.Role), Funding: optionalEnum(posting.Funding), Treatment: optionalEnum(posting.Treatment)}
		if posting.Funding != nil && !posting.Funding.Valid() || posting.Treatment != nil && !posting.Treatment.Valid() || !validPosting(p) {
			return ingestion.TransactionRecord{}, ingestion.ErrInvalidContract
		}
		result.Postings = append(result.Postings, p)
	}
	return result, nil
}

func reference(externalID, product, network, assetCode string) (ingestion.AccountReference, error) {
	result := ingestion.AccountReference{ExternalAccountID: externalID, Product: product, Network: network, AssetCode: assetCode}
	if err := result.Validate(); err != nil || !assetSyntax.MatchString(assetCode) {
		return ingestion.AccountReference{}, err
	}
	return result, nil
}

func instant(value string) (calendar.Instant, error) {
	return calendar.ParseInstant(value)
}

func decodeStrict(data []byte, target any, validShape func(any) bool) error {
	if len(data) == 0 || len(data) > MaxEncodedResultBytes || !validJSONUnicode(data) {
		return ingestion.ErrInvalidContract
	}
	var shape any
	shapeDecoder := json.NewDecoder(bytes.NewReader(data))
	shapeDecoder.UseNumber()
	if err := shapeDecoder.Decode(&shape); err != nil || jsonValueContainsNull(shape) || !validShape(shape) {
		return ingestion.ErrInvalidContract
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return ingestion.ErrInvalidContract
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return ingestion.ErrInvalidContract
	}
	return nil
}

func validManifestShape(value any) bool {
	root, ok := requiredObject(value, "provider", "contractVersion", "actions", "products", "logs", "history")
	if !ok {
		return false
	}
	logs, ok := root["logs"].([]any)
	if !ok {
		return false
	}
	for _, value := range logs {
		if _, ok = requiredObject(value, "product", "namespace", "recordKinds"); !ok {
			return false
		}
	}
	_, ok = requiredObject(root["history"], "paginated")
	return ok
}

func validResultShape(value any) bool {
	root, ok := requiredObject(value, "outcome")
	if !ok {
		return false
	}
	outcome, ok := root["outcome"].(string)
	if !ok {
		return false
	}
	field := outcome
	if field != "page" && field != "failure" {
		return false
	}
	payload, ok := requiredObject(root[field], "jobId", "attempt", "leaseToken", "connectionGeneration", "binding", "admissionRevision", "cursor")
	if !ok || !validBindingShape(payload["binding"]) {
		return false
	}
	if field == "failure" {
		if _, ok = requiredObject(payload, "kind", "retryable", "evidence"); !ok {
			return false
		}
		return validEvidenceList(payload["evidence"])
	}
	if _, ok = requiredObject(payload, "complete", "coverage", "evidence", "records"); !ok || !validCoverageShape(payload["coverage"]) || !validEvidenceList(payload["evidence"]) {
		return false
	}
	records, ok := payload["records"].([]any)
	if !ok {
		return false
	}
	for _, record := range records {
		if !validRecordShape(record) {
			return false
		}
	}
	return true
}

func validBindingShape(value any) bool {
	_, ok := requiredObject(value, "provider", "environment", "adapterBuildDigest", "collectorImageDigest", "contractVersion", "allowlistRevision", "nonSecretConfigRevision", "operatorPermissionRevision")
	return ok
}

func validCoverageShape(value any) bool {
	_, ok := requiredObject(value, "state", "gaps")
	return ok
}

func validEvidenceList(value any) bool {
	items, ok := value.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		if _, ok = requiredObject(item, "id", "mediaType", "data", "sha256", "locator"); !ok {
			return false
		}
	}
	return true
}

func validRecordShape(value any) bool {
	record, ok := requiredObject(value, "recordType")
	if !ok {
		return false
	}
	kind, ok := record["recordType"].(string)
	if !ok {
		return false
	}
	switch kind {
	case "account":
		payload, valid := requiredObject(record["account"], "externalAccountId", "product", "logNamespace", "assetCode", "name", "openingDate", "evidenceId")
		if !valid {
			return false
		}
		if aliases, exists := payload["aliases"]; exists {
			items, valid := aliases.([]any)
			if !valid {
				return false
			}
			for _, item := range items {
				if _, valid = requiredObject(item, "id", "label", "lastFour"); !valid {
					return false
				}
			}
		}
		return true
	case "balance_snapshot":
		payload, valid := requiredObject(record["balanceSnapshot"], "externalAccountId", "product", "logNamespace", "assetCode", "sourceAsOf", "owned", "available", "locked", "debt", "creditLimit", "ownAvailable", "coverage", "freshness", "evidenceId")
		if !valid || !validCoverageShape(payload["coverage"]) {
			return false
		}
		for _, field := range []string{"owned", "available", "locked", "debt", "creditLimit"} {
			if !validAmountShape(payload[field]) {
				return false
			}
		}
		return true
	case "transaction":
		payload, valid := requiredObject(record["transaction"], "externalAccountId", "product", "logNamespace", "providerRecordId", "classification", "providerState", "economicType", "occurredAt", "feeKnowledge", "evidenceId", "postings")
		if !valid {
			return false
		}
		postings, valid := payload["postings"].([]any)
		if !valid {
			return false
		}
		for _, posting := range postings {
			if _, valid = requiredObject(posting, "externalAccountId", "product", "assetCode", "money", "role"); !valid {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func validAmountShape(value any) bool {
	amount, ok := requiredObject(value, "state", "assetCode")
	if !ok {
		return false
	}
	state, ok := amount["state"].(string)
	if !ok {
		return false
	}
	if state == "known" {
		_, hasAmount := amount["amount"]
		_, hasReason := amount["reason"]
		return hasAmount && !hasReason
	}
	if state == "unknown" || state == "unavailable" {
		_, hasAmount := amount["amount"]
		_, hasReason := amount["reason"]
		return hasReason && !hasAmount
	}
	return false
}

func requiredObject(value any, fields ...string) (map[string]any, bool) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	for _, field := range fields {
		if _, ok = object[field]; !ok {
			return nil, false
		}
	}
	return object, true
}

func jsonValueContainsNull(value any) bool {
	switch value := value.(type) {
	case nil:
		return true
	case []any:
		for _, item := range value {
			if jsonValueContainsNull(item) {
				return true
			}
		}
	case map[string]any:
		for _, item := range value {
			if jsonValueContainsNull(item) {
				return true
			}
		}
	}
	return false
}

func validJSONUnicode(data []byte) bool {
	if !utf8.Valid(data) {
		return false
	}
	inString := false
	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '"':
			inString = !inString
		case '\\':
			if !inString || i+1 >= len(data) {
				continue
			}
			if data[i+1] != 'u' {
				i++
				continue
			}
			value, ok := escapedCodeUnit(data, i+2)
			if !ok {
				return false
			}
			switch {
			case value >= 0xD800 && value <= 0xDBFF:
				if i+12 > len(data) || data[i+6] != '\\' || data[i+7] != 'u' {
					return false
				}
				low, valid := escapedCodeUnit(data, i+8)
				if !valid || low < 0xDC00 || low > 0xDFFF {
					return false
				}
				i += 11
			case value >= 0xDC00 && value <= 0xDFFF:
				return false
			default:
				i += 5
			}
		}
	}
	return !inString
}

func escapedCodeUnit(data []byte, start int) (uint16, bool) {
	if start+4 > len(data) {
		return 0, false
	}
	var value uint16
	for _, b := range data[start : start+4] {
		value <<= 4
		switch {
		case b >= '0' && b <= '9':
			value += uint16(b - '0')
		case b >= 'a' && b <= 'f':
			value += uint16(b-'a') + 10
		case b >= 'A' && b <= 'F':
			value += uint16(b-'A') + 10
		default:
			return 0, false
		}
	}
	return value, true
}

func validPosting(posting ingestion.Posting) bool {
	switch posting.Role {
	case "principal", "fee", "interest", "funding", "pnl", "reward":
	default:
		return false
	}
	if posting.Funding != "" && posting.Funding != "own" && posting.Funding != "credit" && posting.Funding != "unknown" {
		return false
	}
	if posting.Treatment != "" && posting.Treatment != "movement" && posting.Treatment != "included" && posting.Treatment != "valuation" {
		return false
	}
	return true
}

func validProviderState(value string) bool {
	switch value {
	case "unknown", "draft", "pending", "posted", "cancelled", "reversed":
		return true
	default:
		return false
	}
}

func validEconomicType(value string) bool {
	switch value {
	case "income", "expense", "transfer", "exchange", "refund", "yield", "trade_result":
		return true
	default:
		return false
	}
}

func validDecimal(value string) bool {
	return len(value) <= money.MaxDecimalLength && decimalSyntax.MatchString(value)
}

func validText(value string) bool {
	return validTextLimit(value, ingestion.MaxTextLength) && strings.TrimSpace(value) != ""
}

func validTextLimit(value string, maximum int) bool {
	return value != "" && !strings.ContainsRune(value, 0) && utf8.ValidString(value) && utf8.RuneCountInString(value) <= maximum
}

func validOptionalText(value string, maximum int) bool {
	return value == "" || !strings.ContainsRune(value, 0) && utf8.ValidString(value) && utf8.RuneCountInString(value) <= maximum
}

func validEvidenceID(value string) bool { return validTextLimit(value, 128) }

func optional(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func optionalEnum[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func parseUUID(value string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, ingestion.ErrInvalidContract
	}
	return parsed, nil
}
