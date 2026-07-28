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

  it('builds a width-filling spotlight and restores the centered static rows', async () => {
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
    expect(wrapper.find('.home-entry-spotlight').exists()).toBe(false)

    await cards[0].trigger('mouseenter')
    await flushPromises()

    expect(wrapper.get('.home-entry-spotlight').classes()).toContain('home-entry-spotlight--columns-2')
    expect(wrapper.get('[data-home-tab-id="workbench"]').classes()).toContain('home-entry-card--active')
    expect(wrapper.find('.home-entry-card--dimmed').exists()).toBe(false)

    await wrapper.get('.home-entry-area').trigger('mouseleave')
    await flushPromises()

    expect(wrapper.find('.home-entry-spotlight').exists()).toBe(false)
    expect(wrapper.findAll('.home-entry-row')).toHaveLength(1)
    expect(wrapper.findAll('.home-entry-card')).toHaveLength(2)
    wrapper.unmount()
  })

  it('moves the spotlight ahead of every remaining static row', async () => {
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

    await wrapper.get('[data-home-tab-id="tab-6"]').trigger('mouseenter')
    await flushPromises()

    expect(wrapper.get('.home-entry-area').element.firstElementChild?.classList)
      .toContain('home-entry-spotlight')
    expect(wrapper.get('.home-entry-spotlight').classes()).toContain('home-entry-spotlight--center')
    wrapper.unmount()
  })
})
