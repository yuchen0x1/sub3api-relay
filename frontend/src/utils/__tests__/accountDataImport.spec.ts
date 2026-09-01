import { describe, expect, it } from 'vitest'
import { normalizeAccountDataPayload, parseAccountDataText } from '@/utils/accountDataImport'

describe('accountDataImport', () => {
  it('normalizes native Codex auth.json tokens and account identity', () => {
    const payload = normalizeAccountDataPayload({
      auth_mode: 'ChatGPT',
      last_refresh: '2026-08-30T00:00:00Z',
      tokens: {
        access_token: 'access-token',
        refresh_token: 'refresh-token',
        id_token: 'id-token',
        account_id: 'workspace-1'
      }
    }, 'auth.json')

    expect(payload.accounts).toHaveLength(1)
    expect(payload.accounts[0]).toMatchObject({
      platform: 'openai',
      type: 'oauth',
      credentials: {
        access_token: 'access-token',
        refresh_token: 'refresh-token',
        id_token: 'id-token',
        chatgpt_account_id: 'workspace-1'
      }
    })
  })

  it('normalizes ChatGPT session JSON with nested user/account fields', () => {
    const payload = normalizeAccountDataPayload({
      accessToken: 'session-access-token',
      idToken: 'session-id-token',
      expires: 1735689600000,
      user: { id: 'user-1', email: 'user@example.com', name: 'User' },
      account: { id: 'workspace-1', plan_type: 'plus' }
    }, 'session.json')

    expect(payload.accounts[0]).toMatchObject({
      name: 'User',
      platform: 'openai',
      credentials: {
        access_token: 'session-access-token',
        id_token: 'session-id-token',
        chatgpt_account_id: 'workspace-1',
        chatgpt_user_id: 'user-1',
        email: 'user@example.com',
        plan_type: 'plus'
      },
      expires_at: 1735689600
    })
  })

  it('normalizes CLIProxyAPI/Sub2API refresh-token arrays', () => {
    const payload = normalizeAccountDataPayload([
      { refresh_token: 'rt-one', email: 'one@example.com' },
      { refreshToken: 'rt-two', user: { email: 'two@example.com' } }
    ], 'credentials.json')

    expect(payload.accounts).toHaveLength(2)
    expect(payload.accounts.map((item) => item.credentials.refresh_token)).toEqual(['rt-one', 'rt-two'])
    expect(payload.accounts.map((item) => item.platform)).toEqual(['openai', 'openai'])
  })

  it('defaults provider-less OpenAI-compatible api keys to the OpenAI channel', () => {
    const payload = normalizeAccountDataPayload({
      name: 'custom gateway',
      base_url: 'https://api.example.com/v1',
      key: 'opaque-api-key'
    }, 'provider.json')
    expect(payload.accounts[0]).toMatchObject({
      platform: 'openai',
      type: 'apikey',
      credentials: { base_url: 'https://api.example.com/v1', api_key: 'opaque-api-key' }
    })
  })

  it('normalizes Cockpit API-account exports without dropping base URLs', () => {
    const payload = normalizeAccountDataPayload({
      account_name: 'Cockpit API account',
      api_provider_mode: 'openai',
      api_base_url: 'https://gateway.example.com/v1',
      openai_api_key: 'sk-cockpit-example'
    }, 'cockpit-codex-export.json')

    expect(payload.accounts[0]).toMatchObject({
      name: 'Cockpit API account',
      platform: 'openai',
      type: 'apikey',
      credentials: {
        base_url: 'https://gateway.example.com/v1',
        api_key: 'sk-cockpit-example'
      }
    })
  })

  it('normalizes Gemini service-account and Google ADC files', () => {
    const gemini = normalizeAccountDataPayload({
      type: 'service_account',
      project_id: 'demo-project',
      client_email: 'bot@demo-project.iam.gserviceaccount.com',
      private_key: '-----BEGIN PRIVATE KEY-----\\nkey\\n-----END PRIVATE KEY-----'
    }, 'service-account.json')
    expect(gemini.accounts[0]).toMatchObject({ platform: 'gemini', type: 'service_account' })
    expect(gemini.accounts[0]?.credentials.project_id).toBe('demo-project')

    const googleADC = normalizeAccountDataPayload({
      type: 'authorized_user',
      client_id: 'google-client',
      client_secret: 'google-secret',
      refresh_token: 'google-refresh'
    }, 'application_default_credentials.json')
    expect(googleADC.accounts[0]).toMatchObject({ platform: 'gemini', type: 'oauth' })
  })

  it('parses token-per-line TXT and JSONL formats', () => {
    const parsed = parseAccountDataText('rt-one\nrt-two\n', 'tokens.txt')
    const payload = normalizeAccountDataPayload(parsed, 'tokens.txt')
    expect(payload.accounts.map((item) => item.credentials.refresh_token)).toEqual(['rt-one', 'rt-two'])

    const jsonl = parseAccountDataText('{"access_token":"at-one"}\n{"access_token":"at-two"}', 'accounts.jsonl')
    const jsonlPayload = normalizeAccountDataPayload(jsonl, 'accounts.jsonl')
    expect(jsonlPayload.accounts.map((item) => item.credentials.access_token)).toEqual(['at-one', 'at-two'])

    const atTxt = parseAccountDataText('opaque-access-token-1\nopaque-access-token-2', 'access_tokens.at_txt')
    const atPayload = normalizeAccountDataPayload(atTxt, 'access_tokens.at_txt')
    expect(atPayload.accounts.map((item) => item.credentials.access_token)).toEqual([
      'opaque-access-token-1',
      'opaque-access-token-2'
    ])
  })

  it('rejects malformed JSON that is not a token-line file', () => {
    expect(() => parseAccountDataText('not json', 'accounts.json')).toThrow()
    expect(() => parseAccountDataText('hello', 'notes.txt')).toThrow()
    expect(() => normalizeAccountDataPayload({ name: 'missing credentials' }, 'accounts.json')).toThrow()
  })

  it('accepts account maps and generates proxies from proxy URLs', () => {
    const payload = normalizeAccountDataPayload({
      'one@example.com': { refresh_token: 'rt-one', email: 'one@example.com', proxy_url: 'socks5://u:p@127.0.0.1:1080' }
    }, 'pool.json')
    expect(payload.accounts[0]?.proxy_key).toContain('socks5|127.0.0.1|1080|u|p')
    expect(payload.proxies[0]).toMatchObject({ protocol: 'socks5', host: '127.0.0.1', port: 1080 })

    const httpsProxy = normalizeAccountDataPayload({
      api_key: 'opaque-key',
      proxy: 'https://proxy.example.com'
    }, 'https-proxy.json')
    expect(httpsProxy.proxies[0]).toMatchObject({ protocol: 'https', host: 'proxy.example.com', port: 443 })
  })

  it('rejects unsupported bundle envelopes even when records need normalization', () => {
    expect(() => normalizeAccountDataPayload({
      type: 'third-party-export',
      version: 1,
      proxies: [],
      accounts: [{ accessToken: 'token' }]
    }, 'bundle.json')).toThrow(/数据类型不受支持/)
  })
})
