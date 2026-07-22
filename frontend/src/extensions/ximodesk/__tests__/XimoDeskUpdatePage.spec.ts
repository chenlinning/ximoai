import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const api = vi.hoisted(() => ({
  get: vi.fn(),
  update: vi.fn(),
  uploadPackage: vi.fn(),
  deleteRelease: vi.fn(),
  deleteApp: vi.fn()
}))
const showError = vi.hoisted(() => vi.fn())
const showSuccess = vi.hoisted(() => vi.fn())

vi.mock('@/api/admin', () => ({
  adminAPI: { ximodeskUpdate: api }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess })
}))

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({
    t: (key: string, fallback?: string) => fallback || key
  })
}))

import XimoDeskUpdatePage from '../XimoDeskUpdatePage.vue'

const config = {
  enabled: true,
  apps: [{ key: 'ximodesk', name: 'XimoDesk', client_type: 'desktop', enabled: true }],
  releases: []
}

async function mountPage() {
  const wrapper = mount(XimoDeskUpdatePage, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: { template: '<span />' }
      }
    }
  })
  await flushPromises()
  return wrapper
}

describe('XimoDeskUpdatePage', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    api.get.mockReset().mockResolvedValue(config)
    api.deleteApp.mockReset().mockResolvedValue({ ...config, apps: [] })
    api.uploadPackage.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
  })

  it('aligns file selection and upload controls at the same height', async () => {
    const wrapper = await mountPage()
    const controls = wrapper.get('[data-testid="package-upload-controls"]')

    expect(controls.classes()).toContain('sm:items-center')
    expect(controls.get('input[type="file"]').classes()).toContain('h-10')
    expect(controls.get('button[type="submit"]').classes()).toContain('h-10')
  })

  it('deletes a persisted app from its registration actions', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const wrapper = await mountPage()

    await wrapper.get('[data-testid="delete-app"]').trigger('click')
    await flushPromises()

    expect(api.deleteApp).toHaveBeenCalledWith('ximodesk')
    expect(wrapper.find('[data-testid="delete-app"]').exists()).toBe(false)
  })

  it('removes an unsaved app locally without calling the delete endpoint', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const wrapper = await mountPage()
    const addApp = wrapper.findAll('button').find((button) => button.text() === 'Add App')
    expect(addApp).toBeDefined()
    await addApp!.trigger('click')

    const deleteButtons = wrapper.findAll('[data-testid="delete-app"]')
    expect(deleteButtons).toHaveLength(2)
    await deleteButtons[1].trigger('click')
    await flushPromises()

    expect(api.deleteApp).not.toHaveBeenCalled()
    expect(wrapper.findAll('[data-testid="delete-app"]')).toHaveLength(1)
  })

  it('shows byte and percentage progress while a large package uploads', async () => {
    let resolveUpload: ((value: unknown) => void) | undefined
    api.uploadPackage.mockImplementation((_payload: FormData, options: { onProgress: (event: { loaded: number; total: number }) => void }) => {
      options.onProgress({ loaded: 210 * 1024 * 1024, total: 420 * 1024 * 1024 })
      return new Promise((resolve) => { resolveUpload = resolve })
    })
    const wrapper = await mountPage()
    const fileInput = wrapper.get('input[type="file"]')
    const file = new File(['package'], 'large.msi', { type: 'application/octet-stream' })
    Object.defineProperty(fileInput.element, 'files', { value: [file], configurable: true })
    await fileInput.trigger('change')

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    const progress = wrapper.get('[role="progressbar"]')
    expect(progress.attributes('aria-valuenow')).toBe('50')
    expect(progress.text()).toContain('50%')
    expect(progress.text()).toContain('210.0 MB')
    expect(progress.text()).toContain('420.0 MB')

    resolveUpload?.({ release: {}, config })
  })
})
