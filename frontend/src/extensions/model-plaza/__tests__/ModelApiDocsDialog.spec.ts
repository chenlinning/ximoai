import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ModelApiDocsDialog from '../ModelApiDocsDialog.vue'
import { modelApiDocsAPI } from '@/api/modelApiDocs'
import type { ModelAPIDocsResponse } from '@/api/modelApiDocs'

const state = vi.hoisted(() => ({ isAdmin: false }))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => state
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() })
}))

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('@/api/modelApiDocs', () => ({
  modelApiDocsAPI: {
    save: vi.fn(),
    reset: vi.fn()
  }
}))

const response: ModelAPIDocsResponse = {
  platform: 'custom-openai',
  protocol: 'openai_compatible',
  model: 'custom-model',
  source: 'automatic' as const,
  binding: {
    platform: 'custom-openai',
    protocol: 'openai_compatible',
    model: 'custom-model',
    categories: [{
      category: 'conversation',
      endpoints: [{ profile: 'openai_responses', variants: ['sync', 'stream'] }]
    }]
  },
  profiles: [{
    id: 'openai_responses',
    category: 'conversation',
    protocol: 'openai_responses',
    title: 'OpenAI Responses',
    description: 'Responses docs',
    variants: [
      {
        id: 'sync', label: 'Synchronous', mode: 'sync', transport: 'http', delivery: 'json',
        termination: 'The response completes with one JSON document.',
        steps: [{ id: 'request', title: 'Request', method: 'POST', path: '/v1/responses', content_type: 'application/json', request_example: '{"stream":false}', response_example: '{"status":"completed"}', parameters: [
          { name: 'Authorization', location: 'header', required: true, type: 'string', description: 'Bearer API key.' },
          { name: 'model', location: 'body', required: true, type: 'string', description: 'Public model name.' }
        ] }]
      },
      {
        id: 'stream', label: 'Streaming', mode: 'stream', transport: 'http', delivery: 'sse',
        termination: 'Stop after response.completed.',
        steps: [{ id: 'request', title: 'Request', method: 'POST', path: '/v1/responses', content_type: 'application/json', request_example: '{"stream":true}', response_example: 'event: response.completed', parameters: [] }]
      }
    ]
  }],
  editor: null
}

function mountDialog(documentation: ModelAPIDocsResponse = response) {
  return mount(ModelApiDocsDialog, {
    props: {
      show: true,
      modelName: 'custom-model',
      documentation: structuredClone(documentation)
    },
    global: {
      stubs: {
        BaseDialog: { template: '<div v-if="$attrs.show"><slot/><slot name="footer"/></div>' },
        Icon: true
      }
    }
  })
}

