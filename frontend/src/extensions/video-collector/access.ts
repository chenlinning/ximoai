import { readonly, ref } from 'vue'
import { membershipAPI, type MembershipSummary } from '@/api/membership'

const canUse = ref<boolean | undefined>(undefined)
let pending: Promise<boolean> | null = null

export function isActiveDiamondMembership(summary: MembershipSummary | null | undefined, now = new Date()): boolean {
  if (!summary?.level || summary.level.code !== 'diamond' || !summary.level.enabled) {
    return false
  }
  const startsAt = new Date(summary.starts_at)
  if (!Number.isFinite(startsAt.getTime()) || startsAt.getTime() > now.getTime()) {
    return false
  }
  if (summary.expires_at) {
    const expiresAt = new Date(summary.expires_at)
    if (!Number.isFinite(expiresAt.getTime()) || expiresAt.getTime() <= now.getTime()) {
      return false
    }
  }
  return true
}

export async function refreshVideoCollectorAccess(force = false): Promise<boolean> {
  if (pending && !force) {
    return pending
  }
  pending = membershipAPI.getCurrent()
    .then(summary => {
      canUse.value = isActiveDiamondMembership(summary)
      return canUse.value
    })
    .catch(() => {
      canUse.value = false
      return false
    })
    .finally(() => {
      pending = null
    })
  return pending
}

export function useVideoCollectorAccess() {
  return {
    canUseVideoCollector: readonly(canUse),
    refreshVideoCollectorAccess
  }
}
