<template>
  <AppLayout>
    <div class="mx-auto flex h-full w-full max-w-none flex-col gap-4 p-4 sm:p-6">
      <div class="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
            {{ t('ximoappUpdate.title', 'Application Center') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('ximoappUpdate.description', 'Maintain update versions, packages, and checksums for each app.') }}
          </p>
        </div>
        <button type="button" class="btn btn-secondary inline-flex items-center gap-2" :disabled="loading" @click="loadConfig">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          {{ t('common.refresh', 'Refresh') }}
        </button>
      </div>

      <div class="rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-800 dark:border-blue-800/40 dark:bg-blue-900/20 dark:text-blue-200">
        {{ t('ximoappUpdate.publicEndpoint', 'Update endpoint: POST /api/ximoapp/:appKey/version/latest; download path: /downloads/ximoapp/:file.') }}
      </div>

      <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('ximoappUpdate.globalSettings', 'Global Settings') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('ximoappUpdate.globalSettingsHint', 'When disabled, all update-check endpoints return 204.') }}
            </p>
          </div>
          <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input v-model="form.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            {{ t('ximoappUpdate.enabled', 'Enable update source') }}
          </label>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('ximoappUpdate.apps', 'Apps') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('ximoappUpdate.appsHint', 'Use appKey to separate clients. XimoDesk uses ximodesk.') }}
            </p>
          </div>
          <button type="button" class="btn btn-secondary" @click="addApp">
            {{ t('ximoappUpdate.addApp', 'Add App') }}
          </button>
        </div>

        <div class="ximoapp-card-grid grid gap-4 p-5">
          <article
            v-for="(app, index) in form.apps"
            :key="`${app.key || 'app'}:${index}`"
            class="rounded-lg border border-gray-200 bg-gray-50/70 p-4 dark:border-dark-700 dark:bg-dark-900/40"
          >
            <div class="mb-4 flex items-start justify-between gap-3">
              <div class="min-w-0">
                <h3 class="truncate text-base font-semibold text-gray-900 dark:text-white">
                  {{ app.name || app.key || t('ximoappUpdate.addApp', 'Add App') }}
                </h3>
                <p class="mt-1 truncate font-mono text-xs text-gray-500 dark:text-gray-400">
                  {{ app.key || 'app-key' }}
                </p>
              </div>
              <label class="inline-flex shrink-0 items-center gap-2 text-xs text-gray-700 dark:text-gray-300">
                <input v-model="app.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                {{ t('ximoappUpdate.releaseEnabled', 'Enabled') }}
              </label>
            </div>

            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div>
                <label class="input-label">{{ t('ximoappUpdate.appKey', 'App Key') }}</label>
                <input v-model.trim="app.key" class="input h-9 font-mono text-xs" placeholder="ximodesk" />
              </div>
              <div>
                <label class="input-label">{{ t('ximoappUpdate.appName', 'Display Name') }}</label>
                <input v-model.trim="app.name" class="input h-9" placeholder="XimoDesk" />
              </div>
              <div>
                <label class="input-label">{{ t('ximoappUpdate.clientType', 'Client Type') }}</label>
                <select v-model="app.client_type" class="input h-9">
                  <option value="desktop">desktop</option>
                  <option value="mobile">mobile</option>
                  <option value="custom">custom</option>
                </select>
              </div>
              <div class="sm:col-span-2">
                <label class="input-label">{{ t('ximoappUpdate.appDescription', 'App Description') }}</label>
                <textarea
                  v-model.trim="app.description"
                  class="input min-h-[82px] text-sm"
                  :placeholder="t('ximoappUpdate.appDescriptionPlaceholder', 'Shown in the user download center')"
                />
              </div>
            </div>
          </article>
        </div>

        <div class="flex justify-end border-t border-gray-100 p-5 dark:border-dark-700">
          <button type="button" class="btn btn-primary" :disabled="saving" @click="saveConfig">
            <Icon v-if="saving" name="refresh" size="sm" class="mr-2 animate-spin" />
            {{ saving ? t('common.saving', 'Saving') : t('common.save', 'Save') }}
          </button>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('ximoappUpdate.uploadPackage', 'Upload Package') }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('ximoappUpdate.uploadPackageHint', 'Uploads automatically calculate sha256 and generate a download URL.') }}
          </p>
        </div>

        <form class="space-y-5 p-5" @submit.prevent="uploadPackage">
          <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
            <div>
              <label class="input-label">{{ t('ximoappUpdate.appKey', 'App Key') }}</label>
              <input v-model.trim="uploadForm.app_key" class="input font-mono" list="ximoapp-keys" placeholder="ximodesk" required />
              <datalist id="ximoapp-keys">
                <option v-for="app in appOptions" :key="app.key" :value="app.key">{{ app.name }}</option>
              </datalist>
            </div>
            <div>
              <label class="input-label">{{ t('ximoappUpdate.version', 'Version') }}</label>
              <input v-model.trim="uploadForm.version" class="input font-mono" placeholder="1.0.1" required />
            </div>
            <div>
              <label class="input-label">{{ t('ximoappUpdate.channel', 'Channel') }}</label>
              <select v-model="uploadForm.channel" class="input">
                <option value="stable">stable</option>
                <option value="beta">beta</option>
              </select>
            </div>
          </div>

          <div class="grid grid-cols-1 gap-4 md:grid-cols-4">
            <div>
              <label class="input-label">{{ t('ximoappUpdate.os', 'OS') }}</label>
              <select v-model="uploadForm.os" class="input">
                <option value="windows">windows</option>
                <option value="macos">macos</option>
                <option value="linux">linux</option>
                <option value="android">android</option>
                <option value="ios">ios</option>
              </select>
            </div>
            <div>
              <label class="input-label">{{ t('ximoappUpdate.arch', 'Architecture') }}</label>
              <select v-model="uploadForm.arch" class="input">
                <option value="x86_64">x86_64</option>
                <option value="aarch64">aarch64</option>
                <option value="universal">universal</option>
              </select>
            </div>
            <div>
              <label class="input-label">{{ t('ximoappUpdate.locale', 'Language') }}</label>
              <input v-model.trim="uploadForm.locale" class="input font-mono" placeholder="zh-CN / en-US / all" />
            </div>
            <div>
              <label class="input-label">{{ t('ximoappUpdate.packageType', 'Package Type') }}</label>
              <select v-model="uploadForm.package_type" class="input">
                <option value="">{{ t('ximoappUpdate.autoDetect', 'Auto detect') }}</option>
                <option value="msi">msi</option>
                <option value="zip">zip</option>
                <option value="exe">exe</option>
                <option value="dmg">dmg</option>
                <option value="pkg">pkg</option>
                <option value="apk">apk</option>
                <option value="aab">aab</option>
                <option value="ipa">ipa</option>
              </select>
            </div>
          </div>

          <div class="grid grid-cols-1 gap-4 md:grid-cols-4">
            <div>
              <label class="input-label">{{ t('ximoappUpdate.minSupportedVersion', 'Minimum Supported Version') }}</label>
              <input v-model.trim="uploadForm.min_supported_version" class="input font-mono" placeholder="1.0.0" />
            </div>
            <div>
              <label class="input-label">{{ t('ximoappUpdate.minSupportedVersionCode', 'Minimum Supported Version Code') }}</label>
              <input v-model.trim="uploadForm.min_supported_version_code" class="input font-mono" placeholder="90" />
            </div>
            <div>
              <label class="input-label">{{ t('ximoappUpdate.publishedAt', 'Published at') }}</label>
              <input v-model.trim="uploadForm.published_at" class="input font-mono" placeholder="2026-06-21T00:00:00Z" />
            </div>
            <div class="flex items-end gap-4">
              <label class="inline-flex items-center gap-2 pb-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="uploadForm.force" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                {{ t('ximoappUpdate.force', 'Force update') }}
              </label>
              <label class="inline-flex items-center gap-2 pb-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="uploadForm.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                {{ t('ximoappUpdate.releaseEnabled', 'Enabled') }}
              </label>
            </div>
          </div>

          <div>
            <label class="input-label">{{ t('ximoappUpdate.notes', 'Release Notes') }}</label>
            <textarea v-model.trim="uploadForm.notes" class="input min-h-[96px]" placeholder="Fix connection stability and polish the interface" />
          </div>

          <div class="grid grid-cols-1 gap-4 md:grid-cols-[1fr_auto] md:items-end">
            <div>
              <label class="input-label">{{ t('ximoappUpdate.packageFile', 'Package File') }}</label>
              <input
                ref="fileInput"
                type="file"
                class="input"
                accept=".msi,.zip,.exe,.dmg,.pkg,.apk,.aab,.ipa"
                required
                @change="onFileChange"
              />
              <p class="input-hint">
                {{ selectedFileName || t('ximoappUpdate.packageFileHint', 'Supports .msi / .zip / .exe / .dmg / .pkg / .apk / .aab / .ipa.') }}
              </p>
            </div>
            <button type="submit" class="btn btn-primary min-w-[132px]" :disabled="uploading">
              <Icon v-if="uploading" name="refresh" size="sm" class="mr-2 animate-spin" />
              {{ uploading ? t('ximoappUpdate.uploading', 'Uploading') : t('ximoappUpdate.upload', 'Upload package') }}
            </button>
          </div>
        </form>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('ximoappUpdate.releaseList', 'Published Packages') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('ximoappUpdate.releaseListHint', 'Packages are grouped by app to avoid mixing different clients.') }}
            </p>
          </div>
          <button type="button" class="btn btn-secondary" :disabled="saving || loading" @click="saveConfig">
            {{ t('ximoappUpdate.saveList', 'Save list settings') }}
          </button>
        </div>

        <div v-if="loading" class="flex items-center justify-center py-16 text-gray-500 dark:text-gray-400">
          <Icon name="refresh" size="md" class="mr-2 animate-spin" />
          {{ t('common.loading', 'Loading') }}
        </div>

        <div v-else-if="form.releases.length === 0" class="px-5 py-12 text-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('ximoappUpdate.emptyReleases', 'No packages uploaded yet.') }}
        </div>

        <div v-else class="ximoapp-card-grid grid gap-4 p-5">
          <article
            v-for="group in releaseAppGroups"
            :key="group.key"
            class="flex min-w-0 flex-col overflow-hidden rounded-lg border border-gray-200 bg-gray-50/70 dark:border-dark-700 dark:bg-dark-900/40"
          >
            <header class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <h3 class="break-words text-base font-semibold text-gray-900 dark:text-white">
                      {{ group.name || group.key }}
                    </h3>
                    <span class="rounded-full bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/25 dark:text-primary-200">
                      {{ group.client_type || 'custom' }}
                    </span>
                  </div>
                  <p class="mt-1 font-mono text-xs text-gray-500 dark:text-gray-400">{{ group.key }}</p>
                </div>
                <span class="shrink-0 text-xs text-gray-500 dark:text-gray-400">
                  {{ group.releases.length }}
                </span>
              </div>
            </header>

            <div class="divide-y divide-gray-100 dark:divide-dark-700">
              <section
                v-for="release in group.releases"
                :key="release.id || release.download_url"
                class="space-y-3 px-4 py-4"
              >
                <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                  <div class="min-w-0">
                    <div class="flex flex-wrap items-center gap-2">
                      <div class="font-semibold text-gray-900 dark:text-white">{{ release.version }}</div>
                      <span v-if="release.version_code" class="rounded-md bg-gray-100 px-2 py-0.5 font-mono text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                        {{ release.version_code }}
                      </span>
                    </div>
                    <div class="mt-1 font-mono text-xs text-gray-600 dark:text-gray-300">
                      {{ release.channel }} / {{ release.os }} / {{ release.arch }}
                    </div>
                    <div class="mt-1 font-mono text-xs text-gray-500 dark:text-gray-400">
                      {{ release.locale || 'all' }} / {{ release.package_type || '-' }}
                    </div>
                  </div>

                  <div class="flex shrink-0 flex-wrap gap-2">
                    <button type="button" class="btn btn-secondary btn-sm" @click="copyUrl(release.download_url)">
                      {{ t('common.copy', 'Copy') }}
                    </button>
                    <button type="button" class="btn btn-danger btn-sm" :disabled="deletingId === release.id" @click="deleteRelease(release)">
                      {{ deletingId === release.id ? t('common.deleting', 'Deleting') : t('common.delete', 'Delete') }}
                    </button>
                  </div>
                </div>

                <div class="grid grid-cols-1 gap-3 text-xs sm:grid-cols-2">
                  <div class="min-w-0">
                    <div class="text-gray-500 dark:text-gray-400">{{ t('ximoappUpdate.packageFile', 'Package File') }}</div>
                    <div class="mt-1 break-all font-mono text-gray-800 dark:text-gray-200">{{ release.file_name || '-' }}</div>
                    <div class="mt-1 text-gray-500 dark:text-gray-400">{{ formatBytes(release.file_size) }}</div>
                  </div>
                  <div class="min-w-0">
                    <div class="text-gray-500 dark:text-gray-400">{{ t('ximoappUpdate.downloadUrl', 'Download URL') }}</div>
                    <div class="mt-1 break-all font-mono text-gray-800 dark:text-gray-200">{{ release.download_url }}</div>
                    <div v-if="releaseTime(release)" class="mt-1 font-mono text-gray-500 dark:text-gray-400">{{ releaseTime(release) }}</div>
                  </div>
                  <div class="min-w-0 sm:col-span-2">
                    <div class="text-gray-500 dark:text-gray-400">sha256</div>
                    <div class="mt-1 break-all font-mono text-gray-500 dark:text-gray-400">{{ release.sha256 }}</div>
                  </div>
                </div>

                <input v-model.trim="release.notes" class="input h-9 text-xs" :placeholder="t('ximoappUpdate.notes', 'Release Notes')" />

                <div class="flex flex-wrap gap-4">
                  <label class="flex items-center gap-2 text-xs text-gray-700 dark:text-gray-300">
                    <input v-model="release.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                    {{ t('ximoappUpdate.releaseEnabled', 'Enabled') }}
                  </label>
                  <label class="flex items-center gap-2 text-xs text-gray-700 dark:text-gray-300">
                    <input v-model="release.force" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                    {{ t('ximoappUpdate.force', 'Force update') }}
                  </label>
                </div>
              </section>
            </div>
          </article>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { XimoAppUpdateApp, XimoDeskUpdateConfig, XimoDeskUpdateRelease } from '@/api/admin/ximodeskUpdate'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