describe('ModelApiDocsDialog', () => {
  beforeEach(() => {
    state.isAdmin = false
    vi.clearAllMocks()
    vi.mocked(modelApiDocsAPI.save).mockResolvedValue(response.binding)
    vi.mocked(modelApiDocsAPI.reset).mockResolvedValue(undefined)
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } })
  })

  it('shows all selected variants to a regular user without edit controls', async () => {
    const wrapper = mountDialog()
    await flushPromises()

    expect(wrapper.text()).toContain('OpenAI Responses')
    expect(wrapper.text()).toContain('modelPlaza.apiDocs.modes.sync')
    expect(wrapper.text()).toContain('modelPlaza.apiDocs.modes.stream')
    expect(wrapper.text()).toContain('modelPlaza.apiDocs.parameters')
    expect(wrapper.text()).toContain('Authorization')
    expect(wrapper.text()).toContain('modelPlaza.apiDocs.termination')
    expect(wrapper.text()).toContain('The response completes with one JSON document.')
    expect(wrapper.find('[data-testid="model-api-docs-edit"]').exists()).toBe(false)
  })

  it('lets an administrator keep sync and stream selected concurrently', async () => {
    state.isAdmin = true
    const wrapper = mountDialog({
      ...structuredClone(response),
      editor: {
        automatic_binding: structuredClone(response.binding),
        available_profiles: structuredClone(response.profiles)
      }
    })
    await flushPromises()

    await wrapper.get('[data-testid="model-api-docs-edit"]').trigger('click')
    const sync = wrapper.get<HTMLInputElement>('[data-testid="variant-openai_responses-sync"]')
    const stream = wrapper.get<HTMLInputElement>('[data-testid="variant-openai_responses-stream"]')
    expect(sync.element.checked).toBe(true)
    expect(stream.element.checked).toBe(true)

    await stream.setValue(false)
    expect(sync.element.checked).toBe(true)
    expect(stream.element.checked).toBe(false)
  })

  it('copies the public base URL, model name, and authorization placeholder', async () => {
    const wrapper = mountDialog()
    await flushPromises()

    for (const testID of ['copy-base-url', 'copy-model-name', 'copy-auth-header']) {
      await wrapper.get(`[data-testid="${testID}"]`).trigger('click')
    }

    expect(navigator.clipboard.writeText).toHaveBeenNthCalledWith(1, window.location.origin)
    expect(navigator.clipboard.writeText).toHaveBeenNthCalledWith(2, 'custom-model')
    expect(navigator.clipboard.writeText).toHaveBeenNthCalledWith(3, 'Authorization: Bearer $XIMOAI_API_KEY')
  })

  it('renders Volcengine TTS HTTP and WebSocket workflows concurrently', async () => {
    const documentation: ModelAPIDocsResponse = {
      platform: 'volcengine-agent-plan', protocol: 'native', model: 'seed-tts-2.0', source: 'automatic',
      binding: {
        platform: 'volcengine-agent-plan', protocol: 'native', model: 'seed-tts-2.0',
        categories: [{ category: 'tts', endpoints: [
          { profile: 'volcengine_tts_unidirectional', variants: ['sync'] },
          { profile: 'volcengine_tts_unidirectional_stream', variants: ['stream'] },
          { profile: 'volcengine_tts_bidirectional', variants: ['bidirectional'] }
        ] }]
      },
      profiles: [
        {
          id: 'volcengine_tts_unidirectional', category: 'tts', protocol: 'volcengine_agent_plan_native', title: 'HTTP TTS', description: '',
          variants: [{ id: 'sync', label: 'Sync', mode: 'sync', transport: 'http', delivery: 'binary', steps: [{ id: 'request', title: 'Request', method: 'POST', path: '/v1/volcengine/audio/tts/unidirectional' }] }]
        },
        {
          id: 'volcengine_tts_unidirectional_stream', category: 'tts', protocol: 'volcengine_agent_plan_native', title: 'WSS Stream TTS', description: '',
          variants: [{ id: 'stream', label: 'Stream', mode: 'stream', transport: 'websocket', delivery: 'websocket_frames', steps: [{ id: 'session', title: 'Session', method: 'GET', path: '/v1/volcengine/audio/tts/unidirectional/stream' }] }]
        },
        {
          id: 'volcengine_tts_bidirectional', category: 'tts', protocol: 'volcengine_agent_plan_native', title: 'WSS Bidirectional TTS', description: '',
          variants: [{ id: 'bidirectional', label: 'Bidirectional', mode: 'bidirectional', transport: 'websocket', delivery: 'websocket_frames', steps: [{ id: 'session', title: 'Session', method: 'GET', path: '/v1/volcengine/audio/tts/bidirection' }] }]
        }
      ],
      editor: null
    }

    const wrapper = mount(ModelApiDocsDialog, {
      props: {
        show: true,
        modelName: 'seed-tts-2.0',
        documentation
      },
      global: { stubs: { BaseDialog: { template: '<div v-if="$attrs.show"><slot/><slot name="footer"/></div>' }, Icon: true } }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('HTTP TTS')
    expect(wrapper.text()).toContain('WSS Stream TTS')
    expect(wrapper.text()).toContain('WSS Bidirectional TTS')
  })
})
