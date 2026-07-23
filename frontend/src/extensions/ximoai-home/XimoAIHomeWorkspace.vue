<template>
  <main class="relative min-h-screen overflow-hidden bg-gray-50 dark:bg-dark-950">
    <template v-if="!activeTabID">
      <LoginGalaxyBackground class="fixed inset-0 z-0" />
      <AppHeader
        variant="floating"
        show-theme-toggle
        show-console-button
      />

      <section class="relative z-10 flex min-h-screen items-center justify-center px-4 py-24 sm:px-6">
        <div class="w-full max-w-7xl">
          <div class="mb-8 text-center">
            <img
              v-if="siteLogo"
              :src="siteLogo"
              :alt="siteName"
              class="mx-auto mb-4 h-16 w-16 object-contain"
            />
            <h1 class="text-3xl font-semibold text-gray-950 dark:text-white sm:text-4xl">
              {{ siteName }}
            </h1>
          </div>

          <div class="home-entry-grid">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              type="button"
              class="group overflow-hidden rounded-lg border border-white/60 bg-white/90 text-left shadow-lg backdrop-blur transition hover:-translate-y-1 hover:border-primary-300 hover:shadow-xl focus:outline-none focus:ring-2 focus:ring-primary-500 dark:border-dark-700/80 dark:bg-dark-900/90 dark:hover:border-primary-600"
              @click="activateTab(tab.id)"
            >
              <div class="aspect-[16/9] overflow-hidden bg-gray-100 dark:bg-dark-800">
                <img
                  v-if="tab.cover_url"
                  :src="tab.cover_url"
                  :alt="tab.label"
                  class="h-full w-full object-cover transition duration-300 group-hover:scale-[1.03]"
                />
                <div
                  v-else
                  class="flex h-full items-center justify-center bg-primary-50 text-5xl font-semibold text-primary-600 dark:bg-primary-900/20 dark:text-primary-300"
                >
                  {{ tab.label.slice(0, 1).toUpperCase() }}
                </div>
              </div>
              <div class="flex h-14 items-center justify-between gap-3 px-4">
                <span class="truncate text-base font-semibold text-gray-900 dark:text-white">
                  {{ tab.label }}
                </span>
                <Icon name="chevronRight" size="sm" class="shrink-0 text-gray-400 group-hover:text-primary-500" />
              </div>
            </button>
          </div>
        </div>
      </section>
    </template>

    <template v-else>
      <AppHeader
        variant="workspace"
        show-theme-toggle
        show-console-button
      >
        <template #left>
          <div class="flex min-w-0 flex-1 items-center gap-2">
            <nav class="flex min-w-0 flex-1 items-center gap-1 overflow-x-auto" :aria-label="t('ximoaiHome.tabs')">
              <button
                v-for="tab in tabs"
                :key="tab.id"
                type="button"
                class="shrink-0 rounded-md px-3 py-2 text-sm font-medium transition-colors"
                :class="activeTabID === tab.id
                  ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
                  : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white'"
                @click="activateTab(tab.id)"
              >
                {{ tab.label }}
              </button>
            </nav>
          </div>
        </template>
      </AppHeader>

      <section class="relative h-[calc(100vh-4rem)] bg-white dark:bg-dark-950">
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
            class="h-full w-full border-0"
            allow="clipboard-read; clipboard-write; fullscreen"
            allowfullscreen
            @load="handleFrameLoad(tab.id)"
          />
        </div>
      </section>
    </template>
  </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { workbenchAPI, type XimoAIHomeTab } from '@/api'
import LoginGalaxyBackground from '@/components/auth/LoginGalaxyBackground.vue'
import Icon from '@/components/icons/Icon.vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import {
  addLoadedTab,
  buildPreferencesMessage,
  frameOrigin,
  isPreferencesReadyMessage
} from './homeState'

const props = defineProps<{ tabs: XimoAIHomeTab[] }>()
const appStore = useAppStore()
const { t, locale } = useI18n()

const activeTabID = ref('')
const loadedTabIDs = ref<Set<string>>(new Set())
const loadingTabIDs = ref<Set<string>>(new Set())
const frameURLs = reactive<Record<string, string>>({})
const tabErrors = reactive<Record<string, string>>({})
const frameRefs = new Map<string, HTMLIFrameElement>()
const theme = ref<'light' | 'dark'>(document.documentElement.classList.contains('dark') ? 'dark' : 'light')
let themeObserver: MutationObserver | null = null

const siteName = computed(() => appStore.siteName || 'XimoAI')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))

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
      postPreferences(tab)
      return
    }
  }
}

watch([theme, locale], broadcastPreferences)

onMounted(() => {
  themeObserver = new MutationObserver(() => {
    theme.value = document.documentElement.classList.contains('dark') ? 'dark' : 'light'
  })
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  window.addEventListener('message', handleMessage)
})

onBeforeUnmount(() => {
  themeObserver?.disconnect()
  window.removeEventListener('message', handleMessage)
})
</script>

<style scoped>
.home-entry-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 260px), 320px));
  justify-content: center;
  gap: 1rem;
}
</style>
