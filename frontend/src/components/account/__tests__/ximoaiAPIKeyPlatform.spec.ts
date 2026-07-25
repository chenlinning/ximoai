import { describe, expect, it } from 'vitest'
import type { Account, Platform } from '@/types'
import {
  resolveXimoAIPlatformKind,
  requiresXimoAIAPIKeyBaseURL
} from '../ximoaiAPIKeyPlatform'

function platform(input: Partial<Platform>): Platform {
  return {
    slug: 'custom-platform',
    display_name: 'Custom platform',
    protocol: 'openai_compatible',
    base_url: '',
    auth_modes: ['apikey'],
    capabilities: [],
    color: '#111827',
    enabled: true,
    builtin: true,
    created_at: '',
    updated_at: '',
    ...input
  }
}

describe('XimoAI built-in platform runtime kind', () => {
  it('uses the immutable platform kind after a platform rename', () => {
    const renamed = platform({ slug: 'video-provider', kind: 'grok_video' })

    expect(resolveXimoAIPlatformKind(renamed)).toBe('grok_video')
  })

  it('uses the persisted account kind when platform metadata is unavailable', () => {
    const account = {
      platform: 'renamed-kling',
      credentials: { platform_kind: 'kling_audio' }
    } as Account

    expect(resolveXimoAIPlatformKind(undefined, account)).toBe('kling_audio')
  })

  it('keeps compatibility with accounts created before runtime kinds existed', () => {
    const account = { platform: 'openai-audio', credentials: {} } as Account

    expect(resolveXimoAIPlatformKind(undefined, account)).toBe('openai_audio')
  })

  it('requires an explicit base URL only for providers without a safe default', () => {
    expect(requiresXimoAIAPIKeyBaseURL(platform({ kind: 'grok_video' }))).toBe(true)
    expect(requiresXimoAIAPIKeyBaseURL(platform({ kind: 'kling_audio' }))).toBe(true)
    expect(requiresXimoAIAPIKeyBaseURL(platform({ kind: 'openai_audio' }))).toBe(false)
    expect(requiresXimoAIAPIKeyBaseURL(platform({ kind: 'volcengine_agent_plan' }))).toBe(false)
  })

  it('recognizes the Volcengine Agent Plan runtime kind', () => {
    const definition = platform({
      slug: 'volcengine-agent-plan',
      kind: 'volcengine_agent_plan',
      protocol: 'native'
    })

    expect(resolveXimoAIPlatformKind(definition)).toBe('volcengine_agent_plan')
  })
})
