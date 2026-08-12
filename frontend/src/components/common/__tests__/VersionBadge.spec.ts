import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

import VersionBadge from '../VersionBadge.vue'
import { useAppStore, useAuthStore } from '@/stores'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

vi.mock('@/api/admin/system', async () => {
  const actual = await vi.importActual<typeof import('@/api/admin/system')>('@/api/admin/system')
  return {
    ...actual,
    checkUpdates: vi.fn(),
    performUpdate: vi.fn(),
    restartService: vi.fn(),
    getRollbackVersions: vi.fn(),
    rollback: vi.fn()
  }
})

function mountVersionBadge(body: string) {
  const authStore = useAuthStore()
  const appStore = useAppStore()

  authStore.user = {
    id: 1,
    username: 'admin',
    email: 'admin@example.com',
    role: 'admin',
    balance: 0,
    concurrency: 1,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: false,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-08-12T00:00:00Z',
    updated_at: '2026-08-12T00:00:00Z'
  }
  appStore.versionLoaded = true
  appStore.currentVersion = '0.1.180'
  appStore.latestVersion = '0.1.181'
  appStore.hasUpdate = true
  appStore.buildType = 'release'
  appStore.releaseInfo = {
    name: 'Sub3API 0.1.181',
    body,
    published_at: '2026-08-12T00:00:00Z',
    html_url: 'https://github.com/yuchen0x1/sub3api-relay/releases/tag/v0.1.181'
  }

  return mount(VersionBadge)
}

describe('VersionBadge release notes', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders release markdown inside the version panel', async () => {
    const wrapper = mountVersionBadge('## Highlights\n\n- Faster updates')

    await wrapper.get('button').trigger('click')
    await wrapper.get('[data-testid="release-notes-toggle"]').trigger('click')

    const content = wrapper.get('[data-testid="release-notes-content"]')
    expect(content.find('h2').text()).toBe('Highlights')
    expect(content.find('li').text()).toBe('Faster updates')
  })

  it('shows an empty state when the release body is blank', async () => {
    const wrapper = mountVersionBadge('   ')

    await wrapper.get('button').trigger('click')
    await wrapper.get('[data-testid="release-notes-toggle"]').trigger('click')

    expect(wrapper.get('[data-testid="release-notes-empty"]').text()).toBe(
      'version.noReleaseNotes'
    )
  })

  it('sanitizes unsafe release HTML', async () => {
    const wrapper = mountVersionBadge(
      'Safe text<script>window.__releaseXss = true</script><img src=x onerror="window.__releaseXss = true">'
    )

    await wrapper.get('button').trigger('click')
    await wrapper.get('[data-testid="release-notes-toggle"]').trigger('click')

    const content = wrapper.get('[data-testid="release-notes-content"]')
    expect(content.text()).toContain('Safe text')
    expect(content.find('script').exists()).toBe(false)
    expect(content.find('img').attributes('onerror')).toBeUndefined()
  })

  it('shows a persistent Compose update command for Docker deployments', async () => {
    const wrapper = mountVersionBadge('Docker release')
    const appStore = useAppStore()
    appStore.buildType = 'docker'

    await wrapper.get('button').trigger('click')

    expect(wrapper.get('[data-testid="docker-update-command"]').text()).toContain(
      'docker compose pull'
    )
    expect(wrapper.get('[data-testid="docker-update-command"]').text()).toContain(
      'docker compose up -d'
    )
    expect(wrapper.text()).not.toContain('version.updateNow')
  })
})
