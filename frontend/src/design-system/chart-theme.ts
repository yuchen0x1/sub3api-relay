/**
 * Chart Palette — Phase 1 Design Token Integration
 *
 * Single source of truth for all chart color sequences.
 * Replaces inline hex/rgb values across 10+ Chart.js component files.
 *
 * Usage:
 *   import { chartColors, chartColorsAlpha, chartPalette } from '@/design-system/chart-theme'
 *
 *   new Chart(ctx, {
 *     datasets: [{ data: [...], backgroundColor: chartColors(3) }]
 *   })
 *
 * The values below mirror the CSS variables defined in tokens.css
 * but as static arrays for programmatic use (Chart.js needs actual values).
 */

/** RGB triples for the 10-color chart sequence (light mode) */
const LIGHT_RGB: [number, number, number][] = [
  [8, 145, 178],    // 1  cyan 600
  [124, 58, 237],   // 2  violet 600
  [6, 182, 212],    // 3  cyan 500
  [132, 204, 22],   // 4  lime 500
  [245, 158, 11],   // 5  amber 500
  [236, 72, 153],   // 6  pink 500
  [16, 185, 129],   // 7  emerald 500
  [59, 130, 246],   // 8  blue 500
  [168, 85, 247],   // 9  purple 500
  [244, 63, 94],    // 10 rose 500
]

/** RGB triples for dark mode (brighter variant for dark backgrounds) */
const DARK_RGB: [number, number, number][] = [
  [34, 211, 238],   // 1  cyan 400
  [167, 139, 250],  // 2  violet 400
  [94, 234, 212],   // 3  teal 300
  [163, 230, 53],   // 4  lime 400
  [251, 191, 36],   // 5  amber 400
  [244, 114, 182],  // 6  pink 400
  [52, 211, 153],   // 7  emerald 400
  [96, 165, 250],   // 8  blue 400
  [192, 132, 252],  // 9  purple 400
  [251, 113, 133],  // 10 rose 400
]

/**
 * Detect current theme (light/dark) by checking <html> class.
 * Defaults to light if no class set.
 */
function isDark(): boolean {
  if (typeof document === 'undefined') return false
  return document.documentElement.classList.contains('dark')
}

/**
 * Get the active RGB palette based on current theme.
 */
function activePalette(): [number, number, number][] {
  return isDark() ? DARK_RGB : LIGHT_RGB
}

/**
 * Convert RGB triple to CSS rgba() string with optional alpha.
 */
function rgba(rgb: [number, number, number], alpha = 1): string {
  return `rgba(${rgb[0]},${rgb[1]},${rgb[2]},${alpha})`
}

/**
 * Convert RGB triple to hex string.
 */
function hex(rgb: [number, number, number]): string {
  const toHex = (n: number) => n.toString(16).padStart(2, '0')
  return `#${toHex(rgb[0])}${toHex(rgb[1])}${toHex(rgb[2])}`
}

// ============================================================
// Public API
// ============================================================

/**
 * Return N solid hex colors from the current theme palette.
 * Cycles if N > palette length.
 */
export function chartColors(count: number): string[] {
  const palette = activePalette()
  return Array.from({ length: count }, (_, i) => hex(palette[i % palette.length]))
}

/**
 * Return N rgba colors with the given alpha from the current theme palette.
 * Cycles if N > palette length.
 */
export function chartColorsAlpha(count: number, alpha = 0.5): string[] {
  const palette = activePalette()
  return Array.from({ length: count }, (_, i) => rgba(palette[i % palette.length], alpha))
}

/**
 * Return a single hex color by index (1-based for readability).
 */
export function chartColor(index: number): string {
  const palette = activePalette()
  return hex(palette[(index - 1) % palette.length])
}

/**
 * Return a single rgba color by index with custom alpha.
 */
export function chartColorAlpha(index: number, alpha = 0.5): string {
  const palette = activePalette()
  return rgba(palette[(index - 1) % palette.length], alpha)
}

/**
 * Static color map for semantic chart purposes.
 * Names are stable across theme — hex values shift with dark mode.
 */
export const chartPalette = {
  primary:   () => chartColor(1),
  secondary: () => chartColor(2),
  accent:    () => chartColor(3),
  highlight: () => chartColor(4),
  warning:   () => chartColor(5),
  success:   () => chartColor(7),
  info:      () => chartColor(8),
  danger:    () => chartColor(10),
} as const

/**
 * Convenience: full palette as hex array (light mode only).
 * For dark‑mode‑aware usage, call chartColors() at render time.
 */
export const CHART_COLORS = LIGHT_RGB.map(hex)
export const CHART_COLORS_ALPHA = LIGHT_RGB.map(rgb => rgba(rgb, 0.5))
export const DARK_MODE_CHART_COLORS = DARK_RGB.map(hex)

// ============================================================
// Theme change listener helper
// ============================================================

type ThemeCallback = (isDarkNow: boolean) => void
const listeners = new Set<ThemeCallback>()
let observer: MutationObserver | null = null

/**
 * Subscribe to theme changes. Returns an unsubscribe function.
 * Useful for reactive chart re-render.
 */
export function onThemeChange(cb: ThemeCallback): () => void {
  listeners.add(cb)
  if (!observer) {
    observer = new MutationObserver(() => {
      const dark = document.documentElement.classList.contains('dark')
      for (const fn of listeners) fn(dark)
    })
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  }
  return () => {
    listeners.delete(cb)
    if (listeners.size === 0 && observer) {
      observer.disconnect()
      observer = null
    }
  }
}