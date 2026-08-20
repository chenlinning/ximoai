import type { Platform } from '@/types'

export const fixedPlatforms: Platform[] = [
  { slug: 'anthropic', display_name: 'Anthropic', protocol: 'native', base_url: '', auth_modes: ['oauth', 'setup-token', 'apikey', 'bedrock'], capabilities: ['messages'], color: '#D97706', enabled: true, builtin: true, created_at: '', updated_at: '' },
  { slug: 'openai', display_name: 'OpenAI', protocol: 'openai', base_url: 'https://api.openai.com', auth_modes: ['oauth', 'apikey'], capabilities: ['responses', 'chat_completions', 'embeddings', 'images', 'audio', 'realtime', 'codex'], color: '#10A37F', enabled: true, builtin: true, created_at: '', updated_at: '' },
  { slug: 'gemini', display_name: 'Gemini', protocol: 'native', base_url: '', auth_modes: ['oauth', 'apikey', 'service_account'], capabilities: ['messages', 'native_gemini', 'videos'], color: '#4285F4', enabled: true, builtin: true, created_at: '', updated_at: '' },
  { slug: 'antigravity', display_name: 'Antigravity', protocol: 'native', base_url: '', auth_modes: ['oauth', 'upstream', 'apikey'], capabilities: ['messages', 'native_gemini'], color: '#7C3AED', enabled: true, builtin: true, created_at: '', updated_at: '' },
  { slug: 'grok', display_name: 'Grok', protocol: 'openai_compatible', base_url: 'https://api.x.ai/v1', auth_modes: ['oauth', 'apikey'], capabilities: ['responses', 'chat_completions', 'images', 'videos'], color: '#111827', enabled: true, builtin: true, created_at: '', updated_at: '' },
  { slug: 'kimi', display_name: 'Kimi', protocol: 'openai_compatible', base_url: '', auth_modes: ['apikey'], capabilities: ['chat_completions', 'messages'], color: '#EC4899', enabled: true, builtin: true, created_at: '', updated_at: '' },
  { slug: 'zhipu', display_name: 'Zhipu GLM', protocol: 'openai_compatible', base_url: '', auth_modes: ['apikey'], capabilities: ['chat_completions', 'messages'], color: '#6366F1', enabled: true, builtin: true, created_at: '', updated_at: '' },
  { slug: 'deepseek', display_name: 'DeepSeek', protocol: 'openai_compatible', base_url: '', auth_modes: ['apikey'], capabilities: ['responses', 'chat_completions', 'messages'], color: '#14B8A6', enabled: true, builtin: true, created_at: '', updated_at: '' },
  { slug: 'grok-video', kind: 'grok_video', display_name: 'Grok Video', protocol: 'openai_compatible', base_url: '', auth_modes: ['apikey'], capabilities: ['videos'], color: '#111827', enabled: true, builtin: true, created_at: '', updated_at: '' },
  { slug: 'openai-audio', kind: 'openai_audio', display_name: 'OpenAI Audio', protocol: 'openai_compatible', base_url: '', auth_modes: ['apikey'], capabilities: ['chat_completions', 'audio'], color: '#0F766E', enabled: true, builtin: true, created_at: '', updated_at: '' },
  { slug: 'kling_audio', kind: 'kling_audio', display_name: 'Kling Audio', protocol: 'openai_compatible', base_url: '', auth_modes: ['apikey'], capabilities: ['audio'], color: '#0EA5E9', enabled: true, builtin: true, created_at: '', updated_at: '' },
  { slug: 'volcengine-agent-plan', kind: 'volcengine_agent_plan', display_name: 'Volcengine Agent Plan', protocol: 'native', base_url: 'https://ark.cn-beijing.volces.com/api/plan/v3', auth_modes: ['apikey'], capabilities: ['images', 'audio'], color: '#E5484D', enabled: true, builtin: true, created_at: '', updated_at: '' },
]

export const fixedPlatformBySlug = (slug?: string | null) =>
  fixedPlatforms.find((platform) => platform.slug === slug)
