import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";

import {
  capabilitySupports,
  parseCapabilityManifest,
  parseSyncRequest,
  parseSyncResult,
  parseSyncResultJSON,
  SyntheticGateway,
} from "../src/index.js";

const fixture = (name: string): unknown =>
  JSON.parse(
    readFileSync(
      new URL(`../contracts/v10/fixtures/${name}`, import.meta.url),
      "utf8",
    ),
  ) as unknown;

const manifest = fixture("manifest.json");
const golden = fixture("golden-page.json");
const page = (golden as { page: Record<string, unknown> }).page;
const request = {
  jobId: page.jobId,
  attempt: page.attempt,
  leaseToken: page.leaseToken,
  connectionId: "22222222-2222-4222-8222-222222222222",
  connectionGeneration: page.connectionGeneration,
  binding: page.binding,
  admissionRevision: page.admissionRevision,
};

describe("collector ingestion contract v10", () => {
  it("checks capability meaning and exact server-issued tokens", () => {
    const parsedManifest = parseCapabilityManifest(manifest);
    expect(
      capabilitySupports(
        parsedManifest,
        "read_transactions",
        "wallet",
        "trades",
        "transaction",
      ),
    ).toBe(true);
    expect(
      capabilitySupports(
        parsedManifest,
        "read_rates",
        "wallet",
        "trades",
        "transaction",
      ),
    ).toBe(false);

    const gateway = new SyntheticGateway(manifest, [golden]);
    const result = gateway.read(request);
    expect(result.outcome).toBe("page");
    expect(result.page?.records).toHaveLength(17);

    const stale = structuredClone(golden) as {
      page: Record<string, unknown>;
    };
    stale.page.admissionRevision = 4;
    expect(() => parseSyncResult(stale, parseSyncRequest(request))).toThrow(
      "invalid ingestion contract",
    );
  });

  it("keeps decimal values as strings and unsupported assets explicit", () => {
    const result = parseSyncResult(golden, parseSyncRequest(request));
    const records = result.page?.records ?? [];
    const balances = records.flatMap((record) =>
      record.balanceSnapshot === undefined ? [] : [record.balanceSnapshot],
    );
    expect(balances.map((record) => record.assetCode)).toEqual([
      "RUB",
      "USD",
      "USDT",
      "USDC",
      "BTC",
      "ETH",
      "USDC.E",
    ]);
    expect(balances[3]?.owned.amount).toBe("0.123456789123456789");
    expect(balances[4]?.owned.amount).toBe("0.000000000000000001");
    expect(balances[1]?.available).toEqual({
      state: "unknown",
      assetCode: "USD",
      reason: "provider_field_missing",
    });
  });

  it("rejects unknown fields, numeric money, malformed base64 and trailing JSON", () => {
    const extra = structuredClone(golden) as { page: Record<string, unknown> };
    extra.page.householdId = "forged";
    expect(() => parseSyncResult(extra, parseSyncRequest(request))).toThrow();

    const numeric = structuredClone(golden) as {
      page: {
        records: Array<{ balanceSnapshot?: { owned: { amount?: unknown } } }>;
      };
    };
    const balance = numeric.page.records.find(
      (record) => record.balanceSnapshot !== undefined,
    )?.balanceSnapshot;
    if (balance === undefined) throw new Error("invalid test fixture");
    balance.owned.amount = 1;
    expect(() => parseSyncResult(numeric, parseSyncRequest(request))).toThrow();

    const invalidDate = structuredClone(golden) as {
      page: { records: Array<{ account?: { openingDate: string } }> };
    };
    const account = invalidDate.page.records.find(
      (record) => record.account !== undefined,
    )?.account;
    if (account === undefined) throw new Error("invalid test fixture");
    account.openingDate = "2026-02-30";
    expect(() =>
      parseSyncResult(invalidDate, parseSyncRequest(request)),
    ).toThrow();

    const missingDescriptor = structuredClone(golden) as {
      page: { records: Array<{ recordType: string }> };
    };
    missingDescriptor.page.records = missingDescriptor.page.records.filter(
      (_, index) => index !== 0,
    );
    expect(() =>
      parseSyncResult(missingDescriptor, parseSyncRequest(request)),
    ).toThrow();

    const malformed = structuredClone(golden) as {
      page: { evidence: Array<{ data: string }> };
    };
    malformed.page.evidence[0]!.data = "%%%=";
    expect(() =>
      parseSyncResult(malformed, parseSyncRequest(request)),
    ).toThrow();

    expect(() =>
      parseSyncResultJSON(
        `${JSON.stringify(golden)} {}`,
        parseSyncRequest(request),
      ),
    ).toThrow();
  });

  it("rejects write capabilities and inconsistent failure semantics", () => {
    const write = structuredClone(manifest) as { actions: unknown[] };
    write.actions.push("create_payment");
    expect(() => parseCapabilityManifest(write)).toThrow();

    const failure = {
      outcome: "failure",
      failure: {
        jobId: request.jobId,
        attempt: request.attempt,
        leaseToken: request.leaseToken,
        connectionGeneration: request.connectionGeneration,
        binding: request.binding,
        admissionRevision: request.admissionRevision,
        kind: "mfa_required",
        retryable: true,
        evidence: (golden as { page: { evidence: unknown[] } }).page.evidence,
      },
    };
    expect(() => parseSyncResult(failure, parseSyncRequest(request))).toThrow();
  });
});
