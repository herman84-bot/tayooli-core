import { defineConfig, devices } from '@playwright/test';

// Audit config: runs against PRODUCTION, never starts a local server.
// Usage: npx playwright test -c frontend/e2e-audit/playwright.audit.config.ts
export default defineConfig({
  testDir: '.',
  timeout: 90_000,
  workers: 1,
  reporter: [['list'], ['json', { outputFile: '../../test-results/audit/results.json' }]],
  outputDir: '../../test-results/audit/artifacts',
  expect: { timeout: 15_000 },
  use: {
    baseURL: process.env.AUDIT_BASE_URL || 'https://tayooli.my.id',
    screenshot: 'off',
    trace: 'off',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
});
