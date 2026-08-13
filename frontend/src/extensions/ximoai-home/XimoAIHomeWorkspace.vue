<template>
  <main
    class="home-workspace-shell relative flex min-h-screen flex-col overflow-x-hidden bg-gray-50 dark:bg-dark-950"
  >
    <AppHeader
      variant="workspace"
      show-theme-toggle
      show-console-button
    >
      <template #left>
        <div class="home-workspace-tabs-scroll w-full min-w-0 max-w-full overflow-x-auto">
          <nav class="home-workspace-tabs flex w-max min-w-full flex-nowrap items-center justify-center gap-1" :aria-label="t('ximoaiHome.tabs')">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              type="button"
              class="shrink-0 whitespace-nowrap rounded-md border px-2 py-1.5 text-xs font-medium transition-colors sm:px-3 sm:py-2 sm:text-sm"
              :class="activeTabID === tab.id
                ? 'border-primary-500 bg-primary-500 text-white shadow-sm hover:bg-primary-600 hover:shadow-md'
                : 'border-gray-300 text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:border-dark-600 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white'"
              @click="activateTab(tab.id)"
            >
              {{ tab.label }}
            </button>
          </nav>
        </div>
      </template>
    </AppHeader>

    <section class="relative min-h-0 flex-1 overflow-hidden bg-white dark:bg-dark-950">
      <div
        v-for="tab in tabs"
        v-show="activeTabID === tab.id"
        :key="tab.id"
        class="absolute inset-0"
      >
        <div v-if="loadingTabIDs.has(tab.id)" class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-dark-300">
          {{ t('common.loading') }}
        </div>
        <div v-else-if="tabErrors[tab.id]" class="flex h-full flex-col items-center justify-center gap-4 px-6 text-center">
          <p class="text-sm text-red-600 dark:text-red-400">{{ tabErrors[tab.id] }}</p>
          <button type="button" class="btn btn-primary" @click="retryTab(tab.id)">
            <Icon name="refresh" size="sm" />
            {{ t('ximoaiHome.retry') }}
          </button>
        </div>
        <iframe
          v-else-if="loadedTabIDs.has(tab.id) && frameURLs[tab.id]"
          :ref="(element) => setFrameRef(tab.id, element)"
          :src="frameURLs[tab.id]"
          :title="tab.label"
          class="absolute inset-0 block h-full w-full border-0"
          allow="clipboard-read; clipboard-write; fullscreen"
          allowfullscreen
          @load="handleFrameLoad(tab.id)"
        />
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { workbenchAPI, type XimoAIHomeTab } from '@/api'
import Icon from '@/components/icons/Icon.vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import {
  addLoadedTab,
  buildPreferencesMessage,
  frameOrigin,
  isPreferencesReadyMessage,
  resolveDefaultXimoAIHomeTab
} from './homeState'

const props = defineProps<{ tabs: XimoAIHomeTab[] }>()
const { t, locale } = useI18n()

const activeTabID = ref('')
const loadedTabIDs = ref<Set<string>>(new Set())
const loadingTabIDs = ref<Set<string>>(new Set())
const frameURLs = reactive<Record<string, string>>({})
const tabErrors = reactive<Record<string, string>>({})
const frameRefs = new Map<string, HTMLIFrameElement>()
const theme = ref<'light' | 'dark'>(document.documentElement.classList.contains('dark') ? 'dark' : 'light')
let themeObserver: MutationObserver | null = null

function setLoading(tabID: string, loading: boolean) {
  const next = new Set(loadingTabIDs.value)
  if (loading) next.add(tabID)
  else next.delete(tabID)
  loadingTabIDs.value = next
}

async function ensureTabLoaded(tab: XimoAIHomeTab) {
  if (loadedTabIDs.value.has(tab.id) || loadingTabIDs.value.has(tab.id)) return

  setLoading(tab.id, true)
  delete tabErrors[tab.id]
  try {
    frameURLs[tab.id] = tab.workbench_sso
      ? (await workbenchAPI.createSSOTicket(tab.url)).entry_url
      : tab.url
    loadedTabIDs.value = addLoadedTab(loadedTabIDs.value, tab.id)
  } catch {
    tabErrors[tab.id] = t('ximoaiHome.loadFailed')
  } finally {
    setLoading(tab.id, false)
  }
}

function activateTab(tabID: string) {
  const tab = props.tabs.find((item) => item.id === tabID)
  if (!tab) return
  activeTabID.value = tab.id
  void ensureTabLoaded(tab)
}

function retryTab(tabID: string) {
  const tab = props.tabs.find((item) => item.id === tabID)
  if (!tab) return
  delete frameURLs[tabID]
  loadedTabIDs.value = new Set([...loadedTabIDs.value].filter((id) => id !== tabID))
  void ensureTabLoaded(tab)
}

function setFrameRef(tabID: string, element: unknown) {
  if (element instanceof HTMLIFrameElement) frameRefs.set(tabID, element)
  else frameRefs.delete(tabID)
}

function postPreferences(tab: XimoAIHomeTab) {
  const frame = frameRefs.get(tab.id)
  const origin = frameOrigin(tab.url)
  if (!frame?.contentWindow || !origin) return
  frame.contentWindow.postMessage(buildPreferencesMessage(theme.value, locale.value), origin)
}

function broadcastPreferences() {
  props.tabs.forEach((tab) => {
    if (loadedTabIDs.value.has(tab.id)) postPreferences(tab)
  })
}

function handleFrameLoad(tabID: string) {
  const tab = props.tabs.find((item) => item.id === tabID)
  if (tab) postPreferences(tab)
}

function handleMessage(event: MessageEvent) {
  for (const tab of props.tabs) {
    const frameWindow = frameRefs.get(tab.id)?.contentWindow
    if (frameWindow && isPreferencesReadyMessage(event, tab.url, frameWindow)) {
      if (tab.workbench_sso && frameURLs[tab.id] !== tab.url) {
        frameURLs[tab.id] = tab.url
        return
      }
      postPreferences(tab)
      return
    }
  }
}

watch([theme, locale], broadcastPreferences)

watch(() => props.tabs, (nextTabs) => {
  if (activeTabID.value && !nextTabs.some((tab) => tab.id === activeTabID.value)) {
    const defaultTab = resolveDefaultXimoAIHomeTab(nextTabs)
    activeTabID.value = defaultTab?.id || ''
    if (defaultTab) void ensureTabLoaded(defaultTab)
  }
}, { deep: true })

onMounted(() => {
  themeObserver = new MutationObserver(() => {
    theme.value = document.documentElement.classList.contains('dark') ? 'dark' : 'light'
  })
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  const defaultTab = resolveDefaultXimoAIHomeTab(props.tabs)
  activeTabID.value = defaultTab?.id || ''
  if (defaultTab) void ensureTabLoaded(defaultTab)
  window.addEventListener('message', handleMessage)
})

onBeforeUnmount(() => {
  themeObserver?.disconnect()
  window.removeEventListener('message', handleMessage)
})
</script>

<style scoped>
.home-workspace-shell {
  height: 100vh;
  min-height: 0;
  overflow: hidden;
}

@supports (height: 100dvh) {
  .home-workspace-shell {
    height: 100dvh;
  }
}

.home-workspace-tabs-scroll {
  overscroll-behavior-inline: contain;
  scrollbar-width: none;
}

.home-workspace-tabs-scroll::-webkit-scrollbar {
  display: none;
}

</style>