interface ReleaseAppGroup {
  key: string
  name: string
  description?: string
  client_type?: string
  releases: XimoDeskUpdateRelease[]
}

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const uploading = ref(false)
const deletingId = ref('')
const selectedFile = ref<File | null>(null)
const selectedFileName = ref('')
const fileInput = ref<HTMLInputElement | null>(null)

const form = reactive<XimoDeskUpdateConfig>({
  enabled: false,
  apps: [],
  releases: []
})

const uploadForm = reactive({
  app_key: 'ximodesk',
  channel: 'stable',
  os: 'windows',
  arch: 'x86_64',
  locale: 'zh-CN',
  version: '',
  min_supported_version: '',
  min_supported_version_code: '',
  package_type: '',
  notes: '',
  force: false,
  enabled: true,
  published_at: ''
})

const appOptions = computed(() => {
  const map = new Map<string, XimoAppUpdateApp>()
  for (const app of form.apps || []) {
    if (app.key) {
      map.set(app.key, app)
    }
  }
  for (const release of form.releases || []) {
    const key = release.app_key || 'ximodesk'
    if (!map.has(key)) {
      map.set(key, { key, name: key, client_type: 'custom', enabled: true })
    }
  }
  return [...map.values()].sort((left, right) => (left.name || left.key).localeCompare(right.name || right.key))
})

