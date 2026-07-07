import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import AccountBatchTestModal from '../AccountBatchTestModal.vue'

const { getByIdMock, getAvailableModelsMock, batchTestMock } = vi.hoisted(() => ({
  getByIdMock: vi.fn(),
  getAvailableModelsMock: vi.fn(),
  batchTestMock: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getById: getByIdMock,
      getAvailableModels: getAvailableModelsMock,
      batchTest: batchTestMock
    }
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
    disabled: { type: Boolean, default: false },
    placeholder: { type: String, default: '' }
  },
  emits: ['update:modelValue'],
  template: `
    <select
      :value="modelValue"
      :disabled="disabled"
      :data-placeholder="placeholder"
      @change="$emit('update:modelValue', $event.target.value)"
    >
      <option v-for="option in options" :key="option.value" :value="option.value">
        {{ option.label }}
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
  template: '<textarea :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
})

function buildAccount(id = 1) {
  return {
    id,
    name: `OpenAI OAuth ${id}`,
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

function mountModal(accountIds = [1], show = true) {
  return mount(AccountBatchTestModal, {
    props: {
      show,
      accountIds,
      accounts: accountIds.map((id) => buildAccount(id))
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
}

describe('AccountBatchTestModal', () => {
  const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined)

  beforeEach(() => {
    getByIdMock.mockReset()
    getAvailableModelsMock.mockReset()
    batchTestMock.mockReset()
    consoleErrorSpy.mockClear()
    getAvailableModelsMock.mockResolvedValue([
      { id: 'gpt-5.4-mini', display_name: 'GPT-5.4 Mini' }
    ])
    batchTestMock.mockResolvedValue({ total: 1, success: 1, failed: 0, results: [] })
  })

  it('keeps the selected model visible and usable while model loading continues', async () => {
    const wrapper = mountModal()
    await flushPromises()

    ;(wrapper.vm as any).modelOptions = [
      { value: 'gpt-5.4-mini', label: 'GPT-5.4 Mini' }
    ]
    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4-mini'
    ;(wrapper.vm as any).loadingModels = true
    await nextTick()

    const modelSelect = wrapper.findAllComponents(SelectStub)[0]
    expect(modelSelect.props('placeholder')).toBe('GPT-5.4 Mini')
    expect(modelSelect.props('disabled')).toBe(false)

    const startButton = wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.batchTest.start'))
    expect(startButton?.attributes('disabled')).toBeUndefined()
  })

  it('loads models from the first selected account that returns a non-empty list', async () => {
    getAvailableModelsMock
      .mockRejectedValueOnce(new Error('first failed'))
      .mockResolvedValueOnce([
        { id: 'gpt-5.4-mini', display_name: 'GPT-5.4 Mini' }
      ])
      .mockResolvedValueOnce([
        { id: 'gpt-5.5', display_name: 'GPT-5.5' }
      ])

    const wrapper = mountModal([1, 2, 3], false)
    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(getAvailableModelsMock).toHaveBeenCalledTimes(2)
    expect(getAvailableModelsMock).toHaveBeenNthCalledWith(1, 1)
    expect(getAvailableModelsMock).toHaveBeenNthCalledWith(2, 2)
    expect((wrapper.vm as any).modelOptions).toEqual([
      { value: 'gpt-5.4-mini', label: 'GPT-5.4 Mini' }
    ])
    expect((wrapper.vm as any).selectedModelId).toBe('gpt-5.4-mini')
  })
})
