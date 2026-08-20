import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('GroupsView platform options', () => {
  it('uses the fixed upstream platform list plus XimoAI built-ins', () => {
    const source = readFileSync(resolve('src/views/admin/GroupsView.vue'), 'utf8')
    const platforms = readFileSync(resolve('src/extensions/platforms/fixedPlatforms.ts'), 'utf8')

    expect(source).not.toContain('adminAPI.platforms')
    expect(source).toContain('fixedPlatforms.map')
    expect(platforms).toContain("slug: 'anthropic'")
    expect(platforms).toContain("slug: 'kimi'")
    expect(platforms).toContain("slug: 'grok-video'")
    expect(platforms).toContain("slug: 'openai-audio'")
    expect(platforms).toContain("slug: 'kling_audio'")
    expect(platforms).toContain("slug: 'volcengine-agent-plan'")
    expect(source).toContain('...platformOptions.value')
  })
})
