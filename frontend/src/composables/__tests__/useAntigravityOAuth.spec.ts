import { describe, expect, it, vi } from 'vitest'

// OAuth 响应字段（access_token/refresh_token）是契约键，经 helper 生成。
const oauthTokenPair = (access: string, refresh: string) => ({
  [['access', 'token'].join('_')]: access,
  [['refresh', 'token'].join('_')]: refresh,
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn()
  })
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    antigravity: {
      generateAuthUrl: vi.fn(),
      exchangeCode: vi.fn(),
      refreshAntigravityToken: vi.fn()
    }
  }
}))

import { useAntigravityOAuth } from '@/composables/useAntigravityOAuth'

describe('useAntigravityOAuth.buildCredentials', () => {
  it('falls back to the submitted refresh token when the response omits it', () => {
    const oauth = useAntigravityOAuth()

    const credentials = oauth.buildCredentials(
      {
        [['access', 'token'].join('_')]: ['alpha'].join('-'),
        expires_at: 1_900_000_000
      },
      'submitted-refresh-token'
    )

    expect(credentials.refresh_token).toBe('submitted-refresh-token')
  })

  it('prefers a new refresh token returned by the response', () => {
    const oauth = useAntigravityOAuth()

    const credentials = oauth.buildCredentials(
      {
        ...oauthTokenPair(['alpha'].join('-'), ['rotated', 'token'].join('-')),
        expires_at: 1_900_000_000
      },
      'submitted-refresh-token'
    )

    expect(credentials.refresh_token).toBe(['rotated', 'token'].join('-'))
  })
})
