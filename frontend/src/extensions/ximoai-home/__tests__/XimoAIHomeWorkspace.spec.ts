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
    cover_url: 'data:image/png;base64,AAAA',
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

describe('XimoAIHomeWorkspace multi-site SSO', () => {
  beforeEach(() => {
    document.documentElement.classList.remove('dark')
    createSSOTicket.mockReset()
    createSSOTicket.mockImplementation(async (audience: string) => ({
      ticket: `ticket-${audience}`,
      expires_in: 60,
      entry_url: `${new URL(audience).origin}/sso/entry?ticket=test`
    }))
  })

  it('syncs HTML cover color scheme with the main theme', async () => {
    const wrapper = mount(XimoAIHomeWorkspace, {
      props: {
        tabs: [{
          id: 'html-cover',
          label: 'HTML Cover',
          url: 'https://cover.example/app',
          cover_url: 'data:text/html;base64,PGRpdj5Db3ZlcjwvZGl2Pg==',
          enabled: true,
          sort_order: 0
        }]
      },
      global: {
        stubs: {
          AppHeader: { template: '<header><slot name="left" /></header>' },
          LoginGalaxyBackground: true,
          Icon: true
        }
      }
    })

    const coverFrame = wrapper.get('.home-entry-cover-frame')
    expect((coverFrame.element as HTMLIFrameElement).style.colorScheme).toBe('light')

    document.documentElement.classList.add('dark')
    await flushPromises()

    expect((coverFrame.element as HTMLIFrameElement).style.colorScheme).toBe('dark')
    wrapper.unmount()
  })

  it('loads each audience once and preserves both iframe sessions while switching', async () => {
    const wrapper = mount(XimoAIHomeWorkspace, {
      props: { tabs },
      global: {
        stubs: {
          AppHeader: { template: '<header><slot name="left" /></header>' },
          LoginGalaxyBackground: true,
          Icon: true
        }
      }
    })

    const findButton = (label: string) => {
      const button = wrapper.findAll('button').find((item) => item.text().endsWith(label))
      if (!button) throw new Error(`Missing ${label} button:\n${wrapper.html()}`)
      return button
    }

    const entryMedia = wrapper.get('.home-entry-media')
    const entryCover = entryMedia.get('img')
    expect(entryMedia.classes()).toContain('aspect-[5/3]')
    expect(entryCover.classes()).toContain('object-contain')
    expect(entryCover.classes()).not.toContain('object-cover')
    expect(entryMedia.find('.home-entry-label').exists()).toBe(false)
    expect(wrapper.get('.home-entry-label').text()).toBe('Workbench')

    await findButton('Workbench').trigger('click')
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
          LoginGalaxyBackground: true,
          Icon: true
        }
      }
    })

    await wrapper.get('.home-entry-card button').trigger('click')
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

  it('keeps the home cards in centered static rows', async () => {
    const wrapper = mount(XimoAIHomeWorkspace, {
      props: { tabs },
      global: {
        stubs: {
          AppHeader: { template: '<header><slot name="left" /></header>' },
          LoginGalaxyBackground: true,
          Icon: true
        }
      }
    })

    const cards = wrapper.findAll('.home-entry-card')
    expect(cards).toHaveLength(2)
    expect(wrapper.findAll('.home-entry-row')).toHaveLength(1)
    expect(wrapper.get('.home-entry-row').classes()).toContain('home-entry-row--primary')

    await cards[0].find('button').trigger('click')
    await flushPromises()

    expect(wrapper.findAll('.home-entry-card')).toHaveLength(0)
    expect(wrapper.findAll('iframe')).toHaveLength(1)
    wrapper.unmount()
  })

  it('keeps later rows compact without creating hover layout state', () => {
    const manyTabs = Array.from({ length: 7 }, (_, index) => ({
      id: `tab-${index}`,
      label: `Tab ${index}`,
      url: `https://tab-${index}.example/app`,
      enabled: true,
      workbench_sso: false,
      sort_order: index
    }))
    const wrapper = mount(XimoAIHomeWorkspace, {
      props: { tabs: manyTabs },
      global: {
        stubs: {
          AppHeader: { template: '<header><slot name="left" /></header>' },
          LoginGalaxyBackground: true,
          Icon: true
        }
      }
    })

    expect(wrapper.findAll('.home-entry-row')).toHaveLength(4)
    expect(wrapper.findAll('.home-entry-row--primary')).toHaveLength(1)
    expect(wrapper.findAll('.home-entry-row--compact')).toHaveLength(3)
    wrapper.unmount()
  })
})
