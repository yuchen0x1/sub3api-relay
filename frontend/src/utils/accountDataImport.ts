import type {
  AccountPlatform,
  AccountType,
  AdminDataAccount,
  AdminDataPayload,
  AdminDataProxy,
} from '@/types'

type RecordValue = Record<string, unknown>

const DATA_TYPE = 'sub3api-data'
const DATA_VERSION = 1

const isRecord = (value: unknown): value is RecordValue =>
  Boolean(value) && typeof value === 'object' && !Array.isArray(value)

const firstString = (...values: unknown[]): string | undefined => {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) return value.trim()
  }
  return undefined
}

const numberValue = (value: unknown, fallback = 0): number => {
  const parsed = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

const timestampSeconds = (value: unknown): number | undefined => {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return Math.floor(value > 1e11 ? value / 1000 : value)
  }
  if (typeof value === 'string' && value.trim()) {
    const numeric = Number(value)
    if (Number.isFinite(numeric)) return Math.floor(numeric > 1e11 ? numeric / 1000 : numeric)
    const parsed = Date.parse(value)
    if (Number.isFinite(parsed)) return Math.floor(parsed / 1000)
  }
  return undefined
}

const canonicalPlatform = (value: unknown): AccountPlatform | undefined => {
  const normalized = firstString(value)?.toLowerCase().replace(/_/g, '-')
  switch (normalized) {
    case 'openai':
    case 'codex':
    case 'chatgpt':
    case 'sora':
      return 'openai'
    case 'anthropic':
    case 'claude':
      return 'anthropic'
    case 'gemini':
    case 'google':
    case 'google-ai':
    case 'vertex':
      return 'gemini'
    case 'antigravity':
      return 'antigravity'
    case 'grok':
    case 'xai':
      return 'grok'
    case 'manus':
      return 'manus'
    default:
      return undefined
  }
}

const canonicalType = (value: unknown): AccountType | undefined => {
  const normalized = firstString(value)?.toLowerCase().replace(/_/g, '-').replace(/ /g, '-')
  switch (normalized) {
    case 'oauth':
    case 'oauth2':
      return 'oauth'
    case 'setup-token':
    case 'setuptoken':
    case 'session-token':
    case 'sessiontoken':
      return 'setup-token'
    case 'apikey':
    case 'api-key':
    case 'key':
      return 'apikey'
    case 'upstream':
    case 'proxy':
      return 'upstream'
    case 'bedrock':
      return 'bedrock'
    case 'service-account':
    case 'serviceaccount':
      return 'service_account'
    default:
      return undefined
  }
}

const credentialAliases: Record<string, string> = {
  accessToken: 'access_token',
  refreshToken: 'refresh_token',
  idToken: 'id_token',
  sessionToken: 'session_token',
  setupToken: 'setup_token',
  apiKey: 'api_key',
  key: 'api_key',
  openai_api_key: 'api_key',
  openAIKey: 'api_key',
  openaiApiKey: 'api_key',
  baseURL: 'base_url',
  api_base_url: 'base_url',
  apiBaseURL: 'base_url',
  apiBaseUrl: 'base_url',
  bedrock_api_key: 'api_key',
  personal_access_token: 'access_token',
  baseUrl: 'base_url',
  accountId: 'chatgpt_account_id',
  chatgptAccountId: 'chatgpt_account_id',
  account_id: 'chatgpt_account_id',
  userId: 'chatgpt_user_id',
  chatgptUserId: 'chatgpt_user_id',
  user_id: 'chatgpt_user_id',
  organizationId: 'organization_id',
  orgId: 'organization_id',
  projectId: 'project_id',
  clientId: 'client_id',
  clientSecret: 'client_secret',
  privateKey: 'private_key',
  privateKeyId: 'private_key_id',
  clientEmail: 'client_email',
  accessKeyId: 'aws_access_key_id',
  secretAccessKey: 'aws_secret_access_key',
  awsAccessKeyId: 'aws_access_key_id',
  awsSecretAccessKey: 'aws_secret_access_key',
  awsSessionToken: 'aws_session_token',
  awsRegion: 'aws_region',
  access_key_id: 'aws_access_key_id',
  secret_access_key: 'aws_secret_access_key',
  aws_access_key: 'aws_access_key_id',
  aws_secret_key: 'aws_secret_access_key',
  region: 'aws_region',
  planType: 'plan_type',
  expiresAt: 'expires_at',
  expiry: 'expires_at',
  expiration: 'expires_at',
  expiry_date: 'expires_at',
  tokenType: 'token_type',
}

