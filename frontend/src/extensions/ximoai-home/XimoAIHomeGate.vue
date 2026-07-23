<template>
  <div v-if="mode === 'loading'" class="min-h-screen bg-gray-50 dark:bg-dark-950" />
  <XimoAIHomeWorkspace v-else-if="mode === 'workspace'" :tabs="enabledTabs" />
  <HomeView v-else />
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ximoaiHomeAPI, type XimoAIHomeTab } from '@/api'
import { useAppStore, useAuthStore } from '@/stores'
import HomeView from '@/views/HomeView.vue'
import XimoAIHomeWorkspace from './XimoAIHomeWorkspace.vue'
import { resolveXimoAIHomeMode } from './homeState'

const appStore = useAppStore()
const authStore = useAuthStore()
const tabs = ref<XimoAIHomeTab[]>([])
const tabsLoaded = ref(false)

const enabledTabs = computed(() => tabs.value
  .filter((tab) => tab.enabled)
  .sort((left, right) => left.sort_order - right.sort_order))

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
    tabs.value = await ximoaiHomeAPI.list()
  } catch {
    tabs.value = []
  } finally {
    tabsLoaded.value = true
  }
})
</script>
