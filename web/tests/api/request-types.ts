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
