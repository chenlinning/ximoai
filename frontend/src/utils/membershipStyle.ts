import { i18n } from '@/i18n'

export const defaultMembershipColor = '#a15a2b'

export interface MembershipLevelPresentation {
  nameKey: string
  color: string
}

const fixedMembershipLevelPresentations: Record<string, MembershipLevelPresentation> = {
  bronze: {
    nameKey: 'membership.levels.bronze',
    color: '#a15a2b'
  },
  silver: {
    nameKey: 'membership.levels.silver',
    color: '#94a3b8'
  },
  gold: {
    nameKey: 'membership.levels.gold',
    color: '#d99a00'
  },
  platinum: {
    nameKey: 'membership.levels.platinum',
    color: '#b89a56'
  },
  diamond: {
    nameKey: 'membership.levels.diamond',
    color: '#0ea5e9'
  }
}

const membershipColorPattern = /^#[0-9a-fA-F]{6}$/

export function membershipLevelPresentation(code?: string | null): MembershipLevelPresentation {
  const key = code?.trim().toLowerCase() || ''
  return fixedMembershipLevelPresentations[key] || {
    nameKey: 'membership.levels.bronze',
    color: defaultMembershipColor
  }
}

export function membershipLevelDisplayName(code?: string | null, fallback?: string | null): string {
  const key = code?.trim().toLowerCase() || ''
  const presentation = fixedMembershipLevelPresentations[key]
  if (!presentation) {
    return fallback || i18n.global.t('membership.levels.bronze')
  }
  return i18n.global.t(presentation.nameKey)
}

export function membershipLevelColor(code?: string | null, fallback?: string | null): string {
  return normalizeMembershipColor(membershipLevelPresentation(code).color || fallback)
}

export function normalizeMembershipColor(color?: string | null): string {
  const value = color?.trim()
  return value && membershipColorPattern.test(value) ? value.toLowerCase() : defaultMembershipColor
}

export function membershipTextColor(color?: string | null): string {
  const hex = normalizeMembershipColor(color).slice(1)
  const red = Number.parseInt(hex.slice(0, 2), 16)
  const green = Number.parseInt(hex.slice(2, 4), 16)
  const blue = Number.parseInt(hex.slice(4, 6), 16)
  const luminance = (red * 299 + green * 587 + blue * 114) / 1000
  return luminance > 150 ? '#111827' : '#ffffff'
}

export function membershipBadgeStyle(color?: string | null): Record<string, string> {
  const backgroundColor = normalizeMembershipColor(color)
  return {
    backgroundColor,
    borderColor: backgroundColor,
    color: membershipTextColor(backgroundColor)
  }
}

export function membershipPanelStyle(color?: string | null): Record<string, string> {
  const base = normalizeMembershipColor(color)
  return {
    backgroundColor: `${base}1a`,
    borderColor: base
  }
}

export function membershipAvatarStyle(color?: string | null): Record<string, string> {
  const base = normalizeMembershipColor(color)
  return {
    boxShadow: `0 0 0 2px ${base}`
  }
}
