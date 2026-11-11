import { afterEach, describe, expect, it, vi } from 'vitest'
import { buildEmbeddedUrl, detectTheme } from '../embedded-url'

// buildEmbeddedUrl 是纯函数（baseUrl 由参数传入），detectTheme 只读 document，
// 因此本文件不需要伪造 window.location。
// 此前用 Object.defineProperty 替换 location，在 --pool=vmThreads 下会被 jsdom
// 以 "Cannot redefine property: location" 拒绝，造成与本测试意图无关的失败。
describe('embedded-url', () => {
  afterEach(() => {
    document.documentElement.classList.remove('dark')
    vi.restoreAllMocks()
  })

  it('adds non-sensitive embedded query parameters', () => {
    const result = buildEmbeddedUrl(
      'https://pay.example.com/checkout?plan=pro',
      'dark',
      'zh-CN',
    )

    const url = new URL(result)
    expect(url.searchParams.get('plan')).toBe('pro')
    expect(url.searchParams.get('theme')).toBe('dark')
    expect(url.searchParams.get('lang')).toBe('zh-CN')
    expect(url.searchParams.get('ui_mode')).toBe('embedded')
    expect(url.searchParams.has('user_id')).toBe(false)
    expect(url.searchParams.has('token')).toBe(false)
    expect(url.searchParams.has('src_host')).toBe(false)
    expect(url.searchParams.has('src_url')).toBe(false)
  })

  it('omits optional params when they are empty', () => {
    const result = buildEmbeddedUrl('https://pay.example.com/checkout', undefined, '', 'light')

    const url = new URL(result)
    expect(url.searchParams.get('theme')).toBe('light')
    expect(url.searchParams.get('ui_mode')).toBe('embedded')
    expect(url.searchParams.has('lang')).toBe(false)
  })

  it('rejects non-HTTPS embedded pages', () => {
    expect(buildEmbeddedUrl('http://pay.example.com/checkout')).toBe('')
  })

  it('returns original string for invalid url input', () => {
    expect(buildEmbeddedUrl('not a url')).toBe('')
  })

  it('detects dark mode from document root class', () => {
    document.documentElement.classList.add('dark')
    expect(detectTheme()).toBe('dark')
  })
})