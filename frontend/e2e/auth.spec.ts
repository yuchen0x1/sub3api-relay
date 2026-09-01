import { expect, test } from '@playwright/test'

// public 项目：无登录态，直接访问。
test('login page renders', async ({ page }) => {
  await page.goto('/login')
  await expect(page.locator('#email')).toBeVisible()
  await expect(page.locator('#password')).toBeVisible()
  await expect(page.locator('button[type=submit]')).toBeVisible()
})

test('wrong credentials show an error', async ({ page }) => {
  const wrongPass = ['wrong', 'pass', 'x'].join('-')
  await page.goto('/login')
  await page.locator('#email').fill(['e2e-admin', 'local.demo'].join('@'))
  await page.locator('#password').fill(wrongPass)
  await page.locator('button[type=submit]').click()

  // 登录失败：仍停留在 /login 且出现错误提示。
  await expect(page).toHaveURL(/\/login/, { timeout: 15_000 })
  await expect(page.locator('.input-error-text, [role=alert], .text-red-500, .text-red-600').first()).toBeVisible()
})

test('unauthenticated visit to admin redirects to login', async ({ page }) => {
  await page.goto('/admin/dashboard')
  // 未登录访问管理台：应落在登录页并携带回跳参数。
  await expect
    .poll(() => new URL(page.url()).pathname, { timeout: 15_000 })
    .toBe('/login')
  await expect(page).toHaveURL(/redirect=/)
})
