export type ThemeColorToken = string

function getRootStyle(): CSSStyleDeclaration | null {
  if (typeof window === 'undefined' || typeof document === 'undefined') return null
  return window.getComputedStyle(document.documentElement)
}

function readThemeColor(token: ThemeColorToken, seen = new Set<ThemeColorToken>()): string {
  const value = getRootStyle()?.getPropertyValue(`--color-${token}`).trim()
  const alias = value?.match(/^var\(--color-([^)]+)\)$/)?.[1]
  if (alias && !seen.has(alias)) {
    seen.add(token)
    return readThemeColor(alias, seen)
  }
  return value || '0 0 0'
}

export function themeColor(token: ThemeColorToken, alpha?: number): string {
  const rgb = readThemeColor(token)
  return alpha === undefined ? `rgb(${rgb})` : `rgb(${rgb} / ${alpha})`
}

export function themeColorVar(token: ThemeColorToken, alpha?: number): string {
  return alpha === undefined
    ? `rgb(var(--color-${token}))`
    : `rgb(var(--color-${token}) / ${alpha})`
}

export function themeChartPalette(tokens: ThemeColorToken[]): string[] {
  return tokens.map((token) => themeColor(token))
}

export const defaultChartColorTokens = [
  'primary-500',
  'accent-500',
  'emerald-500',
  'amber-500',
  'orange-500',
  'red-500',
  'teal-500',
  'indigo-500',
  'cyan-500',
  'violet-500',
  'lime-500',
  'pink-500',
  'purple-500'
]

export function defaultChartColors(): string[] {
  return themeChartPalette(defaultChartColorTokens)
}
