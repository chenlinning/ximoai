<template>
  <AppLayout>
    <div class="mx-auto flex h-full w-full max-w-7xl flex-col gap-4 p-4 sm:p-6">
      <div class="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
            {{ t('admin.platforms.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.platforms.description') }}
          </p>
        </div>
        <button class="btn btn-primary" @click="openCreateDialog">
          <Icon name="plus" size="sm" class="mr-2" />
          {{ t('admin.platforms.create') }}
        </button>
      </div>

      <div class="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-800/40 dark:bg-amber-900/20 dark:text-amber-200">
        {{ t('admin.platforms.openAICompatibleOnly') }}
      </div>

      <div class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div v-if="loading" class="flex items-center justify-center py-16 text-gray-500 dark:text-gray-400">
          <Icon name="refresh" size="md" class="mr-2 animate-spin" />
          {{ t('common.loading') }}
        </div>
        <div v-else-if="platforms.length === 0" class="py-16 text-center text-gray-500 dark:text-gray-400">
          {{ t('admin.platforms.noPlatforms') }}
        </div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-700/60">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
                  {{ t('admin.platforms.displayName') }}
                </th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
                  {{ t('admin.platforms.slug') }}
                </th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
                  {{ t('admin.platforms.protocol') }}
                </th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
                  {{ t('admin.platforms.baseUrl') }}
                </th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
                  {{ t('admin.platforms.enabled') }}
                </th>
                <th class="px-4 py-3 text-right text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">
                  {{ t('admin.platforms.actions') }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="platform in platforms" :key="platform.slug" class="hover:bg-gray-50 dark:hover:bg-dark-700/50">
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2">
                    <span
                      class="h-3 w-3 rounded-full border"
                      :style="{ backgroundColor: displayColor(platform), borderColor: displayColor(platform) }"
                    />
                    <span
                      class="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium"
                      :style="platformBadgeStyle(displayColor(platform))"
                    >
                      {{ platform.display_name }}
                    </span>
                    <span class="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-600 dark:text-gray-300">
                      {{ platform.builtin ? t('admin.platforms.builtin') : t('admin.platforms.custom') }}
                    </span>
                  </div>
                </td>
                <td class="px-4 py-3 font-mono text-sm text-gray-700 dark:text-gray-300">
                  {{ platform.slug }}
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                  {{ protocolLabel(platform.protocol) }}
                </td>
                <td class="max-w-md truncate px-4 py-3 font-mono text-sm text-gray-500 dark:text-gray-400" :title="platform.base_url">
                  {{ platform.base_url || '-' }}
                </td>
                <td class="px-4 py-3">
                  <button
                    type="button"
                    class="relative inline-flex h-6 w-11 rounded-full border-2 border-transparent transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 dark:focus:ring-offset-dark-800"
                    :class="platform.enabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'"
                    @click="toggleEnabled(platform)"
                  >
                    <span
                      class="inline-block h-5 w-5 rounded-full bg-white shadow transition-transform"
                      :class="platform.enabled ? 'translate-x-5' : 'translate-x-0'"
                    />
                  </button>
                </td>
                <td class="px-4 py-3">
                  <div class="flex justify-end gap-2">
                    <button class="btn btn-secondary px-3 py-1.5 text-xs" @click="openEditDialog(platform)">
                      <Icon name="edit" size="xs" class="mr-1" />
                      {{ t('common.edit') }}
                    </button>
                    <button
                      class="btn btn-danger px-3 py-1.5 text-xs"
                      :disabled="platform.builtin"
                      @click="openDeleteDialog(platform)"
                    >
                      <Icon name="trash" size="xs" class="mr-1" />
                      {{ t('common.delete') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <BaseDialog
      :show="showDialog"
      :title="editingPlatform ? t('admin.platforms.edit') : t('admin.platforms.create')"
      width="normal"
      @close="closeDialog"
    >
      <form id="platform-form" class="space-y-4" @submit.prevent="submitPlatform">
        <div v-if="editingPlatform?.builtin" class="rounded-lg border border-blue-200 bg-blue-50 px-3 py-2 text-sm text-blue-800 dark:border-blue-800/40 dark:bg-blue-900/20 dark:text-blue-200">
          {{ t('admin.platforms.readOnlyBuiltin') }}
        </div>
        <div>
          <label class="input-label">{{ t('admin.platforms.slug') }}</label>
          <input v-model.trim="form.slug" class="input font-mono" :disabled="!!editingPlatform" required />
          <p v-if="!editingPlatform" class="input-hint">{{ t('admin.platforms.slugHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.platforms.displayName') }}</label>
          <input v-model.trim="form.display_name" class="input" required />
        </div>
        <div>
          <label class="input-label">{{ t('admin.platforms.protocol') }}</label>
          <input
            v-if="editingPlatform?.builtin"
            :value="protocolLabel(form.protocol)"
            class="input"
            disabled
          />
          <select v-else v-model="form.protocol" class="input" required>
            <option v-for="option in customProtocolOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </div>
        <div>
          <label class="input-label">{{ t('admin.platforms.baseUrl') }}</label>
          <input
            v-model.trim="form.base_url"
            class="input font-mono"
            :disabled="editingPlatform?.builtin"
            :placeholder="baseUrlPlaceholder"
            required
          />
        </div>
        <div class="grid grid-cols-[1fr_auto] items-end gap-3">
          <div>
            <label class="input-label">{{ t('admin.platforms.color') }}</label>
            <input v-model.trim="form.color" class="input font-mono" placeholder="#10A37F" />
          </div>
          <input v-model="form.color" type="color" class="h-10 w-12 rounded border border-gray-300 bg-transparent p-1 dark:border-dark-600" />
        </div>
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input v-model="form.enabled" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600" />
          {{ t('admin.platforms.enabled') }}
        </label>
      </form>

      <template #footer>
        <button class="btn btn-secondary" type="button" @click="closeDialog">
          {{ t('common.cancel') }}
        </button>
        <button class="btn btn-primary" form="platform-form" type="submit" :disabled="saving">
          <Icon v-if="saving" name="refresh" size="sm" class="mr-2 animate-spin" />
          {{ t('common.save') }}
        </button>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('common.delete')"
      :message="t('admin.platforms.deleteConfirm', { name: deletingPlatform?.display_name || deletingPlatform?.slug })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="deletePlatform"
      @cancel="showDeleteDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { Platform } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { customPlatformFallbackColor, platformBadgeStyle } from '@/utils/platformColors'

const { t } = useI18n()
const appStore = useAppStore()

const platforms = ref<Platform[]>([])
const loading = ref(false)
const saving = ref(false)
const showDialog = ref(false)
const showDeleteDialog = ref(false)
const editingPlatform = ref<Platform | null>(null)
const deletingPlatform = ref<Platform | null>(null)

const form = reactive({
  slug: '',
  display_name: '',
  protocol: 'openai_compatible',
  base_url: '',
  color: '#64748B',
  enabled: true,
})

const customProtocolOptions = computed(() => [
  { value: 'openai_compatible', label: t('admin.platforms.protocolOpenAICompatible') },
  { value: 'anthropic', label: t('admin.platforms.protocolAnthropic') },
  { value: 'gemini', label: t('admin.platforms.protocolGemini') },
])

const baseUrlPlaceholder = computed(() => {
  if (form.protocol === 'gemini') return 'https://generativelanguage.googleapis.com'
  if (form.protocol === 'anthropic') return 'https://api.anthropic.com'
  return 'https://api.example.com/v1'
})

const loadPlatforms = async () => {
  loading.value = true
  try {
    platforms.value = await adminAPI.platforms.list(true)
  } catch (error: any) {
    appStore.showError(error.message || t('admin.platforms.failedToLoad'))
  } finally {
    loading.value = false
  }
}

const protocolLabel = (protocol: string) => {
  if (protocol === 'openai_compatible') return t('admin.platforms.protocolOpenAICompatible')
  if (protocol === 'anthropic') return t('admin.platforms.protocolAnthropic')
  if (protocol === 'gemini') return t('admin.platforms.protocolGemini')
  if (protocol === 'openai') return 'OpenAI'
  return protocol || 'native'
}

const capabilitiesForProtocol = (protocol: string) => {
  if (protocol === 'anthropic') return ['messages']
  if (protocol === 'gemini') return ['messages', 'native_gemini']
  return ['responses', 'chat_completions', 'images', 'videos']
}

const displayColor = (platform: Platform) => {
  return platform.color || customPlatformFallbackColor(platform.slug)
}

const resetForm = () => {
  form.slug = ''
  form.display_name = ''
  form.protocol = 'openai_compatible'
  form.base_url = ''
  form.color = '#64748B'
  form.enabled = true
}

const openCreateDialog = () => {
  editingPlatform.value = null
  resetForm()
  showDialog.value = true
}

const openEditDialog = (platform: Platform) => {
  editingPlatform.value = platform
  form.slug = platform.slug
  form.display_name = platform.display_name
  form.protocol = platform.protocol
  form.base_url = platform.base_url
  form.color = platform.color || '#64748B'
  form.enabled = platform.enabled
  showDialog.value = true
}

const closeDialog = () => {
  showDialog.value = false
}

const submitPlatform = async () => {
  saving.value = true
  const payload = {
    slug: form.slug.trim(),
    display_name: form.display_name.trim(),
    protocol: editingPlatform.value?.builtin ? editingPlatform.value.protocol : form.protocol,
    base_url: editingPlatform.value?.builtin ? editingPlatform.value.base_url : form.base_url.trim(),
    auth_modes: ['apikey'],
    capabilities: editingPlatform.value?.builtin
      ? editingPlatform.value.capabilities
      : capabilitiesForProtocol(form.protocol),
    color: form.color.trim(),
    enabled: form.enabled,
  }
  try {
    if (editingPlatform.value) {
      await adminAPI.platforms.update(editingPlatform.value.slug, payload)
      appStore.showSuccess(t('admin.platforms.updateSuccess'))
    } else {
      await adminAPI.platforms.create(payload)
      appStore.showSuccess(t('admin.platforms.createSuccess'))
    }
    showDialog.value = false
    await loadPlatforms()
  } catch (error: any) {
    appStore.showError(error.message || t('admin.platforms.failedToSave'))
  } finally {
    saving.value = false
  }
}

const toggleEnabled = async (platform: Platform) => {
  const payload = {
    slug: platform.slug,
    display_name: platform.display_name,
    protocol: platform.protocol,
    base_url: platform.base_url,
    auth_modes: platform.auth_modes,
    capabilities: platform.capabilities,
    color: platform.color,
    enabled: !platform.enabled,
  }
  try {
    await adminAPI.platforms.update(platform.slug, payload)
    await loadPlatforms()
  } catch (error: any) {
    appStore.showError(error.message || t('admin.platforms.failedToSave'))
  }
}

const openDeleteDialog = (platform: Platform) => {
  if (platform.builtin) return
  deletingPlatform.value = platform
  showDeleteDialog.value = true
}

const deletePlatform = async () => {
  if (!deletingPlatform.value) return
  try {
    await adminAPI.platforms.remove(deletingPlatform.value.slug)
    appStore.showSuccess(t('admin.platforms.deleteSuccess'))
    showDeleteDialog.value = false
    await loadPlatforms()
  } catch (error: any) {
    appStore.showError(error.message || t('admin.platforms.failedToDelete'))
  }
}

onMounted(loadPlatforms)
</script>
