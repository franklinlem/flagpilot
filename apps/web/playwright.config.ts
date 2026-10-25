import { defineConfig } from '@playwright/test';
export default defineConfig({
  testDir: './e2e',
  workers: 1,
  retries: 0,
  timeout: 30000,
  reporter: 'list',
  use: { baseURL: process.env.E2E_BASE_URL, browserName: (process.env.E2E_BROWSER ?? 'chromium') as 'chromium' | 'firefox', trace: 'off', screenshot: 'off' },
});
