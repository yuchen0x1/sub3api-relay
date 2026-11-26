import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import axios from 'axios'

const settings = vi.hoisted(() => ({ baseURL: '/api/v1' }))

vi.mock('@/api/url', () => ({ getAPIBaseURL: () => settings.baseURL }))
vi.mock('axios', () => ({ default: { post: vi.fn() } }))

const mockedPost = vi.mocked(axios.post)
const testToken = (...parts: string[]) => parts.join('-')

function seedLegacySession(): void {
  localStorage.setItem('auth_token', testToken('old', 'access'))
  localStorage.setItem('refresh_token', testToken('old', 'refresh'))
  localStorage.setItem('token_expires_at', String(Date.now() - 1))
  localStorage.setItem('auth_user', JSON.stringify({ id: 7 }))
}

function seedCookieSession(): void {
  document.cookie = 'sub3api_browser_csrf=signed-token; Path=/'
  document.cookie = 'sub3api_browser_session=1; Path=/'
  localStorage.setItem('auth_user', JSON.stringify({ id: 7 }))
}

function refreshedResponse() {
  return {
    data: {
      code: 0,
      message: 'ok',
      data: {
        access_token: testToken('new', 'access'),
        refresh_token: testToken('new', 'refresh'),
        expires_in: 3600,
        token_type: 'Bearer'
      }
    }
  }
}

