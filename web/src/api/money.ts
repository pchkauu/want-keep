import type { components } from "./generated/openapi.gen";

export type Money = components["schemas"]["Money"];

const assets = new Set<Money["asset"]>([
  "RUB",
  "USD",
  "USDT",
  "USDC",
  "BTC",
  "ETH",
]);
const decimal = /^-?(0|[1-9][0-9]*)(\.[0-9]+)?$/;

export function decodeMoney(input: unknown): Money {
  if (typeof input !== "object" || input === null || Array.isArray(input)) {
    throw new TypeError("invalid_money");
  }
  if (!("amount" in input) || !("asset" in input)) {
    throw new TypeError("invalid_money");
  }
  const { amount, asset } = input;
  if (
    typeof amount !== "string" ||
    amount.length > 256 ||
    !decimal.test(amount)
  ) {
    throw new TypeError("invalid_money");
  }
  if (typeof asset !== "string" || !assets.has(asset as Money["asset"])) {
    throw new TypeError("unsupported_asset");
  }
  if (Object.keys(input).some((key) => key !== "amount" && key !== "asset")) {
    throw new TypeError("invalid_money");
  }
  return { amount, asset: asset as Money["asset"] };
}
