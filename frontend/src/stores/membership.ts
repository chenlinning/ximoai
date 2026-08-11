import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

import { membershipAPI, type MembershipSummary } from '@/api/membership'
import { useAuthStore } from './auth'

export const useMembershipStore = defineStore('membership', () => {
  const authStore = useAuthStore()
  const summary = ref<MembershipSummary | null>(null)
  const loadedUserID = ref<number | null>(null)
  const inFlightByUser = new Map<number, Promise<MembershipSummary>>()

  function clear() {
    summary.value = null
    loadedUserID.value = null
  }

  async function fetch(force = false): Promise<MembershipSummary | null> {
    const userID = authStore.user?.id ?? null
    if (userID === null) {
      clear()
      return null
    }

    const inFlight = inFlightByUser.get(userID)
    if (inFlight) return inFlight
    if (!force && loadedUserID.value === userID) return summary.value

    const request = membershipAPI.getCurrent()
      .then((result) => {
        if (authStore.user?.id === userID) {
          summary.value = result
          loadedUserID.value = userID
        }
        return result
      })
      .finally(() => {
        if (inFlightByUser.get(userID) === request) {
          inFlightByUser.delete(userID)
        }
      })

    inFlightByUser.set(userID, request)
    return request
  }

  watch(
    () => authStore.user?.id ?? null,
    (userID) => {
      if (loadedUserID.value !== userID) clear()
    }
  )

  return {
    summary,
    loadedUserID,
    fetch,
    clear
  }
})
