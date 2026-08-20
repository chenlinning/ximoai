import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { CONCRETE_PLATFORM_OPTIONS } from '@/constants/platforms'
import { fixedPlatforms } from '@/extensions/platforms/fixedPlatforms'

function readSource(path: string): string {
  return readFileSync(resolve(path), 'utf8')
}

describe('admin platform filters', () => {
  it('uses the group platform catalog on the subscriptions page', () => {
    const source = readSource('src/views/admin/SubscriptionsView.vue')
    expect(source).toContain("import { GROUP_PLATFORM_OPTIONS } from '@/constants/platforms'")
    expect(source).toMatch(/const platformFilterOptions[\s\S]*?\.\.\.GROUP_PLATFORM_OPTIONS/)
  })

  it('extends the upstream catalog for the groups and account pages', () => {
    const source = readSource('src/views/admin/GroupsView.vue')
    const accountFilters = readSource('src/components/admin/account/AccountTableFilters.vue')
    const fixedPlatformSlugs = fixedPlatforms.map((platform) => platform.slug)

    expect(source).toContain("import { fixedPlatforms")
    expect(source).toContain('fixedPlatforms.map')
    expect(accountFilters).toContain("import { fixedPlatforms } from '@/extensions/platforms/fixedPlatforms'")
    expect(accountFilters).toContain('...fixedPlatforms.map')
    expect(fixedPlatformSlugs).toEqual(expect.arrayContaining(
      CONCRETE_PLATFORM_OPTIONS.map((option) => option.value)
    ))
    expect(fixedPlatformSlugs).toEqual(expect.arrayContaining([
      'grok-video', 'openai-audio', 'kling_audio', 'volcengine-agent-plan'
    ]))
  })

  it('uses the concrete upstream catalog on unextended selectors', () => {
    for (const path of [
      'src/components/admin/ErrorPassthroughRulesModal.vue',
      'src/views/admin/ops/components/OpsDashboardHeader.vue'
    ]) {
      const source = readSource(path)
      expect(source).toContain("import { CONCRETE_PLATFORM_OPTIONS } from '@/constants/platforms'")
      expect(source).toMatch(/platformOptions\s*=.*CONCRETE_PLATFORM_OPTIONS|pOpts.*\.\.\.CONCRETE_PLATFORM_OPTIONS/s)
    }
  })
})
