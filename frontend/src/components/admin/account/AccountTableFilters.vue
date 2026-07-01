<template>
  <div class="flex flex-wrap items-center gap-3">
    <SearchInput
      :model-value="searchQuery"
      :placeholder="t('admin.accounts.searchAccounts')"
      class="w-full sm:w-64"
      @update:model-value="$emit('update:searchQuery', $event)"
      @search="$emit('change')"
    />
    <Select :model-value="filters.platform" class="w-40" :options="pOpts" @update:model-value="updatePlatform" @change="$emit('change')" />
    <Select :model-value="filters.type" class="w-40" :options="tOpts" @update:model-value="updateType" @change="$emit('change')" />
    <Select :model-value="filters.status" class="w-40" :options="sOpts" @update:model-value="updateStatus" @change="$emit('change')" />
    <Select :model-value="filters.privacy_mode" class="w-40" :options="privacyOpts" @update:model-value="updatePrivacyMode" @change="$emit('change')" />
    <Select :model-value="filters.group" class="w-40" :options="gOpts" @update:model-value="updateGroup" @change="$emit('change')" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'; import { useI18n } from 'vue-i18n'; import Select from '@/components/common/Select.vue'; import SearchInput from '@/components/common/SearchInput.vue'
import { adminAPI } from '@/api/admin'
import type { AdminGroup, Platform } from '@/types'
const props = defineProps<{ searchQuery: string; filters: Record<string, any>; groups?: AdminGroup[] }>()
const emit = defineEmits(['update:searchQuery', 'update:filters', 'change']); const { t } = useI18n()
const fallbackPlatforms: Platform[] = [
  { slug: 'anthropic', display_name: 'Anthropic', protocol: 'native', base_url: '', auth_modes: [], capabilities: [], color: '#D97706', enabled: true, builtin: true, created_at: '', updated_at: '' },
  { slug: 'openai', display_name: 'OpenAI', protocol: 'openai', base_url: 'https://api.openai.com', auth_modes: [], capabilities: [], color: '#10A37F', enabled: true, builtin: true, created_at: '', updated_at: '' },
  { slug: 'gemini', display_name: 'Gemini', protocol: 'native', base_url: '', auth_modes: [], capabilities: [], color: '#4285F4', enabled: true, builtin: true, created_at: '', updated_at: '' },
  { slug: 'antigravity', display_name: 'Antigravity', protocol: 'native', base_url: '', auth_modes: [], capabilities: [], color: '#7C3AED', enabled: true, builtin: true, created_at: '', updated_at: '' },
  { slug: 'grok', display_name: 'Grok', protocol: 'openai', base_url: 'https://api.x.ai', auth_modes: [], capabilities: [], color: '#18181B', enabled: true, builtin: true, created_at: '', updated_at: '' },
]
const platforms = ref<Platform[]>([...fallbackPlatforms])
const updatePlatform = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, platform: value }) }
const updateType = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, type: value }) }
const updateStatus = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, status: value }) }
const updatePrivacyMode = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, privacy_mode: value }) }
const updateGroup = (value: string | number | boolean | null) => { emit('update:filters', { ...props.filters, group: value }) }
const pOpts = computed(() => [
  { value: '', label: t('admin.accounts.allPlatforms') },
  ...platforms.value.map(platform => ({ value: platform.slug, label: platform.display_name }))
])
const tOpts = computed(() => [{ value: '', label: t('admin.accounts.allTypes') }, { value: 'oauth', label: t('admin.accounts.oauthType') }, { value: 'setup-token', label: t('admin.accounts.setupToken') }, { value: 'apikey', label: t('admin.accounts.apiKey') }, { value: 'bedrock', label: 'AWS Bedrock' }])
const sOpts = computed(() => [{ value: '', label: t('admin.accounts.allStatus') }, { value: 'active', label: t('admin.accounts.status.active') }, { value: 'inactive', label: t('admin.accounts.status.inactive') }, { value: 'error', label: t('admin.accounts.status.error') }, { value: 'rate_limited', label: t('admin.accounts.status.rateLimited') }, { value: 'temp_unschedulable', label: t('admin.accounts.status.tempUnschedulable') }, { value: 'unschedulable', label: t('admin.accounts.status.unschedulable') }])
const privacyOpts = computed(() => [
  { value: '', label: t('admin.accounts.allPrivacyModes') },
  { value: '__unset__', label: t('admin.accounts.privacyUnset') },
  { value: 'training_off', label: 'Privacy' },
  { value: 'training_set_cf_blocked', label: 'CF' },
  { value: 'training_set_failed', label: 'Fail' }
])
const gOpts = computed(() => [
  { value: '', label: t('admin.accounts.allGroups') },
  { value: 'ungrouped', label: t('admin.accounts.ungroupedGroup') },
  ...(props.groups || []).map(g => ({ value: String(g.id), label: g.name }))
])

onMounted(async () => {
  try {
    const items = await adminAPI.platforms.list(true)
    if (items.length > 0) platforms.value = items
  } catch {
    platforms.value = [...fallbackPlatforms]
  }
})
</script>
