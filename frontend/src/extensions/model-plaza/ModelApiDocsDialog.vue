<template>
  <BaseDialog
    :show="show"
    :title="t('modelPlaza.apiDocs.title', { model: modelName })"
    width="extra-wide"
    @close="emit('close')"
  >
    <template v-if="docs">
      <div class="mb-5 border-b border-gray-200 pb-4 dark:border-dark-700">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex min-w-0 flex-wrap items-center gap-2">
            <span class="rounded-md bg-gray-100 px-2 py-1 font-mono text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-200">
              {{ docs.protocol }}
            </span>
            <span class="rounded-md px-2 py-1 text-xs font-medium" :class="sourceBadgeClass">
              {{ sourceLabel }}
            </span>
          </div>
          <button
            v-if="canEdit && !editing"
            data-testid="model-api-docs-edit"
            type="button"
            class="btn btn-secondary btn-sm"
            @click="startEditing"
          >
            <Icon name="edit" size="sm" />
            {{ t('modelPlaza.apiDocs.edit') }}
          </button>
        </div>

        <div class="mt-4 grid gap-3 sm:grid-cols-3">
          <div v-for="item in referenceValues" :key="item.id" class="min-w-0 border-l-2 border-gray-200 pl-3 dark:border-dark-700">
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ item.label }}</p>
            <div class="mt-1 flex min-w-0 items-center gap-2">
              <code class="min-w-0 flex-1 truncate text-xs text-gray-800 dark:text-gray-200">{{ item.value }}</code>
              <button
                :data-testid="`copy-${item.id}`"
                type="button"
                class="btn btn-ghost btn-icon btn-sm shrink-0"
                :title="t('modelPlaza.apiDocs.copy')"
                @click="copyText(item.value)"
              >
                <Icon name="copy" size="xs" />
              </button>
            </div>
          </div>
        </div>
      </div>

      <div v-if="editing" class="space-y-5">
        <section
          v-for="category in editableCategories"
          :key="category"
          class="border-b border-gray-200 pb-5 last:border-0 dark:border-dark-700"
        >
          <label class="mb-3 flex cursor-pointer items-center gap-3">
            <input
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="isCategorySelected(category)"
              @change="toggleCategory(category, ($event.target as HTMLInputElement).checked)"
            />
            <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ categoryLabel(category) }}</span>
            <span class="text-xs text-gray-500 dark:text-dark-400">
              {{ t('modelPlaza.apiDocs.selectedEndpoints', { count: selectedEndpointCount(category) }) }}
            </span>
          </label>

          <div v-if="availableProfiles(category).length" class="space-y-2 pl-7">
            <div
              v-for="profile in availableProfiles(category)"
              :key="profile.id"
              class="border-l-2 border-gray-200 py-2 pl-4 dark:border-dark-700"
            >
              <div class="flex items-start justify-between gap-3">
                <label class="flex min-w-0 cursor-pointer items-start gap-3">
                  <input
                    type="checkbox"
                    class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                    :checked="isProfileSelected(profile.id)"
                    @change="toggleProfile(profile, ($event.target as HTMLInputElement).checked)"
                  />
                  <span class="min-w-0">
                    <span class="block text-sm font-medium text-gray-900 dark:text-white">{{ profile.title }}</span>
                    <span class="mt-0.5 block text-xs text-gray-500 dark:text-dark-400">{{ profile.description }}</span>
                  </span>
                </label>
                <div v-if="isProfileSelected(profile.id)" class="flex shrink-0 gap-1">
                  <button
                    type="button"
                    class="btn btn-ghost btn-icon btn-sm"
                    :title="t('modelPlaza.apiDocs.moveUp')"
                    @click="moveProfile(profile.id, -1)"
                  >
                    <Icon name="arrowUp" size="xs" />
                  </button>
                  <button
                    type="button"
                    class="btn btn-ghost btn-icon btn-sm"
                    :title="t('modelPlaza.apiDocs.moveDown')"
                    @click="moveProfile(profile.id, 1)"
                  >
                    <Icon name="arrowDown" size="xs" />
                  </button>
                </div>
              </div>

              <div v-if="isProfileSelected(profile.id)" class="mt-3 flex flex-wrap gap-x-5 gap-y-2 pl-7">
                <label
                  v-for="variant in profile.variants"
                  :key="variant.id"
                  class="flex cursor-pointer items-center gap-2 text-xs text-gray-700 dark:text-gray-300"
                >
                  <input
                    :data-testid="`variant-${profile.id}-${variant.id}`"
                    type="checkbox"
                    class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                    :checked="isVariantSelected(profile.id, variant.id)"
                    @change="toggleVariant(profile, variant.id, ($event.target as HTMLInputElement).checked)"
                  />
                  {{ modeLabel(variant.mode) }} · {{ transportLabel(variant.transport, variant.delivery) }}
                </label>
              </div>
            </div>
          </div>
        </section>
      </div>

      <div v-else-if="viewCategories.length === 0" class="flex min-h-48 flex-col items-center justify-center gap-3 text-center">
        <Icon name="document" size="xl" class="text-gray-300 dark:text-dark-600" />
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('modelPlaza.apiDocs.empty') }}</p>
      </div>

      <div v-else>
        <div class="mb-4 flex gap-2 overflow-x-auto border-b border-gray-200 pb-3 dark:border-dark-700">
          <button
            v-for="category in viewCategories"
            :key="category"
            type="button"
            class="inline-flex h-9 shrink-0 items-center rounded-md px-3 text-sm font-medium transition-colors"
            :class="activeCategory === category ? 'bg-primary-600 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
            @click="selectCategory(category)"
          >
            {{ categoryLabel(category) }}
          </button>
        </div>

        <div class="grid min-h-[28rem] gap-5 lg:grid-cols-[15rem_minmax(0,1fr)]">
          <nav class="space-y-1 border-b border-gray-200 pb-4 lg:border-b-0 lg:border-r lg:pb-0 lg:pr-4 dark:border-dark-700">
            <button
              v-for="profile in profilesForActiveCategory"
              :key="profile.id"
              type="button"
              class="w-full rounded-md px-3 py-2 text-left text-sm transition-colors"
              :class="activeProfileID === profile.id ? 'bg-primary-50 font-medium text-primary-700 dark:bg-primary-950/30 dark:text-primary-300' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
              @click="selectProfile(profile)"
            >
              {{ profile.title }}
            </button>
          </nav>

          <section v-if="activeProfile && activeVariant" class="min-w-0">
            <div class="mb-4">
              <h4 class="text-base font-semibold text-gray-900 dark:text-white">{{ activeProfile.title }}</h4>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ activeProfile.description }}</p>
            </div>

            <div class="mb-5 flex flex-wrap gap-2">
              <button
                v-for="variant in activeProfile.variants"
                :key="variant.id"
                type="button"
                class="inline-flex h-8 items-center rounded-md border px-3 text-xs font-medium transition-colors"
                :class="activeVariantID === variant.id ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-950/30 dark:text-primary-300' : 'border-gray-200 text-gray-600 hover:border-primary-300 dark:border-dark-700 dark:text-gray-300'"
                @click="activeVariantID = variant.id"
              >
                {{ modeLabel(variant.mode) }}
              </button>
            </div>

            <div class="mb-5 grid gap-3 sm:grid-cols-3">
              <div class="border-l-2 border-primary-500 pl-3">
                <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.apiDocs.mode') }}</p>
                <p class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ modeLabel(activeVariant.mode) }}</p>
              </div>
              <div class="border-l-2 border-emerald-500 pl-3">
                <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.apiDocs.transport') }}</p>
                <p class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ activeVariant.transport.toUpperCase() }}</p>
              </div>
              <div class="border-l-2 border-amber-500 pl-3">
                <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('modelPlaza.apiDocs.delivery') }}</p>
                <p class="mt-1 text-sm font-medium uppercase text-gray-900 dark:text-white">{{ activeVariant.delivery.replace('_', ' ') }}</p>
              </div>
            </div>

            <div class="space-y-6">
              <section v-for="(step, index) in activeVariant.steps" :key="step.id" class="border-t border-gray-200 pt-5 first:border-t-0 first:pt-0 dark:border-dark-700">
                <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
                  <div class="flex items-center gap-2">
                    <span v-if="activeVariant.steps.length > 1" class="flex h-6 w-6 items-center justify-center rounded-full bg-primary-600 text-xs font-semibold text-white">{{ index + 1 }}</span>
                    <h5 class="text-sm font-semibold text-gray-900 dark:text-white">{{ step.title }}</h5>
                  </div>
                  <span class="rounded-md bg-gray-100 px-2 py-1 font-mono text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-200">
                    {{ step.method }}
                  </span>
                </div>

                <div class="mb-4 flex items-center gap-2 rounded-md bg-gray-950 px-3 py-2 text-gray-100">
                  <code class="min-w-0 flex-1 overflow-x-auto whitespace-nowrap text-xs">{{ endpointURL(step.path, activeVariant.transport) }}</code>
                  <button type="button" class="shrink-0 text-gray-400 hover:text-white" :title="t('modelPlaza.apiDocs.copy')" @click="copyText(endpointURL(step.path, activeVariant.transport))">
                    <Icon name="copy" size="sm" />
                  </button>
                </div>

                <div v-if="step.parameters?.length" class="mb-4 min-w-0 overflow-x-auto">
                  <p class="mb-2 text-xs font-medium text-gray-500 dark:text-dark-400">
                    {{ t('modelPlaza.apiDocs.parameters') }}
                  </p>
                  <table class="w-full min-w-[38rem] text-left text-xs">
                    <thead class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                      <tr>
                        <th class="px-2 py-2 font-medium">{{ t('modelPlaza.apiDocs.parameterName') }}</th>
                        <th class="px-2 py-2 font-medium">{{ t('modelPlaza.apiDocs.parameterLocation') }}</th>
                        <th class="px-2 py-2 font-medium">{{ t('modelPlaza.apiDocs.parameterType') }}</th>
                        <th class="px-2 py-2 font-medium">{{ t('modelPlaza.apiDocs.parameterRequirement') }}</th>
                        <th class="px-2 py-2 font-medium">{{ t('modelPlaza.apiDocs.parameterDescription') }}</th>
                      </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-100 text-gray-700 dark:divide-dark-700 dark:text-gray-300">
                      <tr v-for="parameter in step.parameters" :key="`${parameter.location}-${parameter.name}`">
                        <td class="px-2 py-2 font-mono text-gray-900 dark:text-white">{{ parameter.name }}</td>
                        <td class="px-2 py-2">{{ parameterLocationLabel(parameter.location) }}</td>
                        <td class="px-2 py-2 font-mono">{{ parameter.type }}</td>
                        <td class="px-2 py-2">
                          {{ t(parameter.required ? 'modelPlaza.apiDocs.required' : 'modelPlaza.apiDocs.optional') }}
                        </td>
                        <td class="px-2 py-2">{{ parameter.description }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>

                <CodeExample
                  :title="t('modelPlaza.apiDocs.requestExample')"
                  :copy-label="t('modelPlaza.apiDocs.copy')"
                  :value="requestCode(step, activeVariant)"
                  @copy="copyText"
                />
                <CodeExample
                  v-if="step.response_example"
                  class="mt-4"
                  :title="t('modelPlaza.apiDocs.responseExample')"
                  :copy-label="t('modelPlaza.apiDocs.copy')"
                  :value="step.response_example"
                  @copy="copyText"
                />
              </section>
            </div>

            <div v-if="activeVariant.termination" class="mt-5 border-l-2 border-primary-500 pl-3">
              <p class="text-xs font-medium text-gray-500 dark:text-dark-400">
                {{ t('modelPlaza.apiDocs.termination') }}
              </p>
              <p class="mt-1 text-sm text-gray-700 dark:text-gray-300">{{ activeVariant.termination }}</p>
            </div>
          </section>
        </div>
      </div>
    </template>

    <template v-if="docs && editing" #footer>
      <div class="flex w-full flex-wrap items-center justify-between gap-3">
        <button
          v-if="docs.source === 'administrator'"
          type="button"
          class="btn btn-ghost btn-md text-red-600 dark:text-red-400"
          :disabled="saving"
          @click="resetDocs"
        >
          <Icon name="refresh" size="sm" />
          {{ t('modelPlaza.apiDocs.reset') }}
        </button>
        <span v-else></span>
        <div class="flex gap-2">
          <button type="button" class="btn btn-secondary btn-md" :disabled="saving" @click="cancelEditing">
            {{ t('modelPlaza.apiDocs.cancel') }}
          </button>
          <button type="button" class="btn btn-primary btn-md" :disabled="saving || !hasDraftEndpoints" @click="saveDocs">
            <Icon v-if="saving" name="refresh" size="sm" class="animate-spin" />
            {{ t('modelPlaza.apiDocs.save') }}
          </button>
        </div>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, h, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { modelApiDocsAPI } from '@/api/modelApiDocs'
import type {
  ModelAPIDocsBinding,
  ModelAPIDocsCategory,
  ModelAPIDocsEndpointProfile,
  ModelAPIDocsEndpointVariant,
  ModelAPIDocsResponse,
  ModelAPIDocsWorkflowStep
} from '@/api/modelApiDocs'

const CodeExample = (
  props: { title: string; copyLabel: string; value: string },
  { emit }: { emit: (event: 'copy', value: string) => void }
) => h('div', { class: 'min-w-0' }, [
  h('div', { class: 'mb-2 flex items-center justify-between gap-3' }, [
    h('p', { class: 'text-xs font-medium text-gray-500 dark:text-dark-400' }, props.title),
    h('button', {
      type: 'button',
      class: 'text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400',
      onClick: () => emit('copy', props.value)
    }, props.copyLabel)
  ]),
  h('pre', { class: 'max-h-72 overflow-auto rounded-md bg-gray-950 p-4 text-xs leading-5 text-gray-100' }, [
    h('code', props.value)
  ])
])

const props = defineProps<{
  show: boolean
  modelName: string
  documentation: ModelAPIDocsResponse
}>()

const emit = defineEmits<{
  (event: 'close'): void
  (event: 'updated', documentation: ModelAPIDocsResponse): void
}>()
const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const saving = ref(false)
const docs = ref<ModelAPIDocsResponse | null>(null)
const editing = ref(false)
const draft = ref<ModelAPIDocsBinding | null>(null)
const activeCategory = ref<ModelAPIDocsCategory | ''>('')
const activeProfileID = ref('')
const activeVariantID = ref('')

const categoryOrder: ModelAPIDocsCategory[] = ['conversation', 'image', 'video', 'tts', 'asr']

const publicBaseURL = computed(() => window.location.origin.replace(/\/$/, ''))
const websocketBaseURL = computed(() => publicBaseURL.value.replace(/^http/i, 'ws'))
const authorizationHeader = 'Authorization: Bearer $XIMOAI_API_KEY'
const referenceValues = computed(() => [
  { id: 'base-url', label: t('modelPlaza.apiDocs.baseURL'), value: publicBaseURL.value },
  { id: 'model-name', label: t('modelPlaza.apiDocs.modelName'), value: props.modelName },
  { id: 'auth-header', label: t('modelPlaza.apiDocs.authorization'), value: authorizationHeader }
])
const canEdit = computed(() => Boolean(authStore.isAdmin && docs.value?.editor))
const sourceLabel = computed(() => docs.value?.source === 'administrator'
  ? t('modelPlaza.apiDocs.sourceAdministrator')
  : t('modelPlaza.apiDocs.sourceAutomatic'))
const sourceBadgeClass = computed(() => docs.value?.source === 'administrator'
  ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
  : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300')

const viewBinding = computed(() => editing.value && draft.value ? draft.value : docs.value?.binding)
const availableProfileList = computed(() => docs.value?.editor?.available_profiles || docs.value?.profiles || [])
const viewProfiles = computed(() => {
  if (!viewBinding.value) return []
  const byID = new Map(availableProfileList.value.map((profile) => [profile.id, profile]))
  const out: ModelAPIDocsEndpointProfile[] = []
  for (const category of viewBinding.value.categories) {
    for (const endpoint of category.endpoints) {
      const profile = byID.get(endpoint.profile)
      if (!profile) continue
      const variants = profile.variants.filter((variant) => endpoint.variants.includes(variant.id))
      out.push({ ...profile, variants })
    }
  }
  return out
})
const viewCategories = computed(() => categoryOrder.filter((category) => viewProfiles.value.some((profile) => profile.category === category)))
const editableCategories = computed(() => categoryOrder.filter((category) => availableProfileList.value.some((profile) => profile.category === category)))
const profilesForActiveCategory = computed(() => viewProfiles.value.filter((profile) => profile.category === activeCategory.value))
const activeProfile = computed(() => profilesForActiveCategory.value.find((profile) => profile.id === activeProfileID.value) || profilesForActiveCategory.value[0] || null)
const activeVariant = computed(() => activeProfile.value?.variants.find((variant) => variant.id === activeVariantID.value) || activeProfile.value?.variants[0] || null)
const hasDraftEndpoints = computed(() => Boolean(draft.value?.categories.some((category) => category.endpoints.some((endpoint) => endpoint.variants.length > 0))))

function target() {
  return {
    platform: props.documentation.platform,
    protocol: props.documentation.protocol,
    model: props.modelName
  }
}

function loadFromProps() {
  if (!props.show) return
  docs.value = cloneDocumentation(props.documentation)
  editing.value = false
  draft.value = null
  setInitialSelection(docs.value)
}

function setInitialSelection(value: ModelAPIDocsResponse) {
  const category = categoryOrder.find((item) => value.profiles.some((profile) => profile.category === item)) || ''
  activeCategory.value = category
  const profile = value.profiles.find((item) => item.category === category)
  activeProfileID.value = profile?.id || ''
  activeVariantID.value = profile?.variants[0]?.id || ''
}

function selectCategory(category: ModelAPIDocsCategory) {
  activeCategory.value = category
  const profile = viewProfiles.value.find((item) => item.category === category)
  activeProfileID.value = profile?.id || ''
  activeVariantID.value = profile?.variants[0]?.id || ''
}

function selectProfile(profile: ModelAPIDocsEndpointProfile) {
  activeProfileID.value = profile.id
  activeVariantID.value = profile.variants[0]?.id || ''
}

function startEditing() {
  if (!docs.value?.editor) return
  draft.value = cloneBinding(docs.value.binding)
  editing.value = true
}

function cloneBinding(binding: ModelAPIDocsBinding): ModelAPIDocsBinding {
  return JSON.parse(JSON.stringify(binding)) as ModelAPIDocsBinding
}

function cloneDocumentation(documentation: ModelAPIDocsResponse): ModelAPIDocsResponse {
  return JSON.parse(JSON.stringify(documentation)) as ModelAPIDocsResponse
}

function cancelEditing() {
  editing.value = false
  draft.value = null
}

function availableProfiles(category: ModelAPIDocsCategory) {
  return availableProfileList.value.filter((profile) => profile.category === category)
}

function findDraftEndpoint(profileID: string) {
  for (const category of draft.value?.categories || []) {
    const endpoint = category.endpoints.find((item) => item.profile === profileID)
    if (endpoint) return endpoint
  }
  return null
}

function isCategorySelected(category: ModelAPIDocsCategory) {
  return Boolean(draft.value?.categories.some((item) => item.category === category && item.endpoints.length > 0))
}

function isProfileSelected(profileID: string) {
  return Boolean(findDraftEndpoint(profileID))
}

function isVariantSelected(profileID: string, variantID: string) {
  return Boolean(findDraftEndpoint(profileID)?.variants.includes(variantID))
}

function selectedEndpointCount(category: ModelAPIDocsCategory) {
  return draft.value?.categories.find((item) => item.category === category)?.endpoints.length || 0
}

function ensureDraftCategory(category: ModelAPIDocsCategory) {
  if (!draft.value) return null
  let binding = draft.value.categories.find((item) => item.category === category)
  if (!binding) {
    binding = { category, endpoints: [] }
    const desiredIndex = categoryOrder.indexOf(category)
    const insertAt = draft.value.categories.findIndex((item) => categoryOrder.indexOf(item.category) > desiredIndex)
    if (insertAt === -1) draft.value.categories.push(binding)
    else draft.value.categories.splice(insertAt, 0, binding)
  }
  return binding
}

function toggleCategory(category: ModelAPIDocsCategory, checked: boolean) {
  if (!draft.value) return
  if (!checked) {
    draft.value.categories = draft.value.categories.filter((item) => item.category !== category)
    return
  }
  const binding = ensureDraftCategory(category)
  if (!binding || binding.endpoints.length > 0) return
  binding.endpoints = availableProfiles(category).map((profile) => ({
    profile: profile.id,
    variants: profile.variants.map((variant) => variant.id)
  }))
}

function toggleProfile(profile: ModelAPIDocsEndpointProfile, checked: boolean) {
  if (!draft.value) return
  if (!checked) {
    for (const category of draft.value.categories) {
      category.endpoints = category.endpoints.filter((item) => item.profile !== profile.id)
    }
    draft.value.categories = draft.value.categories.filter((category) => category.endpoints.length > 0)
    return
  }
  const category = ensureDraftCategory(profile.category)
  if (!category || category.endpoints.some((item) => item.profile === profile.id)) return
  category.endpoints.push({ profile: profile.id, variants: profile.variants.map((variant) => variant.id) })
}

function toggleVariant(profile: ModelAPIDocsEndpointProfile, variantID: string, checked: boolean) {
  if (!draft.value) return
  if (!isProfileSelected(profile.id)) toggleProfile(profile, true)
  const endpoint = findDraftEndpoint(profile.id)
  if (!endpoint) return
  if (checked && !endpoint.variants.includes(variantID)) endpoint.variants.push(variantID)
  if (!checked) endpoint.variants = endpoint.variants.filter((item) => item !== variantID)
  if (endpoint.variants.length === 0) toggleProfile(profile, false)
}

function moveProfile(profileID: string, direction: -1 | 1) {
  for (const category of draft.value?.categories || []) {
    const index = category.endpoints.findIndex((item) => item.profile === profileID)
    if (index < 0) continue
    const targetIndex = index + direction
    if (targetIndex < 0 || targetIndex >= category.endpoints.length) return
    const [item] = category.endpoints.splice(index, 1)
    category.endpoints.splice(targetIndex, 0, item)
    return
  }
}

async function saveDocs() {
  const requestTarget = target()
  if (!requestTarget || !draft.value || !hasDraftEndpoints.value) return
  saving.value = true
  try {
    const binding = await modelApiDocsAPI.save(requestTarget, draft.value.categories)
    appStore.showSuccess(t('modelPlaza.apiDocs.saved'))
    applySavedBinding(binding, 'administrator')
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('modelPlaza.apiDocs.saveError')))
  } finally {
    saving.value = false
  }
}

