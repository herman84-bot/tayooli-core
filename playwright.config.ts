import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './frontend/e2e',
  timeout: 60_000,
  fullyParallel: true,
  retries: process.env.CI ? 2 : 0,
  reporter: [['list']],
  expect: {
    // Dev-mode compile latency (Next 15 hot reload) routinely pushes first
    // assertion after login past the 5s default. 15s absorbs cold compiles
    // without masking real regressions.
    timeout: 15_000,
  },
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
