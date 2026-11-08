/**
 * 强调色（Accent Color）预设。
 *
 * Tailwind 工具类（bg-primary-500 等）编译为 var(--color-primary-500)，
 * 因此运行时覆写 <html> 上的同名变量即可整体换肤，无需重新构建。
 */

export type AccentId = 'cyan' | 'blue' | 'violet' | 'emerald' | 'amber' | 'rose'

export type AccentShade = 50 | 100 | 200 | 300 | 400 | 500 | 600 | 700 | 800 | 900 | 950

export interface AccentPreset {
  id: AccentId
  /** i18n key of the human readable label */
  labelKey: string
  /** swatch color shown in the picker */
  swatch: string
  /** hex ramp, matching the Tailwind palette */
  ramp: Record<AccentShade, string>
}

export const accentPresets: AccentPreset[] = [
  {
    id: 'cyan',
    labelKey: 'nav.accentCyan',
    swatch: '#22d3ee',
    ramp: {
      50: '#ecfeff', 100: '#cffafe', 200: '#a5f3fc', 300: '#67e8f9', 400: '#22d3ee',
      500: '#06b6d4', 600: '#0891b2', 700: '#0e7490', 800: '#155e75', 900: '#164e63', 950: '#083344'
    }
  },
  {
    id: 'blue',
    labelKey: 'nav.accentBlue',
    swatch: '#60a5fa',
    ramp: {
      50: '#eff6ff', 100: '#dbeafe', 200: '#bfdbfe', 300: '#93c5fd', 400: '#60a5fa',
      500: '#3b82f6', 600: '#2563eb', 700: '#1d4ed8', 800: '#1e40af', 900: '#1e3a8a', 950: '#172554'
    }
  },
  {
    id: 'violet',
    labelKey: 'nav.accentViolet',
    swatch: '#a78bfa',
    ramp: {
      50: '#f5f3ff', 100: '#ede9fe', 200: '#ddd6fe', 300: '#c4b5fd', 400: '#a78bfa',
      500: '#8b5cf6', 600: '#7c3aed', 700: '#6d28d9', 800: '#5b21b6', 900: '#4c1d95', 950: '#2e1065'
    }
  },
  {
    id: 'emerald',
    labelKey: 'nav.accentEmerald',
    swatch: '#34d399',
    ramp: {
      50: '#ecfdf5', 100: '#d1fae5', 200: '#a7f3d0', 300: '#6ee7b7', 400: '#34d399',
      500: '#10b981', 600: '#059669', 700: '#047857', 800: '#065f46', 900: '#064e3b', 950: '#022c22'
    }
  },
  {
    id: 'amber',
    labelKey: 'nav.accentAmber',
    swatch: '#fbbf24',
    ramp: {
      50: '#fffbeb', 100: '#fef3c7', 200: '#fde68a', 300: '#fcd34d', 400: '#fbbf24',
      500: '#f59e0b', 600: '#d97706', 700: '#b45309', 800: '#92400e', 900: '#78350f', 950: '#451a03'
    }
  },
  {
    id: 'rose',
    labelKey: 'nav.accentRose',
    swatch: '#fb7185',
    ramp: {
      50: '#fff1f2', 100: '#ffe4e6', 200: '#fecdd3', 300: '#fda4af', 400: '#fb7185',
      500: '#f43f5e', 600: '#e11d48', 700: '#be123c', 800: '#9f1239', 900: '#881337', 950: '#4c0519'
    }
  }
]

export const defaultAccentId: AccentId = 'cyan'

export function getAccentPreset(id: string): AccentPreset {
  return accentPresets.find((preset) => preset.id === id) ?? accentPresets[0]
}

export function isAccentId(value: unknown): value is AccentId {
  return typeof value === 'string' && accentPresets.some((preset) => preset.id === value)
}

/** Convert a #rrggbb color into the "R G B" triplet used by the token layer. */
function hexToRgbTriplet(hex: string): string {
  const normalized = hex.replace('#', '')
  const r = parseInt(normalized.slice(0, 2), 16)
  const g = parseInt(normalized.slice(2, 4), 16)
  const b = parseInt(normalized.slice(4, 6), 16)
  return `${r} ${g} ${b}`
}

/**
 * Apply an accent preset to the document root.
 *
 * --color-primary-* drives the compiled Tailwind utilities, while --brand-*
 * feeds the semantic token layer (light/dark aware).
 */
export function applyAccent(id: AccentId): void {
  if (typeof document === 'undefined') return
  const preset = getAccentPreset(id)
  const root = document.documentElement.style

  for (const [shade, hex] of Object.entries(preset.ramp)) {
    root.setProperty(`--color-primary-${shade}`, hex)
    root.setProperty(`--brand-${shade}`, hexToRgbTriplet(hex))
  }
  root.setProperty('--accent-swatch', preset.swatch)
  document.documentElement.dataset.accent = preset.id
}

/** Remove the inline overrides so the stylesheet defaults take over again. */
export function resetAccent(): void {
  if (typeof document === 'undefined') return
  const root = document.documentElement.style
  for (const shade of [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950]) {
    root.removeProperty(`--color-primary-${shade}`)
    root.removeProperty(`--brand-${shade}`)
  }
  root.removeProperty('--accent-swatch')
  delete document.documentElement.dataset.accent
}