const releaseAppGroups = computed<ReleaseAppGroup[]>(() => {
  const map = new Map<string, ReleaseAppGroup>()
  for (const app of appOptions.value) {
    map.set(app.key, {
      key: app.key,
      name: app.name || app.key,
      description: app.description,
      client_type: app.client_type || 'custom',
      releases: []
    })
  }
  for (const release of form.releases || []) {
    const key = release.app_key || 'ximodesk'
    if (!map.has(key)) {
      map.set(key, { key, name: key, client_type: 'custom', releases: [] })
    }
    map.get(key)!.releases.push(release)
  }
  return [...map.values()]
    .filter((group) => group.releases.length > 0)
    .map((group) => ({
      ...group,
      releases: [...group.releases].sort(compareReleases)
    }))
    .sort((left, right) => (left.name || left.key).localeCompare(right.name || right.key))
})

function normalizeApp(app: XimoAppUpdateApp): XimoAppUpdateApp {
  return {
    key: app.key || '',
    name: app.name || app.key || '',
    description: app.description || '',
    client_type: app.client_type || 'custom',
    response_mode: app.response_mode || 'standard',
    enabled: app.enabled !== false
  }
}

function normalizeRelease(release: XimoDeskUpdateRelease): XimoDeskUpdateRelease {
  return {
    ...release,
    app_key: release.app_key || 'ximodesk',
    enabled: release.enabled !== false,
    locale: release.locale || 'all',
    package_type: release.package_type || ''
  }
}

