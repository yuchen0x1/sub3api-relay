import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/components/account/CreateAccountModal.vue'),
  'utf8'
)
const oauthFlowSource = readFileSync(
  resolve(process.cwd(), 'src/components/account/OAuthAuthorizationFlow.vue'),
  'utf8'
)

describe('CreateAccountModal Grok account types', () => {
  it('offers Build OAuth, Web SSO import, and Console API-key setup', () => {
    expect(source).toContain('data-testid="grok-account-type-build-oauth"')
    expect(source).toContain('data-testid="grok-account-type-web-sso"')
    expect(source).toContain('data-testid="grok-account-type-api-key"')
    expect(source).toContain("grokPreferredInputMethod = 'manual'")
    expect(source).toContain("grokPreferredInputMethod = 'sso_cookie'")
    expect(source).toContain("@click=\"accountCategory = 'apikey'\"")
    expect(source).toContain(":initial-input-method=\"form.platform === 'grok' ? grokPreferredInputMethod : 'manual'\"")
    expect(source).toContain("t('admin.accounts.grokProviderHint')")
    expect(source).toContain("newPlatform === 'grok'")
    expect(source).toContain("? 'https://api.x.ai/v1'")
    expect(source).toContain("form.platform === 'grok'")
    expect(source).toContain("? 'xai-...'")
  })

  it('exposes custom upstream URL and header override for the OAuth create flow', () => {
    expect(source).toContain('data-testid="grok-custom-base-url-toggle"')
    expect(source).toContain('data-testid="grok-custom-base-url-input"')
    expect(source).toContain('form.platform === \'grok\' && isOAuthFlow')
  })

  it('loads one-per-line Grok SSO keys from TXT files and removes duplicates', () => {
    expect(oauthFlowSource).toContain('accept=".txt,text/plain"')
    expect(oauthFlowSource).toContain('@change="handleSSOFile"')
    expect(oauthFlowSource).toContain("ssoCookieInput.value = [...new Set(tokens)].join('\\n')")
    expect(oauthFlowSource).toContain("t(getOAuthKey('selectSSOFile'))")
  })

  it('checks Grok SSO keys in observable chunks and reports valid and invalid counts', () => {
    expect(source).toContain('const chunkSize = 3')
    expect(source).toContain('index_offset: offset')
    expect(source).toContain('total_count: ssoTokens.length')
    expect(source).toContain(':sso-import-progress="grokSSOImportProgress"')
    expect(oauthFlowSource).toContain('data-testid="grok-sso-import-progress"')
    expect(oauthFlowSource).toContain("t(getOAuthKey('ssoValidCount')")
    expect(oauthFlowSource).toContain("t(getOAuthKey('ssoInvalidCount')")
  })

  it('validates and applies upstream config on all three Grok OAuth create paths', () => {
    // 授权码兑换 / RT 批量 / SSO 批量 3 处调用（定义为箭头函数，不计入）
    expect(source.match(/validateGrokOAuthUpstreamConfig\(\)/g)?.length).toBe(3)
    expect(source.match(/applyGrokOAuthUpstreamConfig\(credentials\)/g)?.length).toBe(3)
  })
})
