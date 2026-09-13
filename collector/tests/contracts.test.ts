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

    const wrongCursor = structuredClone(golden) as {
      page: Record<string, unknown>;
    };
    wrongCursor.page.cursor = "another-page";
    expect(() =>
      parseSyncResult(wrongCursor, parseSyncRequest(request)),
    ).toThrow("invalid ingestion contract");
  });

  it("enforces the manifest against every returned record", () => {
    const missingBalances = structuredClone(manifest) as { actions: string[] };
    missingBalances.actions = missingBalances.actions.filter(
      (action) => action !== "read_balances",
    );
    expect(() =>
      new SyntheticGateway(missingBalances, [golden]).read(request),
    ).toThrow("invalid ingestion contract");

    const wrongNamespace = structuredClone(golden) as {
      page: { records: Array<{ account?: { logNamespace: string } }> };
    };
    const account = wrongNamespace.page.records.find(
      (record) => record.account !== undefined,
    )?.account;
    if (account === undefined) throw new Error("invalid test fixture");
    account.logNamespace = "undeclared";
    expect(() =>
      new SyntheticGateway(manifest, [wrongNamespace]).read(request),
    ).toThrow("invalid ingestion contract");

    const wrongProvider = structuredClone(manifest) as { provider: string };
    wrongProvider.provider = "alfa";
    expect(() =>
      new SyntheticGateway(wrongProvider, [golden]).read(request),
    ).toThrow("invalid ingestion contract");
  });

  it("owns immutable manifest and result snapshots", () => {
    const mutableManifest = structuredClone(manifest) as {
      provider: string;
      actions: string[];
    };
    const mutableGolden = structuredClone(golden) as {
      page: { leaseToken: string };
    };
    const gateway = new SyntheticGateway(mutableManifest, [mutableGolden]);
    mutableManifest.provider = "alfa";
    mutableManifest.actions.length = 0;
    mutableGolden.page.leaseToken = "mutated";

    const exposed = gateway.manifest() as { actions: string[] };
    exposed.actions.length = 0;
    expect(gateway.read(request).outcome).toBe("page");
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

  it("keeps manifest and coverage limits identical across runtimes", () => {
    const zeroLookback = structuredClone(manifest) as {
      history: { maximumLookbackDays?: number };
    };
    zeroLookback.history.maximumLookbackDays = 0;
    expect(() => parseCapabilityManifest(zeroLookback)).toThrow();
    delete zeroLookback.history.maximumLookbackDays;
    expect(() => parseCapabilityManifest(zeroLookback)).not.toThrow();

    const blankNamespace = structuredClone(manifest) as {
      logs: Array<{ namespace: string }>;
    };
    blankNamespace.logs[0]!.namespace = " \t";
    expect(() => parseCapabilityManifest(blankNamespace)).toThrow();

    for (const gaps of [
      ["gap", "gap"],
      Array.from({ length: 101 }, (_, index) => `gap-${index}`),
      ["invalid\u0000gap"],
      ["ё".repeat(2_001)],
    ]) {
      const invalid = structuredClone(golden) as {
        page: { coverage: { gaps: string[] } };
      };
      invalid.page.coverage.gaps = gaps;
      expect(() =>
        parseSyncResult(invalid, parseSyncRequest(request)),
      ).toThrow();
    }
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

    const invalidInstant = structuredClone(golden) as {
      page: {
        records: Array<{ balanceSnapshot?: { sourceAsOf: string } }>;
      };
    };
    const snapshot = invalidInstant.page.records.find(
      (record) => record.balanceSnapshot !== undefined,
    )?.balanceSnapshot;
    if (snapshot === undefined) throw new Error("invalid test fixture");
    snapshot.sourceAsOf = "2026-02-30T00:00:00Z";
    expect(() =>
      parseSyncResult(invalidInstant, parseSyncRequest(request)),
    ).toThrow();
    snapshot.sourceAsOf = "2026-09-08T09:00:00+00:00";
    expect(() =>
      parseSyncResult(invalidInstant, parseSyncRequest(request)),
    ).toThrow();
    snapshot.sourceAsOf = "0000-01-01T00:00:00Z";
    expect(() =>
      parseSyncResult(invalidInstant, parseSyncRequest(request)),
    ).toThrow();

    account.openingDate = "0000-01-01";
    expect(() =>
      parseSyncResult(invalidDate, parseSyncRequest(request)),
    ).toThrow();

    const unicode = structuredClone(golden) as {
      page: { records: Array<{ account?: { name: string } }> };
    };
    const namedAccount = unicode.page.records.find(
      (record) => record.account !== undefined,
    )?.account;
    if (namedAccount === undefined) throw new Error("invalid test fixture");
    namedAccount.name = "ё".repeat(1_500);
    expect(() =>
      parseSyncResult(unicode, parseSyncRequest(request)),
    ).not.toThrow();
    namedAccount.name = "ё".repeat(2_001);
    expect(() => parseSyncResult(unicode, parseSyncRequest(request))).toThrow();
    namedAccount.name = "invalid\ud800text";
    expect(() => parseSyncResult(unicode, parseSyncRequest(request))).toThrow();
    namedAccount.name = "invalid\u0000text";
    expect(() => parseSyncResult(unicode, parseSyncRequest(request))).toThrow();

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

    const lineBroken = structuredClone(golden) as {
      page: { evidence: Array<{ data: string }> };
    };
    const encodedEvidence = lineBroken.page.evidence[0]!.data;
    lineBroken.page.evidence[0]!.data = `${encodedEvidence.slice(0, 4)}\n${encodedEvidence.slice(4)}`;
    expect(() =>
      parseSyncResult(lineBroken, parseSyncRequest(request)),
    ).toThrow();

    const emptyNextCursor = structuredClone(golden) as {
      page: { nextCursor?: string };
    };
    emptyNextCursor.page.nextCursor = "";
    expect(() =>
      parseSyncResult(emptyNextCursor, parseSyncRequest(request)),
    ).toThrow();

    const invalidEvidenceIdentity = structuredClone(golden) as {
      page: { evidence: Array<{ id: string }> };
    };
    invalidEvidenceIdentity.page.evidence[0]!.id = "invalid\ud800id";
    expect(() =>
      parseSyncResult(invalidEvidenceIdentity, parseSyncRequest(request)),
    ).toThrow();

    const unsupportedFeeID = structuredClone(golden) as {
      page: {
        records: Array<{
          transaction?: { postings: Array<Record<string, unknown>> };
        }>;
      };
    };
    const posting = unsupportedFeeID.page.records.find(
      (record) => record.transaction?.postings[0] !== undefined,
    )?.transaction?.postings[0];
    if (posting === undefined) throw new Error("invalid test fixture");
    posting.feeId = "fee-1";
    expect(() =>
      parseSyncResult(unsupportedFeeID, parseSyncRequest(request)),
    ).toThrow();

    expect(() =>
      parseSyncResultJSON(
        `${JSON.stringify(golden)} {}`,
        parseSyncRequest(request),
      ),
    ).toThrow();
  });

  it("allows only a descriptive card label and its exact last four", () => {
    const aliased = structuredClone(golden) as {
      page: {
        records: Array<{
          account?: { aliases?: Array<{ label: string; lastFour: string }> };
        }>;
      };
    };
    const alias = aliased.page.records.find(
      (record) => record.account?.aliases?.[0] !== undefined,
    )?.account?.aliases?.[0];
    if (alias === undefined) throw new Error("invalid test fixture");

    for (const label of [
      "4242.4242.4242.4242",
      "4242/4242/4242/4242",
      "4242\u200b4242\u200b4242\u200b4242",
      "٤٢٤٢",
      "Card 123",
    ]) {
      alias.label = label;
      expect(() => parseSyncResult(aliased, parseSyncRequest(request))).toThrow(
        "invalid ingestion contract",
      );
    }
    alias.label = "Business card 4242";
    expect(() =>
      parseSyncResult(aliased, parseSyncRequest(request)),
    ).not.toThrow();
  });

  it("rejects write capabilities and inconsistent failure semantics", () => {
    const write = structuredClone(manifest) as { actions: unknown[] };
    write.actions.push("create_payment");
    expect(() => parseCapabilityManifest(write)).toThrow();
    const expectedRequest = parseSyncRequest(request);

    const failure = {
      outcome: "failure",
      failure: {
        jobId: request.jobId,
        attempt: request.attempt,
        leaseToken: request.leaseToken,
        connectionGeneration: request.connectionGeneration,
        binding: request.binding,
        admissionRevision: request.admissionRevision,
        cursor: expectedRequest.cursor ?? "",
        kind: "mfa_required",
        retryable: true,
        evidence: (golden as { page: { evidence: unknown[] } }).page.evidence,
      },
    };
    expect(() => parseSyncResult(failure, expectedRequest)).toThrow();

    const validFailure = structuredClone(failure) as {
      failure: { retryable: boolean; cursor?: string; safeMessage?: string };
    };
    validFailure.failure.retryable = false;
    validFailure.failure.safeMessage = "";
    expect(() => parseSyncResult(validFailure, expectedRequest)).not.toThrow();
    validFailure.failure.cursor = "stale-cursor";
    expect(() => parseSyncResult(validFailure, expectedRequest)).toThrow(
      "invalid ingestion contract",
    );
    delete validFailure.failure.cursor;
    expect(() => parseSyncResult(validFailure, expectedRequest)).toThrow(
      "invalid ingestion contract",
    );
  });
});
