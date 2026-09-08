import { createHash } from "node:crypto";

import type { components } from "./generated/ingestion.gen.js";

export type SyncRequest = components["schemas"]["SyncRequest"];
export type SyncResult = components["schemas"]["SyncResult"];
export type CapabilityManifest = components["schemas"]["CapabilityManifest"];

const MAX_SAFE_REVISION = 9_007_199_254_740_991;
const MAX_TEXT = 2_000;
const MAX_RECORDS = 1_000;
const MAX_EVIDENCE = 16;
const MAX_EVIDENCE_BYTES = 10 * 1024 * 1024;
const MAX_ENCODED_RESULT_BYTES = 14 * 1024 * 1024;
const UUID =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const DIGEST = /^sha256:[0-9a-f]{64}$/;
const RAW_DIGEST = /^[0-9a-f]{64}$/;
const DECIMAL = /^-?(0|[1-9][0-9]*)(\.[0-9]+)?$/;
const ASSET = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$/;
const INSTANT =
  /^([0-9]{4})-([0-9]{2})-([0-9]{2})T([0-9]{2}):([0-9]{2}):([0-9]{2})(?:\.([0-9]{1,9}))?Z$/;
const DATE = /^[0-9]{4}-[0-9]{2}-[0-9]{2}$/;

const providers = new Set([
  "alfa",
  "raiffeisen",
  "ozon",
  "bybit",
  "aifory",
  "emcd",
]);
const products = new Set([
  "current",
  "debit_card",
  "credit_card",
  "savings",
  "deposit",
  "wallet",
  "funding",
  "spot",
  "earn",
  "coinhold",
  "crypto_card",
  "futures",
  "mining",
]);
const readActions = new Set([
  "read_accounts",
  "read_balances",
  "read_transactions",
  "read_history",
  "read_rates",
]);
const recordKinds = new Set(["account", "balance_snapshot", "transaction"]);

export class ContractError extends Error {
  constructor() {
    super("invalid ingestion contract");
    this.name = "ContractError";
  }
}

export function parseSyncRequest(value: unknown): SyncRequest {
  const object = strictObject(value, [
    "jobId",
    "attempt",
    "leaseToken",
    "connectionId",
    "connectionGeneration",
    "binding",
    "admissionRevision",
    "cursor",
    "replayRange",
  ]);
  requiredKeys(object, [
    "jobId",
    "attempt",
    "leaseToken",
    "connectionId",
    "connectionGeneration",
    "binding",
    "admissionRevision",
  ]);
  uuid(object.jobId);
  integer(object.attempt, 1, 100);
  text(object.leaseToken);
  uuid(object.connectionId);
  safeRevision(object.connectionGeneration);
  binding(object.binding);
  safeRevision(object.admissionRevision);
  optionalText(object.cursor);
  if (object.replayRange !== undefined) replayRange(object.replayRange);
  return object as SyncRequest;
}

export function parseCapabilityManifest(value: unknown): CapabilityManifest {
  const object = strictObject(value, [
    "provider",
    "contractVersion",
    "actions",
    "products",
    "logs",
    "history",
  ]);
  requiredKeys(object, [
    "provider",
    "contractVersion",
    "actions",
    "products",
    "logs",
    "history",
  ]);
  oneOf(object.provider, providers);
  equal(object.contractVersion, "10");
  uniqueArray(object.actions, 1, 5).forEach((action) =>
    oneOf(action, readActions),
  );
  uniqueArray(object.products, 1, 32).forEach((product) =>
    oneOf(product, products),
  );
  const productSet = new Set(object.products as string[]);
  const logKeys = new Set<string>();
  array(object.logs, 1, 64).forEach((item) => {
    const log = strictObject(item, ["product", "namespace", "recordKinds"]);
    requiredKeys(log, ["product", "namespace", "recordKinds"]);
    oneOf(log.product, products);
    if (!productSet.has(log.product as string)) fail();
    text(log.namespace);
    const key = `${String(log.product)}\u0000${String(log.namespace)}`;
    if (logKeys.has(key)) fail();
    logKeys.add(key);
    uniqueArray(log.recordKinds, 1, 3).forEach((kind) =>
      oneOf(kind, recordKinds),
    );
  });
  const history = strictObject(object.history, [
    "paginated",
    "maximumLookbackDays",
  ]);
  requiredKeys(history, ["paginated"]);
  boolean(history.paginated);
  if (history.maximumLookbackDays !== undefined)
    integer(history.maximumLookbackDays, 1, 36_500);
  return object as CapabilityManifest;
}

