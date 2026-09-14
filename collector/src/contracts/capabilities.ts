import type { components } from "./generated/ingestion.gen.js";
import {
  products,
  providers,
  readActions,
  recordKinds,
  type CapabilityManifest,
  type SyncRequest,
  type SyncResult,
} from "./types.js";
import {
  array,
  boolean,
  equal,
  fail,
  integer,
  nonblankText,
  oneOf,
  parseInstant,
  requiredKeys,
  strictObject,
  uniqueArray,
} from "./validation.js";

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
    nonblankText(log.namespace);
    const key = JSON.stringify([log.product, log.namespace]);
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

export function requireCapabilities(
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
