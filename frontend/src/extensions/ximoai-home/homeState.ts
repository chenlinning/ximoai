export type XimoAIHomeMode = 'loading' | 'legacy' | 'workspace'

interface HomeModeInput {
  settingsLoaded: boolean
  tabsLoaded: boolean
  authenticated: boolean
  homeContent: string
  enabledTabCount: number
}

export interface XimoAIPreferencesMessage {
  source: 'ximoai'
  version: 1
  type: 'preferences:update'
  payload: {
    theme: 'light' | 'dark'
    locale: 'en' | 'zh-CN'
  }
}

export function resolveDefaultXimoAIHomeTab<T extends { url: string }>(tabs: readonly T[]): T | undefined {
  return tabs.find((tab) => {
    try {
      return new URL(tab.url).hostname.toLowerCase() === 'agent.ximoai.cn'
    } catch {
      return false
    }
  }) || tabs[0]
}

export function resolveXimoAIHomeMode(input: HomeModeInput): XimoAIHomeMode {
  if (!input.settingsLoaded) return 'loading'
  if (input.homeContent.trim()) return 'legacy'
  if (!input.authenticated) return 'legacy'
  if (!input.tabsLoaded) return 'loading'
  return input.enabledTabCount > 0 ? 'workspace' : 'legacy'
}

export function addLoadedTab(current: ReadonlySet<string>, tabID: string): Set<string> {
  const next = new Set(current)
  next.add(tabID)
  return next
}

export function frameOrigin(url: string): string | null {
  try {
    return new URL(url).origin
  } catch {
    return null
  }
}

export function buildPreferencesMessage(
  theme: 'light' | 'dark',
  locale: string
): XimoAIPreferencesMessage {
  return {
    source: 'ximoai',
    version: 1,
    type: 'preferences:update',
    payload: {
      theme,
      locale: locale.startsWith('zh') ? 'zh-CN' : 'en'
    }
  }
}

interface MessageEventLike {
  data: unknown
  origin: string
  source: MessageEventSource | null
}

export function isPreferencesReadyMessage(
  event: MessageEventLike,
  tabURL: string,
  frameWindow: Window
): boolean {
  const data = event.data as Record<string, unknown> | null
  return event.origin === frameOrigin(tabURL)
    && event.source === frameWindow
    && data?.source === 'ximoai-embedded'
    && data?.version === 1
    && data?.type === 'preferences:ready'
}
