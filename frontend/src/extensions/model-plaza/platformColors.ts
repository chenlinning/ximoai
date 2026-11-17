import type { CSSProperties } from 'vue'
import { platformLabel } from '@/utils/platformColors'

export interface PlatformInfo {
  slug: string
  display_name?: string
  color?: string
}
const UNKNOWN_PLATFORM_COLORS = ['#0F766E', '#2563EB', '#7C3AED', '#DB2777', '#EA580C', '#059669', '#4F46E5', '#0891B2']

export function normalizePlatformColor(color?: string | null): string {
  const value = String(color || '').trim()
  if (/^#[0-9a-f]{6}$/i.test(value) || /^#[0-9a-f]{3}$/i.test(value)) {
    return value
  }
  return ''
}

export function platformColorStyle(color?: string | null): CSSProperties {
  const normalized = normalizePlatformColor(color)
  return normalized ? { color: normalized } : {}
}

export function platformBadgeStyle(color?: string | null): CSSProperties {
  const normalized = normalizePlatformColor(color)
  if (!normalized) return {}
  return {
    color: normalized,
    borderColor: `${normalized}55`,
    backgroundColor: `${normalized}18`,
  }
}

export function platformButtonStyle(color?: string | null): CSSProperties {
  const normalized = normalizePlatformColor(color)
  if (!normalized) return {}
  return {
    color: normalized,
    borderColor: `${normalized}55`,
    backgroundColor: `${normalized}14`,
  }
}

export function platformDisplayName(platforms: PlatformInfo[] | undefined, slug: string): string {
  const normalized = String(slug || '').trim()
  if (!normalized) return 'API'
  const found = (platforms || []).find(p => p.slug === normalized)
  return found?.display_name || platformLabel(normalized)
}

export function platformDisplayColor(platforms: PlatformInfo[] | undefined, slug: string): string {
  const normalized = String(slug || '').trim()
  const found = (platforms || []).find(p => p.slug === normalized)
  return normalizePlatformColor(found?.color) || unknownPlatformFallbackColor(normalized)
}

export function unknownPlatformFallbackColor(slug: string): string {
  const normalized = String(slug || '').trim()
  if (!normalized) return '#64748B'
  let hash = 0
  for (const char of normalized) {
    hash = Math.imul(hash, 31) + char.charCodeAt(0)
  }
  return UNKNOWN_PLATFORM_COLORS[Math.abs(hash) % UNKNOWN_PLATFORM_COLORS.length]
}
