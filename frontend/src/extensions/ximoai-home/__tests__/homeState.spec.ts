import { describe, expect, it } from 'vitest'
import {
  addLoadedTab,
  buildPreferencesMessage,
  isPreferencesReadyMessage,
  resolveXimoAIHomeMode
} from '../homeState'

describe('XimoAI home state', () => {
  it('keeps configured home_content ahead of the XimoAI tab workspace', () => {
    expect(resolveXimoAIHomeMode({
      settingsLoaded: true,
      tabsLoaded: true,
      authenticated: true,
      homeContent: 'https://existing.example',
      enabledTabCount: 2
    })).toBe('legacy')
  })

  it('keeps previously loaded tabs when another tab is selected', () => {
    const first = addLoadedTab(new Set(), 'docs')
    const second = addLoadedTab(first, 'workbench')

    expect([...second]).toEqual(['docs', 'workbench'])
    expect([...first]).toEqual(['docs'])
  })

  it('only accepts ready messages from the exact configured iframe', () => {
    const frameWindow = {} as Window
    const message = { source: 'ximoai-embedded', version: 1, type: 'preferences:ready' }

    expect(isPreferencesReadyMessage({
      data: message,
      origin: 'https://workbench.ximoai.cn',
      source: frameWindow
    }, 'https://workbench.ximoai.cn/app', frameWindow)).toBe(true)

    expect(isPreferencesReadyMessage({
      data: message,
      origin: 'https://attacker.example',
      source: frameWindow
    }, 'https://workbench.ximoai.cn/app', frameWindow)).toBe(false)

    expect(buildPreferencesMessage('dark', 'zh')).toEqual({
      source: 'ximoai',
      version: 1,
      type: 'preferences:update',
      payload: { theme: 'dark', locale: 'zh-CN' }
    })
  })
})
