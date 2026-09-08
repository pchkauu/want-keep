export const MAX_SAFE_REVISION = 9_007_199_254_740_991;
export const MAX_TEXT = 2_000;
export const MAX_RECORDS = 1_000;
export const MAX_EVIDENCE = 16;
export const MAX_EVIDENCE_BYTES = 10 * 1024 * 1024;
export const MAX_ENCODED_RESULT_BYTES = 14 * 1024 * 1024;
export const UUID =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
export const DIGEST = /^sha256:[0-9a-f]{64}$/;
export const RAW_DIGEST = /^[0-9a-f]{64}$/;
export const DECIMAL = /^-?(0|[1-9][0-9]*)(\.[0-9]+)?$/;
export const ASSET = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$/;
export const INSTANT =
  /^([0-9]{4})-([0-9]{2})-([0-9]{2})T([0-9]{2}):([0-9]{2}):([0-9]{2})(?:\.([0-9]{1,9}))?Z$/;
export const DATE = /^[0-9]{4}-[0-9]{2}-[0-9]{2}$/;

export class ContractError extends Error {
  constructor() {
    super("invalid ingestion contract");
    this.name = "ContractError";
  }
}

export function strictBase64(value: unknown): Buffer {
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

export function strictObject(
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

export function requiredKeys(
  object: Record<string, unknown>,
  keys: readonly string[],
): void {
  if (keys.some((key) => object[key] === undefined)) fail();
}

export function array(
  value: unknown,
  minimum: number,
  maximum: number,
): unknown[] {
  if (!Array.isArray(value) || value.length < minimum || value.length > maximum)
    fail();
  return value;
}

export function uniqueArray(
  value: unknown,
  minimum: number,
  maximum: number,
): unknown[] {
  const values = array(value, minimum, maximum);
  if (new Set(values).size !== values.length) fail();
  return values;
}

export function text(
  value: unknown,
  maximum = MAX_TEXT,
): asserts value is string {
  if (
    typeof value !== "string" ||
    value.length < 1 ||
    value.includes("\u0000") ||
    !hasWellFormedUnicode(value) ||
    Array.from(value).length > maximum
  )
    fail();
}

export function textOrEmpty(value: unknown): asserts value is string {
  if (
    typeof value !== "string" ||
    value.includes("\u0000") ||
    !hasWellFormedUnicode(value) ||
    Array.from(value).length > MAX_TEXT
  )
    fail();
}

export function optionalText(value: unknown): void {
  if (value !== undefined) text(value);
}

export function optionalTextOrEmpty(value: unknown): void {
  if (value !== undefined) textOrEmpty(value);
}

export function string(value: unknown): string {
  if (typeof value !== "string") fail();
  return value;
}

export function boolean(value: unknown): asserts value is boolean {
  if (typeof value !== "boolean") fail();
}

export function integer(
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

export function safeRevision(value: unknown): void {
  integer(value, 1, MAX_SAFE_REVISION);
}

export function decimal(value: unknown): void {
  if (typeof value !== "string" || value.length > 256 || !DECIMAL.test(value))
    fail();
}

export function asset(value: unknown): void {
  if (typeof value !== "string" || !ASSET.test(value)) fail();
}

export function uuid(value: unknown): void {
  if (typeof value !== "string" || !UUID.test(value)) fail();
}

export function instant(value: unknown): void {
  parseInstant(value);
}

export function parseInstant(value: unknown): bigint {
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
    year < 1 ||
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

export function daysInMonth(year: number, month: number): number {
  if (month === 2)
    return year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0) ? 29 : 28;
  return [4, 6, 9, 11].includes(month) ? 30 : 31;
}

export function date(value: unknown): void {
  if (typeof value !== "string" || !DATE.test(value)) fail();
  const [year, month, day] = value.split("-").map(Number);
  if (
    year === undefined ||
    month === undefined ||
    day === undefined ||
    year < 1 ||
    month < 1 ||
    month > 12 ||
    day < 1 ||
    day > daysInMonth(year, month)
  )
    fail();
}

function hasWellFormedUnicode(value: string): boolean {
  for (let index = 0; index < value.length; index += 1) {
    const unit = value.charCodeAt(index);
    if (unit >= 0xd800 && unit <= 0xdbff) {
      const next = value.charCodeAt(index + 1);
      if (next < 0xdc00 || next > 0xdfff) return false;
      index += 1;
    } else if (unit >= 0xdc00 && unit <= 0xdfff) {
      return false;
    }
  }
  return true;
}

export function evidenceReference(
  value: unknown,
  evidenceIDs: Set<string>,
): void {
  if (typeof value !== "string" || !evidenceIDs.has(value)) fail();
}

export function oneOf(value: unknown, values: ReadonlySet<string>): void {
  if (typeof value !== "string" || !values.has(value)) fail();
}

export function equal(left: unknown, right: unknown): void {
  if (left !== right) fail();
}

export function fail(): never {
  throw new ContractError();
}
