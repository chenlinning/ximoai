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

vi.mock('../ModelMetadataDialog.vue', () => ({
  default: {
    props: ['show', 'modelName', 'platform', 'brand', 'types', 'invocationModes', 'editor'],
    emits: ['updated', 'close'],
    template: '<button v-if="show" data-testid="metadata-dialog" @click="$emit(\'updated\', { brand: \'XimoAI Lab\', types: [\'video\'], invocation_modes: [\'async\'], editor })">{{ modelName }}</button>'
  }
}))

describe('ModelPlazaPage compact metadata', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.channels.mockResolvedValue([{
      name: 'Default channel', description: '', platforms: [{
        platform: 'custom-openai', display_name: 'Custom OpenAI', color: '#10a37f', protocol: 'openai_compatible',
        groups: [{
          id: 1, name: 'Default', platform: 'custom-openai', subscription_type: 'standard',
          rate_multiplier: 1, peak_rate_enabled: false, peak_start: '', peak_end: '',
          peak_rate_multiplier: 1, is_exclusive: false
        }],
        supported_models: [{
          name: 'custom-model', platform: 'custom-openai', brand: 'OpenAI',
          types: ['conversation'], invocation_modes: ['sync', 'stream'],
          metadata_editor: {
            automatic: { brand: 'Other', types: ['conversation'], invocation_modes: ['sync', 'stream'] },
            override: { brand: 'OpenAI' },
            options: {
              types: ['conversation', 'embedding', 'image', 'video', 'tts', 'asr'],
              invocation_modes: ['sync', 'stream', 'async', 'bidirectional', 'batch']
            }
          },
          pricing: {
            billing_mode: 'token', input_price: 0.000001, output_price: 0.000002,
            cache_write_price: null, cache_read_price: null, image_input_price: null,
            image_output_price: null, per_request_price: null, intervals: []
          }
        }, {
          name: 'image-model', platform: 'custom-openai', brand: 'Google',
          types: ['image'], invocation_modes: ['sync', 'async'],
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

  it('renders brand, type, and invocation mode as compact card badges', async () => {
    const wrapper = mountPage()
    await flushPromises()
    const card = wrapper.get('[data-model-key="custom-openai:custom-model"]')

    expect(card.get('[data-model-brand-chip="OpenAI"]').exists()).toBe(true)
    expect(card.get('[data-model-type-chip="conversation"]').exists()).toBe(true)
    expect(card.get('[data-model-mode-chip="sync"]').exists()).toBe(true)
    expect(card.get('[data-model-mode-chip="stream"]').exists()).toBe(true)
    expect(card.attributes('role')).toBeUndefined()
    expect(card.attributes('tabindex')).toBeUndefined()
  })

  it('keeps all three left-aligned filter levels visible and combines them', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.findAll('[data-filter-level]').map((item) => item.attributes('data-filter-level'))).toEqual(['brand', 'type', 'mode'])
    await wrapper.get('[data-model-brand="OpenAI"]').trigger('click')
    await wrapper.get('[data-model-category="conversation"]').trigger('click')
    await wrapper.get('[data-model-mode="stream"]').trigger('click')

    expect(wrapper.find('[data-model-key="custom-openai:custom-model"]').exists()).toBe(true)
    expect(wrapper.find('[data-model-key="custom-openai:image-model"]').exists()).toBe(false)
    expect(wrapper.find('[data-model-mode="async"]').exists()).toBe(true)
  })

  it('opens one metadata editor for administrators and applies all fields', async () => {
    const wrapper = mountPage()
    await flushPromises()

    await wrapper.get('[data-model-metadata-edit="custom-openai:custom-model"]').trigger('click')
    await wrapper.get('[data-testid="metadata-dialog"]').trigger('click')

    const card = wrapper.get('[data-model-key="custom-openai:custom-model"]')
    expect(card.text()).toContain('XimoAI Lab')
    expect(card.find('[data-model-type-chip="video"]').exists()).toBe(true)
    expect(card.find('[data-model-mode-chip="async"]').exists()).toBe(true)
  })

  it('keeps the grid width policy independent from the filtered item count', async () => {
    const wrapper = mountPage()
    await flushPromises()

    const grid = wrapper.get('[data-model-grid]')
    expect(grid.classes()).toContain('model-card-grid')
    expect(grid.attributes('data-card-min-width')).toBe('20rem')
    expect(grid.attributes('data-card-max-width')).toBe('24rem')
  })
})