const credentialKeys = new Set([
  'access_token',
  'refresh_token',
  'id_token',
  'session_token',
  'setup_token',
  'api_key',
  'openai_api_key',
  'key',
  'token',
  'base_url',
  'chatgpt_account_id',
  'chatgpt_user_id',
  'organization_id',
  'project_id',
  'client_id',
  'client_secret',
  'private_key',
  'private_key_id',
  'client_email',
  'aws_access_key_id',
  'aws_secret_access_key',
  'aws_session_token',
  'aws_region',
  'bedrock_api_key',
  'bedrock_access_keys',
  'personal_access_token',
  'access_key_id',
  'secret_access_key',
  'region',
  'service_account',
  'service_account_json',
  'expires_at',
  'expires_in',
  'expiry',
  'scope',
  'scopes',
  'token_type',
  'plan_type',
  'model_mapping',
  'header_overrides',
  'aws_access_key_id',
  'aws_secret_access_key',
  'aws_session_token',
  'bedrock_api_key',
])

const withAliases = (source: RecordValue): RecordValue => {
  const result: RecordValue = {}
  for (const [key, value] of Object.entries(source)) {
    if (value === undefined || value === null || value === '') continue
    result[credentialAliases[key] || key] = value
  }
  return result
}

const buildCredentials = (source: RecordValue): Record<string, unknown> => {
  const credentials: RecordValue = {}
  if (isRecord(source.credentials)) Object.assign(credentials, withAliases(source.credentials))

  for (const nestedKey of [
    'tokens',
    'token',
    'auth',
    'oauth',
    'auth_info',
    'claudeAiOauth',
    'claude_ai_oauth',
    'geminiOauth',
    'oauth_creds',
  ]) {
    const nested = source[nestedKey]
    if (isRecord(nested)) Object.assign(credentials, withAliases(nested))
  }

  for (const [key, value] of Object.entries(source)) {
    const normalizedKey = credentialAliases[key] || key
    if (credentialKeys.has(normalizedKey) && value !== undefined && value !== null && value !== '') {
      credentials[normalizedKey] = value
    }
  }
  // AWS SDK credential files commonly use sessionToken/region without the
  // aws_ prefix. Preserve the project-specific credential names expected by
  // the Bedrock signer when the source is explicitly a Bedrock record.
  if (canonicalType(source.type) === 'bedrock') {
    if (credentials.session_token && !credentials.aws_session_token) {
      credentials.aws_session_token = credentials.session_token
      delete credentials.session_token
    }
    if (credentials.region && !credentials.aws_region) {
      credentials.aws_region = credentials.region
      delete credentials.region
    }
  }
  const openAIKey = firstString(
    source.OPENAI_API_KEY,
    source.openai_api_key,
    source.openAIKey,
    source.openaiApiKey
  )
  if (openAIKey) credentials.api_key = openAIKey

  // ChatGPT /api/auth/session keeps identity under user/account objects.
  const nestedIdentity: Array<[string, unknown]> = [
    ['email', isRecord(source.user) ? source.user.email : undefined],
    ['chatgpt_user_id', isRecord(source.user) ? (source.user.id ?? source.user.user_id) : undefined],
    ['chatgpt_account_id', isRecord(source.account) ? (source.account.id ?? source.account.account_id) : undefined],
    ['plan_type', isRecord(source.account) ? (source.account.plan_type ?? source.account.planType) : undefined],
    ['organization_id', isRecord(source.user) && Array.isArray(source.user.organizations)
      ? (source.user.organizations.find((org) => isRecord(org) && org.id)?.id)
      : undefined],
  ]
  for (const [key, value] of nestedIdentity) {
    if (credentials[key] === undefined && value !== undefined && value !== null && value !== '') {
      credentials[key] = value
    }
  }

  const bedrockKeys = isRecord(source.bedrock_access_keys) ? source.bedrock_access_keys : undefined
  if (bedrockKeys) {
    const accessKeyID = firstString(bedrockKeys.access_key_id, bedrockKeys.aws_access_key_id)
    const secretAccessKey = firstString(bedrockKeys.secret_access_key, bedrockKeys.aws_secret_access_key)
    const region = firstString(bedrockKeys.region, bedrockKeys.aws_region)
    if (accessKeyID) credentials.aws_access_key_id = accessKeyID
    if (secretAccessKey) credentials.aws_secret_access_key = secretAccessKey
    if (region) credentials.aws_region = region
  }

  // A service-account file is itself the credential object. Keep it nested so
  // the backend can apply its token_uri allowlist and key validation.
  if (canonicalType(source.type) === 'service_account' && !credentials.service_account_json) {
    const serviceAccount = { ...source }
    delete serviceAccount.name
    delete serviceAccount.type
    credentials.service_account_json = JSON.stringify(serviceAccount)
  }
  return credentials
}

