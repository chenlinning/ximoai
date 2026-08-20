import type { Account, Platform } from '@/types'

type XimoAIPlatformKind = 'grok_video' | 'openai_audio' | 'kling_audio' | 'volcengine_agent_plan'

const ximoAIFixedPlatformKinds: Record<string, XimoAIPlatformKind> = {
  'grok-video': 'grok_video',
  'openai-audio': 'openai_audio',
  kling_audio: 'kling_audio',
  'volcengine-agent-plan': 'volcengine_agent_plan'
}

export function resolveXimoAIPlatformKind(
  platform?: Platform | null,
  account?: Account | null
): XimoAIPlatformKind | '' {
  const slug = (platform?.slug || account?.platform || '').trim().toLowerCase()
  return ximoAIFixedPlatformKinds[slug] || ''
}

export function requiresXimoAIAPIKeyBaseURL(
  platform?: Platform | null,
  account?: Account | null
): boolean {
  const kind = resolveXimoAIPlatformKind(platform, account)
  return kind === 'grok_video' || kind === 'kling_audio'
}
