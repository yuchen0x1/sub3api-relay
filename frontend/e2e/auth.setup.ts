import { expect, test as setup } from '@playwright/test'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

// 管理员凭据由环境变量注入，缺省值仅用于本地 E2E 环境。
const E2E_EMAIL = process.env.E2E_ADMIN_EMAIL || ['e2e-admin', 'local.demo'].join('@')
const E2E_PASS = process.env.E2E_ADMIN_PASSWORD || ['e2e', 'admin', 'local'].join('-')

const stateFile = path.join(path.dirname(fileURLToPath(import.meta.url)), '.auth', 'admin.json')

setup('admin login', async ({ page }) => {
  await page.goto('/login')
  await page.locator('#email').fill(E2E_EMAIL)
  await page.locator('#password').fill(E2E_PASS)
  await page.locator('button[type=submit]').click()

  await expect(page).toHaveURL(/\/dashboard/, { timeout: 15_000 })

  fs.mkdirSync(path.dirname(stateFile), { recursive: true })
  await page.context().storageState({ path: stateFile })
})
