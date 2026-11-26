/** Browser authentication session state.
 *
 * Tokens for new browser sessions stay in this module's memory. The
 * legacy localStorage value is read only as a compatibility fallback for
 * sessions created by older clients and is never written for new sessions.
 */
import { getAPIBaseURL } from './url'

const LEGACY_ACCESS_TOKEN_KEY = 'auth_token'
const BROWSER_SESSION_HEADER = 'X-Sub3API-Browser-Session'
const BROWSER_SESSION_VALUE = 'cookie'
const CSRF_COOKIE_NAME = 'sub3api_browser_csrf'
const SESSION_COOKIE_NAME = 'sub3api_browser_session'

let accessToken: string | null = null
let refreshToken: string | null = null
let browserSession = false
let pendingRefreshToken: string | null = null
let sessionRevision = 0

function supportsCookieSession(): boolean {
  return new URL(getAPIBaseURL(), window.location.origin).origin === window.location.origin
}

export function markBrowserSession(): void {
  // Cross-origin deployments keep bearer tokens in memory because a readable
  // double-submit cookie cannot be shared across unrelated frontend hosts.
  browserSession = supportsCookieSession()
}

export function isBrowserSession(): boolean {
  return browserSession || hasBrowserSessionCookie()
}

export function hasBrowserSessionCookie(): boolean {
  return supportsCookieSession() && hasCookie(SESSION_COOKIE_NAME)
}

export function setAccessToken(token: string | null): void {
  const next = token && token.trim() ? token : null
  if (accessToken !== next) sessionRevision++
  accessToken = next
  localStorage.removeItem(LEGACY_ACCESS_TOKEN_KEY)
  localStorage.removeItem('refresh_token')
}

export function getAccessToken(): string | null {
  return accessToken || (isBrowserSession() ? null : localStorage.getItem(LEGACY_ACCESS_TOKEN_KEY))
}

export function setSessionRefreshToken(token: string | null): void {
  refreshToken = !isBrowserSession() && token && token.trim() ? token : null
  localStorage.removeItem('refresh_token')
}

export function getSessionRefreshToken(): string | null {
  return isBrowserSession() ? null : refreshToken || localStorage.getItem('refresh_token')
}

export function clearAccessToken(): void {
  accessToken = null
  refreshToken = null
  pendingRefreshToken = null
  browserSession = false
  sessionRevision++
  localStorage.removeItem(LEGACY_ACCESS_TOKEN_KEY)
  localStorage.removeItem('refresh_token')
  // Only the server can clear the HttpOnly refresh cookie. Keep its signed
  // CSRF companion until then so a failed logout/refresh cannot block login.
  document.cookie = `${SESSION_COOKIE_NAME}=; Max-Age=0; Path=/`
}

export function getSessionRevision(): number {
  return sessionRevision
}

export function setPendingRefreshToken(token: string | null): void {
  pendingRefreshToken = token
}

export function peekPendingRefreshToken(): string | null {
  return pendingRefreshToken
}

export function takePendingRefreshToken(): string | null {
  const token = pendingRefreshToken
  pendingRefreshToken = null
  return token
}

export function browserSessionHeader(): Record<string, string> {
  return isBrowserSession() ? { [BROWSER_SESSION_HEADER]: BROWSER_SESSION_VALUE } : {}
}

export function getCsrfToken(): string | null {
  const prefix = `${CSRF_COOKIE_NAME}=`
  const cookie = document.cookie.split(';').map((value) => value.trim()).find((value) => value.startsWith(prefix))
  if (!cookie) return null
  try {
    return decodeURIComponent(cookie.slice(prefix.length))
  } catch {
    return null
  }
}

function hasCookie(name: string): boolean {
  return document.cookie.split(';').some((value) => value.trim().startsWith(`${name}=`))
}

function storedUserID(value: string | null): unknown {
  try {
    return value ? JSON.parse(value)?.id : null
  } catch {
    return null
  }
}

// Cookies are shared across same-origin tabs, while access tokens are not.
// Drop the old tab's memory when another tab logs out or changes account.
window.addEventListener('storage', (event: StorageEvent) => {
  if (event.storageArea && event.storageArea !== window.localStorage) return
  if (event.key !== null && (event.key !== 'auth_user' || storedUserID(event.oldValue) === storedUserID(event.newValue))) return
  accessToken = null
  refreshToken = null
  pendingRefreshToken = null
  browserSession = false
  sessionRevision++
})

export const browserSessionConstants = {
  header: BROWSER_SESSION_HEADER,
  value: BROWSER_SESSION_VALUE,
  csrfCookie: CSRF_COOKIE_NAME,
  sessionCookie: SESSION_COOKIE_NAME
} as const
