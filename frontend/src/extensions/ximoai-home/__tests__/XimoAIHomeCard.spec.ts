import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import XimoAIHomeCard from '../XimoAIHomeCard.vue'

describe('XimoAIHomeCard HTML covers', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('scales a fixed 5:3 design surface instead of reflowing the HTML on narrow cards', async () => {
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
    wrapper.unmount()
  })
})
