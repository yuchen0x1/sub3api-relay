import { describe, expect, it, vi } from 'vitest'

const authStore = vi.hoisted(() => ({
  checkAuth: vi.fn(),
  isAuthenticated: false,
  isAdmin: false,
  isSimpleMode: false,
}))

const appStore = vi.hoisted(() => ({
  siteName: 'Sub3API',
  backendModeEnabled: false,
  cachedPublicSettings: null as null | Record<string, unknown>,
}))

vi.mock('@/stores/auth', () => ({ useAuthStore: () => authStore }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))
vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({ customMenuItems: [] }),
}))
vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
    isLoading: { value: false },
  }),
}))
vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

describe('public channel status route', () => {
  it('keeps /monitor available without authentication', async () => {
    const { default: router } = await import('@/router')
    const route = router.getRoutes().find((record) => record.name === 'ChannelStatus')

    expect(route?.path).toBe('/monitor')
    expect(route?.meta.requiresAuth).toBe(false)
    expect(route?.meta.requiresAdmin).toBe(false)
  })
})
