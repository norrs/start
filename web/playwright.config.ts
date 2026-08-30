import { defineConfig, devices } from '@playwright/test'
import path from 'node:path'

const baseURL = process.env.START_E2E_BASE_URL ?? 'http://127.0.0.1:18081'
const server = process.env.START_E2E_SERVER
const headed = process.env.START_E2E_HEADED === 'true'
const recordAll = process.env.START_E2E_RECORD_ALL === 'true'
const slowMo = Number(process.env.START_E2E_SLOW_MO ?? (recordAll ? 500 : 0))

if (!server) throw new Error('START_E2E_SERVER must point to the Bazel test server')

export default defineConfig({
  testDir: './e2e/specs',
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: [['line']],
  outputDir: path.join(process.env.TEST_UNDECLARED_OUTPUTS_DIR ?? 'test-results', 'playwright'),
  expect: { timeout: 5_000 },
  use: {
    baseURL,
    headless: !headed,
    screenshot: recordAll ? 'on' : 'only-on-failure',
    trace: recordAll ? 'on' : 'retain-on-failure',
    video: {
      mode: recordAll ? 'on' : 'retain-on-failure',
      show: {
        actions: {
          duration: recordAll ? 1600 : 900,
          position: 'bottom-right',
          fontSize: 18,
        },
        test: {
          level: 'step',
          position: 'top-left',
          fontSize: 18,
        },
      },
    },
  },
  webServer: {
    command: server,
    url: `${baseURL}/readyz`,
    reuseExistingServer: process.env.START_E2E_REUSE_SERVER === 'true',
    timeout: 120_000,
    stdout: 'pipe',
    stderr: 'pipe',
  },
  projects: [{
    name: 'chromium',
    use: {
      ...devices['Desktop Chrome'],
      launchOptions: { args: ['--no-sandbox'], slowMo },
    },
  }],
})
