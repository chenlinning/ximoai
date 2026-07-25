import { describe, expect, it, vi } from 'vitest'
import { streamResponseToWriter } from '../download'

describe('streamResponseToWriter', () => {
  it('streams chunks and reports browser-save progress', async () => {
    const writes: Uint8Array[] = []
    const writer = {
      write: vi.fn(async (chunk: Uint8Array) => writes.push(chunk)),
      close: vi.fn(async () => undefined),
      abort: vi.fn(async () => undefined)
    }
    const response = new Response(new TextEncoder().encode('video-data'), {
      headers: { 'Content-Length': '10' }
    })
    const progress: number[] = []

    await streamResponseToWriter(response, writer, value => progress.push(value))

    expect(writer.write).toHaveBeenCalled()
    expect(writer.close).toHaveBeenCalledOnce()
    expect(new TextDecoder().decode(writes[0])).toBe('video-data')
    expect(progress.at(-1)).toBe(100)
  })
})
