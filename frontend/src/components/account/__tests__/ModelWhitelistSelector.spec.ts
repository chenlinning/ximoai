import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ModelWhitelistSelector from '../ModelWhitelistSelector.vue'

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/api/admin/accounts', () => ({
  accountsAPI: {
    syncUpstreamModels: vi.fn()
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const mountSelector = (props: Record<string, unknown>) =>
  mount(ModelWhitelistSelector, {
    props: {
      modelValue: [],
      ...props
    } as any,
    global: {
      stubs: {
        ModelIcon: true,
        Icon: true
      }
    }
  })

describe('ModelWhitelistSelector', () => {
  it('keeps preview synced upstream models selectable without an account id', async () => {
    const wrapper = mountSelector({
      platform: 'mengfactory',
      canSyncUpstream: true,
      syncUpstreamModels: vi.fn().mockResolvedValue(['vendor-audio', 'vendor-video'])
    })

    const syncButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.accounts.syncUpstreamModels'))
    expect(syncButton).toBeTruthy()

    await syncButton!.trigger('click')
    await flushPromises()
    await wrapper.find('div.cursor-pointer').trigger('click')

    expect(wrapper.text()).toContain('vendor-audio')
    expect(wrapper.text()).toContain('vendor-video')

    const clearButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.accounts.clearAllModels'))
    await clearButton!.trigger('click')

    expect(wrapper.text()).toContain('vendor-audio')
    expect(wrapper.text()).toContain('vendor-video')
  })
})
