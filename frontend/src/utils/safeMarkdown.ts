import DOMPurify from 'dompurify'

export interface MarkdownTocItem {
  id: string
  text: string
  level: number
}

export interface SanitizedMarkdownContent {
  html: string
  toc: MarkdownTocItem[]
}

export function generateMarkdownHeadingId(text: string, index: number): string {
  const base = text
    .toLowerCase()
    .replace(/[^\w一-鿿]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return base ? `${base}-${index}` : `heading-${index}`
}

/**
 * Sanitize Markdown HTML before inspecting its DOM, then sanitize the serialized
 * result again after adding generated heading IDs. Heading text is read from
 * DOM textContent instead of stripping tags with a partial HTML regex.
 */
export function sanitizeMarkdownAndBuildToc(html: string): SanitizedMarkdownContent {
  const fragment = DOMPurify.sanitize(html, { RETURN_DOM_FRAGMENT: true }) as DocumentFragment
  const container = document.createElement('div')
  container.append(fragment)

  const toc: MarkdownTocItem[] = []
  container.querySelectorAll<HTMLElement>('h1, h2, h3, h4').forEach((heading, index) => {
    const text = heading.textContent?.trim() ?? ''
    const level = Number(heading.tagName.slice(1))
    const id = generateMarkdownHeadingId(text, index)
    heading.id = id
    toc.push({ id, text, level })
  })

  return {
    html: DOMPurify.sanitize(container.innerHTML) as string,
    toc,
  }
}
