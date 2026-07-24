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
type XimoAIPlatformKind = 'grok_video' | 'openai_audio' | 'kling_audio'

const legacyXimoAIPlatformKinds: Record<string, XimoAIPlatformKind> = {
  'grok-video': 'grok_video',
  'openai-audio': 'openai_audio',
  kling_audio: 'kling_audio'
}

function normalizedXimoAIPlatformKind(value: unknown): XimoAIPlatformKind | '' {
  const kind = typeof value === 'string' ? value.trim().toLowerCase() : ''
  if (kind === 'grok_video' || kind === 'openai_audio' || kind === 'kling_audio') {
    return kind
  }
  return ''
}

export function resolveXimoAIPlatformKind(
  platform?: Platform | null,
  account?: Account | null
): XimoAIPlatformKind | '' {
  const platformKind = normalizedXimoAIPlatformKind(platform?.kind)
  if (platformKind) return platformKind

  const credentials = account?.credentials as Record<string, unknown> | undefined
  const accountKind = normalizedXimoAIPlatformKind(credentials?.platform_kind)
  if (accountKind) return accountKind

  const slug = (platform?.slug || account?.platform || '').trim().toLowerCase()
  return legacyXimoAIPlatformKinds[slug] || ''
}

export function requiresXimoAIAPIKeyBaseURL(
  platform?: Platform | null,
  account?: Account | null
): boolean {
  const kind = resolveXimoAIPlatformKind(platform, account)
  return kind === 'grok_video' || kind === 'kling_audio'
}

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
