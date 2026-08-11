import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import XimoAIHomeGate from '../XimoAIHomeGate.vue'

const listHomeTabs = vi.hoisted(() => vi.fn())
const authStore = vi.hoisted(() => ({ isAuthenticated: true, user: { id: 1 } }))
const membershipStore = vi.hoisted(() => ({
  summary: null as { level: { code: string } } | null,
  fetch: vi.fn()
}))
const appStore = vi.hoisted(() => ({
  publicSettingsLoaded: true,
  cachedPublicSettings: { home_content: '' }
}))

vi.mock('@/api', () => ({
  ximoaiHomeAPI: { list: listHomeTabs },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
  useAuthStore: () => authStore,
  useMembershipStore: () => membershipStore
}))

describe('XimoAIHomeGate membership visibility', () => {
  beforeEach(() => {
    authStore.isAuthenticated = true
    appStore.publicSettingsLoaded = true
    appStore.cachedPublicSettings = { home_content: '' }
    listHomeTabs.mockResolvedValue([
      { id: 'docs', label: 'Docs', url: 'https://docs.example', enabled: true, sort_order: 0 },
      { id: 'diamond', label: 'Diamond', url: 'https://diamond.example', enabled: true, diamond_only: true, sort_order: 1 },
      { id: 'disabled', label: 'Disabled', url: 'https://disabled.example', enabled: false, diamond_only: true, sort_order: 2 }
    ])
    membershipStore.summary = { level: { code: 'platinum' } }
    membershipStore.fetch.mockReset()
    membershipStore.fetch.mockResolvedValue(membershipStore.summary)
  })

  function mountGate() {
    return mount(XimoAIHomeGate, {
      global: {
        stubs: {
          HomeView: { template: '<div data-testid="home" />' },
          XimoAIHomeWorkspace: {
            props: { tabs: { type: Array, required: true } },
            template: `<div data-testid="workspace">{{ tabs.map((tab) => tab.label).join(',') }}</div>`
          }
        }
      }
    })
  }

  it('hides diamond-only tabs from non-diamond members', async () => {
    const wrapper = mountGate()
    await flushPromises()

    expect(wrapper.find('[data-testid="workspace"]').text()).toBe('Docs')
  })

  it('shows diamond-only tabs to diamond members', async () => {
    membershipStore.summary = { level: { code: 'diamond' } }
    membershipStore.fetch.mockResolvedValue(membershipStore.summary)
    const wrapper = mountGate()
    await flushPromises()

    expect(wrapper.find('[data-testid="workspace"]').text()).toBe('Docs,Diamond')
  })
})
