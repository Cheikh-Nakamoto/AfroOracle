import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  fullyParallel: false,
  reporter: "list",
  use: {
    baseURL: "http://127.0.0.1:3210",
    browserName: "chromium",
    headless: true,
    viewport: { width: 1280, height: 900 },
  },
  webServer: [
    {
      command: "node tests/mock-api.mjs",
      url: "http://127.0.0.1:8111/health",
      reuseExistingServer: false,
      timeout: 15_000,
    },
    {
      command: "npm run dev -- --hostname 127.0.0.1 --port 3210",
      url: "http://127.0.0.1:3210",
      reuseExistingServer: false,
      timeout: 120_000,
      env: { AFRIORACLE_BACKEND_URL: "http://127.0.0.1:8111" },
    },
  ],
});
