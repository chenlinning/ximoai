import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('GroupsView platform options', () => {
  it('uses enabled registry platforms once for create options and filters', () => {
    const source = readFileSync(resolve('src/views/admin/GroupsView.vue'), 'utf8')

    expect(source).toContain('platforms.value.filter((platform) => platform.enabled)')
    expect(source).not.toContain('adminAPI.platforms.list(true)')
    expect(source).not.toContain('{ value: "anthropic", label: "Anthropic" }')
    expect(source).toContain('...platformOptions.value')
  })
})
