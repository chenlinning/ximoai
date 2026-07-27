<template>
  <div v-if="mode === 'loading'" class="min-h-screen bg-gray-50 dark:bg-dark-950" />
  <XimoAIHomeWorkspace v-else-if="mode === 'workspace'" :tabs="enabledTabs" />
  <HomeView v-else />
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ximoaiHomeAPI, type XimoAIHomeTab } from '@/api'
import { membershipAPI, type MembershipSummary } from '@/api/membership'
import { useAppStore, useAuthStore } from '@/stores'
import HomeView from '@/views/HomeView.vue'
import XimoAIHomeWorkspace from './XimoAIHomeWorkspace.vue'
import { resolveXimoAIHomeMode } from './homeState'
import { onMembershipUpdated } from '@/utils/membershipEvents'

const appStore = useAppStore()
const authStore = useAuthStore()
const tabs = ref<XimoAIHomeTab[]>([])
const membershipSummary = ref<MembershipSummary | null>(null)
const tabsLoaded = ref(false)
let stopMembershipUpdatedListener: (() => void) | null = null

const isDiamondMember = computed(() => membershipSummary.value?.level?.code?.toLowerCase() === 'diamond')

const enabledTabs = computed(() => tabs.value
  .filter((tab) => tab.enabled && (!tab.diamond_only || isDiamondMember.value))
  .sort((left, right) => left.sort_order - right.sort_order))

async function loadMembership() {
  if (!authStore.isAuthenticated) {
    membershipSummary.value = null
    return
  }
  try {
    membershipSummary.value = await membershipAPI.getCurrent()
  } catch {
    membershipSummary.value = null
  }
}

const mode = computed(() => resolveXimoAIHomeMode({
  settingsLoaded: appStore.publicSettingsLoaded,
  tabsLoaded: tabsLoaded.value,
  authenticated: authStore.isAuthenticated,
  homeContent: appStore.cachedPublicSettings?.home_content || '',
  enabledTabCount: enabledTabs.value.length
}))

onMounted(async () => {
  if (!appStore.publicSettingsLoaded) {
    await appStore.fetchPublicSettings()
  }
  try {
    const [homeTabs] = await Promise.all([ximoaiHomeAPI.list(), loadMembership()])
    tabs.value = homeTabs
  } catch {
    tabs.value = []
  } finally {
    tabsLoaded.value = true
  }
  stopMembershipUpdatedListener = onMembershipUpdated(loadMembership)
})

watch(() => authStore.isAuthenticated, loadMembership)

onBeforeUnmount(() => {
  stopMembershipUpdatedListener?.()
  stopMembershipUpdatedListener = null
})
</script>
