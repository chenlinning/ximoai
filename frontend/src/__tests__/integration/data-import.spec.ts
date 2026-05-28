import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ImportDataModal from '@/components/admin/account/ImportDataModal.vue'
import { adminAPI } from '@/api/admin'

const showError = vi.fn()
const showSuccess = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      importData: vi.fn()
    }
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

describe('ImportDataModal', () => {
  beforeEach(() => {
    showError.mockReset()
    showSuccess.mockReset()
    vi.mocked(adminAPI.accounts.importData).mockReset()
  })

  it('未选择文件时提示错误', async () => {
    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    await wrapper.find('form').trigger('submit')
    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportSelectFile')
  })

  it('无效 JSON 时提示解析失败', async () => {
    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    const input = wrapper.find('input[type="file"]')
    const file = new File(['invalid json'], 'data.json', { type: 'application/json' })
    Object.defineProperty(file, 'text', {
      value: () => Promise.resolve('invalid json')
    })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportParseFailed')
  })

  it('选择多个文件后按文件顺序依次导入', async () => {
    let resolveFirstImport: (() => void) | null = null
    const firstImportDone = new Promise<void>((resolve) => {
      resolveFirstImport = resolve
    })

    vi.mocked(adminAPI.accounts.importData).mockImplementation(async ({ data }) => {
      if ((data as { name?: string }).name === 'first') {
        await firstImportDone
      }
      return {
        proxy_created: 0,
        proxy_reused: 0,
        proxy_failed: 0,
        account_created: 1,
        account_failed: 0,
        errors: []
      }
    })

    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    const input = wrapper.find('input[type="file"]')
    expect(input.attributes('multiple')).toBeDefined()

    const firstFile = new File(['{"name":"first"}'], 'first.json', { type: 'application/json' })
    const secondFile = new File(['{"name":"second"}'], 'second.json', { type: 'application/json' })
    Object.defineProperty(firstFile, 'text', {
      value: () => Promise.resolve('{"name":"first"}')
    })
    Object.defineProperty(secondFile, 'text', {
      value: () => Promise.resolve('{"name":"second"}')
    })
    Object.defineProperty(input.element, 'files', {
      value: [firstFile, secondFile]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(adminAPI.accounts.importData).toHaveBeenCalledTimes(1)
    expect(adminAPI.accounts.importData).toHaveBeenNthCalledWith(1, {
      data: { name: 'first' },
      skip_default_group_bind: true
    })

    resolveFirstImport?.()
    await flushPromises()

    expect(adminAPI.accounts.importData).toHaveBeenCalledTimes(2)
    expect(adminAPI.accounts.importData).toHaveBeenNthCalledWith(2, {
      data: { name: 'second' },
      skip_default_group_bind: true
    })
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.dataImportSuccess')
  })

  it('多个文件导入结果会汇总并保留错误详情', async () => {
    vi.mocked(adminAPI.accounts.importData)
      .mockResolvedValueOnce({
        proxy_created: 1,
        proxy_reused: 0,
        proxy_failed: 0,
        account_created: 2,
        account_failed: 0,
        errors: []
      })
      .mockResolvedValueOnce({
        proxy_created: 0,
        proxy_reused: 1,
        proxy_failed: 1,
        account_created: 0,
        account_failed: 1,
        errors: [
          { kind: 'account', name: 'bad-account', message: 'failed account' }
        ]
      })

    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    const input = wrapper.find('input[type="file"]')
    const firstFile = new File(['{"name":"first"}'], 'first.json', { type: 'application/json' })
    const secondFile = new File(['{"name":"second"}'], 'second.json', { type: 'application/json' })
    Object.defineProperty(firstFile, 'text', {
      value: () => Promise.resolve('{"name":"first"}')
    })
    Object.defineProperty(secondFile, 'text', {
      value: () => Promise.resolve('{"name":"second"}')
    })
    Object.defineProperty(input.element, 'files', {
      value: [firstFile, secondFile]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect((wrapper.vm as any).result).toMatchObject({
      proxy_created: 1,
      proxy_reused: 1,
      proxy_failed: 1,
      account_created: 2,
      account_failed: 1,
      errors: [
        { kind: 'account', name: 'bad-account', message: 'second.json: failed account' }
      ]
    })
    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportCompletedWithErrors')
  })
})
