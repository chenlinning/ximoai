import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('PlatformsView custom protocol capabilities', () => {
  it('uses the platform capability editor without overwriting saved selections', () => {
    const source = readFileSync(resolve('src/views/admin/PlatformsView.vue'), 'utf8')

    expect(source).toContain('v-model="form.capabilities"')
    expect(source).toContain('normalizePlatformCapabilities(platform.protocol, platform.capabilities || [], platform.kind)')
    expect(source).not.toContain('const capabilitiesForProtocol =')
    expect(source).not.toContain('capabilities: editingPlatform.value?.builtin')
  })

  it('keeps core built-in platform slugs immutable', () => {
    const source = readFileSync(resolve('src/views/admin/PlatformsView.vue'), 'utf8')

    expect(source).toContain(':disabled="isCoreBuiltinPlatform(editingPlatform)"')
  })

  it('manages built-in platforms without custom create or delete actions', () => {
    const source = readFileSync(resolve('src/views/admin/PlatformsView.vue'), 'utf8')

    expect(source).not.toContain('openCreateDialog')
    expect(source).not.toContain('openDeleteDialog')
    expect(source).not.toContain('adminAPI.platforms.create')
    expect(source).not.toContain('adminAPI.platforms.remove')
    expect(source).toContain('adminAPI.platforms.update')
  })
})
