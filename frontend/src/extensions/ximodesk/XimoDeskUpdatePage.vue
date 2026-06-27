<template>
  <AppLayout>
    <div class="mx-auto flex h-full w-full max-w-6xl flex-col gap-4 p-4 sm:p-6">
      <div class="flex flex-col justify-between gap-3 sm:flex-row sm:items-center">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
            {{ t('ximoappUpdate.title', 'XimoAPP 更新中心') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('ximoappUpdate.description', '统一维护 XimoDesk、手机端和后续客户端的自动更新版本、安装包和校验信息。') }}
          </p>
        </div>
        <button type="button" class="btn btn-secondary inline-flex items-center gap-2" :disabled="loading" @click="loadConfig">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          {{ t('common.refresh', '刷新') }}
        </button>
      </div>

      <div class="rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-800 dark:border-blue-800/40 dark:bg-blue-900/20 dark:text-blue-200">
        {{ t('ximoappUpdate.publicEndpoint', '更新接口：POST /api/ximoapp/:appKey/version/latest；下载入口：/downloads/ximoapp/:file。') }}
      </div>

      <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('ximoappUpdate.globalSettings', '全局设置') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('ximoappUpdate.globalSettingsHint', '关闭后，所有更新检查接口返回 204。') }}
            </p>
          </div>
          <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input v-model="form.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            {{ t('ximoappUpdate.enabled', '启用更新源') }}
          </label>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('ximoappUpdate.apps', '应用登记') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('ximoappUpdate.appsHint', '按 appKey 区分不同客户端，XimoDesk 使用 ximodesk。') }}
            </p>
          </div>
          <button type="button" class="btn btn-secondary" @click="addApp">
            {{ t('ximoappUpdate.addApp', '新增应用') }}
          </button>
        </div>

        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:bg-dark-700/60 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3">{{ t('ximoappUpdate.appKey', '应用标识') }}</th>
                <th class="px-5 py-3">{{ t('ximoappUpdate.appName', '显示名称') }}</th>
                <th class="px-5 py-3">{{ t('ximoappUpdate.appDescription', '应用介绍') }}</th>
                <th class="px-5 py-3">{{ t('ximoappUpdate.clientType', '客户端类型') }}</th>
                <th class="px-5 py-3">{{ t('ximoappUpdate.releaseEnabled', '启用') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 text-sm dark:divide-dark-700">
              <tr v-for="app in form.apps" :key="app.key">
                <td class="px-5 py-3">
                  <input v-model.trim="app.key" class="input h-9 font-mono text-xs" placeholder="ximodesk" />
                </td>
                <td class="px-5 py-3">
                  <input v-model.trim="app.name" class="input h-9" placeholder="XimoDesk" />
                </td>
                <td class="px-5 py-3">
                  <textarea
                    v-model.trim="app.description"
                    class="input min-h-[72px] min-w-[220px] text-sm"
                    :placeholder="t('ximoappUpdate.appDescriptionPlaceholder', '用于下载中心展示的软件介绍')"
                  />
                </td>
                <td class="px-5 py-3">
                  <select v-model="app.client_type" class="input h-9">
                    <option value="desktop">desktop</option>
                    <option value="mobile">mobile</option>
                    <option value="custom">custom</option>
                  </select>
                </td>
                <td class="px-5 py-3">
                  <input v-model="app.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="flex justify-end p-5">
          <button type="button" class="btn btn-primary" :disabled="saving" @click="saveConfig">
            <Icon v-if="saving" name="refresh" size="sm" class="mr-2 animate-spin" />
            {{ saving ? t('common.saving', '保存中') : t('common.save', '保存') }}
          </button>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('ximoappUpdate.uploadPackage', '上传安装包') }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('ximoappUpdate.uploadPackageHint', '上传后自动计算 sha256，并生成 ximoai.cn 下载链接。') }}
          </p>
        </div>

        <form class="space-y-5 p-5" @submit.prevent="uploadPackage">
          <div class="grid grid-cols-1 gap-4 md:grid-cols-4">
            <div>
              <label class="input-label">{{ t('ximoappUpdate.appKey', '应用标识') }}</label>
              <input v-model.trim="uploadForm.app_key" class="input font-mono" list="ximoapp-keys" placeholder="ximodesk" required />
              <datalist id="ximoapp-keys">
                <option v-for="app in appOptions" :key="app.key" :value="app.key">{{ app.name }}</option>
              </datalist>
            </div>
            <div>
              <label class="input-label">{{ t('ximoappUpdate.version', '版本号') }}</label>
              <input v-model.trim="uploadForm.version" class="input font-mono" placeholder="1.0.1" required />
            </div>
            <div>
              <label class="input-label">{{ t('ximoappUpdate.versionCode', '版本构建号') }}</label>
              <input v-model.trim="uploadForm.version_code" class="input font-mono" placeholder="100" />
            </div>
            <div>
              <label class="input-label">{{ t('ximoappUpdate.channel', '发布渠道') }}</label>
              <select v-model="uploadForm.channel" class="input">
                <option value="stable">stable</option>
                <option value="beta">beta</option>
              </select>
            </div>
          </div>

          <div class="grid grid-cols-1 gap-4 md:grid-cols-4">
            <div>
              <label class="input-label">{{ t('ximoappUpdate.os', '系统') }}</label>
              <select v-model="uploadForm.os" class="input">
                <option value="windows">windows</option>
                <option value="macos">macos</option>
                <option value="linux">linux</option>
                <option value="android">android</option>
                <option value="ios">ios</option>
              </select>
            </div>
            <div>
              <label class="input-label">{{ t('ximoappUpdate.arch', '架构') }}</label>
              <select v-model="uploadForm.arch" class="input">
                <option value="x86_64">x86_64</option>
                <option value="aarch64">aarch64</option>
                <option value="universal">universal</option>
              </select>
            </div>
            <div>
              <label class="input-label">{{ t('ximoappUpdate.locale', '语言版本') }}</label>
              <input v-model.trim="uploadForm.locale" class="input font-mono" placeholder="zh-CN / en-US / all" />
            </div>
            <div>
              <label class="input-label">{{ t('ximoappUpdate.packageType', '安装包类型') }}</label>
              <select v-model="uploadForm.package_type" class="input">
                <option value="">{{ t('ximoappUpdate.autoDetect', '自动识别') }}</option>
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
              <label class="input-label">{{ t('ximoappUpdate.minSupportedVersion', '最低支持版本') }}</label>
              <input v-model.trim="uploadForm.min_supported_version" class="input font-mono" placeholder="1.0.0" />
            </div>
            <div>
              <label class="input-label">{{ t('ximoappUpdate.minSupportedVersionCode', '最低支持构建号') }}</label>
              <input v-model.trim="uploadForm.min_supported_version_code" class="input font-mono" placeholder="90" />
            </div>
            <div>
              <label class="input-label">{{ t('ximoappUpdate.publishedAt', '发布时间') }}</label>
              <input v-model.trim="uploadForm.published_at" class="input font-mono" placeholder="2026-06-21T00:00:00Z" />
            </div>
            <div class="flex items-end gap-4">
              <label class="inline-flex items-center gap-2 pb-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="uploadForm.force" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                {{ t('ximoappUpdate.force', '强制更新') }}
              </label>
              <label class="inline-flex items-center gap-2 pb-3 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="uploadForm.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                {{ t('ximoappUpdate.releaseEnabled', '启用版本') }}
              </label>
            </div>
          </div>

          <div>
            <label class="input-label">{{ t('ximoappUpdate.notes', '更新说明') }}</label>
            <textarea v-model.trim="uploadForm.notes" class="input min-h-[96px]" placeholder="修复连接稳定性，优化界面显示" />
          </div>

          <div class="grid grid-cols-1 gap-4 md:grid-cols-[1fr_auto] md:items-end">
            <div>
              <label class="input-label">{{ t('ximoappUpdate.packageFile', '安装包文件') }}</label>
              <input
                ref="fileInput"
                type="file"
                class="input"
                accept=".msi,.zip,.exe,.dmg,.pkg,.apk,.aab,.ipa"
                required
                @change="onFileChange"
              />
              <p class="input-hint">
                {{ selectedFileName || t('ximoappUpdate.packageFileHint', '支持 .msi / .zip / .exe / .dmg / .pkg / .apk / .aab / .ipa。') }}
              </p>
            </div>
            <button type="submit" class="btn btn-primary min-w-[132px]" :disabled="uploading">
              <Icon v-if="uploading" name="refresh" size="sm" class="mr-2 animate-spin" />
              {{ uploading ? t('ximoappUpdate.uploading', '上传中') : t('ximoappUpdate.upload', '上传安装包') }}
            </button>
          </div>
        </form>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('ximoappUpdate.releaseList', '已发布安装包') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('ximoappUpdate.releaseListHint', '按应用、渠道、系统、架构和语言匹配，返回最高版本。') }}
            </p>
          </div>
          <button type="button" class="btn btn-secondary" :disabled="saving || loading" @click="saveConfig">
            {{ t('ximoappUpdate.saveList', '保存列表配置') }}
          </button>
        </div>

        <div v-if="loading" class="flex items-center justify-center py-16 text-gray-500 dark:text-gray-400">
          <Icon name="refresh" size="md" class="mr-2 animate-spin" />
          {{ t('common.loading', '加载中') }}
        </div>

        <div v-else-if="form.releases.length === 0" class="px-5 py-12 text-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('ximoappUpdate.emptyReleases', '还没有上传安装包。') }}
        </div>

        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:bg-dark-700/60 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3">{{ t('ximoappUpdate.version', '版本') }}</th>
                <th class="px-5 py-3">{{ t('ximoappUpdate.target', '目标') }}</th>
                <th class="px-5 py-3">{{ t('ximoappUpdate.packageFile', '安装包') }}</th>
                <th class="px-5 py-3">{{ t('ximoappUpdate.downloadUrl', '下载地址') }}</th>
                <th class="px-5 py-3">{{ t('ximoappUpdate.options', '选项') }}</th>
                <th class="px-5 py-3">{{ t('common.actions', '操作') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 text-sm dark:divide-dark-700">
              <tr v-for="release in form.releases" :key="release.id || release.download_url">
                <td class="px-5 py-4 align-top">
                  <div class="font-semibold text-gray-900 dark:text-white">{{ release.version }}</div>
                  <div class="mt-1 font-mono text-xs text-gray-500 dark:text-gray-400">
                    {{ release.app_key || 'ximodesk' }}
                  </div>
                  <div v-if="release.version_code" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    code {{ release.version_code }}
                  </div>
                </td>
                <td class="px-5 py-4 align-top">
                  <div class="font-mono text-xs text-gray-800 dark:text-gray-200">
                    {{ release.channel }} / {{ release.os }} / {{ release.arch }}
                  </div>
                  <div class="mt-1 font-mono text-xs text-gray-500 dark:text-gray-400">
                    {{ release.locale || 'all' }} / {{ release.package_type || '-' }}
                  </div>
                </td>
                <td class="px-5 py-4 align-top">
                  <div class="max-w-[220px] break-all font-mono text-xs text-gray-800 dark:text-gray-200">
                    {{ release.file_name || '-' }}
                  </div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ formatBytes(release.file_size) }}
                  </div>
                  <div class="mt-1 max-w-[220px] break-all font-mono text-xs text-gray-500 dark:text-gray-400">
                    sha256: {{ release.sha256 }}
                  </div>
                </td>
                <td class="px-5 py-4 align-top">
                  <div class="max-w-[280px] break-all font-mono text-xs text-gray-800 dark:text-gray-200">
                    {{ release.download_url }}
                  </div>
                  <input v-model.trim="release.notes" class="input mt-3 h-9 text-xs" :placeholder="t('ximoappUpdate.notes', '更新说明')" />
                </td>
                <td class="px-5 py-4 align-top">
                  <div class="space-y-2">
                    <label class="flex items-center gap-2 text-xs text-gray-700 dark:text-gray-300">
                      <input v-model="release.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                      {{ t('ximoappUpdate.releaseEnabled', '启用版本') }}
                    </label>
                    <label class="flex items-center gap-2 text-xs text-gray-700 dark:text-gray-300">
                      <input v-model="release.force" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                      {{ t('ximoappUpdate.force', '强制更新') }}
                    </label>
                  </div>
                </td>
                <td class="px-5 py-4 align-top">
                  <div class="flex flex-col gap-2">
                    <button type="button" class="btn btn-secondary btn-sm" @click="copyUrl(release.download_url)">
                      {{ t('common.copy', '复制') }}
                    </button>
                    <button type="button" class="btn btn-danger btn-sm" :disabled="deletingId === release.id" @click="deleteRelease(release)">
                      {{ deletingId === release.id ? t('common.deleting', '删除中') : t('common.delete', '删除') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
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
  version_code: '',
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
  return [...map.values()]
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
  form.apps?.splice(0, form.apps.length, ...((config.apps || []).map(normalizeApp)))
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
  form.apps?.push({
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
    appStore.showError(extractApiErrorMessage(error, t('ximoappUpdate.loadFailed', 'XimoAPP 更新配置加载失败')))
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  saving.value = true
  try {
    const saved = await adminAPI.ximodeskUpdate.update(buildPayload())
    applyConfig(saved)
    appStore.showSuccess(t('ximoappUpdate.saved', 'XimoAPP 更新配置已保存'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('ximoappUpdate.saveFailed', 'XimoAPP 更新配置保存失败')))
  } finally {
    saving.value = false
  }
}

async function uploadPackage() {
  if (!selectedFile.value) {
    appStore.showError(t('ximoappUpdate.packageFileRequired', '请选择安装包文件'))
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
    payload.append('version_code', uploadForm.version_code)
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
    appStore.showSuccess(t('ximoappUpdate.uploaded', '安装包已上传'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('ximoappUpdate.uploadFailed', '安装包上传失败')))
  } finally {
    uploading.value = false
  }
}

async function deleteRelease(release: XimoDeskUpdateRelease) {
  if (!release.id) {
    appStore.showError(t('ximoappUpdate.releaseIdMissing', '版本缺少 ID，无法删除'))
    return
  }
  const ok = window.confirm(t('ximoappUpdate.confirmDelete', '确定删除这个安装包版本吗？'))
  if (!ok) {
    return
  }
  deletingId.value = release.id
  try {
    const config = await adminAPI.ximodeskUpdate.deleteRelease(release.id)
    applyConfig(config)
    appStore.showSuccess(t('ximoappUpdate.deleted', '安装包版本已删除'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('ximoappUpdate.deleteFailed', '安装包版本删除失败')))
  } finally {
    deletingId.value = ''
  }
}

async function copyUrl(url: string) {
  try {
    await navigator.clipboard.writeText(url)
    appStore.showSuccess(t('common.copied', '已复制'))
  } catch {
    appStore.showError(t('common.copyFailed', '复制失败'))
  }
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

loadConfig()
</script>
