<template>
  <BaseDialog
    :show="show"
    :title="t('modelPlaza.metadataEditor.title', { model: modelName })"
    width="wide"
    @close="emit('close')"
  >
    <div class="space-y-6">
      <section>
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <label for="model-brand" class="label mb-0">{{ t('modelPlaza.metadataEditor.brand') }}</label>
          <label class="flex cursor-pointer items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
            <input v-model="brandAutomatic" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            {{ t('modelPlaza.metadataEditor.useAutomatic') }}
          </label>
        </div>
        <input
          id="model-brand"
          v-model="draftBrand"
          data-testid="model-brand-input"
          class="input"
          type="text"
          maxlength="64"
          autocomplete="off"
          :disabled="brandAutomatic"
        />
        <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">
          {{ t('modelPlaza.metadataEditor.automaticValue') }}: {{ editor.automatic.brand }}
        </p>
      </section>

      <section class="border-t border-gray-200 pt-5 dark:border-dark-700">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <span class="label mb-0">{{ t('modelPlaza.modelType') }}</span>
          <label class="flex cursor-pointer items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
            <input v-model="typesAutomatic" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            {{ t('modelPlaza.metadataEditor.useAutomatic') }}
          </label>
        </div>
        <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          <label
            v-for="type in editor.options.types"
            :key="type"
            class="flex min-h-10 cursor-pointer items-center gap-2 rounded-md border border-gray-200 px-3 py-2 text-sm text-gray-700 dark:border-dark-700 dark:text-gray-300"
          >
            <input
              :data-metadata-type="type"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="selectedTypes.includes(type)"
              :disabled="typesAutomatic"
              @change="toggleType(type)"
            />
            {{ typeLabel(type) }}
          </label>
        </div>
      </section>

      <section class="border-t border-gray-200 pt-5 dark:border-dark-700">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <span class="label mb-0">{{ t('modelPlaza.metadataEditor.reasoningLevels') }}</span>
          <label class="flex cursor-pointer items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
            <input v-model="reasoningAutomatic" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            {{ t('modelPlaza.metadataEditor.useAutomatic') }}
          </label>
        </div>
        <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          <label
            v-for="level in editor.options.reasoning_levels || []"
            :key="level"
            class="flex min-h-10 cursor-pointer items-center gap-2 rounded-md border border-gray-200 px-3 py-2 text-sm text-gray-700 dark:border-dark-700 dark:text-gray-300"
          >
            <input
              :data-metadata-reasoning="level"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="selectedReasoningLevels.includes(level)"
              :disabled="reasoningAutomatic"
              @change="toggleReasoningLevel(level)"
            />
            {{ level }}
          </label>
        </div>
      </section>

      <section class="border-t border-gray-200 pt-5 dark:border-dark-700">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <span class="label mb-0">{{ t('modelPlaza.metadataEditor.thinkingSupport') }}</span>
          <label class="flex cursor-pointer items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
            <input v-model="thinkingAutomatic" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            {{ t('modelPlaza.metadataEditor.useAutomatic') }}
          </label>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input
            data-metadata-thinking
            type="checkbox"
            class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            v-model="selectedThinking"
            :disabled="thinkingAutomatic"
          />
          {{ t('modelPlaza.metadataEditor.thinkingEnabled') }}
        </label>
      </section>

      <section class="border-t border-gray-200 pt-5 dark:border-dark-700">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <span class="label mb-0">{{ t('modelPlaza.invocationMode') }}</span>
          <label class="flex cursor-pointer items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
            <input v-model="modesAutomatic" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            {{ t('modelPlaza.metadataEditor.useAutomatic') }}
          </label>
        </div>
        <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          <label
            v-for="mode in editor.options.invocation_modes"
            :key="mode"
            class="flex min-h-10 cursor-pointer items-center gap-2 rounded-md border border-gray-200 px-3 py-2 text-sm text-gray-700 dark:border-dark-700 dark:text-gray-300"
          >
            <input
              :data-metadata-mode="mode"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="selectedModes.includes(mode)"
              :disabled="modesAutomatic"
              @change="toggleMode(mode)"
            />
            {{ modeLabel(mode) }}
          </label>
        </div>
      </section>
    </div>

    <template #footer>
      <div class="flex w-full flex-wrap items-center justify-between gap-3">
        <button
          v-if="hasOverride"
          type="button"
          class="btn btn-ghost btn-md text-red-600 dark:text-red-400"
          :disabled="saving"
          @click="resetMetadata"
        >
          <Icon name="refresh" size="sm" />
          {{ t('modelPlaza.metadataEditor.resetAll') }}
        </button>
        <span v-else></span>
        <div class="flex gap-2">
          <button type="button" class="btn btn-secondary btn-md" :disabled="saving" @click="emit('close')">
            {{ t('modelPlaza.metadataEditor.cancel') }}
          </button>
          <button type="button" class="btn btn-primary btn-md" :disabled="saving || !canSave" @click="saveMetadata">
            <Icon v-if="saving" name="refresh" size="sm" class="animate-spin" />
            {{ t('modelPlaza.metadataEditor.save') }}
          </button>
        </div>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  modelMetadataAPI,
  type ModelInvocationMode,
  type ModelMetadataEditor,
  type ModelMetadataOverride,
  type ModelMetadataState,
  type ModelReasoningLevel,
  type ModelType
} from '@/api/modelMetadata'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{
  show: boolean
  platform: string
  modelName: string
  brand: string
  types: ModelType[]
  invocationModes: ModelInvocationMode[]
  editor: ModelMetadataEditor
}>()