function applyConfig(config: XimoDeskUpdateConfig) {
  form.enabled = !!config.enabled
  form.apps = (config.apps || []).map(normalizeApp)
  form.releases.splice(0, form.releases.length, ...((config.releases || []).map(normalizeRelease)))
}

function buildPayload(): XimoDeskUpdateConfig {
  return {
    enabled: form.enabled,
    apps: (form.apps || []).map((app) => ({
      ...app,
      key: app.key.trim(),
      name: app.name.trim(),
      description: (app.description || '').trim(),
      client_type: app.client_type || 'custom',
      response_mode: app.response_mode || 'standard',
      enabled: app.enabled !== false
    })),
    releases: form.releases.map((release) => ({
      ...release,
      app_key: release.app_key || 'ximodesk',
      enabled: release.enabled !== false,
      locale: release.locale || 'all',
      package_type: release.package_type || undefined,
      published_at: release.published_at || undefined,
      uploaded_at: release.uploaded_at || undefined,
      file_name: release.file_name || undefined
    }))
  }
}

function addApp() {
  const index = (form.apps || []).length + 1
  if (!form.apps) {
    form.apps = []
  }
  form.apps.push({
    key: `custom-app-${index}`,
    name: `Custom App ${index}`,
    description: '',
    client_type: 'custom',
    response_mode: 'standard',
    enabled: true
  })
}

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  selectedFile.value = input.files?.[0] || null
  selectedFileName.value = selectedFile.value?.name || ''
}

