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
  it('uses the fixed platform slug', () => {
    const video = platform({ slug: 'grok-video', kind: 'grok_video' })

    expect(resolveXimoAIPlatformKind(video)).toBe('grok_video')
  })

  it('does not accept a persisted kind for an unknown platform', () => {
    const account = {
      platform: 'renamed-kling',
      credentials: { platform_kind: 'kling_audio' }
    } as Account

    expect(resolveXimoAIPlatformKind(undefined, account)).toBe('')
  })

  it('keeps compatibility with accounts created before runtime kinds existed', () => {
    const account = { platform: 'openai-audio', credentials: {} } as Account

    expect(resolveXimoAIPlatformKind(undefined, account)).toBe('openai_audio')
  })

  it('requires an explicit base URL only for providers without a safe default', () => {
    expect(requiresXimoAIAPIKeyBaseURL(platform({ slug: 'grok-video' }))).toBe(true)
    expect(requiresXimoAIAPIKeyBaseURL(platform({ slug: 'kling_audio' }))).toBe(true)
    expect(requiresXimoAIAPIKeyBaseURL(platform({ slug: 'openai-audio' }))).toBe(false)
    expect(requiresXimoAIAPIKeyBaseURL(platform({ slug: 'volcengine-agent-plan' }))).toBe(false)
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
