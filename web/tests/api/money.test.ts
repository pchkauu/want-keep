import { describe, expect, it } from "vitest";
import fixtures from "../../../api/fixtures/money.json";
import { decodeMoney } from "../../src/api/money";
import type { components } from "../../src/api/generated/openapi.gen";

describe("money transport", () => {
  it.each(fixtures)("preserves $asset $amount through JSON", (fixture) => {
    const result = decodeMoney(JSON.parse(JSON.stringify(fixture)));
    expect(result).toEqual(fixture);
    expect(typeof result.amount).toBe("string");
  });

  it.each([
    0.1,
    null,
    undefined,
    "",
    "1e2",
    "NaN",
    "+1",
    "01",
    " 1",
    "1\n",
    "9".repeat(257),
  ])("rejects invalid amount %s", (amount) => {
    expect(() => decodeMoney({ amount, asset: "BTC" })).toThrow(
      "invalid_money",
    );
  });

  it("does not equate source aliases or bridged assets", () => {
    expect(() => decodeMoney({ amount: "1", asset: "USDC.E" })).toThrow(
      "unsupported_asset",
    );
    expect(() => decodeMoney({ amount: "1", asset: "RUR" })).toThrow(
      "unsupported_asset",
    );
    expect(() =>
      decodeMoney({ amount: "1", asset: "USD", actorId: "forged" }),
    ).toThrow("invalid_money");
  });

  it("keeps missing and known zero distinct", () => {
    const known: components["schemas"]["AmountValue"] = {
      knowledge: "known",
      value: { amount: "0", asset: "RUB" },
    };
    const missing: components["schemas"]["AmountValue"] = {
      knowledge: "unknown",
      reason: "source_missing",
    };
    expect("value" in known).toBe(true);
    expect("value" in missing).toBe(false);
  });
});
