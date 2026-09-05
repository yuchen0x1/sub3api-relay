import { expect, test } from '@playwright/test'

// 管理台核心页面冒烟：每个路由都要求
// 1) 不抛未捕获的页面异常（pageerror）
// 2) 渲染出实质性内容（防 TablePageLayout 插槽错位导致的静默白屏）
const adminPages = [
  '/admin/dashboard',
  '/admin/users',
  '/admin/groups',
  '/admin/channels',
  '/admin/accounts',
  '/admin/settings',
  '/admin/ops',
]

for (const route of adminPages) {
  test(`renders ${route}`, async ({ page }) => {
    const pageErrors: string[] = []
    page.on('pageerror', (err) => pageErrors.push(String(err)))

    await page.goto(route)
    await page.waitForLoadState('networkidle')

    const text = await page.locator('body').innerText()
    expect(text.length, `${route} 渲染内容过少，疑似白屏`).toBeGreaterThan(80)
    expect(pageErrors, `${route} 存在未捕获异常: ${pageErrors.join('; ')}`).toEqual([])
  })
}

test('settings page shows tab shell', async ({ page }) => {
  await page.goto('/admin/settings')
  await page.waitForLoadState('networkidle')
  await expect(page.locator('.settings-tabs-shell').first()).toBeVisible()
})
