import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ChartExportButton from '../ChartExportButton.vue'
import * as chartExport from '@/utils/chartExport'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/utils/chartExport', () => ({
  chartPngFilename: (prefix: string) => `${prefix}.png`,
  downloadChartPng: vi.fn(() => true)
}))

describe('ChartExportButton', () => {
  beforeEach(() => {
    vi.mocked(chartExport.downloadChartPng).mockClear()
  })

  it('renders the localized label and hint', () => {
    const wrapper = mount(ChartExportButton, {
      props: { getChart: () => null, filePrefix: 'ops-latency' }
    })

    expect(wrapper.text()).toBe('admin.ops.charts.downloadChart')
    expect(wrapper.attributes('title')).toBe('admin.ops.charts.downloadChartHint')
  })

  it('exports the chart under a generated filename', async () => {
    const canvas = document.createElement('canvas')
    const wrapper = mount(ChartExportButton, {
      props: { getChart: () => ({ canvas }), filePrefix: 'ops-latency' }
    })

    await wrapper.trigger('click')

    expect(chartExport.downloadChartPng).toHaveBeenCalledTimes(1)
    expect(chartExport.downloadChartPng).toHaveBeenCalledWith({ canvas }, 'ops-latency.png')
  })

  it('resolves the chart lazily so a re-mounted chart is picked up', async () => {
    const wrapper = mount(ChartExportButton, {
      props: { getChart: vi.fn(() => null), filePrefix: 'ops-error-trend' }
    })
    const getChart = wrapper.props('getChart') as ReturnType<typeof vi.fn>

    expect(getChart).not.toHaveBeenCalled()

    await wrapper.trigger('click')

    expect(getChart).toHaveBeenCalledTimes(1)
  })

  it('does not export while disabled', async () => {
    const wrapper = mount(ChartExportButton, {
      props: { getChart: () => null, filePrefix: 'ops-latency', disabled: true }
    })

    expect(wrapper.attributes('disabled')).toBeDefined()

    await wrapper.trigger('click')

    expect(chartExport.downloadChartPng).not.toHaveBeenCalled()
  })
})
