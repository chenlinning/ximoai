import { defineComponent, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { syncPreviewMock } = vi.hoisted(() => ({
  syncPreviewMock: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      syncUpstreamModelsPreview: syncPreviewMock
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn()
  })
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

import { usePreviewUpstreamModelsSync } from '../usePreviewUpstreamModelsSync'

const PreviewHarness = defineComponent({
  props: {
    platform: { type: String, required: true },
    authScheme: { type: String, default: '' }
  },
  setup(props) {
    const apiKey = ref('test-key')
    const baseUrl = ref('https://api.example.com/v1')
    const { previewSyncUpstreamModels } = usePreviewUpstreamModelsSync({
      platform: ref(props.platform),
      accountType: ref('apikey'),
      platformEnabled: ref(true),
      apiKey,
      baseUrl,
      baseUrlFallback: ref(''),
      anthropicAPIKeyAuthScheme: ref(props.authScheme)
    })
    return { previewSyncUpstreamModels }
  },
  template: '<button type="button" @click="previewSyncUpstreamModels">sync</button>'
})

describe('usePreviewUpstreamModelsSync', () => {
  beforeEach(() => {
    syncPreviewMock.mockReset().mockResolvedValue({ models: ['model-a'] })
  })

  it.each([
    'openai',
    'anthropic',
    'gemini',
    'openai-audio',
  ])('uses the fixed protocol selected by platform %s', async (platform) => {
    const wrapper = mount(PreviewHarness, { props: { platform } })

    await wrapper.get('button').trigger('click')

    expect(syncPreviewMock).toHaveBeenCalledWith({
      platform,
      type: 'apikey',
      credentials: {
        base_url: 'https://api.example.com/v1',
        api_key: 'test-key'
      }
    })
  })

  it('forwards the selected Anthropic bearer auth scheme to the preview account', async () => {
    const wrapper = mount(PreviewHarness, {
      props: {
        platform: 'anthropic',
        authScheme: 'authorization_bearer'
      }
    })

    await wrapper.get('button').trigger('click')

    expect(syncPreviewMock).toHaveBeenCalledWith(expect.objectContaining({
      extra: {
        anthropic_apikey_auth_scheme: 'authorization_bearer'
      }
    }))
  })
})
