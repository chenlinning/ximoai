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

type CustomAPIKeyProtocol = 'openai_compatible' | 'anthropic' | 'gemini'

function isCustomAPIKeyProtocolPlatform(
  platform: Platform | null | undefined,
  protocol: CustomAPIKeyProtocol
): boolean {
  return Boolean(
    platform &&
    !platform.builtin &&
    platform.protocol === protocol &&
    platform.auth_modes.length > 0 &&
    platform.auth_modes.every((mode) => mode === 'apikey')
  )
}

function isCustomAPIKeyProtocolAccount(
  account: Account | null | undefined,
  platform: Platform | null | undefined,
  protocol: CustomAPIKeyProtocol
): boolean {
  if (!account || account.type !== 'apikey') return false
  if (platform) return isCustomAPIKeyProtocolPlatform(platform, protocol)

  const slug = account.platform.trim().toLowerCase()
  if (!slug || reservedPlatformSlugs.has(slug)) return false
  const credentials = account.credentials as Record<string, unknown> | undefined
  return credentials?.platform_protocol === protocol
}

export function isCustomOpenAICompatiblePlatform(platform?: Platform | null): boolean {
  return isCustomAPIKeyProtocolPlatform(platform, 'openai_compatible')
}

export function isCustomOpenAICompatibleAccount(
  account?: Account | null,
  platform?: Platform | null
): boolean {
  return isCustomAPIKeyProtocolAccount(account, platform, 'openai_compatible')
}

export function isCustomAnthropicPlatform(platform?: Platform | null): boolean {
  return isCustomAPIKeyProtocolPlatform(platform, 'anthropic')
}

export function isCustomAnthropicAccount(
  account?: Account | null,
  platform?: Platform | null
): boolean {
  return isCustomAPIKeyProtocolAccount(account, platform, 'anthropic')
}

export function isCustomGeminiPlatform(platform?: Platform | null): boolean {
  return isCustomAPIKeyProtocolPlatform(platform, 'gemini')
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

export function isCustomAnthropicDescriptor(
  slug: string,
  protocol?: string,
  builtin?: boolean
): boolean {
  const normalizedSlug = slug.trim().toLowerCase()
  return Boolean(
    normalizedSlug &&
    builtin !== true &&
    !reservedPlatformSlugs.has(normalizedSlug) &&
    protocol === 'anthropic'
  )
}
