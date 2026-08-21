import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ModelMetadataDialog from '../ModelMetadataDialog.vue'

const mocks = vi.hoisted(() => ({
  save: vi.fn(),
  reset: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('@/api/modelMetadata', () => ({
  modelMetadataAPI: { save: mocks.save, reset: mocks.reset }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: mocks.showSuccess, showError: mocks.showError })
}))

const editor = {
  automatic: { brand: 'Doubao', types: ['tts'], invocation_modes: ['sync', 'stream', 'bidirectional'] },
  override: null,
  options: {
    types: ['conversation', 'embedding', 'image', 'video', 'tts', 'asr'],
    invocation_modes: ['sync', 'stream', 'async', 'bidirectional', 'batch']
  }
}

describe('ModelMetadataDialog', () => {
  beforeEach(() => vi.clearAllMocks())

  function mountDialog() {
    return mount(ModelMetadataDialog, {
      props: {
        show: true,
        platform: 'openai',
        modelName: 'gpt-audio',
        brand: 'OpenAI',
        types: ['tts'],
        invocationModes: ['sync', 'stream', 'bidirectional'],
        editor
      },
      global: {
        stubs: {
          BaseDialog: { template: '<div v-if="$attrs.show"><slot/><slot name="footer"/></div>' },
          Icon: true
        }
      }
    })
  }

  it('offers every type and invocation mode regardless of automatic detection', () => {
    const wrapper = mountDialog()

    for (const type of editor.options.types) {
      expect(wrapper.find(`[data-metadata-type="${type}"]`).exists()).toBe(true)
    }
    for (const mode of editor.options.invocation_modes) {
      expect(wrapper.find(`[data-metadata-mode="${mode}"]`).exists()).toBe(true)
    }
  })

  it('shows all Doubao voice invocation modes together', () => {
    const wrapper = mountDialog()

    expect(wrapper.get('[data-metadata-mode="sync"]').attributes('checked')).toBeDefined()
    expect(wrapper.get('[data-metadata-mode="stream"]').attributes('checked')).toBeDefined()
    expect(wrapper.get('[data-metadata-mode="bidirectional"]').attributes('checked')).toBeDefined()
  })
})