async function resetDocs() {
  const requestTarget = target()
  if (!requestTarget) return
  saving.value = true
  try {
    await modelApiDocsAPI.reset(requestTarget)
    appStore.showSuccess(t('modelPlaza.apiDocs.resetDone'))
    if (docs.value?.editor) applySavedBinding(docs.value.editor.automatic_binding, 'automatic')
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('modelPlaza.apiDocs.resetError')))
  } finally {
    saving.value = false
  }
}

function applySavedBinding(binding: ModelAPIDocsBinding, source: ModelAPIDocsResponse['source']) {
  if (!docs.value) return
  const byID = new Map((docs.value.editor?.available_profiles || []).map((profile) => [profile.id, profile]))
  const profiles: ModelAPIDocsEndpointProfile[] = []
  for (const category of binding.categories) {
    for (const endpoint of category.endpoints) {
      const profile = byID.get(endpoint.profile)
      if (!profile) continue
      profiles.push({
        ...profile,
        variants: profile.variants.filter((variant) => endpoint.variants.includes(variant.id))
      })
    }
  }
  docs.value = { ...docs.value, source, binding: cloneBinding(binding), profiles }
  editing.value = false
  draft.value = null
  setInitialSelection(docs.value)
  emit('updated', cloneDocumentation(docs.value))
}

