import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, RouterLinkStub } from '@vue/test-utils'

import HomeView from '../HomeView.vue'

const { appStore, authStore } = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: {} as Record<string, unknown>,
    siteName: 'Fallback site',
    siteLogo: '',
    docUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
  },
  authStore: {
    isAuthenticated: false,
    isAdmin: false,
    user: null as { email?: string } | null,
    checkAuth: vi.fn(),
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
  useAuthStore: () => authStore,
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key, locale: { value: 'en' } }),
  }
})

function mountHome(settings: Record<string, unknown> = {}) {
  appStore.cachedPublicSettings = {
    site_name: 'Test site',
    site_subtitle: 'Test subtitle',
    ...settings,
  }

  return mount(HomeView, {
    global: {
      stubs: {
        RouterLink: RouterLinkStub,
        LocaleSwitcher: { template: '<div data-testid="locale-switcher" />' },
        Icon: { template: '<span data-testid="icon" />' },
      },
    },
  })
}

describe('HomeView compact mode', () => {
  beforeEach(() => {
    authStore.isAuthenticated = false
    authStore.isAdmin = false
    authStore.user = null
    authStore.checkAuth.mockClear()
    appStore.fetchPublicSettings.mockClear()
    localStorage.clear()
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: false } as MediaQueryList)
  })

  it('renders custom HTML ahead of compact mode', () => {
    const wrapper = mountHome({
      compact_home_enabled: true,
      home_content: '<section id="custom-home">Custom home</section>',
    })

    expect(wrapper.get('#custom-home').text()).toBe('Custom home')
    expect(wrapper.find('[data-testid="compact-home"]').exists()).toBe(false)
  })

  it('sanitizes executable markup from custom HTML', () => {
    const wrapper = mountHome({
      home_content: '<section id="safe" onclick="alert(1)">Safe</section><script>alert(1)</script>',
    })

    expect(wrapper.get('#safe').attributes('onclick')).toBeUndefined()
    expect(wrapper.find('script').exists()).toBe(false)
  })

  it('disables private internal links in custom HTML', () => {
    const wrapper = mountHome({
      home_content: [
        '<a id="login-link" href="/login">Login</a>',
        '<a id="keys-link" href="/keys">Keys</a>',
        '<a id="admin-link" href="/admin/dashboard">Admin</a>',
        '<a id="relative-login-link" href="./login">Relative login</a>',
        '<a id="status-link" href="/monitor">Status</a>',
      ].join(''),
    })

    expect(wrapper.get('#login-link').attributes('href')).toBeUndefined()
    expect(wrapper.get('#keys-link').attributes('href')).toBeUndefined()
    expect(wrapper.get('#admin-link').attributes('href')).toBeUndefined()
    expect(wrapper.get('#relative-login-link').attributes('href')).toBeUndefined()
    expect(wrapper.get('#status-link').attributes('href')).toBe('/monitor')
  })

  it('does not render non-http custom content as an iframe', () => {
    const wrapper = mountHome({ home_content: 'javascript:alert(1)' })

    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.text()).toContain('javascript:alert(1)')
  })

  it('renders custom URL content ahead of compact mode', () => {
    const wrapper = mountHome({
      compact_home_enabled: true,
      home_content: ' https://example.com/home ',
    })

    expect(wrapper.get('iframe').attributes('src')).toBe('https://example.com/home')
    expect(wrapper.find('[data-testid="compact-home"]').exists()).toBe(false)
  })

  it('treats whitespace-only custom content as empty and selects compact mode', () => {
    const wrapper = mountHome({ compact_home_enabled: true, home_content: ' \n\t ' })

    expect(wrapper.get('[data-testid="compact-home"]').text()).toContain('Test site')
  })

  it.each([undefined, false])('selects the default home when compact mode is %s', (enabled) => {
    const settings = enabled === undefined ? {} : { compact_home_enabled: enabled }
    const wrapper = mountHome(settings)

    expect(wrapper.find('[data-testid="compact-home"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="default-home"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Test site')
  })

  it('does not expose login or dashboard links in compact mode', () => {
    authStore.isAuthenticated = true
    authStore.isAdmin = true

    const wrapper = mountHome({ compact_home_enabled: true })
    expect(wrapper.findComponent(RouterLinkStub).exists()).toBe(false)
    expect(wrapper.html()).not.toContain('/login')
    expect(wrapper.html()).not.toContain('/dashboard')
    expect(wrapper.text()).not.toContain('home.login')
    expect(wrapper.text()).not.toContain('home.goToDashboard')
    expect(authStore.checkAuth).toHaveBeenCalledOnce()
    expect(appStore.fetchPublicSettings).not.toHaveBeenCalled()
  })
})
