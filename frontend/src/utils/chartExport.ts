/**
 * Chart.js PNG export.
 *
 * Chart canvases are transparent, so `chart.toBase64Image()` alone yields an
 * image that reads as broken in most viewers. We repaint the chart onto an
 * offscreen canvas over the theme surface color first.
 */

export interface ExportableChart {
  canvas?: HTMLCanvasElement | null
}

function resolveSurfaceColor(): string {
  if (typeof document === 'undefined') return '#ffffff'
  const value = getComputedStyle(document.documentElement).getPropertyValue('--color-surface').trim()
  return value || '#ffffff'
}

export function chartToPngDataUrl(chart: ExportableChart | null | undefined): string | null {
  const source = chart?.canvas
  if (!source || !source.width || !source.height) return null

  const output = document.createElement('canvas')
  output.width = source.width
  output.height = source.height

  const ctx = output.getContext('2d')
  if (!ctx) return null

  ctx.fillStyle = resolveSurfaceColor()
  ctx.fillRect(0, 0, output.width, output.height)
  ctx.drawImage(source, 0, 0)

  try {
    return output.toDataURL('image/png')
  } catch {
    // Cross-origin pixels taint the canvas and make it unreadable.
    return null
  }
}

export function chartPngFilename(prefix: string): string {
  const stamp = new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')
  return `${prefix}-${stamp}.png`
}

export function downloadChartPng(chart: ExportableChart | null | undefined, filename: string): boolean {
  const url = chartToPngDataUrl(chart)
  if (!url) return false

  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  return true
}