export function parseSyncResultJSON(
  textValue: string,
  expected: SyncRequest,
): SyncResult {
  if (Buffer.byteLength(textValue, "utf8") > MAX_ENCODED_RESULT_BYTES) fail();
  let value: unknown;
  try {
    value = JSON.parse(textValue) as unknown;
  } catch {
    fail();
  }
  return parseSyncResult(value, expected);
}

export function parseSyncResult(
  value: unknown,
  expected: SyncRequest,
): SyncResult {
  const validatedExpected = parseSyncRequest(expected);
  const object = strictObject(value, ["outcome", "page", "failure"]);
  requiredKeys(object, ["outcome"]);
  if (object.outcome === "page") {
    if (object.page === undefined || object.failure !== undefined) fail();
    page(object.page, validatedExpected);
  } else if (object.outcome === "failure") {
    if (object.failure === undefined || object.page !== undefined) fail();
    failure(object.failure, validatedExpected);
  } else {
    fail();
  }
  return object as SyncResult;
}

export function capabilitySupports(
  manifest: CapabilityManifest,
  action: CapabilityManifest["actions"][number],
  product: CapabilityManifest["products"][number],
  namespace: string,
  kind: components["schemas"]["RecordKind"],
): boolean {
  parseCapabilityManifest(manifest);
  return supportsCapability(manifest, action, product, namespace, kind);
}

function supportsCapability(
  manifest: CapabilityManifest,
  action: CapabilityManifest["actions"][number],
  product: CapabilityManifest["products"][number],
  namespace: string,
  kind: components["schemas"]["RecordKind"],
): boolean {
  return (
    manifest.actions.includes(action) &&
    manifest.products.includes(product) &&
    manifest.logs.some(
      (log) =>
        log.product === product &&
        log.namespace === namespace &&
        log.recordKinds.includes(kind),
    )
  );
}

function requireCapabilities(
  manifest: CapabilityManifest,
  result: SyncResult,
  request: SyncRequest,
): void {
  if (result.page === undefined) return;
  if (
    request.replayRange !== undefined &&
    !manifest.actions.includes("read_history")
  )
    fail();
  if (result.page.nextCursor !== undefined && !manifest.history.paginated)
    fail();
  if (
    request.replayRange !== undefined &&
    manifest.history.maximumLookbackDays !== undefined
  ) {
    const from = parseInstant(request.replayRange.from);
    const to = parseInstant(request.replayRange.to);
    const maximum =
      BigInt(manifest.history.maximumLookbackDays) *
      24n *
      60n *
      60n *
      1_000_000_000n;
    if (to - from > maximum) fail();
  }
  for (const record of result.page.records) {
    let supported = false;
    if (record.account !== undefined) {
      supported = supportsCapability(
        manifest,
        "read_accounts",
        record.account.product,
        record.account.logNamespace,
        "account",
      );
    } else if (record.balanceSnapshot !== undefined) {
      supported = supportsCapability(
        manifest,
        "read_balances",
        record.balanceSnapshot.product,
        record.balanceSnapshot.logNamespace,
        "balance_snapshot",
      );
    } else if (record.transaction !== undefined) {
      supported = supportsCapability(
        manifest,
        "read_transactions",
        record.transaction.product,
        record.transaction.logNamespace,
        "transaction",
      );
    }
    if (!supported) fail();
  }
}

export class SyntheticGateway {
  readonly #manifest: CapabilityManifest;
  readonly #results: readonly unknown[];
  #index = 0;

