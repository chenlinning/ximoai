import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('Model Plaza route integration', () => {
  it('keeps upstream and XimoAI model plaza routes separate', () => {
    const router = readFileSync(resolve('src/router/index.ts'), 'utf8')
    const extensionRoutes = readFileSync(resolve('src/extensions/routes.ts'), 'utf8')
    const upstreamRegistrations = router.match(/path:\s*['"]\/model-plaza['"]/g) || []
    const customRegistrations = extensionRoutes.match(/path:\s*['"]\/ximoai-model-plaza['"]/g) || []

    expect(upstreamRegistrations).toHaveLength(1)
    expect(customRegistrations).toHaveLength(1)
    expect(router).toContain("component: () => import('@/views/ModelPlazaView.vue')")
    expect(extensionRoutes).toContain("component: () => import('@/extensions/model-plaza/ModelPlazaPage.vue')")
    expect(extensionRoutes).toContain("name: 'XimoAIModelPlaza'")
  })
})
