import axios from 'axios'
import type { ApiResponse } from '@/types'
import { getAPIBaseURL } from './url'
import {
  browserSessionHeader,
  getAccessToken,
  getCsrfToken,
  getSessionRefreshToken,
  getSessionRevision,
  hasBrowserSessionCookie,
  isBrowserSession,
  markBrowserSession,
  setAccessToken,
  setSessionRefreshToken,
} from './browserSession'

const AUTH_USER_KEY = 'auth_user'
const TOKEN_EXPIRES_AT_KEY = 'token_expires_at'
const TOKEN_REFRESH_LOCK_NAME = 'sub3api-auth-token-refresh'
const BROWSER_REFRESH_LEASE_KEY = 'sub3api_browser_refresh_lease'
const TOKEN_REFRESH_TIMEOUT_MS = 30_000
const PEER_REFRESH_GRACE_MS = 1_000
const PEER_REFRESH_POLL_MS = 25

export interface RefreshTokenResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  token_type: string
}

export interface RefreshAuthTokensOptions {
  /** Access token attached to the request that received a 401 response. */
  failedAccessToken?: string | null
}

interface AuthSnapshot {
  accessToken: string | null
  refreshToken: string
  expiresAt: number
  userID: number | null
  sessionRevision: number
}

let inFlightRefresh: Promise<RefreshTokenResponse> | null = null

function getStoredUserID(): number | null {
  const rawUser = localStorage.getItem(AUTH_USER_KEY)
  if (!rawUser) {
    return null
  }

  try {
    const id = Number((JSON.parse(rawUser) as { id?: unknown }).id)
    return Number.isFinite(id) && id > 0 ? id : null
  } catch {
    return null
  }
}

function readAuthSnapshot(): AuthSnapshot {
  const refreshToken = getSessionRefreshToken()
  if (!refreshToken && !isBrowserSession()) {
    throw new Error('No refresh token available')
  }

  return {
    accessToken: getAccessToken(),
    refreshToken: refreshToken || '',
    expiresAt: Number(localStorage.getItem(TOKEN_EXPIRES_AT_KEY)),
    userID: getStoredUserID(),
    sessionRevision: getSessionRevision()
  }
}

function persistTokenPair(tokens: RefreshTokenResponse): void {
  setAccessToken(tokens.access_token)
  setSessionRefreshToken(tokens.refresh_token)
  localStorage.setItem(TOKEN_EXPIRES_AT_KEY, String(Date.now() + tokens.expires_in * 1000))
}

async function requestTokenPair(
  snapshot: AuthSnapshot,
  browser = false
): Promise<RefreshTokenResponse> {
  // A peer may have completed legacy migration while this tab waited for the
  // lock. Read the current cookie marker and CSRF token inside the lock.
  const refreshToken = browser && hasBrowserSessionCookie() ? '' : snapshot.refreshToken
  const response = await axios.post<ApiResponse<RefreshTokenResponse>>(
    `${getAPIBaseURL()}/auth/refresh`,
    refreshToken ? { refresh_token: refreshToken } : {},
    {
      headers: {
        'Content-Type': 'application/json',
        ...(browser ? browserSessionHeader() : {}),
        ...(browser && getCsrfToken() ? { 'X-CSRF-Token': getCsrfToken()! } : {})
      },
      timeout: TOKEN_REFRESH_TIMEOUT_MS,
      withCredentials: true
    }
  )
  const payload = response.data
  if (payload.code !== 0 || !payload.data) {
    throw new Error(payload.message || 'Token refresh failed')
  }

  if (
    getSessionRevision() !== snapshot.sessionRevision ||
    getStoredUserID() !== snapshot.userID
  ) {
    throw new Error('Session changed during token refresh')
  }

  persistTokenPair(payload.data)
  return payload.data
}

async function withBrowserRefreshLease<T>(refresh: () => Promise<T>): Promise<T> {
  // randomUUID and Web Locks require a secure context; HTTP self-hosted
  // installations still need a unique (non-secret) coordination identifier.
  const owner = globalThis.crypto?.randomUUID?.() || `${Date.now()}-${Math.random()}`
  const deadline = Date.now() + TOKEN_REFRESH_TIMEOUT_MS * 2
  while (Date.now() < deadline) {
    let lease: { owner?: string; expiresAt?: number } | null = null
    try {
      lease = JSON.parse(localStorage.getItem(BROWSER_REFRESH_LEASE_KEY) || 'null')
    } catch {
      // Ignore a stale or corrupt coordination record. It contains no tokens.
    }
    if (!lease || !lease.expiresAt || lease.expiresAt <= Date.now()) {
      const value = JSON.stringify({ owner, expiresAt: Date.now() + TOKEN_REFRESH_TIMEOUT_MS + PEER_REFRESH_GRACE_MS })
      localStorage.setItem(BROWSER_REFRESH_LEASE_KEY, value)
      await new Promise((resolve) => window.setTimeout(resolve, PEER_REFRESH_POLL_MS))
      if (localStorage.getItem(BROWSER_REFRESH_LEASE_KEY) === value) {
        try {
          return await refresh()
        } finally {
          if (localStorage.getItem(BROWSER_REFRESH_LEASE_KEY) === value) {
            localStorage.removeItem(BROWSER_REFRESH_LEASE_KEY)
          }
        }
      }
    }
    await new Promise((resolve) => window.setTimeout(resolve, PEER_REFRESH_POLL_MS))
  }
  throw new Error('Timed out waiting for browser session refresh')
}

async function runRefresh(options: RefreshAuthTokensOptions): Promise<RefreshTokenResponse> {
  const snapshot = readAuthSnapshot()
  if (options.failedAccessToken && snapshot.accessToken && snapshot.accessToken !== options.failedAccessToken && snapshot.expiresAt > Date.now()) {
    return {
      access_token: snapshot.accessToken,
      refresh_token: snapshot.refreshToken,
      expires_in: Math.max(1, Math.ceil((snapshot.expiresAt - Date.now()) / 1000)),
      token_type: 'Bearer'
    }
  }
  markBrowserSession()
  const browser = isBrowserSession()
  const refresh = () => requestTokenPair(snapshot, browser)

  if (typeof navigator !== 'undefined' && navigator.locks) {
    return navigator.locks.request(TOKEN_REFRESH_LOCK_NAME, refresh)
  }

  return browser ? withBrowserRefreshLease(refresh) : refresh()
}

/**
 * Refresh and persist the browser session.
 *
 * Calls in the same document share one promise. Cookie refreshes are serialized
 * across tabs; every response is checked against logout/account changes.
 */
export function refreshAuthTokens(
  options: RefreshAuthTokensOptions = {}
): Promise<RefreshTokenResponse> {
  if (inFlightRefresh) {
    return inFlightRefresh
  }

  const pending = runRefresh(options)
  inFlightRefresh = pending
  const clearPending = (): void => {
    if (inFlightRefresh === pending) {
      inFlightRefresh = null
    }
  }
  void pending.then(clearPending, clearPending)
  return pending
}
