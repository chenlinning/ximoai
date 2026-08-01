import { describe, expect, it } from 'vitest'

import { ximoAIRoutes } from '@/extensions/routes'

describe('desktop authorization route', () => {
  it('uses the existing authenticated browser session', () => {
    const route = ximoAIRoutes.find((entry) => entry.path === '/desktop/authorize')
    expect(route).toBeDefined()
    expect(route?.meta?.requiresAuth).toBe(true)
    expect(route?.meta?.requiresAdmin).toBe(false)
  })
})
