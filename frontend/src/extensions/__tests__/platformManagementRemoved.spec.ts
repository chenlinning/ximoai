import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('platform management removal', () => {
  it('does not expose an independent platform page or navigation entry', () => {
    expect(existsSync(resolve('src/views/admin/PlatformsView.vue'))).toBe(false)
    expect(existsSync(resolve('src/api/admin/platforms.ts'))).toBe(false)
    expect(existsSync(resolve('src/api/platforms.ts'))).toBe(false)

    const routes = readFileSync(resolve('src/extensions/routes.ts'), 'utf8')
    const navigation = readFileSync(resolve('src/extensions/navigation.ts'), 'utf8')
    expect(routes).not.toContain('/admin/platforms')
    expect(navigation).not.toContain('/admin/platforms')
  })
})
