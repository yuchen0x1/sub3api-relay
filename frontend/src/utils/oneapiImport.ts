/**
 * Parsers for importing upstream credentials exported from OneAPI / NewAPI
 * deployments (or plain key lists) into Sub3API accounts.
 *
 * The parse itself is intentionally frontend-only: the pasted payload never
 * leaves the browser until the user confirms the preview and the accounts are
 * created through the regular admin batch-create endpoint.
 */

export interface ParsedImportAccount {
  platform: 'openai' | 'gemini'
  type: 'apikey'
  name: string
  baseUrl: string // '' = provider default (official API)
  apiKey: string
  sourceStatus: number // OneAPI channel status: 1 enabled, 2 manually disabled, 3 auto disabled
  sourceGroup: string
  models: string
}

export interface ParsedImportSkip {
  name: string
  reason: string
}

export type ImportPayloadFormat = 'oneapi_json' | 'text_keys'

export interface ParsedImportResult {
  format: ImportPayloadFormat
  accounts: ParsedImportAccount[]
  skipped: ParsedImportSkip[]
}

// OneAPI channel type constants that map onto Sub3API platforms we support.
// Everything else (anthropic, vertex, cloudflare, ...) is reported as skipped.
const ONEAPI_TYPE_OPENAI = 1
const ONEAPI_TYPE_CUSTOM = 8
const ONEAPI_TYPE_GEMINI = 24

interface OneApiChannel {
  id?: number
  type?: number
  name?: string
  key?: string
  status?: number
  base_url?: string
  models?: string
  group?: string
}

function uniqueName(base: string, used: Set<string>): string {
  if (!used.has(base)) {
    used.add(base)
    return base
  }
  let n = 2
  while (used.has(`${base}-${n}`)) n += 1
  used.add(`${base}-${n}`)
  return `${base}-${n}`
}

function normalizeBaseUrl(raw: string): string {
  const trimmed = raw.trim().replace(/\/+$/, '')
  return trimmed
}

function parseOneApiChannel(
  channel: OneApiChannel,
  result: ParsedImportResult,
  usedNames: Set<string>
): void {
  const name = (channel.name || '').trim() || `channel-${channel.id ?? 'unknown'}`
  const type = Number(channel.type)

  let platform: ParsedImportAccount['platform']
  if (type === ONEAPI_TYPE_OPENAI || type === ONEAPI_TYPE_CUSTOM) {
    platform = 'openai'
  } else if (type === ONEAPI_TYPE_GEMINI) {
    platform = 'gemini'
  } else {
    result.skipped.push({ name, reason: `unsupported channel type: ${type}` })
    return
  }

  const keys = (channel.key || '')
    .split('\n')
    .map((k) => k.trim())
    .filter(Boolean)
  if (keys.length === 0) {
    result.skipped.push({ name, reason: 'missing api key' })
    return
  }

  const baseUrl = normalizeBaseUrl(channel.base_url || '')
  for (const key of keys) {
    result.accounts.push({
      platform,
      type: 'apikey',
      name: uniqueName(name, usedNames),
      baseUrl,
      apiKey: key,
      sourceStatus: Number(channel.status ?? 1),
      sourceGroup: (channel.group || '').trim(),
      models: (channel.models || '').trim()
    })
  }
}

function parseTextKeys(text: string, result: ParsedImportResult): void {
  let idx = 0
  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine.trim()
    if (!line || line.startsWith('#')) continue

    const tokens = line.split(/[|\s]+/).filter(Boolean)
    let baseUrl = ''
    let keys = tokens
    if (tokens.length > 1 && /^https?:\/\//i.test(tokens[0])) {
      baseUrl = normalizeBaseUrl(tokens[0])
      keys = tokens.slice(1)
    }

    for (const key of keys) {
      idx += 1
      // AIza-prefixed keys are Gemini API keys; everything else is treated as
      // an OpenAI-compatible credential.
      const platform: ParsedImportAccount['platform'] = key.startsWith('AIza') ? 'gemini' : 'openai'
      result.accounts.push({
        platform,
        type: 'apikey',
        name: `imported-${idx}`,
        baseUrl,
        apiKey: key,
        sourceStatus: 1,
        sourceGroup: '',
        models: ''
      })
    }
  }
}

/**
 * Parse a pasted payload into account drafts. Accepts either the OneAPI
 * channel JSON (the `{data: [...]}` API envelope or a bare channel array) or
 * a plain text list where each line is `KEY`, `URL KEY`, `URL|KEY` or
 * `URL,KEY`-style (comma separated by an URL in front).
 */
export function parseImportPayload(text: string): ParsedImportResult {
  const result: ParsedImportResult = { format: 'text_keys', accounts: [], skipped: [] }
  const trimmed = (text || '').trim()
  if (!trimmed) return result

  const parsed = tryParseJsonEnvelope(trimmed)
  if (parsed && Array.isArray(parsed)) {
    result.format = 'oneapi_json'
    const usedNames = new Set<string>()
    for (const channel of parsed) {
      parseOneApiChannel((channel ?? {}) as OneApiChannel, result, usedNames)
    }
    return result
  }

  parseTextKeys(trimmed, result)
  return result
}

function tryParseJsonEnvelope(text: string): unknown[] | null {
  let value: unknown
  try {
    value = JSON.parse(text)
  } catch {
    return null
  }
  if (Array.isArray(value)) return value
  if (value && typeof value === 'object' && Array.isArray((value as { data?: unknown }).data)) {
    return (value as { data: unknown[] }).data
  }
  return null
}
