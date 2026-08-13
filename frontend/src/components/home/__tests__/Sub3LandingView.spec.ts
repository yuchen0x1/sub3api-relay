import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'

const i18nMock = vi.hoisted(() => {
  const locale = { value: 'zh' as 'zh' | 'en' }
  const setLocale = vi.fn(async (nextLocale: 'zh' | 'en') => {
    locale.value = nextLocale
  })
  return { locale, setLocale }
})

vi.mock('@/i18n', () => ({
  getLocale: () => i18nMock.locale.value,
  i18n: { global: { locale: i18nMock.locale } },
  setLocale: i18nMock.setLocale,
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    fetchPublicSettings: vi.fn(async () => ({
      qq_group_number: '1016190748',
      qq_group_url: 'https://qm.qq.com/q/gBW5t6RmU0',
      version: '0.1.182',
    })),
  }),
}))

import Sub3LandingView from '../Sub3LandingView.vue'

const paths = ['/home', '/login', '/keys', '/usage', '/available-channels', '/monitor']

async function mountLanding() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: paths.map((path) => ({ path, component: { template: '<div />' } })),
  })
  await router.push('/home')
  await router.isReady()

  return {
    router,
    wrapper: mount(Sub3LandingView, { global: { plugins: [router] } }),
  }
}

describe('Sub3LandingView navigation', () => {
  beforeEach(() => {
    i18nMock.locale.value = 'zh'
    i18nMock.setLocale.mockClear()
  })

  it('does not expose login or authenticated dashboard links', async () => {
    const { wrapper } = await mountLanding()

    const internalLinks = wrapper
      .findAll('a[href^="/"]')
      .map((link) => link.attributes('href'))

    expect(new Set(internalLinks)).toEqual(new Set(['/', '/monitor']))
    for (const path of ['/login', '/keys', '/usage', '/available-channels', '/admin']) {
      expect(wrapper.find(`a[href="${path}"]`).exists()).toBe(false)
    }
    expect(wrapper.text()).not.toContain('立即开始')
    expect(wrapper.text()).not.toContain('登录')
    expect(wrapper.text()).not.toContain('Get Started')
    expect(wrapper.text()).not.toContain('Sign in')
  })

  it('opens the repository releases page from the secondary CTA', async () => {
    const { wrapper } = await mountLanding()
    const docsLink = wrapper.get('.hero-cta .btn-ghost')

    expect(docsLink.attributes('href')).toBe('https://github.com/yuchen0x1/sub3api-relay/releases')
    expect(docsLink.attributes('target')).toBe('_blank')
    expect(docsLink.attributes('rel')).toBe('noopener noreferrer')
    expect(docsLink.text()).toBe('仓库下载')
  })

  it('keeps only public status and community links', async () => {
    const { wrapper } = await mountLanding()
    const featureLinks = wrapper.findAll('.f-more')
    const footerLinks = wrapper.findAll('.ft-right a')

    expect(featureLinks.map((link) => link.attributes('href'))).toEqual(['/monitor'])
    expect(featureLinks.map((link) => link.text())).toEqual(['查看服务状态'])
    expect(footerLinks.map((link) => link.attributes('href'))).toEqual([
      '/monitor',
      'https://qm.qq.com/q/gBW5t6RmU0',
    ])
    const qqLink = wrapper.get('.qq-group-link')
    expect(qqLink.attributes('target')).toBe('_blank')
    expect(qqLink.attributes('rel')).toBe('noopener noreferrer')
    expect(qqLink.text()).toBe('QQ群：1016190748')
  })

  it('routes the public status link through Vue Router', async () => {
    const { router, wrapper } = await mountLanding()

    await wrapper.get('a[href="/monitor"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/monitor')
  })

  it('switches the complete landing page between Chinese and English', async () => {
    const { wrapper } = await mountLanding()
    const zhButton = wrapper.get<HTMLButtonElement>('.lang button[data-locale="zh"]')
    const enButton = wrapper.get<HTMLButtonElement>('.lang button[data-locale="en"]')

    expect(zhButton.attributes('aria-pressed')).toBe('true')
    expect(enButton.attributes('aria-pressed')).toBe('false')

    await enButton.trigger('click')
    await flushPromises()

    expect(i18nMock.setLocale).toHaveBeenCalledWith('en')
    expect(wrapper.get('.nav-links a').text()).toBe('Service Status')
    expect(wrapper.get('.hero-sub').text()).toContain('One API key')
    expect(wrapper.get('.f-card h3').text()).toContain('One-key')
    expect(wrapper.findAll('.f-more').map((link) => link.text())).toEqual(['View Service Status'])
    expect(enButton.attributes('aria-pressed')).toBe('true')
    expect(zhButton.attributes('aria-pressed')).toBe('false')

    await zhButton.trigger('click')
    await flushPromises()

    expect(i18nMock.setLocale).toHaveBeenLastCalledWith('zh')
    expect(wrapper.get('.nav-links a').text()).toBe('服务状态')
    expect(zhButton.attributes('aria-pressed')).toBe('true')
    expect(enButton.attributes('aria-pressed')).toBe('false')
  })
})

describe('Sub3LandingView live status', () => {
  it('renders the current version badge from public settings', async () => {
    const { wrapper } = await mountLanding()
    await flushPromises()

    expect(wrapper.get('.monitor-title .badge').text()).toBe('v0.1.182')
  })
  it('renders the channel status line', async () => {
    const { wrapper } = await mountLanding()
    const line = wrapper.get('#chLine')
    expect(line.exists()).toBe(true)
    expect(line.text()).toContain('/')
  })
})