describe('refreshAuthTokens', () => {
  beforeEach(() => {
    localStorage.clear()
    document.cookie = 'sub3api_browser_csrf=; Max-Age=0; Path=/'
    document.cookie = 'sub3api_browser_session=; Max-Age=0; Path=/'
    mockedPost.mockReset()
    vi.resetModules()
    settings.baseURL = '/api/v1'
    Object.defineProperty(navigator, 'locks', {
      configurable: true,
      value: { request: vi.fn((_name, callback) => callback()) }
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('refreshes a reloaded cookie session without persisting either token', async () => {
    seedCookieSession()
    mockedPost.mockResolvedValueOnce(refreshedResponse())
    const { refreshAuthTokens } = await import('@/api/tokenRefresh')
    const { getAccessToken, getSessionRefreshToken } = await import('@/api/browserSession')

    await refreshAuthTokens()

    expect(mockedPost).toHaveBeenCalledWith('/api/v1/auth/refresh', {}, expect.objectContaining({
      headers: expect.objectContaining({
        'X-Sub3API-Browser-Session': 'cookie',
        'X-CSRF-Token': 'signed-token'
      }),
      withCredentials: true
    }))
    expect(getAccessToken()).toBe(testToken('new', 'access'))
    expect(getSessionRefreshToken()).toBeNull()
    expect(localStorage.getItem('auth_token')).toBeNull()
    expect(localStorage.getItem('refresh_token')).toBeNull()
  })

  it('migrates a legacy stored refresh token to cookie mode on its first refresh', async () => {
    seedLegacySession()
    mockedPost.mockResolvedValueOnce(refreshedResponse())
    const { refreshAuthTokens } = await import('@/api/tokenRefresh')

    await refreshAuthTokens({ failedAccessToken: testToken('old', 'access') })

    expect(mockedPost).toHaveBeenCalledWith('/api/v1/auth/refresh', {
      refresh_token: testToken('old', 'refresh')
    }, expect.objectContaining({
      headers: expect.objectContaining({ 'X-Sub3API-Browser-Session': 'cookie' })
    }))
    expect(localStorage.getItem('auth_token')).toBeNull()
    expect(localStorage.getItem('refresh_token')).toBeNull()
  })

  it('shares one refresh request between concurrent callers in the same document', async () => {
    seedCookieSession()
    let resolveRequest!: (value: ReturnType<typeof refreshedResponse>) => void
    mockedPost.mockImplementationOnce(() => new Promise((resolve) => { resolveRequest = resolve }))
    const { refreshAuthTokens } = await import('@/api/tokenRefresh')

    const first = refreshAuthTokens()
    const second = refreshAuthTokens()

    expect(mockedPost).toHaveBeenCalledTimes(1)
    resolveRequest(refreshedResponse())
    await expect(first).resolves.toMatchObject({ access_token: testToken('new', 'access') })
    await expect(second).resolves.toMatchObject({ access_token: testToken('new', 'access') })
    expect(localStorage.getItem('refresh_token')).toBeNull()
  })

  it('uses a peer-migrated cookie instead of replaying the legacy body after acquiring the lock', async () => {
    seedLegacySession()
    Object.defineProperty(navigator, 'locks', {
      configurable: true,
      value: { request: vi.fn((_name, callback) => {
        document.cookie = 'sub3api_browser_session=1; Path=/'
        document.cookie = 'sub3api_browser_csrf=peer-csrf; Path=/'
        localStorage.removeItem('auth_token')
        localStorage.removeItem('refresh_token')
        return callback()
      }) }
    })
    mockedPost.mockResolvedValueOnce(refreshedResponse())
    const { refreshAuthTokens } = await import('@/api/tokenRefresh')

    await refreshAuthTokens()

    expect(mockedPost).toHaveBeenCalledWith('/api/v1/auth/refresh', {}, expect.objectContaining({
      headers: expect.objectContaining({ 'X-CSRF-Token': 'peer-csrf' })
    }))
  })

  it('keeps cross-origin bearer token rotation entirely in memory', async () => {
    settings.baseURL = 'https://api.example/api/v1'
    seedLegacySession()
    mockedPost.mockResolvedValue(refreshedResponse())
    const { refreshAuthTokens } = await import('@/api/tokenRefresh')
    const { getAccessToken, getSessionRefreshToken } = await import('@/api/browserSession')

    await refreshAuthTokens()
    expect(getAccessToken()).toBe(testToken('new', 'access'))
    expect(getSessionRefreshToken()).toBe(testToken('new', 'refresh'))
    expect(localStorage.getItem('auth_token')).toBeNull()
    expect(localStorage.getItem('refresh_token')).toBeNull()
    await refreshAuthTokens()
    expect(mockedPost).toHaveBeenLastCalledWith('https://api.example/api/v1/auth/refresh', {
      refresh_token: testToken('new', 'refresh')
    }, expect.objectContaining({ headers: { 'Content-Type': 'application/json' } }))

    vi.resetModules()
    const reloaded = await import('@/api/browserSession')
    expect(reloaded.getAccessToken()).toBeNull()
    expect(reloaded.getSessionRefreshToken()).toBeNull()
  })

  it('refreshes HTTP cookie sessions without secure-context randomUUID or Web Locks', async () => {
    vi.stubGlobal('crypto', { randomUUID: undefined })
    Object.defineProperty(navigator, 'locks', { configurable: true, value: undefined })
    seedCookieSession()
    mockedPost.mockResolvedValueOnce(refreshedResponse())
    const { refreshAuthTokens } = await import('@/api/tokenRefresh')

    await expect(refreshAuthTokens()).resolves.toMatchObject({ access_token: testToken('new', 'access') })
    expect(localStorage.getItem('sub3api_browser_refresh_lease')).toBeNull()
    expect(localStorage.getItem('auth_token')).toBeNull()
  })

  it('waits for a peer cookie refresh lease instead of racing a one-time token', async () => {
    vi.useFakeTimers()
    Object.defineProperty(navigator, 'locks', { configurable: true, value: undefined })
    seedCookieSession()
    localStorage.setItem('sub3api_browser_refresh_lease', JSON.stringify({ owner: 'peer', expiresAt: Date.now() + 2000 }))
    mockedPost.mockResolvedValueOnce(refreshedResponse())
    const { refreshAuthTokens } = await import('@/api/tokenRefresh')

    const pending = refreshAuthTokens()
    await vi.advanceTimersByTimeAsync(1000)
    expect(mockedPost).not.toHaveBeenCalled()
    localStorage.removeItem('sub3api_browser_refresh_lease')
    await vi.advanceTimersByTimeAsync(100)
    await expect(pending).resolves.toMatchObject({ access_token: testToken('new', 'access') })
    expect(mockedPost).toHaveBeenCalledTimes(1)
  })

  it('does not restore a session logged out while refresh was in flight', async () => {
    seedCookieSession()
    let resolveRequest!: (value: ReturnType<typeof refreshedResponse>) => void
    mockedPost.mockImplementationOnce(() => new Promise((resolve) => { resolveRequest = resolve }))
    const { refreshAuthTokens } = await import('@/api/tokenRefresh')
    const { clearAccessToken, getAccessToken } = await import('@/api/browserSession')

    const pending = refreshAuthTokens()
    clearAccessToken()
    resolveRequest(refreshedResponse())

    await expect(pending).rejects.toThrow('Session changed during token refresh')
    expect(getAccessToken()).toBeNull()
    expect(localStorage.getItem('auth_token')).toBeNull()
  })

  it('rejects a stale refresh after another account signs in', async () => {
    seedCookieSession()
    let resolveRequest!: (value: ReturnType<typeof refreshedResponse>) => void
    mockedPost.mockImplementationOnce(() => new Promise((resolve) => { resolveRequest = resolve }))
    const { refreshAuthTokens } = await import('@/api/tokenRefresh')
    const { getAccessToken, setAccessToken } = await import('@/api/browserSession')

    const pending = refreshAuthTokens()
    localStorage.setItem('auth_user', JSON.stringify({ id: 8 }))
    setAccessToken('other-access')
    resolveRequest(refreshedResponse())

    await expect(pending).rejects.toThrow('Session changed during token refresh')
    expect(getAccessToken()).toBe('other-access')
  })
})
