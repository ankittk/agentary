import { defineConfig, devices } from '@playwright/test';

/**
 * E2E tests for the Agentary React UI. By default starts the Go server with the built SPA.
 * Set AGENTARY_E2E_URL to run against an existing server (e.g. http://127.0.0.1:3548).
 * @see https://playwright.dev/docs/test-webserver
 */
export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: process.env.AGENTARY_E2E_URL ?? 'http://127.0.0.1:3548',
    trace: 'on-first-retry',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  ...(process.env.AGENTARY_E2E_URL
    ? {}
    : {
        webServer: {
          command: 'npm run build && cp -r dist ../internal/ui/dist && cd .. && go run ./cmd/agentary start --foreground --port=3548',
          url: 'http://127.0.0.1:3548',
          reuseExistingServer: !process.env.CI,
          timeout: 90_000,
        },
      }),
});
