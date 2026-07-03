import { mount, flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import HomeView from '../HomeView.vue'
import { useAppStore } from '@/stores/app'

const { createSSOTicket, getCurrentUser } = vi.hoisted(() => ({
  createSSOTicket: vi.fn(),
  getCurrentUser: vi.fn()
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      locale: { value: 'zh' },
      t: (key: string) => key
    },
    install: vi.fn()
  }),
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('@/api', () => ({
  workbenchAPI: {
    createSSOTicket
  },
  authAPI: {
    getCurrentUser,
    refreshToken: vi.fn()
  },
  isTotp2FARequired: vi.fn()
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: vi.fn()
}))

function seedPublicSettings(enabled = true) {
  const appStore = useAppStore()
  appStore.cachedPublicSettings = {
    registration_enabled: true,
    email_verify_enabled: false,
    force_email_on_third_party_signup: false,
    registration_email_suffix_whitelist: [],
    promo_code_enabled: true,
    password_reset_enabled: false,
    invitation_code_enabled: false,
    turnstile_enabled: false,
    turnstile_site_key: '',
    site_name: 'XimoAI',
    site_logo: '',
    site_subtitle: '',
    api_base_url: '',
    contact_info: '',
    doc_url: '',
    home_content: 'https://custom.example',
    hide_ccs_import_button: false,
    workbench_sso_enabled: enabled,
    workbench_base_url: 'http://127.0.0.1:4173',
    payment_enabled: false,
    risk_control_enabled: false,
    table_default_page_size: 20,
    table_page_size_options: [20],
    custom_menu_items: [],
    custom_endpoints: [],
    linuxdo_oauth_enabled: false,
    wechat_oauth_enabled: false,
    oidc_oauth_enabled: false,
    oidc_oauth_provider_name: 'OIDC',
    github_oauth_enabled: false,
    google_oauth_enabled: false,
    backend_mode_enabled: false,
    version: 'test',
    balance_low_notify_enabled: false,
    account_quota_notify_enabled: false,
    balance_low_notify_threshold: 0,
    channel_monitor_enabled: true,
    channel_monitor_default_interval_seconds: 60,
    available_channels_enabled: false,
    service_quota_enabled: false,
    affiliate_enabled: false
  }
  appStore.publicSettingsLoaded = true
}

function mountHome() {
  return mount(HomeView, {
    global: {
      stubs: {
        RouterLink: { template: '<a><slot /></a>' },
        LocaleSwitcher: { template: '<div />' },
        Icon: { template: '<span />' }
      }
    }
  })
}

describe('HomeView Workbench SSO', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: vi.fn().mockReturnValue({
        matches: false,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn()
      })
    })
    createSSOTicket.mockReset()
    getCurrentUser.mockReset()
    getCurrentUser.mockResolvedValue({
      data: { id: 1, email: 'user@example.com', role: 'user', run_mode: 'standard' }
    })
  })

  it('requests a ticket and renders iframe when logged in and workbench sso is enabled', async () => {
    seedPublicSettings(true)
    localStorage.setItem('auth_token', 'main-site-token')
    localStorage.setItem('auth_user', JSON.stringify({ id: 1, email: 'user@example.com', role: 'user' }))
    createSSOTicket.mockResolvedValue({
      ticket: 'ticket-1',
      expires_in: 60,
      entry_url: 'http://127.0.0.1:4173/sso/entry?ticket=ticket-1'
    })

    const wrapper = mountHome()
    await flushPromises()

    expect(createSSOTicket).toHaveBeenCalledWith('http://127.0.0.1:4173')
    const iframe = wrapper.get('iframe')
    expect(iframe.attributes('src')).toBe('http://127.0.0.1:4173/sso/entry?ticket=ticket-1')
    expect(iframe.attributes('src')).not.toContain('auth_token')
  })

  it('does not request a ticket when the user is not logged in', async () => {
    seedPublicSettings(true)

    mountHome()
    await flushPromises()

    expect(createSSOTicket).not.toHaveBeenCalled()
  })

  it('shows retry state when ticket request fails', async () => {
    seedPublicSettings(true)
    localStorage.setItem('auth_token', 'main-site-token')
    localStorage.setItem('auth_user', JSON.stringify({ id: 1, email: 'user@example.com', role: 'user' }))
    createSSOTicket.mockRejectedValueOnce(new Error('failed'))

    const wrapper = mountHome()
    await flushPromises()

    expect(wrapper.text()).toContain('Workbench login failed')
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.get('button').text()).toContain('Retry')
  })
})