const inferPlatform = (source: RecordValue, credentials: RecordValue, sourceName = ''): AccountPlatform | undefined => {
  if (isRecord(source.claudeAiOauth) || /claude/i.test(sourceName)) return 'anthropic'
  // Google ADC / Gemini OAuth files use the same `authorized_user` marker.
  if (source.type === 'authorized_user') return 'gemini'
  if (/(?:gemini|google|oauth_creds|vertex)/i.test(sourceName)) return 'gemini'
  if (canonicalType(source.type) === 'bedrock') return 'anthropic'
  const explicit = canonicalPlatform(source.platform ?? source.provider ?? source.vendor ?? source.service)
  if (explicit) return explicit

  const providerType = canonicalPlatform(source.type)
  if (providerType) return providerType

  const key = firstString(
    credentials.api_key,
    credentials.token,
    credentials.access_token,
    credentials.refresh_token
  )
  if (key?.startsWith('sk-ant-')) return 'anthropic'
  if (key?.startsWith('AIza')) return 'gemini'
  if (key?.startsWith('xai-')) return 'grok'
  if (credentials.project_id || credentials.service_account || credentials.service_account_json) return 'gemini'
  if (credentials.aws_access_key_id || credentials.aws_secret_access_key || source.type === 'bedrock' || source.bedrock_api_key || source.bedrock_access_keys) return 'anthropic'
  if (credentials.chatgpt_account_id || credentials.id_token || credentials.refresh_token || credentials.access_token) {
    return 'openai'
  }
  // A provider-less api_key/base_url pair is the common OpenAI-compatible
  // account shape. OpenAI is the safest canonical platform fallback because
  // the backend's generic API-key path accepts custom base URLs.
  if (credentials.api_key || credentials.base_url || credentials.token) return 'openai'
  return undefined
}

const inferType = (source: RecordValue, credentials: RecordValue): AccountType | undefined => {
  const explicit = canonicalType(source.account_type ?? source.auth_type ?? source.type)
  if (explicit) return explicit
  if (credentials.service_account || credentials.service_account_json || credentials.private_key) return 'service_account'
  if (credentials.aws_access_key_id || credentials.aws_secret_access_key || source.type === 'bedrock') return 'bedrock'
  if (credentials.setup_token) return 'setup-token'
  // Claude Code setup tokens use the sk-ant-oat prefix and are sometimes
  // exported under a generic token/refresh_token key.
  const token = firstString(credentials.token, credentials.access_token, credentials.refresh_token)
  if (token?.toLowerCase().startsWith('sk-ant-oat')) return 'setup-token'
  if (credentials.access_token || credentials.refresh_token || credentials.id_token) return 'oauth'
  if (credentials.session_token) return 'setup-token'
  if (credentials.api_key || credentials.token) return 'apikey'
  return undefined
}

