import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import LoginGalaxyBackground from '../LoginGalaxyBackground.vue'

const renderGalaxy = vi.hoisted(() => vi.fn())

vi.mock('ogl', () => {
  class Renderer {
    gl: Record<string, any>

    constructor() {
      this.gl = {
        canvas: document.createElement('canvas'),
        enable: vi.fn(),
        blendFunc: vi.fn(),
        clearColor: vi.fn(),
        getExtension: vi.fn(() => ({ loseContext: vi.fn() })),
        SRC_ALPHA: 1,
        ONE_MINUS_SRC_ALPHA: 2,
        BLEND: 3
      }
    }

    setSize() {}
    render() {
      renderGalaxy()
    }
  }

  class Program {
    uniforms: Record<string, { value: any }>

    constructor(_gl: unknown, options: { uniforms: Record<string, { value: any }> }) {
      this.uniforms = options.uniforms
    }
  }

  return {
    Color: class {},
    Mesh: class {},
    Program,
    Renderer,
    Triangle: class {}
  }
})

describe('LoginGalaxyBackground', () => {
  beforeEach(() => {
    renderGalaxy.mockReset()
    vi.stubGlobal('requestAnimationFrame', vi.fn(() => 1))
    vi.stubGlobal('cancelAnimationFrame', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('tracks mouse movement at window level and removes the listener on unmount', () => {
    const addEventListener = vi.spyOn(window, 'addEventListener')
    const removeEventListener = vi.spyOn(window, 'removeEventListener')

    const wrapper = mount(LoginGalaxyBackground)

    expect(addEventListener).toHaveBeenCalledWith('mousemove', expect.any(Function))
    wrapper.unmount()
    expect(removeEventListener).toHaveBeenCalledWith('mousemove', expect.any(Function))
  })

  it('limits GPU rendering cadence and skips rendering while the page is hidden', () => {
    const callbacks: FrameRequestCallback[] = []
    vi.stubGlobal('requestAnimationFrame', vi.fn((callback: FrameRequestCallback) => {
      callbacks.push(callback)
      return callbacks.length
    }))
    const hidden = vi.spyOn(document, 'hidden', 'get').mockReturnValue(false)

    const wrapper = mount(LoginGalaxyBackground, { props: { maxFps: 30 } })

    callbacks.shift()!(0)
    expect(renderGalaxy).toHaveBeenCalledTimes(1)

    callbacks.shift()!(16)
    expect(renderGalaxy).toHaveBeenCalledTimes(1)

    callbacks.shift()!(34)
    expect(renderGalaxy).toHaveBeenCalledTimes(2)

    hidden.mockReturnValue(true)
    callbacks.shift()!(68)
    expect(renderGalaxy).toHaveBeenCalledTimes(2)

    wrapper.unmount()
  })
})
