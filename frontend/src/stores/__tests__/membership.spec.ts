import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { MembershipSummary } from '@/api/membership'
import { useMembershipStore } from '@/stores/membership'

const getCurrentMembership = vi.hoisted(() => vi.fn())
const authStore = vi.hoisted(() => ({
  user: null as { id: number } | null
}))

vi.mock('@/api/membership', () => ({
  membershipAPI: { getCurrent: getCurrentMembership }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore
}))

function createSummary(code: string): MembershipSummary {
  return {
    level: {
      id: 1,
      name: code,
      code,
      color: '#000000',
      discount_rate: 1,
      enabled: true,
      is_default: false,
      sort_order: 0,
      description: '',
      created_at: '',
      updated_at: ''
    },
    starts_at: '',
    expires_at: null,
    levels: [],
    groups: [],
    managed_keys: []
  }
}

function createDeferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((resolvePromise) => {
    resolve = resolvePromise
  })
  return { promise, resolve }
}

describe('useMembershipStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    authStore.user = { id: 1 }
    getCurrentMembership.mockReset()
  })

  it('reuses one in-flight request and caches the result for the current user', async () => {
    const deferred = createDeferred<MembershipSummary>()
    getCurrentMembership.mockReturnValue(deferred.promise)
    const store = useMembershipStore()

    const first = store.fetch()
    const second = store.fetch()
    expect(getCurrentMembership).toHaveBeenCalledTimes(1)

    deferred.resolve(createSummary('diamond'))
    await expect(first).resolves.toMatchObject({ level: { code: 'diamond' } })
    await expect(second).resolves.toMatchObject({ level: { code: 'diamond' } })
    await expect(store.fetch()).resolves.toMatchObject({ level: { code: 'diamond' } })
    expect(getCurrentMembership).toHaveBeenCalledTimes(1)
  })

  it('does not expose a completed request after the authenticated user changes', async () => {
    const firstUser = createDeferred<MembershipSummary>()
    getCurrentMembership
      .mockReturnValueOnce(firstUser.promise)
      .mockResolvedValueOnce(createSummary('platinum'))
    const store = useMembershipStore()

    const staleRequest = store.fetch()
    authStore.user = { id: 2 }
    const currentRequest = store.fetch()
    firstUser.resolve(createSummary('diamond'))

    await staleRequest
    await currentRequest
    expect(getCurrentMembership).toHaveBeenCalledTimes(2)
    expect(store.summary?.level?.code).toBe('platinum')
    expect(store.loadedUserID).toBe(2)
  })

  it('clears cached membership without a request after logout', async () => {
    getCurrentMembership.mockResolvedValue(createSummary('diamond'))
    const store = useMembershipStore()
    await store.fetch()

    authStore.user = null
    await expect(store.fetch()).resolves.toBeNull()
    expect(store.summary).toBeNull()
    expect(store.loadedUserID).toBeNull()
    expect(getCurrentMembership).toHaveBeenCalledTimes(1)
  })

  it('refreshes once when forced while still deduplicating concurrent callers', async () => {
    getCurrentMembership.mockResolvedValueOnce(createSummary('platinum'))
    const store = useMembershipStore()
    await store.fetch()

    const deferred = createDeferred<MembershipSummary>()
    getCurrentMembership.mockReturnValueOnce(deferred.promise)
    const first = store.fetch(true)
    const second = store.fetch(true)
    expect(getCurrentMembership).toHaveBeenCalledTimes(2)

    deferred.resolve(createSummary('diamond'))
    await Promise.all([first, second])
    expect(store.summary?.level?.code).toBe('diamond')
  })
})
