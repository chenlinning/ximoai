<template>
  <AppLayout>
    <div class="mx-auto w-full max-w-7xl space-y-5">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('adminMemberships.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            {{ t('adminMemberships.description') }}
          </p>
        </div>
        <div class="flex flex-wrap gap-3">
          <button class="btn btn-secondary" :disabled="loading" @click="loadAll">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </div>

      <div class="membership-table-card membership-levels-card rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <DataTable :columns="columns" :data="levels" :loading="loading">
          <template #cell-name="{ row }">
            <div class="space-y-1">
              <div class="flex flex-wrap items-center gap-2">
                <MembershipLevelMark :code="row.code" :color="row.color" size="md" />
                <span class="font-medium text-gray-900 dark:text-white">{{ membershipLevelDisplayName(row.code, row.name) }}</span>
                <span v-if="row.is_default" class="badge badge-primary">{{ t('adminMemberships.default') }}</span>
              </div>
              <div class="text-xs text-gray-500 dark:text-dark-400">{{ row.code }}</div>
            </div>
          </template>

          <template #cell-discount_rate="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">{{ formatRate(value) }}</span>
          </template>

          <template #cell-groups="{ row }">
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="group in row.groups || []"
                :key="group.id"
                class="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-700 dark:bg-dark-700 dark:text-dark-300"
              >
                {{ group.name }}
              </span>
              <span v-if="!row.groups?.length" class="text-sm text-gray-400">{{ t('adminMemberships.unconfigured') }}</span>
            </div>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex flex-wrap items-center gap-1">
              <button class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700" @click="openEdit(row)">
                <Icon name="edit" size="sm" />
              </button>
              <button class="rounded-lg p-1.5 text-gray-500 hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20" @click="syncLevel(row)">
                <Icon name="sync" size="sm" />
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState :title="t('adminMemberships.noLevelsTitle')" :description="t('adminMemberships.noLevelsDescription')" />
          </template>
        </DataTable>
      </div>

      <section class="w-full rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-5 lg:items-start">
          <div class="sm:col-span-2 lg:col-span-1">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('adminMemberships.assignTitle') }}</h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('adminMemberships.assignDescription') }}</p>
          </div>
          <div>
            <label class="input-label">{{ t('adminMemberships.userId') }}</label>
            <input v-model.number="assignForm.user_id" type="number" min="1" class="input" :placeholder="t('adminMemberships.userId')" />
          </div>
          <div>
            <label class="input-label">{{ t('adminMemberships.level') }}</label>
            <Select v-model="assignForm.membership_level_id" :options="levelOptions" :placeholder="t('adminMemberships.selectLevel')" />
          </div>
          <div>
            <label class="input-label">{{ t('adminMemberships.expiresAt') }}</label>
            <input v-model="assignForm.expires_at" type="datetime-local" class="input" />
            <p class="input-hint">{{ t('adminMemberships.emptyExpiresAtHint') }}</p>
          </div>
          <div class="lg:pt-6">
            <button class="btn btn-primary h-11 w-full whitespace-nowrap" :disabled="assigning" @click="assignMembership">{{ t('adminMemberships.assignButton') }}</button>
          </div>
        </div>
      </section>

      <div class="membership-table-card membership-assignments-card rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('adminMemberships.assignmentsTitle') }}</h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('adminMemberships.assignmentsDescription') }}</p>
        </div>
        <DataTable :columns="assignmentColumns" :data="assignments" :loading="loading">
          <template #cell-user="{ row }">
            <div class="space-y-1">
              <div class="font-medium text-gray-900 dark:text-white">
                {{ row.user?.email || row.user?.username || t('adminMemberships.userFallback', { id: row.user_id }) }}
              </div>
              <div class="text-xs text-gray-500 dark:text-dark-400">ID {{ row.user_id }}</div>
            </div>
          </template>

          <template #cell-level="{ row }">
            <div class="space-y-1">
              <div class="flex items-center gap-2 font-medium text-gray-900 dark:text-white">
                <MembershipLevelMark
                  v-if="row.level"
                  :code="row.level.code"
                  :color="row.level.color"
                  size="sm"
                />
                <span>{{ row.level ? membershipLevelDisplayName(row.level.code, row.level.name) : t('adminMemberships.levelFallback', { id: row.membership_level_id }) }}</span>
              </div>
              <div class="text-xs text-gray-500 dark:text-dark-400">
                {{ row.level?.code || '-' }} · {{ t('adminMemberships.discount') }} {{ formatRate(row.level?.discount_rate ?? 1) }}
              </div>
            </div>
          </template>

          <template #cell-source="{ value }">
            <span class="badge badge-gray">{{ sourceText(value) }}</span>
          </template>

          <template #cell-starts_at="{ value }">
            <span class="text-sm text-gray-700 dark:text-dark-200">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-expires_at="{ value }">
            <span class="text-sm text-gray-700 dark:text-dark-200">{{ value ? formatDateTime(value) : t('adminMemberships.longTerm') }}</span>
          </template>

          <template #cell-actions="{ row }">
            <button
              class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700"
              :title="t('adminMemberships.editLevelTitle')"
              @click="openAssignmentEdit(row)"
            >
              <Icon name="edit" size="sm" />
            </button>
          </template>

          <template #empty>
            <EmptyState :title="t('adminMemberships.noAssignmentsTitle')" :description="t('adminMemberships.noAssignmentsDescription')" />
          </template>
        </DataTable>
      </div>
    </div>

    <BaseDialog :show="showEditor" :title="t('adminMemberships.configureLevel')" width="wide" @close="closeEditor">
      <form id="membership-form" class="space-y-5" @submit.prevent="saveLevel">
        <div
          v-if="editingLevel"
          class="flex items-center gap-3 rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-900"
        >
          <MembershipLevelMark :code="editingLevel.code" :color="editingLevel.color" size="xl" />
          <div>
            <div class="font-semibold text-gray-900 dark:text-white">
              {{ membershipLevelDisplayName(editingLevel.code, editingLevel.name) }}
            </div>
            <div class="mt-0.5 text-xs font-mono text-gray-500 dark:text-dark-400">{{ editingLevel.code }}</div>
          </div>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('adminMemberships.columns.discountRate') }}</label>
            <input v-model.number="form.discount_rate" required type="number" min="0" step="0.0001" class="input" />
            <p class="input-hint">{{ t('adminMemberships.userRateFormula') }}</p>
          </div>
        </div>

        <div>
          <div class="mb-2 flex items-center justify-between">
            <label class="input-label mb-0">{{ t('adminMemberships.availableGroups') }}</label>
            <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('adminMemberships.selectedCount', { count: form.group_ids.length }) }}</span>
          </div>
          <div class="max-h-72 overflow-auto rounded-lg border border-gray-200 dark:border-dark-700">
            <label
              v-for="group in groups"
              :key="group.id"
              class="flex cursor-pointer items-center justify-between gap-3 border-b border-gray-100 px-3 py-2 text-sm last:border-b-0 hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-700"
            >
              <span>
                <span class="font-medium text-gray-900 dark:text-white">{{ group.name }}</span>
                <span class="ml-2 text-xs text-gray-500 dark:text-dark-400">
                  {{ group.platform }} · {{ group.rate_multiplier }}x · {{ group.is_exclusive ? t('adminMemberships.exclusive') : t('adminMemberships.public') }}
                </span>
              </span>
              <input :checked="form.group_ids.includes(group.id)" type="checkbox" class="rounded border-gray-300" @change="toggleGroup(group.id)" />
            </label>
            <div v-if="groups.length === 0" class="px-3 py-8 text-center text-sm text-gray-500 dark:text-dark-400">
              {{ t('adminMemberships.noSelectableGroups') }}
            </div>
          </div>
        </div>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="closeEditor">{{ t('adminMemberships.cancel') }}</button>
        <button type="submit" form="membership-form" class="btn btn-primary" :disabled="saving">
          {{ t('adminMemberships.save') }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="showAssignmentEditor" :title="t('adminMemberships.editUserLevel')" @close="closeAssignmentEditor">
      <form id="assignment-form" class="space-y-5" @submit.prevent="saveAssignmentEdit">
        <div v-if="editingAssignment" class="rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm dark:border-dark-700 dark:bg-dark-900">
          <div class="font-medium text-gray-900 dark:text-white">
            {{ editingAssignment.user?.email || editingAssignment.user?.username || t('adminMemberships.userFallback', { id: editingAssignment.user_id }) }}
          </div>
          <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('adminMemberships.userId') }} {{ editingAssignment.user_id }}</div>
        </div>

        <div>
          <label class="input-label">{{ t('adminMemberships.level') }}</label>
          <Select v-model="assignmentForm.membership_level_id" :options="levelOptions" :placeholder="t('adminMemberships.selectLevel')" />
        </div>

        <div>
          <label class="input-label">{{ t('adminMemberships.expiresAt') }}</label>
          <input v-model="assignmentForm.expires_at" type="datetime-local" class="input" />
          <p class="input-hint">{{ t('adminMemberships.emptyExpiresAtHint') }}</p>
        </div>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="closeAssignmentEditor">{{ t('adminMemberships.cancel') }}</button>
        <button type="submit" form="assignment-form" class="btn btn-primary" :disabled="assignmentSaving">
          {{ t('adminMemberships.save') }}
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import MembershipLevelMark from '@/components/membership/MembershipLevelMark.vue'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { AdminGroup } from '@/types'
import type { MembershipAssignment, MembershipLevel } from '@/api/membership'
import { formatDateTime } from '@/utils/format'
import {
  defaultMembershipColor,
  membershipLevelDisplayName,
  normalizeMembershipColor
} from '@/utils/membershipStyle'
import { notifyMembershipUpdated } from '@/utils/membershipEvents'

