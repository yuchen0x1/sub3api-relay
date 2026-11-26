import { beforeEach, describe, expect, it, vi } from 'vitest'

describe('browser session storage', () => {
  beforeEach(() => {
    localStorage.clear()
    document.cookie = 'sub3api_browser_csrf=; Max-Age=0; Path=/'
    document.cookie = 'sub3api_browser_session=; Max-Age=0; Path=/'
    vi.resetModules()
  })

  it('keeps new tokens in memory and removes legacy secrets', async () => {
    const session = await import('@/api/browserSession')
    localStorage.setItem('auth_token', 'legacy-access')
    localStorage.setItem('refresh_token', 'legacy-refresh')
    session.markBrowserSession()
    session.setAccessToken('new-access')
    expect(session.getAccessToken()).toBe('new-access')
    expect(localStorage.getItem('auth_token')).toBeNull()
    expect(localStorage.getItem('refresh_token')).toBeNull()
    expect(session.browserSessionHeader()).toEqual({ 'X-Sub3API-Browser-Session': 'cookie' })
  })

  it('recognizes a cookie session after the document reloads', async () => {
    localStorage.setItem('auth_token', 'stale-access')
    document.cookie = 'sub3api_browser_csrf=signed-token; Path=/'
    document.cookie = 'sub3api_browser_session=1; Path=/'
    const session = await import('@/api/browserSession')
    expect(session.isBrowserSession()).toBe(true)
    expect(session.getCsrfToken()).toBe('signed-token')
    expect(session.getAccessToken()).toBeNull()
    session.clearAccessToken()
    expect(session.isBrowserSession()).toBe(false)
    expect(session.getCsrfToken()).toBe('signed-token')
    session.markBrowserSession()
    expect(session.getCsrfToken()).toBe('signed-token')
  })

  it('does not mark legacy bearer sessions as cookie sessions', async () => {
    const session = await import('@/api/browserSession')
    localStorage.setItem('auth_token', 'legacy-access')
    expect(session.getAccessToken()).toBe('legacy-access')
    expect(session.browserSessionHeader()).toEqual({})
  })

  it('clears memory when a peer signs out or switches accounts', async () => {
    const session = await import('@/api/browserSession')
    session.setAccessToken('first-access')
    window.dispatchEvent(new StorageEvent('storage', {
      key: 'auth_user', oldValue: JSON.stringify({ id: 7 }), newValue: JSON.stringify({ id: 8 })
    }))
    expect(session.getAccessToken()).toBeNull()
    session.setAccessToken('second-access')
    window.dispatchEvent(new StorageEvent('storage', {
      key: 'auth_user', oldValue: JSON.stringify({ id: 8 }), newValue: null
    }))
    expect(session.getAccessToken()).toBeNull()
  })
})
