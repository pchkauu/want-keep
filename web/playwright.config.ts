import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  forbidOnly: Boolean(process.env.CI),
  retries: 0,
  outputDir: "./.cache/playwright-results",
  use: { baseURL: "http://127.0.0.1:4179", trace: "retain-on-failure" },
  projects: [{ name: "chromium", use: { browserName: "chromium" } }],
  webServer: {
    command: "npm run dev -- --host 127.0.0.1 --port 4179 --strictPort",
    url: "http://127.0.0.1:4179",
    reuseExistingServer: !process.env.CI,
  },
});