const accountName = (source: RecordValue, credentials: RecordValue, index: number): string =>
  firstString(
    source.name,
    source.account_name,
    source.accountName,
    source.email,
    source.user_email,
    isRecord(source.user) ? source.user.name : undefined,
    credentials.email,
    credentials.email_address,
    source.account_id,
    credentials.chatgpt_account_id,
    `imported-account-${index + 1}`
  ) as string

const normalizeAccount = (input: unknown, index: number, sourceName = ''): AdminDataAccount => {
  const source: RecordValue = typeof input === 'string' ? { access_token: input } : { ...(isRecord(input) ? input : {}) }
  const credentials = buildCredentials(source)
  const platform = inferPlatform(source, credentials, sourceName)
  const type = inferType(source, credentials)
  if (!platform || !type) {
    throw new Error(`accounts[${index}] 无法识别 platform/type，请提供 provider/platform 和 credentials`)
  }
  if (type === 'setup-token' && !firstString(credentials.setup_token)) {
    const setupToken = firstString(
      credentials.token,
      credentials.access_token,
      credentials.refresh_token,
      credentials.session_token
    )
    if (setupToken) credentials.setup_token = setupToken
  }

  const expiresAt = timestampSeconds(
    source.expires_at ?? source.expiresAt ?? source.expires ?? source.expired ?? source.expiry ?? source.expiry_date ?? credentials.expires_at ?? credentials.expiry
  )
  if (expiresAt !== undefined) credentials.expires_at = expiresAt

  const extra: RecordValue = isRecord(source.extra) ? { ...source.extra } : {}
  if (isRecord(source.attributes)) Object.assign(extra, source.attributes)
  for (const key of ['last_refresh', 'source', 'mailbox', 'disabled', 'subscription_tier', 'entitlement_status']) {
    if (source[key] !== undefined && source[key] !== null && source[key] !== '') extra[key] = source[key]
  }

  return {
    name: accountName(source, credentials, index),
    notes: typeof source.notes === 'string' ? source.notes : null,
    platform,
    type,
    credentials,
    extra: Object.keys(extra).length ? extra : undefined,
    proxy_key: firstString(source.proxy_key, source.proxyKey) || null,
    concurrency: Math.max(0, Math.floor(numberValue(source.concurrency, 0))),
    priority: Math.max(0, Math.floor(numberValue(source.priority, 0))),
    rate_multiplier: source.rate_multiplier === undefined ? undefined : Math.max(0, numberValue(source.rate_multiplier, 1)),
    expires_at: expiresAt,
    auto_pause_on_expired: typeof source.auto_pause_on_expired === 'boolean' ? source.auto_pause_on_expired : undefined,
  }
}

const proxyKey = (protocol: string, host: string, port: number, username = '', password = '') =>
  `${protocol}|${host}|${port}|${username}|${password}`

