import { describe, expect, it } from 'vitest'
import { sanitizeMarkdownAndBuildToc } from '@/utils/safeMarkdown'

describe('sanitizeMarkdownAndBuildToc', () => {
  it('keeps inline heading markup and derives plain text and unique IDs from the DOM', () => {
    const result = sanitizeMarkdownAndBuildToc('<h1>Guide <em>&amp; examples</em></h1><h2>重复标题</h2><h2>重复标题</h2>')
    const container = document.createElement('div')
    container.innerHTML = result.html
    expect(result.toc).toEqual([
      { id: 'guide-examples-0', text: 'Guide & examples', level: 1 },
      { id: '重复标题-1', text: '重复标题', level: 2 },
      { id: '重复标题-2', text: '重复标题', level: 2 },
    ])
    expect(container.querySelector('h1 em')?.textContent).toBe('& examples')
    expect([...container.querySelectorAll('h1, h2')].map(heading => heading.id)).toEqual(result.toc.map(item => item.id))
  })

  it('removes active content and unsafe attributes before and after adding IDs', () => {
    const result = sanitizeMarkdownAndBuildToc('<h2 onclick="alert(1)">Title<img src="x" onerror="alert(2)"></h2><script>alert(3)</script><iframe srcdoc="evil"></iframe><a href="javascript:alert(4)">link</a>')
    const container = document.createElement('div')
    container.innerHTML = result.html
    expect(container.querySelector('script, iframe, [onclick], [onerror]')).toBeNull()
    expect(container.querySelector('a')?.getAttribute('href')).toBeNull()
    expect(result.toc).toEqual([{ id: 'title-0', text: 'Title', level: 2 }])
  })

  it('handles multiline headings without reconstructing HTML from partial tag matches', () => {
    const result = sanitizeMarkdownAndBuildToc('<h3>Line one\n<strong>Line two</strong></h3><h4></h4>')
    expect(result.toc).toEqual([
      { id: 'line-one-line-two-0', text: 'Line one\nLine two', level: 3 },
      { id: 'heading-1', text: '', level: 4 },
    ])
  })
})