async function loadConfig() {
  loading.value = true
  try {
    const config = await adminAPI.ximodeskUpdate.get()
    applyConfig(config)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('ximoappUpdate.loadFailed', 'Failed to load XimoAPP update config')))
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  saving.value = true
  try {
    const saved = await adminAPI.ximodeskUpdate.update(buildPayload())
    applyConfig(saved)
    appStore.showSuccess(t('ximoappUpdate.saved', 'XimoAPP update config saved'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('ximoappUpdate.saveFailed', 'Failed to save XimoAPP update config')))
  } finally {
    saving.value = false
  }
}

async function uploadPackage() {
  if (!selectedFile.value) {
    appStore.showError(t('ximoappUpdate.packageFileRequired', 'Please choose a package file'))
    return
  }
  uploading.value = true
  try {
    const payload = new FormData()
    payload.append('file', selectedFile.value)
    payload.append('app_key', uploadForm.app_key)
    payload.append('channel', uploadForm.channel)
    payload.append('os', uploadForm.os)
    payload.append('arch', uploadForm.arch)
    payload.append('locale', uploadForm.locale || 'all')
    payload.append('version', uploadForm.version)
    payload.append('min_supported_version', uploadForm.min_supported_version)
    payload.append('min_supported_version_code', uploadForm.min_supported_version_code)
    payload.append('package_type', uploadForm.package_type)
    payload.append('notes', uploadForm.notes)
    payload.append('force', String(uploadForm.force))
    payload.append('enabled', String(uploadForm.enabled))
    payload.append('published_at', uploadForm.published_at)

    const result = await adminAPI.ximodeskUpdate.uploadPackage(payload)
    applyConfig(result.config)
    selectedFile.value = null
    selectedFileName.value = ''
    if (fileInput.value) {
      fileInput.value.value = ''
    }
    appStore.showSuccess(t('ximoappUpdate.uploaded', 'Package uploaded'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('ximoappUpdate.uploadFailed', 'Failed to upload package')))
  } finally {
    uploading.value = false
  }
}