const appStore = useAppStore()
const { t } = useI18n()
const loading = ref(false)
const saving = ref(false)
const assigning = ref(false)
const assignmentSaving = ref(false)
const levels = ref<MembershipLevel[]>([])
const groups = ref<AdminGroup[]>([])
const assignments = ref<MembershipAssignment[]>([])
const showEditor = ref(false)
const editingLevel = ref<MembershipLevel | null>(null)
const showAssignmentEditor = ref(false)
const editingAssignment = ref<MembershipAssignment | null>(null)

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('adminMemberships.columns.name'), sortable: false },
  { key: 'discount_rate', label: t('adminMemberships.columns.discountRate'), sortable: false },
  { key: 'groups', label: t('adminMemberships.columns.groups'), sortable: false },
  { key: 'sort_order', label: t('adminMemberships.columns.sortOrder'), sortable: false },
  { key: 'actions', label: t('adminMemberships.columns.actions'), sortable: false }
])

const assignmentColumns = computed<Column[]>(() => [
  { key: 'user', label: t('adminMemberships.columns.user'), sortable: false },
  { key: 'level', label: t('adminMemberships.columns.level'), sortable: false },
  { key: 'source', label: t('adminMemberships.columns.source'), sortable: false },
  { key: 'starts_at', label: t('adminMemberships.columns.startsAt'), sortable: false },
  { key: 'expires_at', label: t('adminMemberships.columns.expiresAt'), sortable: false },
  { key: 'actions', label: t('adminMemberships.columns.actions'), sortable: false }
])

