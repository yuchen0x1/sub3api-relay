import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { chartPngFilename, chartToPngDataUrl, downloadChartPng } from '../chartExport'

const PNG = 'data:image/png;base64,STUB'

function makeCanvas(width = 200, height = 100): HTMLCanvasElement {
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  return canvas
}

describe('chartExport', () => {
  let fillRect: ReturnType<typeof vi.fn>
  let drawImage: ReturnType<typeof vi.fn>

  beforeEach(() => {
    fillRect = vi.fn()
    drawImage = vi.fn()
    const ctx = { fillStyle: '', fillRect, drawImage }
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(ctx as unknown as CanvasRenderingContext2D)
    vi.spyOn(HTMLCanvasElement.prototype, 'toDataURL').mockReturnValue(PNG)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('builds a sortable timestamped filename', () => {
    expect(chartPngFilename('ops-latency')).toMatch(/^ops-latency-\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2}\.png$/)
  })

  it('returns null when there is no readable canvas', () => {
    expect(chartToPngDataUrl(null)).toBeNull()
    expect(chartToPngDataUrl(undefined)).toBeNull()
    expect(chartToPngDataUrl({})).toBeNull()
    expect(chartToPngDataUrl({ canvas: null })).toBeNull()
  })

  it('returns null for a canvas that has not been drawn to', () => {
    expect(chartToPngDataUrl({ canvas: makeCanvas(0, 0) })).toBeNull()
  })

  it('paints the surface background before drawing the chart', () => {
    const source = makeCanvas()

    expect(chartToPngDataUrl({ canvas: source })).toBe(PNG)
    expect(fillRect).toHaveBeenCalledWith(0, 0, 200, 100)
    expect(drawImage).toHaveBeenCalledWith(source, 0, 0)
    expect(fillRect.mock.invocationCallOrder[0]).toBeLessThan(drawImage.mock.invocationCallOrder[0])
  })

  it('returns null when the canvas cannot be read back', () => {
    vi.spyOn(HTMLCanvasElement.prototype, 'toDataURL').mockImplementation(() => {
      throw new Error('tainted')
    })

    expect(chartToPngDataUrl({ canvas: makeCanvas() })).toBeNull()
  })

  it('downloads the composed image under the requested filename', () => {
    let captured: HTMLAnchorElement | null = null
    const createElement = document.createElement.bind(document)
    vi.spyOn(document, 'createElement').mockImplementation(((tagName: string, options?: ElementCreationOptions) => {
      const element = createElement(tagName, options)
      if (tagName.toLowerCase() === 'a') captured = element as HTMLAnchorElement
      return element
    }) as typeof document.createElement)
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)

    expect(downloadChartPng({ canvas: makeCanvas() }, 'ops-latency-x.png')).toBe(true)
    expect(captured).not.toBeNull()
    expect(captured!.download).toBe('ops-latency-x.png')
    expect(captured!.href).toBe(PNG)
  })

  it('does not attempt a download when export is impossible', () => {
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click')

    expect(downloadChartPng(null, 'x.png')).toBe(false)
    expect(click).not.toHaveBeenCalled()
  })
})
