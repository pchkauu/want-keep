import { chromium } from "playwright";
import { describe, expect, it } from "vitest";

import {
  collectorEnvironments,
  parseCollectorEnvironment,
} from "../src/index.js";

describe("collector foundation", () => {
  it("accepts only named application environments", () => {
    expect(collectorEnvironments).toEqual([
      "development",
      "test",
      "production",
    ]);
    expect(parseCollectorEnvironment("test")).toBe("test");
    expect(() => parseCollectorEnvironment(undefined)).toThrow("WANT_KEEP_ENV");
  });

  it("loads the Playwright API without starting a browser", () => {
    expect(chromium.name()).toBe("chromium");
  });
});
