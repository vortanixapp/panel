import { defineConfig } from "@playwright/test";

const port = Number(process.env.E2E_PORT ?? 3100);
const ci = Boolean(process.env.CI);

export default defineConfig({
  testDir: "./e2e",
  timeout: 45_000,
  expect: { timeout: 10_000 },
  fullyParallel: true,
  forbidOnly: ci,
  retries: ci ? 1 : 0,
  workers: ci ? 2 : undefined,
  reporter: ci ? [["list"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: `http://localhost:${port}`,
    channel: process.env.E2E_CHANNEL || undefined,
    locale: "ru-RU",
    timezoneId: "Europe/Moscow",
    viewport: { width: 1440, height: 900 },
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  webServer: {
    command: `npx next start -p ${port}`,
    url: `http://localhost:${port}/api/health`,
    reuseExistingServer: !ci,
    timeout: 120_000,
    env: {
      API_INTERNAL_URL: "http://127.0.0.1:9",
      NEXT_TELEMETRY_DISABLED: "1",
    },
  },
});
