<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">会员等级管理</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              配置会员等级、折扣倍率、可用分组，并可按用户 ID 分配会员等级。
            </p>
          </div>
          <div class="flex flex-wrap gap-3">
            <button class="btn btn-secondary" :disabled="loading" @click="loadAll">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button class="btn btn-primary" @click="openCreate">
              <Icon name="plus" size="md" class="mr-2" />
              新增会员等级
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="levels" :loading="loading">
          <template #cell-name="{ row }">
            <div class="space-y-1">
              <div class="flex flex-wrap items-center gap-2">
                <span class="font-medium text-gray-900 dark:text-white">{{ row.name }}</span>
                <span v-if="row.is_default" class="badge badge-primary">默认</span>
                <span :class="['badge', row.enabled ? 'badge-success' : 'badge-gray']">
                  {{ row.enabled ? '启用' : '停用' }}
                </span>
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
              <span v-if="!row.groups?.length" class="text-sm text-gray-400">未配置</span>
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
              <button
                class="rounded-lg p-1.5 text-gray-500 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                :disabled="row.is_default"
                @click="disableLevel(row)"
              >
                <Icon name="ban" size="sm" />
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState title="暂无会员等级" description="创建一个默认会员等级后，新用户会自动绑定。" action-text="新增会员等级" @action="openCreate" />
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <section class="mx-auto mt-6 w-full max-w-7xl rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
      <h2 class="text-base font-semibold text-gray-900 dark:text-white">分配用户会员等级</h2>
      <div class="mt-4 grid gap-4 md:grid-cols-[160px_1fr_220px_auto] md:items-end">
        <div>
          <label class="input-label">用户 ID</label>
          <input v-model.number="assignForm.user_id" type="number" min="1" class="input" placeholder="用户 ID" />
        </div>
        <div>
          <label class="input-label">会员等级</label>
          <Select v-model="assignForm.membership_level_id" :options="levelOptions" placeholder="选择会员等级" />
        </div>
        <div>
          <label class="input-label">到期时间</label>
          <input v-model="assignForm.expires_at" type="datetime-local" class="input" />
          <p class="input-hint">留空表示长期有效</p>
        </div>
        <button class="btn btn-primary" :disabled="assigning" @click="assignMembership">分配会员</button>
      </div>
    </section>

    <BaseDialog :show="showEditor" :title="editingLevel ? '编辑会员等级' : '新增会员等级'" width="wide" @close="closeEditor">
      <form id="membership-form" class="space-y-5" @submit.prevent="saveLevel">
        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">名称</label>
            <input v-model.trim="form.name" required class="input" placeholder="例如：VIP" />
          </div>
          <div>
            <label class="input-label">编码</label>
            <input v-model.trim="form.code" required class="input font-mono" placeholder="例如：vip" />
          </div>
          <div>
            <label class="input-label">折扣倍率</label>
            <input v-model.number="form.discount_rate" required type="number" min="0" step="0.0001" class="input" />
            <p class="input-hint">用户专属倍率 = 分组倍率 × 会员折扣</p>
          </div>
          <div>
            <label class="input-label">排序</label>
            <input v-model.number="form.sort_order" type="number" step="1" class="input" />
          </div>
        </div>

        <div>
          <label class="input-label">描述</label>
          <textarea v-model.trim="form.description" rows="3" class="input" placeholder="可选描述" />
        </div>

        <div class="grid gap-3 md:grid-cols-2">
          <label class="flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-700">
            <input v-model="form.enabled" type="checkbox" class="rounded border-gray-300" />
            启用
          </label>
          <label class="flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-700">
            <input v-model="form.is_default" type="checkbox" class="rounded border-gray-300" />
            设为默认会员等级
          </label>
        </div>

        <div>
          <div class="mb-2 flex items-center justify-between">
            <label class="input-label mb-0">可用分组</label>
            <span class="text-xs text-gray-500 dark:text-dark-400">已选 {{ form.group_ids.length }} 个</span>
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
                  {{ group.platform }} · {{ group.rate_multiplier }}x · {{ group.is_exclusive ? '专属' : '公开' }}
                </span>
              </span>
              <input :checked="form.group_ids.includes(group.id)" type="checkbox" class="rounded border-gray-300" @change="toggleGroup(group.id)" />
            </label>
            <div v-if="groups.length === 0" class="px-3 py-8 text-center text-sm text-gray-500 dark:text-dark-400">
              暂无可选分组。
            </div>
          </div>
        </div>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="closeEditor">取消</button>
        <button type="submit" form="membership-form" class="btn btn-primary" :disabled="saving">
          {{ editingLevel ? '保存' : '创建' }}
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { AdminGroup } from '@/types'
import type { MembershipLevel } from '@/api/membership'

