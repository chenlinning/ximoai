import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { fixedPlatforms } from '@/extensions/platforms/fixedPlatforms'

describe('Composite channel platform options', () => {
  it('includes the CN concrete providers for pricing and model mapping', () => {
    const source = readFileSync(resolve('src/views/admin/ChannelsView.vue'), 'utf8')
    const platformSlugs = fixedPlatforms.map((platform) => platform.slug)

    expect(source).toContain('const platformOrder = fixedPlatforms.map')
    expect(source).toContain('const compositePlatforms: GroupPlatform[] = [...platformOrder]')
    expect(platformSlugs).toEqual(expect.arrayContaining([
      'kimi', 'zhipu', 'deepseek',
      'grok-video', 'openai-audio', 'kling_audio', 'volcengine-agent-plan'
    ]))
  })
})
