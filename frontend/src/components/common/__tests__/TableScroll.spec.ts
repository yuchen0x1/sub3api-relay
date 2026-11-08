import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import TableScroll from '../TableScroll.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

const HINT_KEY = 'table-scroll-hint-dismissed'

function stubMetrics(
  element: Element,
  metrics: { scrollWidth: number; clientWidth: number; scrollLeft?: number }
): void {
  Object.defineProperty(element, 'scrollWidth', { configurable: true, get: () => metrics.scrollWidth })
  Object.defineProperty(element, 'clientWidth', { configurable: true, get: () => metrics.clientWidth })
  Object.defineProperty(element, 'scrollLeft', { configurable: true, get: () => metrics.scrollLeft ?? 0 })
}

describe('TableScroll', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  afterEach(() => {
    sessionStorage.clear()
  })

  it('renders slot content inside the scroll viewport', () => {
    const wrapper = mount(TableScroll, {
      slots: { default: '<div class="wide">wide</div>' }
    })

    expect(wrapper.find('.table-scroll__viewport .wide').exists()).toBe(true)
  })

  it('shows the hint and the trailing edge shadow when content overflows', async () => {
    const wrapper = mount(TableScroll, {
      slots: { default: '<div class="wide">wide</div>' }
    })
    const viewport = wrapper.find('.table-scroll__viewport')

    stubMetrics(viewport.element, { scrollWidth: 1200, clientWidth: 400 })
    await viewport.trigger('scroll')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-test="table-scroll-hint"]').exists()).toBe(true)
    expect(
      wrapper.find('.table-scroll__shadow--right').classes()
    ).toContain('table-scroll__shadow--visible')
    expect(
      wrapper.find('.table-scroll__shadow--left').classes()
    ).not.toContain('table-scroll__shadow--visible')
  })

  it('reveals the leading edge shadow once scrolled', async () => {
    const wrapper = mount(TableScroll, {
      slots: { default: '<div class="wide">wide</div>' }
    })
    const viewport = wrapper.find('.table-scroll__viewport')

    stubMetrics(viewport.element, { scrollWidth: 1200, clientWidth: 400, scrollLeft: 800 })
    await viewport.trigger('scroll')
    await wrapper.vm.$nextTick()

    expect(
      wrapper.find('.table-scroll__shadow--left').classes()
    ).toContain('table-scroll__shadow--visible')
    expect(
      wrapper.find('.table-scroll__shadow--right').classes()
    ).not.toContain('table-scroll__shadow--visible')
  })

  it('does not show the hint when the content fits', async () => {
    const wrapper = mount(TableScroll, {
      slots: { default: '<div class="wide">wide</div>' }
    })
    const viewport = wrapper.find('.table-scroll__viewport')

    stubMetrics(viewport.element, { scrollWidth: 400, clientWidth: 400 })
    await viewport.trigger('scroll')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-test="table-scroll-hint"]').exists()).toBe(false)
    expect(
      wrapper.find('.table-scroll__shadow--right').classes()
    ).not.toContain('table-scroll__shadow--visible')
  })

  it('does not show the hint when the hint prop is disabled', async () => {
    const wrapper = mount(TableScroll, {
      props: { hint: false },
      slots: { default: '<div class="wide">wide</div>' }
    })
    const viewport = wrapper.find('.table-scroll__viewport')

    stubMetrics(viewport.element, { scrollWidth: 1200, clientWidth: 400 })
    await viewport.trigger('scroll')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-test="table-scroll-hint"]').exists()).toBe(false)
  })

  it('dismisses the hint and remembers the choice for the session', async () => {
    const wrapper = mount(TableScroll, {
      slots: { default: '<div class="wide">wide</div>' }
    })
    const viewport = wrapper.find('.table-scroll__viewport')

    stubMetrics(viewport.element, { scrollWidth: 1200, clientWidth: 400 })
    await viewport.trigger('scroll')
    await wrapper.vm.$nextTick()

    await wrapper.find('[data-test="table-scroll-hint-close"]').trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-test="table-scroll-hint"]').exists()).toBe(false)
    expect(sessionStorage.getItem(HINT_KEY)).toBe('1')
  })

  it('keeps the hint hidden after it was dismissed earlier in the session', async () => {
    sessionStorage.setItem(HINT_KEY, '1')

    const wrapper = mount(TableScroll, {
      slots: { default: '<div class="wide">wide</div>' }
    })
    const viewport = wrapper.find('.table-scroll__viewport')

    stubMetrics(viewport.element, { scrollWidth: 1200, clientWidth: 400 })
    await viewport.trigger('scroll')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-test="table-scroll-hint"]').exists()).toBe(false)
  })

  it('forwards the scroll event to the parent', async () => {
    const wrapper = mount(TableScroll, {
      slots: { default: '<div class="wide">wide</div>' }
    })

    await wrapper.find('.table-scroll__viewport').trigger('scroll')

    expect(wrapper.emitted('scroll')).toHaveLength(1)
  })

  it('applies viewportClass to the scroll viewport', () => {
    const wrapper = mount(TableScroll, {
      props: { viewportClass: 'max-h-80' },
      slots: { default: '<div class="wide">wide</div>' }
    })

    expect(wrapper.find('.table-scroll__viewport').classes()).toContain('max-h-80')
  })
})
