import { defineConfig } from "@playwright/test";

const skipWebServer = process.env.E2E_SKIP_WEBSERVER === "1";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  workers: 1,
  timeout: 180_000,
  expect: { timeout: 15_000 },
  reporter: "list",
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:3000",
    browserName: "chromium",
    viewport: { width: 375, height: 667 },
    isMobile: true,
    hasTouch: true,
  },
  webServer: skipWebServer
    ? undefined
    : {
        command: "bun run dev",
        url: "http://localhost:3000",
        reuseExistingServer: true,
        timeout: 120_000,
      },
});
