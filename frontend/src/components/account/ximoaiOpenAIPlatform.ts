import type { Account, Platform } from '@/types'

const reservedPlatformSlugs = new Set([
  'anthropic',
  'openai',
  'gemini',
  'antigravity',
  'grok',
  'grok-video',
  'openai-audio',
  'kling_audio'
])

export function isCustomOpenAICompatiblePlatform(platform?: Platform | null): boolean {
  return Boolean(
    platform &&
    !platform.builtin &&
    platform.protocol === 'openai_compatible' &&
    platform.auth_modes.length > 0 &&
    platform.auth_modes.every((mode) => mode === 'apikey')
  )
}

export function isCustomOpenAICompatibleAccount(
  account?: Account | null,
  platform?: Platform | null
): boolean {
  if (!account || account.type !== 'apikey') return false
  if (platform) return isCustomOpenAICompatiblePlatform(platform)

  const slug = account.platform.trim().toLowerCase()
  if (!slug || reservedPlatformSlugs.has(slug)) return false
  const credentials = account.credentials as Record<string, unknown> | undefined
  return credentials?.platform_protocol === 'openai_compatible'
}

export function isCustomOpenAICompatibleDescriptor(
  slug: string,
  protocol?: string,
  builtin?: boolean
): boolean {
  const normalizedSlug = slug.trim().toLowerCase()
  return Boolean(
    normalizedSlug &&
    builtin !== true &&
    !reservedPlatformSlugs.has(normalizedSlug) &&
    protocol === 'openai_compatible'
  )
}