const normalizeProxy = (input: unknown): AdminDataProxy | undefined => {
  if (typeof input === 'string') input = { url: input }
  if (!isRecord(input)) return undefined
  const rawUrl = firstString(input.url, input.proxy_url, input.proxyUrl)
  let protocol = firstString(input.protocol, input.type)?.toLowerCase()
  let host = firstString(input.host, input.server, input.address)
  let port = numberValue(input.port, 0)
  let username = firstString(input.username, input.user) || ''
  let password = firstString(input.password, input.pass) || ''
  if (rawUrl) {
    try {
      const parsed = new URL(rawUrl)
      protocol = parsed.protocol.replace(':', '')
      host = parsed.hostname
      port = Number(parsed.port) || (protocol.startsWith('socks') ? 1080 : protocol === 'https' ? 443 : 80)
      username = decodeURIComponent(parsed.username)
      password = decodeURIComponent(parsed.password)
    } catch {
      return undefined
    }
  }
  if (protocol === 'socks') protocol = 'socks5'
  if (!protocol || !host || !Number.isInteger(port) || port <= 0 || port > 65535) return undefined
  if (!['http', 'https', 'socks5', 'socks5h'].includes(protocol)) return undefined
  const name = firstString(input.name, input.label, `${protocol}-${host}-${port}`) as string
  return {
    proxy_key: firstString(input.proxy_key, input.proxyKey) || proxyKey(protocol, host, port, username, password),
    name,
    protocol: protocol as AdminDataProxy['protocol'],
    host,
    port,
    username: username || null,
    password: password || null,
    options: isRecord(input.options)
      ? Object.entries(input.options).reduce<Record<string, string>>((out, [key, value]) => {
          if (typeof value === 'string') out[key] = value
          return out
        }, {})
      : undefined,
    tls_insecure: input.tls_insecure === true || input.skip_tls_verify === true,
    status: input.status === 'inactive' || input.disabled === true ? 'inactive' : 'active',
  }
}

const canonicalPayload = (value: unknown): value is AdminDataPayload =>
  isRecord(value) && Array.isArray(value.accounts) && Array.isArray(value.proxies)

const extractRecords = (value: unknown): { accounts: unknown[]; proxies: unknown[] } | undefined => {
  if (Array.isArray(value)) return { accounts: value, proxies: [] }
  if (!isRecord(value)) return undefined
  if (canonicalPayload(value)) return { accounts: value.accounts, proxies: value.proxies }
  if (Array.isArray(value.accounts)) return { accounts: value.accounts, proxies: Array.isArray(value.proxies) ? value.proxies : [] }
  if (Array.isArray(value.items)) return { accounts: value.items, proxies: [] }
  if (Array.isArray(value.users)) return { accounts: value.users, proxies: [] }
  if (Array.isArray(value.profiles)) return { accounts: value.profiles, proxies: [] }
  if (isRecord(value.data)) return extractRecords(value.data)
  if (Array.isArray(value.data)) return { accounts: value.data, proxies: [] }
  if (isRecord(value.account)) return { accounts: [{ ...value.account, ...value }], proxies: [] }
  if (
    isRecord(value.tokens) ||
    isRecord(value.token) ||
    isRecord(value.credentials) ||
    Object.keys(value).some((key) => credentialKeys.has(credentialAliases[key] || key)) ||
    typeof value.OPENAI_API_KEY === 'string'
  ) {
    return { accounts: [value], proxies: [] }
  }
  // Some pool managers export a map keyed by email/account id instead of an
  // array. Accept each credential-bearing value as one account record.
  const nestedAccounts = Object.values(value).filter(
    (candidate) => isRecord(candidate) && (
      isRecord(candidate.credentials) ||
      isRecord(candidate.tokens) ||
      Object.keys(candidate).some((key) => credentialKeys.has(credentialAliases[key] || key))
    )
  )
  if (nestedAccounts.length > 0) return { accounts: nestedAccounts, proxies: [] }
  return undefined
}

export const parseAccountDataText = (text: string, fileName = ''): unknown => {
  const trimmed = text.replace(/^\uFEFF/, '').trim()
  if (!trimmed) throw new Error('文件为空')
  try {
    return JSON.parse(trimmed)
  } catch (jsonError) {
    if (!/(?:\.jsonl|\.ndjson|\.txt|\.tokens|\.at[-_]?txt|\.rt[-_]?txt)$/i.test(fileName)) throw jsonError
    const lines = trimmed.split(/\r?\n/).map((line) => line.trim()).filter(Boolean)
    if (!lines.length) throw jsonError
    const isLineFile = /\.(?:txt|jsonl|ndjson|tokens|at[-_]?txt|rt[-_]?txt)$/i.test(fileName)
    const looksLikeToken = (line: string) =>
      line.split('.').length === 3 || /^(?:eyJ|at[-_]|rt[-_]|sk[-_]|sess(?:ion)?[-_]|oauth[-_])/i.test(line)
    // Keep malformed .json files as parse errors; only explicit line files or
    // token-shaped lines are eligible for the token-per-line fallback.
    if (!isLineFile && !lines.every(looksLikeToken)) throw jsonError
    if (isLineFile && !lines.every((line) => looksLikeToken(line) || line.length >= 16)) throw jsonError
    const isAccessTokenFile = /(?:\.at[-_]?txt|(?:^|[._-])access[-_]?tokens?(?:$|[._-]))/i.test(fileName)
    const records = lines.map((line) => {
      if (line.startsWith('{') || line.startsWith('[')) {
        try { return JSON.parse(line) } catch { /* treat as token line below */ }
      }
      const isJWT = line.split('.').length === 3
      return isAccessTokenFile || isJWT ? { access_token: line } : { refresh_token: line }
    })
    return records
  }
}

