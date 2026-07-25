import { describe, expect, it } from 'vitest'
import {
  capabilityOptionsForProtocol,
  defaultCapabilitiesForProtocol,
  normalizePlatformCapabilities
} from '../capabilities'

describe('platform capability editor options', () => {
  it('offers every documented OpenAI-compatible endpoint family', () => {
    expect(capabilityOptionsForProtocol('openai_compatible')).toEqual([
      'responses',
      'chat_completions',
      'embeddings',
      'images',
      'realtime'
    ])
  })

  it('preserves the fixed capabilities assigned to customized built-ins', () => {
    expect(normalizePlatformCapabilities('gemini', ['videos'], 'grok_video')).toEqual(['videos'])
    expect(normalizePlatformCapabilities('gemini', ['audio', 'chat_completions'], 'openai_audio')).toEqual([
      'chat_completions',
      'audio'
    ])
    expect(capabilityOptionsForProtocol('openai_compatible', 'kling_audio')).toEqual(['audio'])
  })

  it('uses conservative defaults for newly created custom platforms', () => {
    expect(defaultCapabilitiesForProtocol('openai_compatible')).toEqual([
      'responses',
      'chat_completions',
      'images'
    ])
    expect(defaultCapabilitiesForProtocol('gemini')).toEqual(['messages', 'native_gemini', 'videos'])
  })
})
