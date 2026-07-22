import { beforeEach, describe, expect, it, vi } from 'vitest'

const post = vi.hoisted(() => vi.fn())

vi.mock('../../client', () => ({
  apiClient: { post }
}))

import { uploadPackage } from '../ximodeskUpdate'

describe('ximodeskUpdate uploadPackage', () => {
  beforeEach(() => {
    post.mockReset()
    post.mockResolvedValue({ data: { release: {}, config: { enabled: true, apps: [], releases: [] } } })
  })

  it('has no fixed request timeout and forwards upload progress', async () => {
    const onProgress = vi.fn()
    const payload = new FormData()

    await uploadPackage(payload, { onProgress })

    expect(post).toHaveBeenCalledWith(
      '/admin/ximoapp/update/packages',
      payload,
      expect.objectContaining({
        timeout: 0,
        onUploadProgress: onProgress
      })
    )
  })
})
