import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const listMock = vi.hoisted(() => vi.fn())

vi.mock('@/api/channelMonitor', () => ({
  list: listMock,
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

import ChannelStatusView from '../ChannelStatusView.vue'

describe('ChannelStatusView', () => {
  beforeEach(() => {
    listMock.mockReset()
    listMock.mockResolvedValue({ items: [] })
  })

  it('renders as a standalone public status page without admin controls', async () => {
    const wrapper = mount(ChannelStatusView)
    await flushPromises()

    expect(wrapper.find('aside').exists()).toBe(false)
    expect(wrapper.find('[role="tab"]').exists()).toBe(false)
    expect(wrapper.find('button').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('刷新')
    expect(wrapper.text()).not.toContain('自动刷新')
    expect(wrapper.text()).not.toContain('15 天')
    expect(wrapper.text()).not.toContain('30 天')
  })

  it('requests channel status without exposing card interactions', async () => {
    listMock.mockResolvedValue({
      items: [{
        id: 1,
        name: 'Public Channel',
        provider: 'openai',
        primary_model: 'model',
        primary_status: 'operational',
        primary_latency_ms: 100,
        primary_ping_latency_ms: 50,
        availability_7d: 99.9,
        timeline: [],
        extra_models: [],
      }],
    })

    const wrapper = mount(ChannelStatusView)
    await flushPromises()

    expect(listMock).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Public Channel')
    expect(wrapper.find('article').exists()).toBe(true)
    expect(wrapper.find('button').exists()).toBe(false)
  })
})
