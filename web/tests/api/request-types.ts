import type { components } from "../../src/api/generated/openapi.gen";

type Correction = components["schemas"]["TransactionCorrection"];
type Message = components["schemas"]["MessageCreate"];
type Rule = components["schemas"]["RuleInput"];

export const correction: Correction = {
  expectedRevision: 1,
  reason: "Correct receipt amount",
  amount: { amount: "0.000000000000000001", asset: "ETH" },
};

export const numericMoney: Correction = {
  expectedRevision: 1,
  reason: "Invalid numeric transport",
  amount: {
    // @ts-expect-error Regression assertion: generated money must stay a string.
    amount: 0.1,
    asset: "ETH",
  },
};

export const unsupportedAsset: Correction = {
  expectedRevision: 1,
  reason: "Invalid source alias",
  amount: {
    amount: "1",
    // @ts-expect-error Regression assertion: external aliases are not domain assets.
    asset: "USDC.E",
  },
};

// @ts-expect-error Regression assertion: correction requires expectedRevision.
export const missingRevision: Correction = { reason: "Missing revision" };

export const forgedActor: Correction = {
  expectedRevision: 1,
  reason: "Invalid actor assignment",
  // @ts-expect-error Regression assertion: command input cannot choose the actor.
  actorId: "10000000-0000-4000-8000-000000000001",
};

export const message: Message = { text: "A purchase", attachmentIds: [] };
// @ts-expect-error Regression assertion: text must be a string.
export const numericMessage: Message = { text: 123, attachmentIds: [] };
// @ts-expect-error Regression assertion: message fields must not become optional unknown.
export const emptyMessage: Message = {};

export const rule: Rule = {
  name: "Groceries",
  conditions: [
    { field: "merchant", operator: "equals", value: "Example shop" },
  ],
  categoryId: "10000000-0000-4000-8000-000000000001",
  applyTo: "future",
};
// @ts-expect-error Regression assertion: a generated rule has required typed fields.
export const emptyRule: Rule = {};

type Gate = components["schemas"]["DeploymentGate"];
export const pendingGate: Gate = {
  status: "pending",
  reasons: ["provider_pending"],
};
// @ts-expect-error Admitted requires the complete server binding and check time.
export const incompleteAdmission: Gate = { status: "admitted", reasons: [] };
type AdmittedGate = components["schemas"]["AdmittedDeploymentGate"];
export const admittedGate: AdmittedGate = {
  status: "admitted",
  binding: {
    provider: "bybit",
    environment: "production",
    adapterBuildDigest: "sha256:" + "a".repeat(64),
    collectorImageDigest: "sha256:" + "b".repeat(64),
    contractVersion: "10",
    allowlistRevision: "1",
    nonSecretConfigRevision: "1",
    operatorPermissionRevision: "1",
  },
  admissionRevision: 3,
  checkedAt: "2026-09-07T00:00:00Z",
  reasons: [],
};
export const { admissionRevision, ...unversionedGate } = admittedGate;
// @ts-expect-error A configured admission requires its own revision.
export const missingAdmissionRevision: Gate = unversionedGate;
export const stringAdmissionRevision: AdmittedGate = {
  ...admittedGate,
  // @ts-expect-error Revisions use exact safe integers, not strings.
  admissionRevision: "3",
};
export const forgedAdmissionRevision: components["schemas"]["ConnectionAction"] =
  {
    expectedRevision: 1,
    // @ts-expect-error A command cannot assign the server admission revision.
    admissionRevision: 3,
  };
export const forgedAdmission: components["schemas"]["ConnectionAction"] = {
  expectedRevision: 1,
  // @ts-expect-error User commands cannot assign server admission.
  deploymentGate: pendingGate,
};
export const expired: components["schemas"]["ExpiredCommand"] = {
  version: "1",
  code: "command_expired",
  message: "Details expired",
  violations: [],
  retryable: false,
  correlationId: "10000000-0000-4000-8000-000000000001",
};
// @ts-expect-error Expired command responses require the safe error envelope.
export const emptyExpired: components["schemas"]["ExpiredCommand"] = {};
export const numericReturn: components["schemas"]["KnownReturn"] = {
  state: "known",
  // @ts-expect-error XIRR remains a decimal string.
  ratio: 0.1,
};
// @ts-expect-error A quote must describe amount, direction, time, fees and spread.
export const emptyQuote: components["schemas"]["PlatformQuote"] = {
  method: "platform_quote",
};