const form = ref({
  name: '',
  code: '',
  color: defaultMembershipColor,
  discount_rate: 1,
  enabled: true,
  is_default: false,
  sort_order: 0,
  description: '',
  group_ids: [] as number[]
})

const assignForm = ref({
  user_id: null as number | null,
  membership_level_id: null as number | null,
  expires_at: ''
})

const assignmentForm = ref({
  membership_level_id: null as number | null,
  expires_at: ''
})

const levelOptions = computed(() =>
  levels.value
    .filter((level) => level.enabled)
    .map((level) => ({ value: level.id, label: level.name }))
)

const formatRate = (value: number) => `${Number(value ?? 1).toFixed(4).replace(/\.?0+$/, '')}x`
const toDateTimeLocal = (value?: string | null) => {
  if (!value) {
    return ''
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}
const sourceText = (source: string) => {
  const key = `adminMemberships.sources.${source}`
  const translated = t(key)
  return translated === key ? source || '-' : translated
}

const loadAll = async () => {
  loading.value = true
  try {
    const [levelList, groupList, assignmentList] = await Promise.all([
      adminAPI.memberships.list(false),
      adminAPI.groups.getAllIncludingInactive(),
      adminAPI.memberships.listAssignments()
    ])
    levels.value = levelList
    groups.value = groupList
    assignments.value = assignmentList
  } catch (error) {
    appStore.showError(t('adminMemberships.errors.loadFailed'))
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  form.value = {
    name: '',
    code: '',
    color: defaultMembershipColor,
    discount_rate: 1,
    enabled: true,
    is_default: false,
    sort_order: 0,
    description: '',
    group_ids: []
  }
}

const openEdit = (level: MembershipLevel) => {
  editingLevel.value = level
  form.value = {
    name: level.name,
    code: level.code,
    color: normalizeMembershipColor(level.color),
    discount_rate: level.discount_rate,
    enabled: level.enabled,
    is_default: level.is_default,
    sort_order: level.sort_order,
    description: level.description || '',
    group_ids: (level.groups || []).map((group) => group.id)
  }
  showEditor.value = true
}

const closeEditor = () => {
  showEditor.value = false
  editingLevel.value = null
  resetForm()
}

const toggleGroup = (groupId: number) => {
  const current = new Set(form.value.group_ids)
  if (current.has(groupId)) {
    current.delete(groupId)
  } else {
    current.add(groupId)
  }
  form.value.group_ids = Array.from(current)
}

const saveLevel = async () => {
  if (!editingLevel.value) {
    return
  }
  saving.value = true
  try {
    await adminAPI.memberships.update(editingLevel.value.id, {
      discount_rate: form.value.discount_rate,
      group_ids: form.value.group_ids
    })
    appStore.showSuccess(t('adminMemberships.success.updated'))
    closeEditor()
    await loadAll()
    notifyMembershipUpdated()
  } catch (error: any) {
    appStore.showError(error?.message || t('adminMemberships.errors.saveFailed'))
  } finally {
    saving.value = false
  }
}

const syncLevel = async (level: MembershipLevel) => {
  try {
    await adminAPI.memberships.sync(level.id)
    appStore.showSuccess(t('adminMemberships.success.synced'))
    await loadAll()
    notifyMembershipUpdated()
  } catch (error: any) {
    appStore.showError(error?.message || t('adminMemberships.errors.syncFailed'))
  }
}

const assignMembership = async () => {
  if (!assignForm.value.user_id || !assignForm.value.membership_level_id) {
    appStore.showError(t('adminMemberships.errors.missingAssignInput'))
    return
  }
  assigning.value = true
  try {
    const expiresAt = assignForm.value.expires_at
      ? new Date(assignForm.value.expires_at).toISOString()
      : null
    await adminAPI.memberships.assignUser(assignForm.value.user_id, {
      membership_level_id: assignForm.value.membership_level_id,
      expires_at: expiresAt,
      source: 'admin'
    })
    appStore.showSuccess(t('adminMemberships.success.assigned'))
    await loadAll()
    notifyMembershipUpdated()
  } catch (error: any) {
    appStore.showError(error?.message || t('adminMemberships.errors.assignFailed'))
  } finally {
    assigning.value = false
  }
}

const openAssignmentEdit = (assignment: MembershipAssignment) => {
  editingAssignment.value = assignment
  assignmentForm.value = {
    membership_level_id: assignment.membership_level_id,
    expires_at: toDateTimeLocal(assignment.expires_at)
  }
  showAssignmentEditor.value = true
}

const closeAssignmentEditor = () => {
  showAssignmentEditor.value = false
  editingAssignment.value = null
  assignmentForm.value = {
    membership_level_id: null,
    expires_at: ''
  }
}

const saveAssignmentEdit = async () => {
  if (!editingAssignment.value || !assignmentForm.value.membership_level_id) {
    appStore.showError(t('adminMemberships.errors.selectLevel'))
    return
  }
  assignmentSaving.value = true
  try {
    const expiresAt = assignmentForm.value.expires_at
      ? new Date(assignmentForm.value.expires_at).toISOString()
      : null
    await adminAPI.memberships.assignUser(editingAssignment.value.user_id, {
      membership_level_id: assignmentForm.value.membership_level_id,
      expires_at: expiresAt,
      source: 'admin'
    })
    appStore.showSuccess(t('adminMemberships.success.userUpdated'))
    closeAssignmentEditor()
    await loadAll()
    notifyMembershipUpdated()
  } catch (error: any) {
    appStore.showError(error?.message || t('adminMemberships.errors.updateFailed'))
  } finally {
    assignmentSaving.value = false
  }
}

onMounted(loadAll)
</script>

<style scoped>
.membership-table-card {
  overflow: hidden;
}

.membership-levels-card :deep(.table-wrapper) {
  flex: none;
  max-height: none;
  overflow: visible !important;
}

.membership-assignments-card :deep(.table-wrapper) {
  flex: none;
  max-height: min(420px, calc(100vh - 280px));
}
</style>
