import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('PlatformsView custom protocol capabilities', () => {
  it('only advertises OpenAI-compatible endpoints that the public gateway exposes', () => {
    const source = readFileSync(resolve('src/views/admin/PlatformsView.vue'), 'utf8')

    expect(source).toContain("return ['responses', 'chat_completions', 'images']")
    expect(source).not.toContain("return ['responses', 'chat_completions', 'images', 'videos', 'audio', 'realtime']")
  })
})
