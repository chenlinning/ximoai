import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import XimoAIHomeCard from '../XimoAIHomeCard.vue'

describe('XimoAIHomeCard HTML covers', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('scales a fixed 5:3 design surface instead of reflowing the HTML on narrow cards', async () => {
    vi.stubGlobal('IntersectionObserver', undefined)
    class ResizeObserverMock {
      constructor(private readonly callback: ResizeObserverCallback) {}

      observe(target: Element) {
        this.callback([{
          target,
          contentRect: { width: 300 }
        } as ResizeObserverEntry], this as unknown as ResizeObserver)
      }

      disconnect() {}
      unobserve() {}
    }
    vi.stubGlobal('ResizeObserver', ResizeObserverMock)

    const wrapper = mount(XimoAIHomeCard, {
      props: {
        tab: {
          id: 'html-cover',
          label: 'HTML Cover',
          url: 'https://cover.example/app',
          cover_url: 'data:text/html;base64,PGRpdj5Db3ZlcjwvZGl2Pg==',
          enabled: true,
          sort_order: 0
        },
        theme: 'dark'
      }
    })
    await flushPromises()

    const frame = wrapper.get('.home-entry-cover-frame').element as HTMLIFrameElement
    expect(frame.style.width).toBe('1200px')
    expect(frame.style.height).toBe('720px')
    expect(frame.style.transform).toBe('scale(0.25)')
    expect(frame.style.colorScheme).toBe('dark')
    expect(wrapper.get('.home-entry-cover-frame').attributes('loading')).toBe('lazy')
    wrapper.unmount()
  })

  it('uses an external source for cacheable HTML covers', async () => {
    vi.stubGlobal('IntersectionObserver', undefined)
    const wrapper = mount(XimoAIHomeCard, {
      props: {
        tab: {
          id: 'html-cover',
          label: 'HTML Cover',
          url: 'https://cover.example/app',
          cover_url: '/api/v1/settings/assets/home-tabs/html-cover/cover/hash.html',
          enabled: true,
          sort_order: 0
        },
        theme: 'light'
      }
    })
    await flushPromises()

    const frame = wrapper.get('.home-entry-cover-frame')
    expect(frame.attributes('src')).toBe('/api/v1/settings/assets/home-tabs/html-cover/cover/hash.html')
    expect(frame.attributes('srcdoc')).toBeUndefined()
    wrapper.unmount()
  })

  it('only mounts animated covers while their card is near the viewport', async () => {
    let observerCallback: IntersectionObserverCallback | undefined
    class IntersectionObserverMock {
      constructor(callback: IntersectionObserverCallback) {
        observerCallback = callback
      }

      observe() {}
      disconnect() {}
      unobserve() {}
      takeRecords() { return [] }
      readonly root = null
      readonly rootMargin = '25% 0px'
      readonly thresholds = [0]
    }
    vi.stubGlobal('IntersectionObserver', IntersectionObserverMock)

    const wrapper = mount(XimoAIHomeCard, {
      props: {
        tab: {
          id: 'html-cover',
          label: 'HTML Cover',
          url: 'https://cover.example/app',
          cover_url: '/api/v1/settings/assets/home-tabs/html-cover/cover/hash.html',
          enabled: true,
          sort_order: 0
        },
        theme: 'light'
      }
    })
    expect(wrapper.find('.home-entry-cover-frame').exists()).toBe(false)

    observerCallback?.([{ isIntersecting: true } as IntersectionObserverEntry], {} as IntersectionObserver)
    await flushPromises()
    expect(wrapper.find('.home-entry-cover-frame').exists()).toBe(true)

    observerCallback?.([{ isIntersecting: false } as IntersectionObserverEntry], {} as IntersectionObserver)
    await flushPromises()
    expect(wrapper.find('.home-entry-cover-frame').exists()).toBe(false)
    wrapper.unmount()
  })
})
