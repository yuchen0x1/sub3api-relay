import { describe, expect, it } from 'vitest'
import { resolveLocale } from '../index'

describe('locale resolution', () => {
  it('prefers saved locale, then browser locale list, then English', () => {
    expect(resolveLocale('ja', ['zh-CN'])).toBe('ja')
    expect(resolveLocale('auto', ['fr-FR', 'zh-TW'])).toBe('zh')
    expect(resolveLocale(undefined, ['fr-FR', 'ja-JP'])).toBe('ja')
    expect(resolveLocale(undefined, ['fr-FR'])).toBe('en')
  })
})