const appStore = useAppStore()
const loading = ref(false)
const saving = ref(false)
const assigning = ref(false)
const levels = ref<MembershipLevel[]>([])
const groups = ref<AdminGroup[]>([])
const showEditor = ref(false)
const editingLevel = ref<MembershipLevel | null>(null)

const columns = computed<Column[]>(() => [
  { key: 'name', label: '会员等级', sortable: false },
  { key: 'discount_rate', label: '折扣倍率', sortable: false },
  { key: 'groups', label: '可用分组', sortable: false },
  { key: 'sort_order', label: '排序', sortable: false },
  { key: 'actions', label: '操作', sortable: false }
])

const form = ref({
  name: '',
  code: '',
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

const levelOptions = computed(() =>
  levels.value
    .filter((level) => level.enabled)
    .map((level) => ({ value: level.id, label: level.name }))
)

const formatRate = (value: number) => `${Number(value ?? 1).toFixed(4).replace(/\.?0+$/, '')}x`

const loadAll = async () => {
  loading.value = true
  try {
    const [levelList, groupList] = await Promise.all([
      adminAPI.memberships.list(true),
      adminAPI.groups.getAllIncludingInactive()
    ])
    levels.value = levelList
    groups.value = groupList
  } catch (error) {
    appStore.showError('会员等级加载失败')
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  form.value = {
    name: '',
    code: '',
    discount_rate: 1,
    enabled: true,
    is_default: false,
    sort_order: 0,
    description: '',
    group_ids: []
  }
}

const openCreate = () => {
  editingLevel.value = null
  resetForm()
  showEditor.value = true
}

const openEdit = (level: MembershipLevel) => {
  editingLevel.value = level
  form.value = {
    name: level.name,
    code: level.code,
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
  saving.value = true
  try {
    if (editingLevel.value) {
      await adminAPI.memberships.update(editingLevel.value.id, form.value)
      appStore.showSuccess('会员等级已更新')
    } else {
      await adminAPI.memberships.create(form.value)
      appStore.showSuccess('会员等级已创建')
    }
    closeEditor()
    await loadAll()
  } catch (error: any) {
    appStore.showError(error?.message || '会员等级保存失败')
  } finally {
    saving.value = false
  }
}

const disableLevel = async (level: MembershipLevel) => {
  if (level.is_default) {
    appStore.showError('默认会员等级不能停用')
    return
  }
  try {
    await adminAPI.memberships.disable(level.id)
    appStore.showSuccess('会员等级已停用')
    await loadAll()
  } catch (error: any) {
    appStore.showError(error?.message || '会员等级停用失败')
  }
}

const syncLevel = async (level: MembershipLevel) => {
  try {
    await adminAPI.memberships.sync(level.id)
    appStore.showSuccess('会员等级已同步')
  } catch (error: any) {
    appStore.showError(error?.message || '会员等级同步失败')
  }
}

const assignMembership = async () => {
  if (!assignForm.value.user_id || !assignForm.value.membership_level_id) {
    appStore.showError('请填写用户 ID 并选择会员等级')
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
    appStore.showSuccess('用户会员等级已分配')
  } catch (error: any) {
    appStore.showError(error?.message || '分配会员失败')
  } finally {
    assigning.value = false
  }
}

onMounted(loadAll)
</script>
