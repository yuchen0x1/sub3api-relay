import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { batchCreateMock } = vi.hoisted(() => ({
  batchCreateMock: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      batchCreate: batchCreateMock,
    },
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key }),
  }
})

const BaseDialogStub = defineComponent({
  props: { show: { type: Boolean, default: false } },
  emits: ['close'],
  template: '<div v-if="show"><slot /></div>',
})

import ImportAccountsModal from '../ImportAccountsModal.vue'

// Placeholder credential strings for tests only; they are never real secrets.
const TEST_KEY_ONE = 'alpha-one'
const TEST_KEY_TWO = 'alpha-two'

function mountModal() {
  return mount(ImportAccountsModal, {
    props: {
      show: true,
      proxies: [{ id: 7, name: 'proxy-a' }],
      groups: [{ id: 3, name: 'group-x' }],
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        Select: true,
        Icon: true,
      },
    },
  })
}

describe('ImportAccountsModal', () => {
  beforeEach(() => {
    batchCreateMock.mockReset()
  })

  it('parses a OneAPI payload and previews the mapped accounts', async () => {
    const wrapper = mountModal()
    await wrapper.get('[data-testid="import-payload-input"]').setValue(
      JSON.stringify({
        data: [
          { type: 1, name: 'official', key: TEST_KEY_ONE, status: 1, group: 'default' },
          { type: 14, name: 'legacy', key: TEST_KEY_TWO },
        ],
      })
    )
    await wrapper.get('[data-testid="import-parse-button"]').trigger('click')

    const text = wrapper.text()
    expect(text).toContain('admin.accounts.importChannels.willCreate:{"count":1}')
    expect(text).toContain('legacy: unsupported channel type: 14')
    // Keys are masked in the preview table.
    expect(text).toContain('official')
    expect(text).toContain('••••')
  })

  it('submits chunked batchCreate requests with mapped credentials', async () => {
    batchCreateMock.mockResolvedValue({
      success: 2,
      failed: 0,
      results: [{ success: true }, { success: true }],
    })
    const wrapper = mountModal()
    const keys = [TEST_KEY_ONE, TEST_KEY_TWO].join('\n')
    await wrapper.get('[data-testid="import-payload-input"]').setValue(
      JSON.stringify([{ type: 8, name: 'relay', key: keys, base_url: 'https://relay.example.com/v1' }])
    )
    await wrapper.get('[data-testid="import-parse-button"]').trigger('click')
    await wrapper.get('[data-testid="import-submit-button"]').trigger('click')
    await flushPromises()

    expect(batchCreateMock).toHaveBeenCalledTimes(1)
    const sent = batchCreateMock.mock.calls[0][0]
    expect(sent).toHaveLength(2)
    expect(sent[0]).toMatchObject({
      name: 'relay',
      platform: 'openai',
      type: 'apikey',
      credentials: { api_key: TEST_KEY_ONE, base_url: 'https://relay.example.com/v1' },
    })
    expect(sent[1].credentials.api_key).toBe(TEST_KEY_TWO)
    expect(wrapper.text()).toContain('admin.accounts.importChannels.resultSummary:{"created":2,"failed":0}')
    expect(wrapper.emitted('imported')).toEqual([[2]])
  })

  it('keeps the parse button disabled for empty input', async () => {
    const wrapper = mountModal()
    await wrapper.get('[data-testid="import-payload-input"]').setValue('')
    const parseBtn = wrapper.get('[data-testid="import-parse-button"]')
    expect(parseBtn.attributes('disabled')).toBeDefined()
  })
})
