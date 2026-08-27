import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import XimoAIHomeWorkspace from '../XimoAIHomeWorkspace.vue'

const createSSOTicket = vi.hoisted(() => vi.fn())

vi.mock('@/api', () => ({
  workbenchAPI: { createSSOTicket }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ siteName: 'XimoAI', siteLogo: '' })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  const { ref } = await import('vue')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: ref('zh-CN')
    })
  }
})

const tabs = [
  {
    id: 'workbench',
    label: 'Workbench',
    url: 'https://workbench.ximoai.cn/app',
    enabled: true,
    workbench_sso: true,
    sort_order: 0
  },
  {
    id: 'novel',
    label: 'Novel',
    url: 'https://novel.ximoai.cn/workspace',
    enabled: true,
    workbench_sso: true,
    sort_order: 1
  }
]

const tabsWithAgent = [
  tabs[0],
  {
    id: 'agent-random-id',
    label: 'Agent工作台',
    url: 'https://agent.ximoai.cn/',
    enabled: true,
    workbench_sso: true,
    sort_order: 1
  },
  { ...tabs[1], sort_order: 2 }
]

describe('XimoAIHomeWorkspace multi-site SSO', () => {
  beforeEach(() => {
    vi.stubGlobal('IntersectionObserver', undefined)
    document.documentElement.classList.remove('dark')
    createSSOTicket.mockReset()
    createSSOTicket.mockImplementation(async (audience: string) => ({
      ticket: `ticket-${audience}`,
      expires_in: 60,
      entry_url: `${new URL(audience).origin}/sso/entry?ticket=test`
    }))
  })

  it('opens the agent workspace immediately without rendering the card landing page', async () => {
    const wrapper = mount(XimoAIHomeWorkspace, {
      props: { tabs: tabsWithAgent },
      global: {
        stubs: {
          AppHeader: { template: '<header><slot name="left" /></header>' },
          Icon: true
        }
      }
    })

    await flushPromises()

    expect(createSSOTicket).toHaveBeenCalledTimes(1)
    expect(createSSOTicket).toHaveBeenCalledWith('https://agent.ximoai.cn/')
    expect(wrapper.find('.home-entry-card').exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'LoginGalaxyBackground' }).exists()).toBe(false)
    expect(wrapper.get('iframe').attributes('title')).toBe('Agent工作台')
    expect(wrapper.get('main').classes()).toContain('home-workspace-shell')
  })

  it('loads each audience once and preserves both iframe sessions while switching', async () => {
    const wrapper = mount(XimoAIHomeWorkspace, {
      props: { tabs },
      global: {
        stubs: {
          AppHeader: { template: '<header><slot name="left" /></header>' },
          Icon: true
        }
      }
    })

    const findButton = (label: string) => {
      const button = wrapper.findAll('button').find((item) => item.text().endsWith(label))
      if (!button) throw new Error(`Missing ${label} button:\n${wrapper.html()}`)
      return button
    }

    await flushPromises()
    expect(createSSOTicket).toHaveBeenCalledWith('https://workbench.ximoai.cn/app')
    expect(wrapper.get('.home-workspace-tabs-scroll').classes()).toContain('overflow-x-auto')
    expect(wrapper.get('.home-workspace-tabs').classes()).toContain('flex-nowrap')
    expect(wrapper.get('.home-workspace-tabs').classes()).not.toContain('flex-wrap')
    expect(findButton('Workbench').classes()).toContain('border-primary-500')
    expect(findButton('Novel').classes()).toContain('border-gray-300')
    expect(wrapper.findAll('iframe').map((frame) => frame.attributes('src'))).toEqual([
      'https://workbench.ximoai.cn/sso/entry?ticket=test'
    ])

    await findButton('Novel').trigger('click')
    await flushPromises()
    expect(createSSOTicket).toHaveBeenCalledWith('https://novel.ximoai.cn/workspace')
    expect(wrapper.findAll('iframe').map((frame) => frame.attributes('src'))).toEqual([
      'https://workbench.ximoai.cn/sso/entry?ticket=test',
      'https://novel.ximoai.cn/sso/entry?ticket=test'
    ])

    await findButton('Workbench').trigger('click')
    await flushPromises()
    expect(createSSOTicket).toHaveBeenCalledTimes(2)
    expect(wrapper.findAll('iframe')).toHaveLength(2)
  })

  it('replaces a consumed SSO entry URL with the stable configured URL after the child is ready', async () => {
    const wrapper = mount(XimoAIHomeWorkspace, {
      attachTo: document.body,
      props: { tabs: [tabs[0]] },
      global: {
        stubs: {
          AppHeader: { template: '<header><slot name="left" /></header>' },
          Icon: true
        }
      }
    })

    await flushPromises()

    const frame = wrapper.get('iframe')
    expect(frame.attributes('src')).toContain('/sso/entry?ticket=test')
    expect(wrapper.get('main').classes()).toContain('home-workspace-shell')
    expect(wrapper.get('section').classes()).toContain('overflow-hidden')
    expect(frame.classes()).toEqual(expect.arrayContaining(['absolute', 'inset-0', 'block', 'h-full', 'w-full']))

    const readyEvent = new MessageEvent('message', {
      origin: 'https://workbench.ximoai.cn',
      data: {
        source: 'ximoai-embedded',
        version: 1,
        type: 'preferences:ready'
      }
    })
    Object.defineProperty(readyEvent, 'source', { value: frame.element.contentWindow })
    expect(readyEvent.origin).toBe('https://workbench.ximoai.cn')
    expect(readyEvent.source).toBe(frame.element.contentWindow)
    window.dispatchEvent(readyEvent)
    await flushPromises()

    expect(wrapper.get('iframe').attributes('src')).toBe('https://workbench.ximoai.cn/app')
    expect(createSSOTicket).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('falls back to the first visible workspace when agent is unavailable', async () => {
    const wrapper = mount(XimoAIHomeWorkspace, {
      props: { tabs },
      global: {
        stubs: {
          AppHeader: { template: '<header><slot name="left" /></header>' },
          Icon: true
        }
      }
    })

    await flushPromises()

    expect(createSSOTicket).toHaveBeenCalledTimes(1)
    expect(createSSOTicket).toHaveBeenCalledWith('https://workbench.ximoai.cn/app')
    expect(wrapper.findAll('iframe')).toHaveLength(1)
    wrapper.unmount()
  })
})
