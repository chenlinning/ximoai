import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('XimoAI model plaza navigation', () => {
  it('keeps the custom entry isolated behind its own feature flag', () => {
    const sidebar = readFileSync(resolve('src/components/layout/AppSidebar.vue'), 'utf8')
    const flags = readFileSync(resolve('src/utils/featureFlags.ts'), 'utf8')

    expect(sidebar).toContain("path: '/ximoai-model-plaza'")
    expect(sidebar).toContain('FeatureFlags.ximoaiModelPlazaEntry')
    expect(flags).toContain("key: 'ximoai_model_plaza_entry_enabled'")
    expect(flags).toContain("mode: 'opt-out'")
  })
})