const emit = defineEmits<{
  (event: 'close'): void
  (event: 'updated', state: ModelMetadataState): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const saving = ref(false)
const brandAutomatic = ref(true)
const typesAutomatic = ref(true)
const modesAutomatic = ref(true)
const reasoningAutomatic = ref(true)
const thinkingAutomatic = ref(true)
const draftBrand = ref('')
const selectedTypes = ref<ModelType[]>([])
const selectedModes = ref<ModelInvocationMode[]>([])
const selectedReasoningLevels = ref<ModelReasoningLevel[]>([])
const selectedThinking = ref(false)

const hasOverride = computed(() => Boolean(props.editor.override && Object.keys(props.editor.override).length > 0))
const canSave = computed(() => {
  if (!brandAutomatic.value && draftBrand.value.trim().length === 0) return false
  if (!typesAutomatic.value && selectedTypes.value.length === 0) return false
  if (!modesAutomatic.value && selectedModes.value.length === 0) return false
  if (!reasoningAutomatic.value && selectedReasoningLevels.value.length === 0) return false
  return true
})

function target() {
  return { platform: props.platform, model: props.modelName }
}

function toggleType(type: ModelType) {
  if (typesAutomatic.value) return
  selectedTypes.value = selectedTypes.value.includes(type)
    ? selectedTypes.value.filter((item) => item !== type)
    : [...selectedTypes.value, type]
}

function toggleMode(mode: ModelInvocationMode) {
  if (modesAutomatic.value) return
  selectedModes.value = selectedModes.value.includes(mode)
    ? selectedModes.value.filter((item) => item !== mode)
    : [...selectedModes.value, mode]
}

function toggleReasoningLevel(level: ModelReasoningLevel) {
  if (reasoningAutomatic.value) return
  selectedReasoningLevels.value = selectedReasoningLevels.value.includes(level)
    ? selectedReasoningLevels.value.filter((item) => item !== level)
    : [...selectedReasoningLevels.value, level]
}

function buildOverride(): ModelMetadataOverride {
  const override: ModelMetadataOverride = {}
  if (!brandAutomatic.value) override.brand = draftBrand.value.trim()
  if (!typesAutomatic.value) override.types = [...selectedTypes.value]
  if (!modesAutomatic.value) override.invocation_modes = [...selectedModes.value]
  if (!reasoningAutomatic.value) override.reasoning_levels = [...selectedReasoningLevels.value]
  if (!thinkingAutomatic.value) override.thinking_supported = selectedThinking.value
  return override
}

function stateFromOverride(override: ModelMetadataOverride | null): ModelMetadataState {
  return {
    brand: override?.brand ?? props.editor.automatic.brand,
    types: override?.types ?? [...props.editor.automatic.types],
    invocation_modes: override?.invocation_modes ?? [...props.editor.automatic.invocation_modes],
    reasoning_levels: override?.reasoning_levels ?? [...(props.editor.automatic.reasoning_levels ?? [])],
    thinking_supported: override?.thinking_supported ?? Boolean(props.editor.automatic.thinking_supported),
    editor: { ...props.editor, override }
  }
}

async function saveMetadata() {
  if (!canSave.value) return
  saving.value = true
  try {
    const override = buildOverride()
    const saved = await modelMetadataAPI.save(target(), override)
    appStore.showSuccess(t('modelPlaza.metadataEditor.saved'))
    emit('updated', stateFromOverride(Object.keys(saved).length > 0 ? saved : null))
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('modelPlaza.metadataEditor.saveError')))
  } finally {
    saving.value = false
  }
}

async function resetMetadata() {
  saving.value = true
  try {
    await modelMetadataAPI.reset(target())
    appStore.showSuccess(t('modelPlaza.metadataEditor.resetDone'))
    emit('updated', stateFromOverride(null))
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('modelPlaza.metadataEditor.resetError')))
  } finally {
    saving.value = false
  }
}

function typeLabel(type: ModelType) {
  return t(`modelPlaza.types.${type}`)
}

function modeLabel(mode: ModelInvocationMode) {
  return t(`modelPlaza.modes.${mode}`)
}

function loadDraft() {
  const override = props.editor.override
  brandAutomatic.value = override?.brand == null
  typesAutomatic.value = override?.types == null
  modesAutomatic.value = override?.invocation_modes == null
  reasoningAutomatic.value = override?.reasoning_levels == null
  thinkingAutomatic.value = override?.thinking_supported == null
  draftBrand.value = override?.brand ?? props.editor.automatic.brand
  selectedTypes.value = [...(override?.types ?? props.editor.automatic.types)]
  selectedModes.value = [...(override?.invocation_modes ?? props.editor.automatic.invocation_modes)]
  selectedReasoningLevels.value = [...(override?.reasoning_levels ?? props.editor.automatic.reasoning_levels ?? [])]
  selectedThinking.value = override?.thinking_supported ?? Boolean(props.editor.automatic.thinking_supported)
}

watch(
  () => [props.show, props.editor],
  ([show]) => {
    if (show) loadDraft()
  },
  { immediate: true, deep: true }
)
</script>