function endpointURL(path: string, transport: ModelAPIDocsEndpointVariant['transport']) {
  return `${transport === 'websocket' ? websocketBaseURL.value : publicBaseURL.value}${path}`
}

function requestCode(step: ModelAPIDocsWorkflowStep, variant: ModelAPIDocsEndpointVariant) {
  const targetURL = endpointURL(step.path, variant.transport)
  if (variant.transport === 'websocket') {
    return `import asyncio\nimport os\nimport websockets\n\nasync def main():\n    headers = {"Authorization": f"Bearer {os.environ['XIMOAI_API_KEY']}"}\n    async with websockets.connect("${targetURL}", additional_headers=headers) as ws:\n        # ${step.request_example || 'Send provider-native WebSocket frames.'}\n        response = await ws.recv()\n        print(response)\n\nasyncio.run(main())`
  }
  const lines = [
    `curl -X ${step.method} '${targetURL}'`,
    `  -H 'Authorization: Bearer $XIMOAI_API_KEY'`
  ]
  if (step.content_type === 'multipart/form-data') {
    lines.push(`  -F 'file=@audio.mp3'`)
    lines.push(`  -F 'model=${props.modelName.replace(/'/g, `'\\''`)}'`)
    return lines.join(' \\\n')
  }
  if (step.content_type) lines.push(`  -H 'Content-Type: ${step.content_type}'`)
  if (step.request_example) {
    const escaped = step.request_example.replace(/'/g, `'\\''`)
    lines.push(`  --data '${escaped}'`)
  }
  return lines.join(' \\\n')
}

function categoryLabel(category: ModelAPIDocsCategory) {
  return t(`modelPlaza.apiDocs.categories.${category}`)
}

function modeLabel(mode: ModelAPIDocsEndpointVariant['mode']) {
  return t(`modelPlaza.apiDocs.modes.${mode}`)
}

function transportLabel(transport: ModelAPIDocsEndpointVariant['transport'], delivery: ModelAPIDocsEndpointVariant['delivery']) {
  return `${transport.toUpperCase()} / ${delivery.replace('_', ' ').toUpperCase()}`
}

function parameterLocationLabel(location: string) {
  return t(`modelPlaza.apiDocs.parameterLocations.${location}`)
}

async function copyText(value: string) {
  await navigator.clipboard.writeText(value)
  appStore.showSuccess(t('modelPlaza.apiDocs.copied'))
}

watch(
  () => [props.show, props.modelName, props.documentation],
  ([show]) => {
    if (show) loadFromProps()
  },
  { immediate: true }
)
</script>
