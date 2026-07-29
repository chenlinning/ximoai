import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('Model Plaza route integration', () => {
  it('keeps one route and delegates public and authenticated views through the extension gate', () => {
    const router = readFileSync(resolve('src/router/index.ts'), 'utf8')
    const extensionRoutes = readFileSync(resolve('src/extensions/routes.ts'), 'utf8')
    const registrations = `${router}\n${extensionRoutes}`.match(/path:\s*['"]\/model-plaza['"]/g) || []

    expect(registrations).toHaveLength(1)
    expect(router).toContain("import('@/extensions/model-plaza/XimoAIModelPlazaGate.vue')")
    expect(router).toContain("to.path === '/model-plaza' && !authStore.isAuthenticated")
    expect(extensionRoutes).not.toContain("path: '/model-plaza'")
  })
})
