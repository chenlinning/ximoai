import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import AccountTestModal from '../AccountTestModal.vue'

const { getAvailableModelsMock } = vi.hoisted(() => ({
  getAvailableModelsMock: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getAvailableModels: getAvailableModelsMock
    },
    platforms: {
      list: vi.fn().mockResolvedValue([
        { slug: 'openai', protocol: 'openai' }
      ])
    }
  }
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn()
  })
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

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: { show: { type: Boolean, default: false } },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const SelectStub = defineComponent({
  name: 'SelectStub',
  props: {
    modelValue: { type: [String, Number, Boolean, null], default: '' },
    options: { type: Array, default: () => [] },
    valueKey: { type: String, default: 'value' },
    labelKey: { type: String, default: 'label' }
  },
  emits: ['update:modelValue'],
  template: `
    <select
      v-bind="$attrs"
      :value="modelValue"
      @change="$emit('update:modelValue', $event.target.value)"
    >
      <option
        v-for="option in options"
        :key="option[valueKey]"
        :value="option[valueKey]"
      >
        {{ option[labelKey] }}
      </option>
    </select>
  `
})

const TextAreaStub = defineComponent({
  name: 'TextArea',
  props: {
    modelValue: { type: String, default: '' }
  },
  emits: ['update:modelValue'],
  template: `
    <textarea
      v-bind="$attrs"
      :value="modelValue"
      @input="$emit('update:modelValue', $event.target.value)"
    />
  `
})

function buildAccount() {
  return {
    id: 1,
    name: 'OpenAI OAuth',
    platform: 'openai',
    type: 'oauth',
    status: 'active',
    credentials: {},
    extra: {},
    concurrency: 1,
    priority: 1,
    proxy_id: null,
    auto_pause_on_expired: false
  } as any
}

describe('AccountTestModal', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    getAvailableModelsMock.mockReset()
    getAvailableModelsMock.mockResolvedValue([
      { id: 'gpt-5.4', display_name: 'GPT-5.4' }
    ])
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: {
        getReader: () => ({
          read: vi.fn().mockResolvedValue({ done: true, value: undefined })
        })
      }
    } as any)
    localStorage.setItem('auth_token', 'test-token')
  })

  afterEach(() => {
    global.fetch = originalFetch
    localStorage.clear()
  })

  it('posts compact mode for OpenAI compact probe', async () => {
    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: buildAccount()
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Select: SelectStub,
          TextArea: TextAreaStub,
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4'
    ;(wrapper.vm as any).testMode = 'compact'
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, options] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(options.body)).toMatchObject({
      model_id: 'gpt-5.4',
      mode: 'compact'
    })
  })

  it('renders audio and video results returned by account test events', async () => {
    const encoder = new TextEncoder()
    const chunks = [
      'data: {"type":"test_start","model":"gpt-4o-audio-preview"}\n',
      'data: {"type":"audio","audio_url":"data:audio/mpeg;base64,bXAz","mime_type":"audio/mpeg"}\n',
      'data: {"type":"video","video_url":"https://cdn.example/test.mp4","mime_type":"video/mp4"}\n',
      'data: {"type":"test_complete","success":true}\n'
    ].map((line) => encoder.encode(line))
    let index = 0
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: {
        getReader: () => ({
          read: vi.fn().mockImplementation(async () => {
            if (index < chunks.length) {
              return { done: false, value: chunks[index++] }
            }
            return { done: true, value: undefined }
          })
        })
      }
    } as any)

    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: buildAccount()
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Select: SelectStub,
          TextArea: TextAreaStub,
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'gpt-4o-audio-preview'
    await (wrapper.vm as any).startTest()
    await flushPromises()
    await flushPromises()

    const audio = wrapper.find('audio')
    expect(audio.exists()).toBe(true)
    expect(audio.attributes('src')).toBe('data:audio/mpeg;base64,bXAz')

    const video = wrapper.find('video')
    expect(video.exists()).toBe(true)
    expect(video.attributes('src')).toBe('https://cdn.example/test.mp4')
    expect(wrapper.find('a[href="https://cdn.example/test.mp4"]').exists()).toBe(true)
  })

  it('prints non-text test content as separate lines', async () => {
    const encoder = new TextEncoder()
    const chunks = [
      'data: {"type":"test_start","model":"sora-2"}\n',
      'data: {"type":"content","text":"Video status: queued"}\n',
      'data: {"type":"content","text":"Video test still processing: task_123"}\n',
      'data: {"type":"error","error":"Video test still processing: task_123"}\n'
    ].map((line) => encoder.encode(line))
    let index = 0
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: {
        getReader: () => ({
          read: vi.fn().mockImplementation(async () => {
            if (index < chunks.length) {
              return { done: false, value: chunks[index++] }
            }
            return { done: true, value: undefined }
          })
        })
      }
    } as any)

    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: {
          ...buildAccount(),
          type: 'apikey'
        }
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Select: SelectStub,
          TextArea: TextAreaStub,
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'sora-2'
    ;(wrapper.vm as any).testType = 'video'
    await (wrapper.vm as any).startTest()
    await flushPromises()
    await flushPromises()

    expect(wrapper.text()).toContain('Video status: queued')
    expect(wrapper.text()).toContain('Video test still processing: task_123')
    expect((wrapper.vm as any).streamingContent).toBe('')
  })
})
