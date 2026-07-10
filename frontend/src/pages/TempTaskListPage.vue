<template>
  <div class="temp-task-list-page space-y-5">
    <div class="page-header">
      <div>
        <div class="page-kicker">Quick requests</div>
        <h1>临时需求</h1>
        <p class="page-subtitle">捕捉计划外工作，并保持每一个请求可追踪。</p>
      </div>
      <n-button type="primary" size="small" @click="$router.push('/temp-tasks/new')">新增临时需求</n-button>
    </div>

    <div class="filter-bar">
      <n-input v-model:value="query" placeholder="搜索标题、来源、内容" clearable size="small" class="!w-56" @keyup.enter="applyFilters" />
      <n-select :value="status" placeholder="状态" clearable size="small" :options="statusOptions" class="!w-32" @update:value="onStatusChange" />
      <n-button size="small" @click="applyFilters">搜索</n-button>
    </div>

    <div class="table-shell">
      <n-data-table :loading="loading" :columns="columns" :data="tasks" :row-key="taskRowKey" size="small" :bordered="false" />
    </div>

    <div class="pagination-bar flex items-center justify-between text-sm text-gray-500">
      <span>第 {{ page }} 页，当前 {{ tasks.length }} 条</span>
      <div class="flex items-center gap-2">
        <n-button size="small" :disabled="page <= 1 || loading" @click="goPage(page - 1)">上一页</n-button>
        <n-button size="small" :disabled="!hasNextPage || loading" @click="goPage(page + 1)">下一页</n-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { h, ref, watch } from 'vue'
import { NButton, NDataTable, NInput, NSelect, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api/client'
import StatusTag from '../components/StatusTag.vue'
import PriorityTag from '../components/PriorityTag.vue'
import { tempStatusLabel, tempPriorityLabel } from '../utils/tempTaskDisplay'
import { formatDateTime } from '../utils/datetime'
import type { TempTask } from '../types'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const loading = ref(false)
const tasks = ref<TempTask[]>([])
const query = ref('')
const status = ref<string | null>(null)
const page = ref(1)
const pageSize = 50
const hasNextPage = ref(false)

const statusOptions = [
  { label: '待处理', value: 'todo' },
  { label: '处理中', value: 'processing' },
  { label: '已完成', value: 'done' },
  { label: '挂起', value: 'suspended' }
]

const columns: DataTableColumns<TempTask> = [
  {
    title: '标题',
    key: 'title',
    minWidth: 200,
    render: (row) =>
      h(
        'a',
        {
          class: 'task-link',
          href: `/temp-tasks/${row.id}`,
          onClick: (event: MouseEvent) => {
            event.preventDefault()
            router.push(`/temp-tasks/${row.id}`)
          }
        },
        row.title
      )
  },
  { title: '来源', key: 'source', width: 120 },
  { title: '状态', key: 'status', width: 100, render: (row) => h(StatusTag, { status: row.status, label: tempStatusLabel(row.status) }) },
  { title: '优先级', key: 'priority', width: 90, render: (row) => h(PriorityTag, { priority: row.priority, label: tempPriorityLabel(row.priority) }) },
  { title: '开始时间', key: 'started_at', width: 150, render: (row) => formatDate(row.started_at) },
  { title: '结束时间', key: 'completed_at', width: 150, render: (row) => formatDate(row.completed_at) },
  { title: 'Jira', key: 'converted_jira_key', width: 120 },
  { title: '更新时间', key: 'updated_at', width: 150, render: (row) => formatDate(row.updated_at) }
]

async function load() {
  loading.value = true
  try {
    const rows = await api.listTempTasks({
      q: query.value.trim() || undefined,
      status: status.value ?? undefined,
      limit: pageSize + 1,
      offset: (page.value - 1) * pageSize
    })
    hasNextPage.value = rows.length > pageSize
    tasks.value = rows.slice(0, pageSize)
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    loading.value = false
  }
}

function taskRowKey(row: TempTask) {
  return row.id
}

function formatDate(value: string) {
  return formatDateTime(value)
}

function applyFilters() {
  router.push({ path: '/temp-tasks', query: filterQuery(1) })
}

function onStatusChange(value: unknown) {
  status.value = typeof value === 'string' ? value : null
  applyFilters()
}

function goPage(nextPage: number) {
  router.push({ path: '/temp-tasks', query: filterQuery(Math.max(1, nextPage)) })
}

function filterQuery(nextPage = page.value) {
  return {
    ...(query.value.trim() ? { q: query.value.trim() } : {}),
    ...(status.value ? { status: status.value } : {}),
    ...(nextPage > 1 ? { page: String(nextPage) } : {})
  }
}

function syncFiltersFromRoute() {
  query.value = stringParam(route.query.q)
  status.value = stringParam(route.query.status) || null
  page.value = positiveInt(route.query.page)
}

function stringParam(value: unknown) {
  return typeof value === 'string' ? value : ''
}

function positiveInt(value: unknown) {
  if (typeof value !== 'string') return 1
  const parsed = Number(value)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 1
}

watch(
  () => route.query,
  () => {
    syncFiltersFromRoute()
    load()
  },
  { immediate: true }
)
</script>

<style scoped>
.task-link {
  color: var(--accent);
  font-weight: 600;
  font-size: 13px;
}

.task-link:hover {
  text-decoration: underline;
}
</style>
