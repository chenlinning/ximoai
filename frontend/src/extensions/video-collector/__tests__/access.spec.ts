import { describe, expect, it } from 'vitest'
import { isActiveDiamondMembership } from '../access'

const now = new Date('2026-07-25T12:00:00Z')

function summary(overrides: Record<string, unknown> = {}) {
  return {
    level: {
      id: 5,
      name: 'Diamond',
      code: 'diamond',
      color: '#0ea5e9',
      discount_rate: 1,
      enabled: true,
      is_default: false,
      sort_order: 50,
      description: '',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z'
    },
    starts_at: '2026-07-01T00:00:00Z',
    expires_at: '2026-08-01T00:00:00Z',
    levels: [],
    groups: [],
    managed_keys: [],
    ...overrides
  }
}

describe('isActiveDiamondMembership', () => {
  it('accepts an enabled active diamond membership', () => {
    expect(isActiveDiamondMembership(summary(), now)).toBe(true)
  })

  it('rejects other, disabled, future, and expired memberships', () => {
    expect(isActiveDiamondMembership(summary({ level: { ...summary().level, code: 'gold' } }), now)).toBe(false)
    expect(isActiveDiamondMembership(summary({ level: { ...summary().level, enabled: false } }), now)).toBe(false)
    expect(isActiveDiamondMembership(summary({ starts_at: '2026-07-26T00:00:00Z' }), now)).toBe(false)
    expect(isActiveDiamondMembership(summary({ expires_at: '2026-07-25T12:00:00Z' }), now)).toBe(false)
  })
})
