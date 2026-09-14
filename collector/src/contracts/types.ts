import type { components } from "./generated/ingestion.gen.js";

export type SyncRequest = components["schemas"]["SyncRequest"];
export type SyncResult = components["schemas"]["SyncResult"];
export type CapabilityManifest = components["schemas"]["CapabilityManifest"];

export const providers = new Set([
  "alfa",
  "raiffeisen",
  "ozon",
  "bybit",
  "aifory",
  "emcd",
]);
export const products = new Set([
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
export const readActions = new Set([
  "read_accounts",
  "read_balances",
  "read_transactions",
  "read_history",
  "read_rates",
]);
export const recordKinds = new Set([
  "account",
  "balance_snapshot",
  "transaction",
]);
