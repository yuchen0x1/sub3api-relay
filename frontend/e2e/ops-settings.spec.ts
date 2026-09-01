import { expect, test } from '@playwright/test'

// 阶段 4.1-4.3 验收点：运维监控设置弹窗中，
// 启用「渠道监控自动禁用」后应出现「联动暂停同平台账号调度」开关。
test('ops settings exposes monitor account-pause linkage toggle', async ({ page }) => {
  await page.goto('/admin/ops')
  await page.waitForLoadState('networkidle')

  // 打开设置弹窗（头部按钮文案：设置）。
  await page.locator('header button, .ops-panel-hero button')
    .filter({ hasText: '设置' })
    .first()
    .click()
  await expect(page.getByText('运维监控设置')).toBeVisible({ timeout: 15_000 })

  // 展开高级设置折叠区（渠道监控自动禁用位于其中）。
  const advanced = page.locator('details > summary').filter({ hasText: '高级设置' })
  await expect(advanced).toBeVisible()
  await advanced.click()

  // 启用「自动禁用」主开关。
  const enableRow = page
    .getByText('启用自动禁用')
    .locator('xpath=ancestor::div[contains(@class,"justify-between")][1]')
  await expect(enableRow).toBeVisible()
  await enableRow.getByRole('switch').click()
  await expect(enableRow.getByRole('switch')).toHaveAttribute('aria-checked', 'true')

  // 联动开关随之出现（默认关闭，不保存、不改动线上配置）。
  await expect(page.getByText('联动暂停同平台账号调度')).toBeVisible()
  const linkageRow = page
    .getByText('联动暂停同平台账号调度')
    .locator('xpath=ancestor::div[contains(@class,"justify-between")][1]')
  await expect(linkageRow.getByRole('switch')).toHaveAttribute('aria-checked', 'false')
})
