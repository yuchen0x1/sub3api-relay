import { describe, expect, it } from 'vitest'
import zh from '@/i18n/locales/zh'
import ja from '@/i18n/locales/ja'

function paths(obj: unknown, prefix = ''): string[] {
  const acc: string[] = []
  if (!obj || typeof obj !== 'object') return acc
  for (const [k, v] of Object.entries(obj as Record<string, unknown>)) {
    const p = prefix ? `${prefix}.${k}` : k
    if (v && typeof v === 'object' && !Array.isArray(v)) acc.push(...paths(v, p))
    else acc.push(p)
  }
  return acc
}

describe('ja locale parity with zh', () => {
  it('has identical key sets', () => {
    const zhKeys = paths(zh)
    const jaKeySet = new Set(paths(ja))
    const missing = zhKeys.filter((k) => !jaKeySet.has(k))
    const extra = [...jaKeySet].filter((k) => !zhKeys.includes(k))
    console.log(`zh=${zhKeys.length} ja=${jaKeySet.size} missing=${missing.length} extra=${extra.length}`)
    if (missing.length) console.log('MISSING:', missing.slice(0, 40))
    if (extra.length) console.log('EXTRA:', extra.slice(0, 40))
    expect(missing).toEqual([])
    expect(extra).toEqual([])
  })
})
