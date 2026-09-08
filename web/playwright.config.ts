import { defineConfig } from "@playwright/test";

const access = process.env.WANT_KEEP_ACCESS_E2E === "1";
export default defineConfig({
  workers: access ? 1 : undefined,
  testDir: "./e2e",
  forbidOnly: Boolean(process.env.CI),
  retries: 0,
  outputDir: "./.cache/playwright-results",
  use: {
    baseURL: access
      ? process.env.WANT_KEEP_ACCESS_ORIGIN!
      : "http://127.0.0.1:4180",
    trace: access ? "off" : "retain-on-failure",
  },
  projects: [{ name: "chromium", use: { browserName: "chromium" } }],
  webServer: {
    command: access
      ? "npm run dev -- --host localhost --port 4183 --strictPort"
      : "npm run dev -- --host 127.0.0.1 --port 4180 --strictPort",
    url: access
      ? process.env.WANT_KEEP_ACCESS_ORIGIN!
      : "http://127.0.0.1:4180",
    reuseExistingServer: false,
  },
});