async function deleteRelease(release: XimoDeskUpdateRelease) {
  if (!release.id) {
    appStore.showError(t('ximoappUpdate.releaseIdMissing', 'Release ID is missing, cannot delete'))
    return
  }
  const ok = window.confirm(t('ximoappUpdate.confirmDelete', 'Delete this package release?'))
  if (!ok) {
    return
  }
  deletingId.value = release.id
  try {
    const config = await adminAPI.ximodeskUpdate.deleteRelease(release.id)
    applyConfig(config)
    appStore.showSuccess(t('ximoappUpdate.deleted', 'Package release deleted'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('ximoappUpdate.deleteFailed', 'Failed to delete package release')))
  } finally {
    deletingId.value = ''
  }
}

async function copyUrl(url: string) {
  try {
    await navigator.clipboard.writeText(url)
    appStore.showSuccess(t('common.copied', 'Copied'))
  } catch {
    appStore.showError(t('common.copyFailed', 'Copy failed'))
  }
}

function compareReleases(left: XimoDeskUpdateRelease, right: XimoDeskUpdateRelease): number {
  const version = compareVersions(right.version, left.version)
  if (version !== 0) return version
  if ((right.version_code || 0) !== (left.version_code || 0)) {
    return (right.version_code || 0) - (left.version_code || 0)
  }
  return releaseTimestamp(right) - releaseTimestamp(left)
}

function compareVersions(left: string, right: string): number {
  const leftParts = left.trim().replace(/^v/i, '').split('.')
  const rightParts = right.trim().replace(/^v/i, '').split('.')
  const length = Math.max(leftParts.length, rightParts.length)
  for (let idx = 0; idx < length; idx += 1) {
    const leftValue = leadingInt(leftParts[idx] || '')
    const rightValue = leadingInt(rightParts[idx] || '')
    if (leftValue !== rightValue) return leftValue - rightValue
  }
  return left.localeCompare(right)
}

function leadingInt(value: string): number {
  const match = value.match(/^\d+/)
  return match ? Number(match[0]) : 0
}

function releaseTimestamp(release: XimoDeskUpdateRelease): number {
  const date = new Date(release.published_at || release.uploaded_at || '')
  return Number.isNaN(date.getTime()) ? 0 : date.getTime()
}

function releaseTime(release: XimoDeskUpdateRelease): string {
  return formatDate(release.published_at || release.uploaded_at || '')
}

function formatBytes(value?: number) {
  if (!value || value <= 0) {
    return '-'
  }
  const units = ['B', 'KB', 'MB', 'GB']
  let size = value
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit += 1
  }
  return `${size.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`
}

function formatDate(value: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString()
}

loadConfig()
</script>

<style scoped>
.ximoapp-card-grid {
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 28rem), 1fr));
}
</style>
