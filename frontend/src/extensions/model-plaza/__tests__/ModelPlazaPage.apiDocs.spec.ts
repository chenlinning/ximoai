import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ModelPlazaPage from '../ModelPlazaPage.vue'

const mocks = vi.hoisted(() => ({
  channels: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('@/api/channels', () => ({
  default: { getModelPlaza: mocks.channels }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: mocks.showSuccess, showError: mocks.showError })
}))

vi.mock('../ModelApiDocsDialog.vue', () => ({
  default: {
    props: ['show', 'modelName', 'documentation'],
    emits: ['close'],
    template: '<div v-if="show" data-testid="docs-dialog">{{ modelName }}</div>'
  }
}))

vi.mock('../ModelBrandDialog.vue', () => ({
  default: {
    props: ['show', 'modelName', 'platform', 'brand', 'editor'],
    emits: ['updated', 'close'],
    template: '<button v-if="show" data-testid="brand-dialog" @click="$emit(\'updated\', { brand: \'XimoAI Lab\', editor: { automatic_brand: \'OpenAI\', source: \'administrator\' } })">{{ brand }}</button>'
  }
}))

describe('ModelPlazaPage API documentation entry', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.channels.mockResolvedValue([{
      name: 'Default channel', description: '', platforms: [{
        platform: 'custom-openai',
        display_name: 'Custom OpenAI', color: '#10a37f', protocol: 'openai_compatible',
        groups: [{
          id: 1, name: 'Default', platform: 'custom-openai', subscription_type: 'standard',
          rate_multiplier: 1, peak_rate_enabled: false, peak_start: '', peak_end: '',
          peak_rate_multiplier: 1, is_exclusive: false
        }],
        supported_models: [{
          name: 'custom-model', platform: 'custom-openai', brand: 'OpenAI',
          brand_editor: { automatic_brand: 'Other', source: 'automatic' },
          types: ['conversation'], capabilities: ['responses'], protocols: ['openai_responses'], invocation_modes: ['sync', 'stream'],
          api_documentation: modelDocs('custom-model', 'conversation'),
          pricing: {
            billing_mode: 'token', input_price: 0.000001, output_price: 0.000002,
            cache_write_price: null, cache_read_price: null, image_input_price: null,
            image_output_price: null, per_request_price: null, intervals: []
          }
        }, {
          name: 'image-model', platform: 'custom-openai', brand: 'Google',
          types: ['image'], capabilities: ['images'], protocols: ['openai_images'], invocation_modes: ['sync'],
          api_documentation: modelDocs('image-model', 'image'),
          pricing: {
            billing_mode: 'image', input_price: null, output_price: null,
            cache_write_price: null, cache_read_price: null, image_input_price: null,
            image_output_price: 0.01, per_request_price: null, intervals: []
          }
        }]
      }]
    }])
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } })
  })

  function modelDocs(model: string, category: 'conversation' | 'image') {
    return {
      platform: 'custom-openai', protocol: 'openai_compatible', model, source: 'automatic',
      binding: { platform: 'custom-openai', protocol: 'openai_compatible', model, categories: [{ category, endpoints: [] }] },
      profiles: [], editor: null
    }
  }

  function mountPage() {
    return mount(ModelPlazaPage, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot/></div>' },
          Icon: true
        }
      }
    })
  }

  it('opens documentation by mouse and keyboard from the model card', async () => {
    const wrapper = mountPage()
    await flushPromises()
    const card = wrapper.get('[data-model-key="custom-openai:custom-model"]')

    await card.trigger('click')
    expect(wrapper.get('[data-testid="docs-dialog"]').text()).toContain('custom-model')

    await wrapper.get('[data-testid="docs-dialog"]').trigger('close')
    await card.trigger('keydown', { key: 'Enter' })
    expect(wrapper.get('[data-testid="docs-dialog"]').exists()).toBe(true)
  })

  it('keeps copy as an independent card action', async () => {
    const wrapper = mountPage()
    await flushPromises()

    await wrapper.get('button[title="modelPlaza.copyModelName"]').trigger('click')
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('custom-model')
    expect(wrapper.find('[data-testid="docs-dialog"]').exists()).toBe(false)
  })

  it('filters visible models by the effective documentation category', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.find('[data-model-key="custom-openai:custom-model"]').exists()).toBe(true)
    expect(wrapper.find('[data-model-key="custom-openai:image-model"]').exists()).toBe(true)

    await wrapper.get('[data-model-category="image"]').trigger('click')

    expect(wrapper.find('[data-model-key="custom-openai:custom-model"]').exists()).toBe(false)
    expect(wrapper.find('[data-model-key="custom-openai:image-model"]').exists()).toBe(true)
  })

  it('uses brand as the primary filter while keeping model type as the secondary filter', async () => {
    const wrapper = mountPage()
    await flushPromises()

    await wrapper.get('[data-model-brand="OpenAI"]').trigger('click')
    expect(wrapper.find('[data-model-key="custom-openai:custom-model"]').exists()).toBe(true)
    expect(wrapper.find('[data-model-key="custom-openai:image-model"]').exists()).toBe(false)

    await wrapper.get('[data-model-brand="all"]').trigger('click')
    await wrapper.get('[data-model-category="image"]').trigger('click')
    expect(wrapper.find('[data-model-key="custom-openai:custom-model"]').exists()).toBe(false)
    expect(wrapper.find('[data-model-key="custom-openai:image-model"]').exists()).toBe(true)
  })

  it('lets an administrator update the brand without changing the model identity', async () => {
    const wrapper = mountPage()
    await flushPromises()

    await wrapper.get('[data-model-brand-edit="custom-openai:custom-model"]').trigger('click')
    await wrapper.get('[data-testid="brand-dialog"]').trigger('click')

    expect(wrapper.find('[data-model-key="custom-openai:custom-model"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('XimoAI Lab')
  })

  it('loads the complete catalog with a single model-plaza request', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(mocks.channels).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[data-model-key="custom-openai:custom-model"]').exists()).toBe(true)
  })
})
