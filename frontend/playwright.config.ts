import { defineConfig, devices } from '@playwright/test'

// E2E 前置条件（见 e2e/README.md）：
// 1. 本地 embed 后端运行在 BASE_URL（默认 127.0.0.1:18081）
// 2. 数据库中存在 E2E_ADMIN_EMAIL 管理员账号
const BASE_URL = process.env.E2E_BASE_URL || 'http://127.0.0.1:18081'

const publicSpecs = /auth\.spec\.ts|public\.spec\.ts/

export default defineConfig({
  testDir: './e2e',
  timeout: 60_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list']],
  outputDir: 'e2e/.artifacts',
  use: {
    baseURL: BASE_URL,
    locale: 'zh-CN',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'setup',
      testMatch: /auth\.setup\.ts/,
    },
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'], storageState: 'e2e/.auth/admin.json' },
      dependencies: ['setup'],
      testIgnore: publicSpecs,
    },
    {
      name: 'public',
      use: { ...devices['Desktop Chrome'] },
      testMatch: publicSpecs,
    },
  ],
})
