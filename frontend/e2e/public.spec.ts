import { expect, test } from '@playwright/test'

// 公开页面在未登录态可直接访问（public 项目，无登录态）。
test('home landing renders without auth', async ({ page }) => {
  const pageErrors: string[] = []
  page.on('pageerror', (err) => pageErrors.push(String(err)))

  await page.goto('/home')
  await page.waitForLoadState('networkidle')

  const text = await page.locator('body').innerText()
  expect(text.length, '/home 渲染内容过少，疑似白屏').toBeGreaterThan(80)
  expect(pageErrors).toEqual([])
})
