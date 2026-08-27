<template>
  <section class="border-t border-gray-100 pt-6 dark:border-dark-700">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h3 class="font-medium text-gray-900 dark:text-white">{{ t('ximoaiHome.settingsTitle') }}</h3>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('ximoaiHome.settingsHint') }}</p>
      </div>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="loading || tabs.length >= 24" @click="addTab">
        <Icon name="plus" size="sm" />
        {{ t('ximoaiHome.addTab') }}
      </button>
    </div>

    <div v-if="loading" class="py-8 text-center text-sm text-gray-500 dark:text-dark-300">
      {{ t('common.loading') }}
    </div>
    <div v-else-if="tabs.length === 0" class="mt-4 rounded-lg border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-300">
      {{ t('ximoaiHome.emptyTabs') }}
    </div>
    <div v-else class="mt-4 divide-y divide-gray-100 border-y border-gray-100 dark:divide-dark-700 dark:border-dark-700">
      <div v-for="(tab, index) in tabs" :key="tab.editorKey" class="space-y-4 py-5">
        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('ximoaiHome.label') }}</label>
            <input v-model="tab.label" type="text" maxlength="48" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('ximoaiHome.url') }}</label>
            <input v-model="tab.url" type="url" class="input" placeholder="https://" />
          </div>
        </div>

        <div class="flex flex-wrap items-center justify-between gap-4">
          <div class="flex flex-wrap items-center gap-5">
            <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <Toggle v-model="tab.enabled" />
              {{ t('common.enabled') }}
            </label>
            <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <Toggle v-model="tab.workbench_sso" />
              {{ t('ximoaiHome.workbenchSSO') }}
            </label>
            <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <Toggle v-model="tab.diamond_only" />
              {{ t('ximoaiHome.diamondOnly') }}
            </label>
          </div>
          <div class="flex items-center gap-1">
            <button type="button" class="btn-ghost btn-icon" :disabled="index === 0" :title="t('ximoaiHome.moveUp')" @click="moveTab(index, -1)">
              <Icon name="chevronDown" size="sm" class="rotate-180" />
            </button>
            <button type="button" class="btn-ghost btn-icon" :disabled="index === tabs.length - 1" :title="t('ximoaiHome.moveDown')" @click="moveTab(index, 1)">
              <Icon name="chevronDown" size="sm" />
            </button>
            <button type="button" class="btn-ghost btn-icon text-red-600 dark:text-red-400" :title="t('common.delete')" @click="removeTab(index)">
              <Icon name="trash" size="sm" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <div class="mt-5 flex justify-end">
      <button type="button" class="btn btn-primary" :disabled="loading || saving" @click="saveTabs">
        <Icon name="check" size="sm" />
        {{ saving ? t('common.saving') : t('common.save') }}
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ximoaiHomeAPI, type XimoAIHomeTab, type XimoAIHomeTabInput } from '@/api'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'

interface EditorTab extends Omit<XimoAIHomeTabInput, 'workbench_sso' | 'diamond_only'> {
  workbench_sso: boolean
  diamond_only: boolean
  editorKey: string
}

const { t } = useI18n()
const appStore = useAppStore()
const tabs = ref<EditorTab[]>([])
const loading = ref(true)
const saving = ref(false)

function toEditorTab(tab: XimoAIHomeTab): EditorTab {
  return {
    ...tab,
    workbench_sso: !!tab.workbench_sso,
    diamond_only: !!tab.diamond_only,
    editorKey: tab.id
  }
}

function addTab() {
  tabs.value.push({
    id: '',
    label: '',
    url: '',
    enabled: true,
    workbench_sso: false,
    diamond_only: false,
    editorKey: crypto.randomUUID()
  })
}

function removeTab(index: number) {
  tabs.value.splice(index, 1)
}

function moveTab(index: number, offset: number) {
  const target = index + offset
  if (target < 0 || target >= tabs.value.length) return
  const [tab] = tabs.value.splice(index, 1)
  tabs.value.splice(target, 0, tab)
}

async function loadTabs() {
  loading.value = true
  try {
    tabs.value = (await ximoaiHomeAPI.listAdmin()).map(toEditorTab)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('ximoaiHome.loadSettingsFailed')))
  } finally {
    loading.value = false
  }
}

async function saveTabs() {
  saving.value = true
  try {
    const payload = tabs.value.map(({ editorKey: _editorKey, ...tab }, sort_order) => ({ ...tab, sort_order }))
    tabs.value = (await ximoaiHomeAPI.update(payload)).map(toEditorTab)
    appStore.showSuccess(t('ximoaiHome.saved'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('ximoaiHome.saveFailed')))
  } finally {
    saving.value = false
  }
}

onMounted(loadTabs)
</script>
