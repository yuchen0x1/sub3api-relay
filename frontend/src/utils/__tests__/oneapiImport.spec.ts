import { describe, expect, it } from 'vitest'
import { parseImportPayload } from '../oneapiImport'

describe('parseImportPayload', () => {
  it('parses the OneAPI channel envelope and maps openai/custom/gemini types', () => {
    const payload = JSON.stringify({
      data: [
        { id: 1, type: 1, name: 'official', key: 'sk-a', status: 1, models: 'gpt-4', group: 'default', base_url: '' },
        { id: 2, type: 8, name: 'relay', key: 'sk-b', base_url: 'https://relay.example.com/v1/', status: 2 },
        { id: 3, type: 24, name: 'gem', key: 'AIza-c', status: 1 }
      ]
    })

    const result = parseImportPayload(payload)
    expect(result.format).toBe('oneapi_json')
    expect(result.skipped).toEqual([])
    expect(result.accounts).toHaveLength(3)
    expect(result.accounts[0]).toMatchObject({ platform: 'openai', baseUrl: '', apiKey: 'sk-a', name: 'official', sourceStatus: 1, sourceGroup: 'default', models: 'gpt-4' })
    expect(result.accounts[1]).toMatchObject({ platform: 'openai', baseUrl: 'https://relay.example.com/v1', apiKey: 'sk-b', sourceStatus: 2 })
    expect(result.accounts[2]).toMatchObject({ platform: 'gemini', apiKey: 'AIza-c' })
  })

  it('accepts a bare channel array as well as the envelope', () => {
    const result = parseImportPayload('[{"type":1,"name":"x","key":"sk-x"}]')
    expect(result.format).toBe('oneapi_json')
    expect(result.accounts).toHaveLength(1)
    expect(result.accounts[0].name).toBe('x')
  })

  it('splits multi-key channels into separate accounts with unique names', () => {
    const result = parseImportPayload('[{"type":1,"name":"pool","key":"sk-1\\nsk-2\\nsk-3"}]')
    expect(result.accounts.map((a) => a.name)).toEqual(['pool', 'pool-2', 'pool-3'])
    expect(result.accounts.map((a) => a.apiKey)).toEqual(['sk-1', 'sk-2', 'sk-3'])
  })

  it('reports unsupported channel types and missing keys as skipped', () => {
    const result = parseImportPayload(
      JSON.stringify([
        { type: 14, name: 'legacy-anthropic', key: 'sk-a' },
        { type: 1, name: 'no-key' },
        { type: 1, name: 'good', key: 'sk-b' }
      ])
    )
    expect(result.accounts).toHaveLength(1)
    expect(result.accounts[0].name).toBe('good')
    expect(result.skipped).toEqual([
      { name: 'legacy-anthropic', reason: 'unsupported channel type: 14' },
      { name: 'no-key', reason: 'missing api key' }
    ])
  })

  it('parses plain text lists with optional base urls and detects gemini keys', () => {
    const text = [
      '# comment line',
      'sk-plain-1',
      'https://relay.example.com/v1 sk-url-1',
      'https://relay.example.com/v1|sk-url-2',
      'AIzaGeminiKey'
    ].join('\n')

    const result = parseImportPayload(text)
    expect(result.format).toBe('text_keys')
    expect(result.skipped).toEqual([])
    expect(result.accounts).toHaveLength(4)
    expect(result.accounts[0]).toMatchObject({ platform: 'openai', name: 'imported-1', baseUrl: '', apiKey: 'sk-plain-1' })
    expect(result.accounts[1]).toMatchObject({ platform: 'openai', baseUrl: 'https://relay.example.com/v1', apiKey: 'sk-url-1' })
    expect(result.accounts[2]).toMatchObject({ platform: 'openai', baseUrl: 'https://relay.example.com/v1', apiKey: 'sk-url-2' })
    expect(result.accounts[3]).toMatchObject({ platform: 'gemini', baseUrl: '', apiKey: 'AIzaGeminiKey' })
  })

  it('returns an empty result for empty input', () => {
    const result = parseImportPayload('   \n  ')
    expect(result.accounts).toEqual([])
    expect(result.skipped).toEqual([])
  })
})