  constructor(manifest: unknown, results: readonly unknown[]) {
    this.#manifest = parseCapabilityManifest(manifest);
    this.#results = results;
  }

  manifest(): CapabilityManifest {
    return structuredClone(this.#manifest);
  }

  read(request: unknown): SyncResult {
    const expected = parseSyncRequest(request);
    if (this.#index >= this.#results.length) fail();
    const result = parseSyncResult(this.#results[this.#index], expected);
    requireCapabilities(this.#manifest, result, expected);
    this.#index += 1;
    return structuredClone(result);
  }
}

function page(value: unknown, expected: SyncRequest): void {
  const object = strictObject(value, [
    "jobId",
    "attempt",
    "leaseToken",
    "connectionGeneration",
    "binding",
    "admissionRevision",
    "cursor",
    "nextCursor",
    "complete",
    "coverage",
    "evidence",
    "records",
  ]);
  requiredKeys(object, [
    "jobId",
    "attempt",
    "leaseToken",
    "connectionGeneration",
    "binding",
    "admissionRevision",
    "cursor",
    "complete",
    "coverage",
    "evidence",
    "records",
  ]);
  echoedToken(object, expected);
  textOrEmpty(object.cursor);
  equal(object.cursor, expected.cursor ?? "");
  optionalText(object.nextCursor);
  boolean(object.complete);
  if (object.complete === true && object.nextCursor !== undefined) fail();
  if (
    object.complete === false &&
    (typeof object.nextCursor !== "string" ||
      object.nextCursor === object.cursor)
  )
    fail();
  coverage(object.coverage);
  const evidenceIDs = evidence(object.evidence);
  let postingCount = 0;
  const descriptors = new Set<string>();
  const references = new Set<string>();
  array(object.records, 0, MAX_RECORDS).forEach((record) => {
    const summary = ingestionRecord(record, evidenceIDs);
    postingCount += summary.postingCount;
    if (postingCount > MAX_RECORDS) fail();
    if (summary.descriptor !== undefined) {
      if (descriptors.has(summary.descriptor)) fail();
      descriptors.add(summary.descriptor);
    }
    summary.references.forEach((reference) => references.add(reference));
  });
  references.forEach((reference) => {
    if (!descriptors.has(reference)) fail();
  });
}

function failure(value: unknown, expected: SyncRequest): void {
  const object = strictObject(value, [
    "jobId",
    "attempt",
    "leaseToken",
    "connectionGeneration",
    "binding",
    "admissionRevision",
    "kind",
    "retryable",
    "retryAfterSeconds",
    "safeMessage",
    "evidence",
  ]);
  requiredKeys(object, [
    "jobId",
    "attempt",
    "leaseToken",
    "connectionGeneration",
    "binding",
    "admissionRevision",
    "kind",
    "retryable",
    "evidence",
  ]);
  echoedToken(object, expected);
  const kind = string(object.kind);
  oneOf(
    kind,
    new Set([
      "reauthentication_required",
      "mfa_required",
      "captcha_required",
      "rate_limited",
      "temporary_failure",
      "permanent_failure",
      "unsupported_capability",
      "contract_violation",
    ]),
  );
  boolean(object.retryable);
  optionalText(object.safeMessage);
  if (object.retryAfterSeconds !== undefined)
    integer(object.retryAfterSeconds, 1, 86_400);
  if (
    kind === "rate_limited" &&
    (object.retryable !== true || object.retryAfterSeconds === undefined)
  )
    fail();
  if (kind === "temporary_failure" && object.retryable !== true) fail();
  if (
    [
      "reauthentication_required",
      "mfa_required",
      "captcha_required",
      "permanent_failure",
      "unsupported_capability",
      "contract_violation",
    ].includes(kind) &&
    (object.retryable !== false || object.retryAfterSeconds !== undefined)
  )
    fail();
  evidence(object.evidence);
}

function echoedToken(
  object: Record<string, unknown>,
  expected: SyncRequest,
): void {
  uuid(object.jobId);
  integer(object.attempt, 1, 100);
  text(object.leaseToken);
  safeRevision(object.connectionGeneration);
  binding(object.binding);
  safeRevision(object.admissionRevision);
  if (
    object.jobId !== expected.jobId ||
    object.attempt !== expected.attempt ||
    object.leaseToken !== expected.leaseToken ||
    object.connectionGeneration !== expected.connectionGeneration ||
    object.admissionRevision !== expected.admissionRevision ||
    bindingKey(object.binding as components["schemas"]["DeploymentBinding"]) !==
      bindingKey(expected.binding)
  )
    fail();
}

interface RecordSummary {
  postingCount: number;
  descriptor?: string;
  references: string[];
}

function ingestionRecord(
  value: unknown,
  evidenceIDs: Set<string>,
): RecordSummary {
  const object = strictObject(value, [
    "recordType",
    "account",
    "balanceSnapshot",
    "transaction",
  ]);
  requiredKeys(object, ["recordType"]);
  oneOf(object.recordType, recordKinds);
  const present = [
    object.account,
    object.balanceSnapshot,
    object.transaction,
  ].filter((item) => item !== undefined);
  if (present.length !== 1) fail();
  if (object.recordType === "account" && object.account !== undefined) {
    accountRecord(object.account, evidenceIDs);
    return {
      postingCount: 0,
      descriptor: accountKey(object.account as Record<string, unknown>),
      references: [],
    };
  } else if (
    object.recordType === "balance_snapshot" &&
    object.balanceSnapshot !== undefined
  ) {
    balanceRecord(object.balanceSnapshot, evidenceIDs);
    return {
      postingCount: 0,
      references: [
        accountKey(object.balanceSnapshot as Record<string, unknown>),
      ],
    };
  } else if (
    object.recordType === "transaction" &&
    object.transaction !== undefined
  ) {
    const postings = transactionRecord(object.transaction, evidenceIDs);
    return {
      postingCount: postings.length,
      references: postings.map((item) => accountKey(item)),
    };
  }
  fail();
}

function accountRecord(value: unknown, evidenceIDs: Set<string>): void {
  const object = strictObject(value, [
    "externalAccountId",
    "product",
    "logNamespace",
    "assetCode",
    "network",
    "name",
    "openingDate",
    "aliases",
    "evidenceId",
  ]);
  requiredKeys(object, [
    "externalAccountId",
    "product",
    "logNamespace",
    "assetCode",
    "name",
    "openingDate",
    "evidenceId",
  ]);
  accountReference(object);
  text(object.logNamespace);
  text(object.name);
  date(object.openingDate);
  evidenceReference(object.evidenceId, evidenceIDs);
  if (object.aliases !== undefined)
    array(object.aliases, 0, 32).forEach(cardAlias);
}

function balanceRecord(value: unknown, evidenceIDs: Set<string>): void {
  const object = strictObject(value, [
    "externalAccountId",
    "product",
    "logNamespace",
    "assetCode",
    "network",
    "sourceAsOf",
    "owned",
    "available",
    "locked",
    "debt",
    "creditLimit",
    "ownAvailable",
    "coverage",
    "freshness",
    "evidenceId",
  ]);
  requiredKeys(object, [
    "externalAccountId",
    "product",
    "logNamespace",
    "assetCode",
    "sourceAsOf",
    "owned",
    "available",
    "locked",
    "debt",
    "creditLimit",
    "ownAvailable",
    "coverage",
    "freshness",
    "evidenceId",
  ]);
  accountReference(object);
  text(object.logNamespace);
  instant(object.sourceAsOf);
  for (const field of ["owned", "available", "locked", "debt", "creditLimit"])
    sourceAmount(object[field], object.assetCode);
  boolean(object.ownAvailable);
  coverage(object.coverage);
  oneOf(object.freshness, new Set(["fresh", "stale", "unknown"]));
  evidenceReference(object.evidenceId, evidenceIDs);
}

function transactionRecord(
  value: unknown,
  evidenceIDs: Set<string>,
): Array<Record<string, unknown>> {
  const object = strictObject(value, [
    "externalAccountId",
    "product",
    "logNamespace",
    "providerRecordId",
    "classification",
    "providerState",
    "economicType",
    "occurredAt",
    "postedAt",
    "merchant",
    "note",
    "feeKnowledge",
    "pnlBasis",
    "evidenceId",
    "postings",
  ]);
  requiredKeys(object, [
    "externalAccountId",
    "product",
    "logNamespace",
    "providerRecordId",
    "classification",
    "providerState",
    "economicType",
    "occurredAt",
    "feeKnowledge",
    "evidenceId",
    "postings",
  ]);
  text(object.externalAccountId);
  oneOf(object.product, products);
  text(object.logNamespace);
  text(object.providerRecordId);
  oneOf(object.classification, new Set(["new", "correction", "ambiguous"]));
  oneOf(
    object.providerState,
    new Set(["unknown", "draft", "pending", "posted", "cancelled", "reversed"]),
  );
  oneOf(
    object.economicType,
    new Set([
      "income",
      "expense",
      "transfer",
      "exchange",
      "refund",
      "yield",
      "trade_result",
    ]),
  );
  instant(object.occurredAt);
  if (object.postedAt !== undefined) instant(object.postedAt);
  optionalTextOrEmpty(object.merchant);
  optionalTextOrEmpty(object.note);
  oneOf(object.feeKnowledge, new Set(["known", "unknown"]));
  if (object.pnlBasis !== undefined)
    oneOf(object.pnlBasis, new Set(["gross", "net"]));
  if (object.economicType !== "trade_result" && object.pnlBasis !== undefined)
    fail();
  evidenceReference(object.evidenceId, evidenceIDs);
  const postings = array(object.postings, 0, MAX_RECORDS).map((value) =>
    strictObject(value, [
      "externalAccountId",
      "product",
      "network",
      "assetCode",
      "money",
      "role",
      "funding",
      "treatment",
      "feeId",
    ]),
  );
  postings.forEach(posting);
  return postings;
}

function posting(value: unknown): void {
  const object = strictObject(value, [
    "externalAccountId",
    "product",
    "network",
    "assetCode",
    "money",
    "role",
    "funding",
    "treatment",
    "feeId",
  ]);
  requiredKeys(object, [
    "externalAccountId",
    "product",
    "assetCode",
    "money",
    "role",
  ]);
  accountReference(object);
  decimal(object.money);
  oneOf(
    object.role,
    new Set(["principal", "fee", "interest", "funding", "pnl", "reward"]),
  );
  if (object.funding !== undefined)
    oneOf(object.funding, new Set(["own", "credit", "unknown"]));
  if (object.treatment !== undefined)
    oneOf(object.treatment, new Set(["movement", "included", "valuation"]));
  optionalTextOrEmpty(object.feeId);
}

function evidence(value: unknown): Set<string> {
  const ids = new Set<string>();
  let decodedBytes = 0;
  array(value, 1, MAX_EVIDENCE).forEach((item) => {
    const object = strictObject(item, [
      "id",
      "mediaType",
      "data",
      "sha256",
      "locator",
    ]);
    requiredKeys(object, ["id", "mediaType", "data", "sha256", "locator"]);
    const id = string(object.id);
    if (id.length < 1 || Array.from(id).length > 128 || ids.has(id)) fail();
    ids.add(id);
    oneOf(
      object.mediaType,
      new Set(["application/json", "application/xml", "text/csv", "text/html"]),
    );
    const raw = strictBase64(object.data);
    decodedBytes += raw.length;
    const digest = string(object.sha256);
    if (
      !RAW_DIGEST.test(digest) ||
      createHash("sha256").update(raw).digest("hex") !== digest
    )
      fail();
    text(object.locator);
  });
  if (decodedBytes > MAX_EVIDENCE_BYTES) fail();
  return ids;
}

function sourceAmount(value: unknown, assetCode: unknown): void {
  const object = strictObject(value, [
    "state",
    "assetCode",
    "amount",
    "reason",
  ]);
  requiredKeys(object, ["state", "assetCode"]);
  asset(object.assetCode);
  equal(object.assetCode, assetCode);
  if (object.state === "known") {
    requiredKeys(object, ["amount"]);
    decimal(object.amount);
    if (object.reason !== undefined) fail();
  } else if (object.state === "unknown" || object.state === "unavailable") {
    requiredKeys(object, ["reason"]);
    text(object.reason);
    if (object.amount !== undefined) fail();
  } else fail();
}

function coverage(value: unknown): void {
  const object = strictObject(value, ["state", "gaps"]);
  requiredKeys(object, ["state", "gaps"]);
  oneOf(object.state, new Set(["complete", "partial", "unavailable"]));
  const gaps = uniqueArray(object.gaps, 0, 100);
  gaps.forEach((gap) => text(gap));
  if ((object.state === "complete") !== (gaps.length === 0)) fail();
}

function binding(value: unknown): void {
  const object = strictObject(value, [
    "provider",
    "environment",
    "adapterBuildDigest",
    "collectorImageDigest",
    "contractVersion",
    "allowlistRevision",
    "nonSecretConfigRevision",
    "operatorPermissionRevision",
  ]);
  requiredKeys(object, [
    "provider",
    "environment",
    "adapterBuildDigest",
    "collectorImageDigest",
    "contractVersion",
    "allowlistRevision",
    "nonSecretConfigRevision",
    "operatorPermissionRevision",
  ]);
  oneOf(object.provider, providers);
  text(object.environment, 128);
  if (
    typeof object.adapterBuildDigest !== "string" ||
    !DIGEST.test(object.adapterBuildDigest)
  )
    fail();
  if (
    typeof object.collectorImageDigest !== "string" ||
    !DIGEST.test(object.collectorImageDigest)
  )
    fail();
  equal(object.contractVersion, "10");
  for (const field of [
    "allowlistRevision",
    "nonSecretConfigRevision",
    "operatorPermissionRevision",
  ] as const)
    text(object[field], 128);
}

function bindingKey(value: components["schemas"]["DeploymentBinding"]): string {
  return [
    value.provider,
    value.environment,
    value.adapterBuildDigest,
    value.collectorImageDigest,
    value.contractVersion,
    value.allowlistRevision,
    value.nonSecretConfigRevision,
    value.operatorPermissionRevision,
  ].join("\u0000");
}

function replayRange(value: unknown): void {
  const object = strictObject(value, ["from", "to"]);
  requiredKeys(object, ["from", "to"]);
  const from = parseInstant(object.from);
  const to = parseInstant(object.to);
  if (from >= to || to - from > 90n * 24n * 60n * 60n * 1_000_000_000n) fail();
}

function accountReference(object: Record<string, unknown>): void {
  text(object.externalAccountId);
  oneOf(object.product, products);
  asset(object.assetCode);
  optionalTextOrEmpty(object.network);
}

function accountKey(object: Record<string, unknown>): string {
  return [
    object.externalAccountId,
    object.product,
    object.network ?? "",
    object.assetCode,
  ].join("\u0000");
}

function cardAlias(value: unknown): void {
  const object = strictObject(value, ["id", "label", "lastFour"]);
  requiredKeys(object, ["id", "label", "lastFour"]);
  text(object.id);
  text(object.label, 100);
  if (
    /[0-9][0-9 -]{5,}[0-9]/.test(object.label as string) ||
    typeof object.lastFour !== "string" ||
    !/^[0-9]{4}$/.test(object.lastFour)
  )
    fail();
}

function strictBase64(value: unknown): Buffer {
  if (
    typeof value !== "string" ||
    value.length < 1 ||
    value.length > 13_981_016 ||
    value.length % 4 !== 0 ||
    !/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(
      value,
    )
  )
    fail();
  const raw = Buffer.from(value, "base64");
  if (raw.toString("base64") !== value) fail();
  return raw;
}

function strictObject(
  value: unknown,
  allowed: readonly string[],
): Record<string, unknown> {
  if (value === null || typeof value !== "object" || Array.isArray(value))
    fail();
  const object = value as Record<string, unknown>;
  const allowedSet = new Set(allowed);
  if (Object.keys(object).some((key) => !allowedSet.has(key))) fail();
  return object;
}

function requiredKeys(
  object: Record<string, unknown>,
  keys: readonly string[],
): void {
  if (keys.some((key) => object[key] === undefined)) fail();
}

function array(value: unknown, minimum: number, maximum: number): unknown[] {
  if (!Array.isArray(value) || value.length < minimum || value.length > maximum)
    fail();
  return value;
}

function uniqueArray(
  value: unknown,
  minimum: number,
  maximum: number,
): unknown[] {
  const values = array(value, minimum, maximum);
  if (new Set(values).size !== values.length) fail();
  return values;
}

function text(value: unknown, maximum = MAX_TEXT): asserts value is string {
  if (
    typeof value !== "string" ||
    value.length < 1 ||
    Array.from(value).length > maximum
  )
    fail();
}

function textOrEmpty(value: unknown): asserts value is string {
  if (typeof value !== "string" || Array.from(value).length > MAX_TEXT) fail();
}

function optionalText(value: unknown): void {
  if (value !== undefined) text(value);
}

function optionalTextOrEmpty(value: unknown): void {
  if (value !== undefined) textOrEmpty(value);
}

function string(value: unknown): string {
  if (typeof value !== "string") fail();
  return value;
}

function boolean(value: unknown): asserts value is boolean {
  if (typeof value !== "boolean") fail();
}

function integer(
  value: unknown,
  minimum: number,
  maximum: number,
): asserts value is number {
  if (
    typeof value !== "number" ||
    !Number.isSafeInteger(value) ||
    value < minimum ||
    value > maximum
  )
    fail();
}

function safeRevision(value: unknown): void {
  integer(value, 1, MAX_SAFE_REVISION);
}

function decimal(value: unknown): void {
  if (typeof value !== "string" || value.length > 256 || !DECIMAL.test(value))
    fail();
}

function asset(value: unknown): void {
  if (typeof value !== "string" || !ASSET.test(value)) fail();
}

function uuid(value: unknown): void {
  if (typeof value !== "string" || !UUID.test(value)) fail();
}

function instant(value: unknown): void {
  parseInstant(value);
}

function parseInstant(value: unknown): bigint {
  if (typeof value !== "string") fail();
  const match = INSTANT.exec(value);
  if (match === null) fail();
  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  const hour = Number(match[4]);
  const minute = Number(match[5]);
  const second = Number(match[6]);
  if (
    month < 1 ||
    month > 12 ||
    day < 1 ||
    day > daysInMonth(year, month) ||
    hour > 23 ||
    minute > 59 ||
    second > 59
  )
    fail();
  const parsed = new Date(0);
  parsed.setUTCFullYear(year, month - 1, day);
  parsed.setUTCHours(hour, minute, second, 0);
  if (!Number.isFinite(parsed.valueOf())) fail();
  const fraction = (match[7] ?? "").padEnd(9, "0");
  return BigInt(parsed.valueOf()) * 1_000_000n + BigInt(fraction || "0");
}

function daysInMonth(year: number, month: number): number {
  if (month === 2)
    return year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0) ? 29 : 28;
  return [4, 6, 9, 11].includes(month) ? 30 : 31;
}

function date(value: unknown): void {
  if (typeof value !== "string" || !DATE.test(value)) fail();
  const parsed = new Date(`${value}T00:00:00Z`);
  if (
    Number.isNaN(parsed.valueOf()) ||
    parsed.toISOString().slice(0, 10) !== value
  )
    fail();
}

function evidenceReference(value: unknown, evidenceIDs: Set<string>): void {
  if (typeof value !== "string" || !evidenceIDs.has(value)) fail();
}

function oneOf(value: unknown, values: ReadonlySet<string>): void {
  if (typeof value !== "string" || !values.has(value)) fail();
}

function equal(left: unknown, right: unknown): void {
  if (left !== right) fail();
}

function fail(): never {
  throw new ContractError();
}
