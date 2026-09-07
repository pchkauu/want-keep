export const collectorEnvironments = [
  "development",
  "test",
  "production",
] as const;

export type CollectorEnvironment = (typeof collectorEnvironments)[number];

export function parseCollectorEnvironment(
  value: string | undefined,
): CollectorEnvironment {
  switch (value) {
    case "development":
    case "test":
    case "production":
      return value;
    default:
      throw new Error("WANT_KEEP_ENV must be development, test, or production");
  }
}
