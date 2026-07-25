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

describe('XimoAIHomeWorkspace multi-site SSO', () => {
  beforeEach(() => {
    createSSOTicket.mockReset()
    createSSOTicket.mockImplementation(async (audience: string) => ({
      ticket: `ticket-${audience}`,
      expires_in: 60,
      entry_url: `${new URL(audience).origin}/sso/entry?ticket=test`
    }))
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

    await findButton('Workbench').trigger('click')
    await flushPromises()
    expect(createSSOTicket).toHaveBeenCalledWith('https://workbench.ximoai.cn/app')
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
})
