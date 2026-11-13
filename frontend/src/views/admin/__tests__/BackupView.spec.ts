import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import BackupView from '../BackupView.vue'

const { backup, showWarning } = vi.hoisted(() => ({
  backup: {
    getS3Config: vi.fn(), getImageStorageConfig: vi.fn(), getSchedule: vi.fn(),
    listBackups: vi.fn(), getBackup: vi.fn(), updateSchedule: vi.fn(), createBackup: vi.fn(),
  },
  showWarning: vi.fn(),
}))
vi.mock('@/api', () => ({ adminAPI: { backup } }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn(), showWarning }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/composables/useStepUp', () => ({
  useStepUp: () => ({ run: (action: () => unknown) => action() }),
  isStepUpBlocked: () => false, isStepUpCancelled: () => false, stepUpBlockReason: () => '',
}))

const completed = { id: 'backup-1', status: 'completed', file_name: 'backup.sql.gz', size_bytes: 10, triggered_by: 'manual', started_at: '' }
let wrapper: VueWrapper | undefined
const render = async () => {
  wrapper = mount(BackupView, { global: { stubs: { TotpStepUpDialog: true, TableScroll: { template: '<div><slot /></div>' } } } })
  await flushPromises()
  return wrapper
}
const button = (text: string) => wrapper!.findAll('button').find(item => item.text() === text)!

beforeEach(() => {
  vi.useFakeTimers()
  vi.clearAllMocks()
  Object.defineProperty(document, 'hidden', { configurable: true, value: false })
  backup.getS3Config.mockResolvedValue({})
  backup.getImageStorageConfig.mockResolvedValue({ config: {}, secret_configured: false })
  backup.getSchedule.mockResolvedValue({ enabled: false, cron_expr: '0 2 * * *', retain_days: 0, retain_count: 0 })
  backup.listBackups.mockResolvedValue({ items: [completed] })
  backup.updateSchedule.mockResolvedValue({})
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.useRealTimers()
})

describe('Backup retention and operation state', () => {
  it('keeps unlimited settings across loading and saving', async () => {
    await render()
    const saveButtons = wrapper!.findAll('button').filter(item => item.text() === 'common.save')
    await saveButtons[2].trigger('click')
    expect(backup.updateSchedule).toHaveBeenCalledWith(expect.objectContaining({ retain_days: 0, retain_count: 0 }))
  })

  it('displays historical backup and restore errors', async () => {
    backup.listBackups.mockResolvedValue({ items: [{ ...completed, status: 'failed', error_message: 'Upload failed', restore_status: 'failed', restore_error: 'Download failed' }] })
    await render()
    expect(wrapper!.text()).toContain('Upload failed')
    expect(wrapper!.text()).toContain('Download failed')
  })

  it('disables conflicting actions while a restore is running', async () => {
    backup.listBackups.mockResolvedValue({ items: [{ ...completed, restore_status: 'running' }, { ...completed, id: 'backup-2' }] })
    await render()
    expect(button('admin.backup.operations.createBackup').attributes('disabled')).toBeDefined()
    expect(button('admin.backup.actions.restore').attributes('disabled')).toBeDefined()
    expect(button('common.delete').attributes('disabled')).toBeDefined()
  })

  it('handles normalized 409 responses as an in-progress warning', async () => {
    backup.createBackup.mockRejectedValue({ status: 409 })
    await render()
    await button('admin.backup.operations.createBackup').trigger('click')
    await flushPromises()
    expect(showWarning).toHaveBeenCalledWith('admin.backup.operations.alreadyInProgress')
  })

  it('does not overlap polling requests and ignores results after unmount', async () => {
    backup.listBackups.mockResolvedValue({ items: [{ ...completed, status: 'running' }] })
    let resolve!: (value: unknown) => void
    backup.getBackup.mockImplementation(() => new Promise(done => { resolve = done }))
    await render()
    await vi.advanceTimersByTimeAsync(6000)
    expect(backup.getBackup).toHaveBeenCalledTimes(1)
    wrapper!.unmount()
    wrapper = undefined
    resolve(completed)
    await flushPromises()
    expect(backup.listBackups).toHaveBeenCalledTimes(1)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('clears busy state when the task finished while the tab was hidden', async () => {
    backup.listBackups.mockResolvedValueOnce({ items: [{ ...completed, status: 'running' }] })
    await render()
    Object.defineProperty(document, 'hidden', { configurable: true, value: true })
    document.dispatchEvent(new Event('visibilitychange'))
    Object.defineProperty(document, 'hidden', { configurable: true, value: false })
    document.dispatchEvent(new Event('visibilitychange'))
    await flushPromises()
    expect(button('admin.backup.operations.createBackup').attributes('disabled')).toBeUndefined()
    expect(vi.getTimerCount()).toBe(0)
  })
})
