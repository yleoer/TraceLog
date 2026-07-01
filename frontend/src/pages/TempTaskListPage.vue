<template>
  <div class="space-y-5">
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-semibold text-gray-900">临时需求</h1>
      <n-button type="primary" size="small" @click="$router.push('/temp-tasks/new')">新增临时需求</n-button>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <n-input v-model:value="query" placeholder="搜索标题、来源、内容" clearable size="small" class="!w-56" @keyup.enter="load" />
      <n-select v-model:value="status" placeholder="状态" clearable size="small" :options="statusOptions" class="!w-32" />
      <n-button size="small" @click="load">搜索</n-button>
    </div>

    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <n-data-table :loading="loading" :columns="columns" :data="tasks" :row-key="taskRowKey" size="small" :bordered="false" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { h, onMounted, ref, watch } from 'vue'
import { NButton, NDataTable, NInput, NSelect, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useRouter } from 'vue-router'
import { api } from '../api/client'
import StatusTag from '../components/StatusTag.vue'
import PriorityTag from '../components/PriorityTag.vue'
import { tempStatusLabel, tempPriorityLabel } from '../utils/tempTaskDisplay'
import { formatDateTime } from '../utils/datetime'
import type { TempTask } from '../types'

const router = useRouter()
const message = useMessage()
const loading = ref(false)
const tasks = ref<TempTask[]>([])
const query = ref('')
const status = ref<string | null>(null)

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
    tasks.value = await api.listTempTasks({ q: query.value, status: status.value ?? undefined })
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

watch(status, load)
onMounted(load)
</script>

<style scoped>
.task-link {
  color: #2563eb;
  font-weight: 600;
  font-size: 13px;
}

.task-link:hover {
  text-decoration: underline;
}
</style>