export function normalizeAccountDataPayload(input: unknown, sourceName = ''): AdminDataPayload {
  if (canonicalPayload(input)) {
    // A payload with both `accounts` and `proxies` is a data bundle. Validate
    // its envelope before considering whether the individual records use an
    // external account shape; otherwise an invalid bundle header could be
    // silently accepted just because its records need normalization.
    if (input.type && input.type !== DATA_TYPE && input.type !== 'sub3api-bundle') {
      throw new Error(`${sourceName || '文件'} 数据类型不受支持: ${input.type}`)
    }
    if (input.version !== undefined && input.version !== 0 && input.version !== DATA_VERSION) {
      throw new Error(`${sourceName || '文件'} 数据版本不受支持: ${input.version}`)
    }
    const hasExternalAccountShape = input.accounts.some((item) => isRecord(item) && (
      Object.keys(item).some((key) => credentialKeys.has(credentialAliases[key] || key)) ||
      isRecord(item.tokens) || isRecord(item.token) || isRecord(item.credentials) ||
      item.provider !== undefined || item.accessToken !== undefined || item.refreshToken !== undefined
    ))
    const hasExternalProxyShape = input.proxies.some((item) => isRecord(item) && (
      item.url !== undefined || item.proxy_url !== undefined || item.proxyUrl !== undefined
    ))
    if (!hasExternalAccountShape && !hasExternalProxyShape) {
      const supportedType = input.type === 'sub3api-bundle' ? input.type : DATA_TYPE
      return {
        ...input,
        type: supportedType,
        version: input.version || DATA_VERSION,
        exported_at: input.exported_at || new Date().toISOString(),
      }
    }
  }
  const extracted = extractRecords(input)
  if (!extracted || !extracted.accounts.length) throw new Error(`${sourceName || '文件'} 不包含可识别的账号数据`)
  const accounts: AdminDataAccount[] = []
  const generatedProxies: AdminDataProxy[] = []
  extracted.accounts.forEach((record, index) => {
    const normalized = normalizeAccount(record, index, sourceName)
    const source = isRecord(record) ? record : {}
    const proxy = normalizeProxy(source.proxy ?? source.proxy_url ?? source.proxyUrl)
    if (proxy) {
      const existing = generatedProxies.find((item) => item.proxy_key === proxy.proxy_key)
      if (!existing) generatedProxies.push(proxy)
      normalized.proxy_key = proxy.proxy_key
    }
    accounts.push(normalized)
  })
  const importedProxies = extracted.proxies
    .map((proxy) => {
      const raw = isRecord(proxy) ? proxy : undefined
      const hasURL = raw && (raw.url !== undefined || raw.proxy_url !== undefined || raw.proxyUrl !== undefined)
      return hasURL ? normalizeProxy(proxy) : proxy
    })
    .filter((proxy): proxy is AdminDataProxy => Boolean(proxy))
  const proxies = importedProxies.length ? importedProxies : generatedProxies
  return {
    type: DATA_TYPE,
    version: DATA_VERSION,
    exported_at: new Date().toISOString(),
    proxies: proxies as AdminDataProxy[],
    accounts,
  }
}
