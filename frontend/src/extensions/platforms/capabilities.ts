const protocolCapabilities: Record<string, string[]> = {
  openai_compatible: [
    'responses',
    'chat_completions',
    'embeddings',
    'images',
    'realtime'
  ],
  anthropic: ['messages'],
  gemini: ['messages', 'native_gemini', 'videos']
}

const protocolDefaults: Record<string, string[]> = {
  openai_compatible: ['responses', 'chat_completions', 'images'],
  anthropic: ['messages'],
  gemini: ['messages', 'native_gemini', 'videos']
}

const platformKindCapabilities: Record<string, string[]> = {
  grok_video: ['videos'],
  openai_audio: ['chat_completions', 'audio'],
  kling_audio: ['audio'],
  volcengine_agent_plan: ['images', 'audio']
}

export function capabilityOptionsForProtocol(protocol: string, kind = ''): string[] {
  if (platformKindCapabilities[kind]) return [...platformKindCapabilities[kind]]
  return [...(protocolCapabilities[protocol] || [])]
}

export function defaultCapabilitiesForProtocol(protocol: string, kind = ''): string[] {
  if (platformKindCapabilities[kind]) return [...platformKindCapabilities[kind]]
  return [...(protocolDefaults[protocol] || [])]
}

export function normalizePlatformCapabilities(protocol: string, capabilities: string[], kind = ''): string[] {
  const selected = new Set(capabilities)
  return capabilityOptionsForProtocol(protocol, kind).filter((capability) => selected.has(capability))
}
