import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import StaticProxyExitEditor from '../StaticProxyExitEditor.vue'
import type { Proxy, StaticProxyExit } from '@/types'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key} ${Object.values(values).join(' ')}` : key
  })
}))

const showError = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showInfo: vi.fn(),
    showSuccess: vi.fn()
  })
}))

const listByAccount = vi.fn()
const createExit = vi.fn()
const updateExit = vi.fn()
const toggleExitAPI = vi.fn()
const removeExit = vi.fn()

vi.mock('@/api/admin/staticProxyExits', () => ({
  default: {
    listByAccount: (...args: unknown[]) => listByAccount(...args),
    getById: vi.fn(),
    create: (...args: unknown[]) => createExit(...args),
    update: (...args: unknown[]) => updateExit(...args),
    toggle: (...args: unknown[]) => toggleExitAPI(...args),
    remove: (...args: unknown[]) => removeExit(...args),
    capacitySummary: vi.fn()
  }
}))

function makeProxy(id: number, name: string): Proxy {
  return {
    id,
    name,
    protocol: 'http',
    host: `${id}.example.com`,
    port: 8080,
    username: null,
    tls_insecure: false,
    status: 'active'
  }
}

function makeExit(overrides: Partial<StaticProxyExit> = {}): StaticProxyExit {
  return {
    id: 11,
    account_id: 42,
    proxy_id: 1,
    name: 'Exit A',
    position: 1,
    enabled: true,
    capacity: 10,
    health_status: 'healthy',
    failure_count: 0,
    last_failure_at: null,
    last_checked_at: null,
    created_at: '2026-09-01T00:00:00Z',
    updated_at: '2026-09-01T00:00:00Z',
    ...overrides
  }
}

const proxies = [makeProxy(1, 'exit-one'), makeProxy(2, 'exit-two')]

const mountEditor = () =>
  mount(StaticProxyExitEditor, {
    props: { accountId: 42, proxies }
  })

beforeEach(() => {
  vi.clearAllMocks()
  listByAccount.mockResolvedValue([makeExit()])
})

describe('StaticProxyExitEditor', () => {
  it('loads and renders the account exits with per-exit controls', async () => {
    const wrapper = mountEditor()
    await flushPromises()

    expect(listByAccount).toHaveBeenCalledWith(42)
    expect(wrapper.find('[data-test="static-exit-row-11"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="static-exit-toggle-11"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="static-exit-delete-11"]').exists()).toBe(true)
    // One enabled exit of one total → 10/10 effective capacity.
    expect(wrapper.find('[data-test="static-exit-capacity"]').text()).toMatch(/capacity 10 10/)
  })

  it('adds an exit with sequential position and enabled default', async () => {
    createExit.mockResolvedValue(
      makeExit({ id: 12, proxy_id: 2, name: 'Exit B', position: 2 })
    )
    const wrapper = mountEditor()
    await flushPromises()

    await wrapper.find('[data-test="static-exit-add-proxy"]').setValue('2')
    await wrapper.find('[data-test="static-exit-add-name"]').setValue('Exit B')
    await wrapper.find('[data-test="static-exit-add"]').trigger('click')
    await flushPromises()

    expect(createExit).toHaveBeenCalledWith({
      account_id: 42,
      proxy_id: 2,
      name: 'Exit B',
      position: 2,
      enabled: true
    })
    expect(wrapper.find('[data-test="static-exit-row-12"]').exists()).toBe(true)
  })

  it('toggles an exit through the API and updates the row', async () => {
    toggleExitAPI.mockResolvedValue(makeExit({ enabled: false }))
    const wrapper = mountEditor()
    await flushPromises()

    await wrapper.find('[data-test="static-exit-toggle-11"]').trigger('click')
    await flushPromises()

    expect(toggleExitAPI).toHaveBeenCalledWith(11, false)
    expect(wrapper.find('[data-test="static-exit-capacity"]').text()).toMatch(/capacity 0 10/)
  })

  it('requires a second click before deleting an exit', async () => {
    removeExit.mockResolvedValue(undefined)
    const wrapper = mountEditor()
    await flushPromises()

    await wrapper.find('[data-test="static-exit-delete-11"]').trigger('click')
    expect(removeExit).not.toHaveBeenCalled()
    expect(wrapper.find('[data-test="static-exit-delete-11"]').text()).toContain('confirmDelete')

    await wrapper.find('[data-test="static-exit-delete-11"]').trigger('click')
    expect(removeExit).toHaveBeenCalledWith(11)
    await flushPromises()
    expect(wrapper.find('[data-test="static-exit-row-11"]').exists()).toBe(false)
  })
})
