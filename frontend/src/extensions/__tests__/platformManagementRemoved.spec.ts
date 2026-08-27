import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('platform management removal', () => {
  it('does not add an independent platform route through XimoAI extensions', () => {
    const routes = readFileSync(resolve('src/extensions/routes.ts'), 'utf8')
    const navigation = readFileSync(resolve('src/extensions/navigation.ts'), 'utf8')
    expect(routes).not.toContain('/admin/platforms')
    expect(navigation).not.toContain('/admin/platforms')
  })
})
