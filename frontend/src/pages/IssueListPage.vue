<template>
  <div class="space-y-5">
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-semibold text-gray-900">Issues</h1>
      <n-button type="primary" size="small" @click="$router.push('/issues/new')">新增 Issue</n-button>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <n-input v-model:value="query" placeholder="搜索编号、简述、评论" clearable size="small" class="!w-56" @keyup.enter="applyFilters" />
      <n-select
        :value="status || null"
        :options="statusOptions"
        placeholder="状态"
        filterable
        clearable
        size="small"
        class="!w-40"
        @update:value="onStatusChange"
      />
      <n-select
        :value="tag || null"
        :options="tagOptions"
        placeholder="标签"
        filterable
        clearable
        size="small"
        class="!w-36"
        @update:value="onTagChange"
      />
      <n-button size="small" @click="applyFilters">搜索</n-button>
      <n-button v-if="hasFilters" size="small" quaternary @click="clearFilters">清空</n-button>
    </div>

    <div v-if="activeTag" class="flex items-center gap-2 text-sm text-gray-500">
      <span>标签</span>
      <n-tag size="small" closable @close="clearTag">{{ activeTag }}</n-tag>
    </div>

    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <n-data-table :loading="loading" :columns="columns" :data="issues" :row-key="issueRowKey" size="small" :bordered="false" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, h, ref, watch } from 'vue'
import { NTag, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api/client'
import type { Issue } from '../types'
import { parseJiraMeta, statusDisplayName } from '../utils/jiraDisplay'
import StatusTag from '../components/StatusTag.vue'
import TypeTag from '../components/TypeTag.vue'
import PriorityTag from '../components/PriorityTag.vue'
import { formatDateTime } from '../utils/datetime'
import { openExternalClick } from '../utils/openExternal'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const loading = ref(false)
const issues = ref<Issue[]>([])
const query = ref('')
const status = ref('')
const tag = ref('')
const activeTag = computed(() => tag.value.trim())
const hasFilters = computed(() => Boolean(query.value.trim() || status.value.trim() || tag.value.trim()))
const allIssues = ref<Issue[]>([])

const statusOptions = computed(() => {
  const seen = new Set<string>()
  for (const issue of allIssues.value) {
    const label = statusDisplayName(issue.status, issue.background_md)
    if (label && label !== '-') seen.add(label)
  }
  return Array.from(seen).sort().map((value) => ({ label: value, value }))
})

const tagOptions = computed(() => {
  const seen = new Set<string>()
  for (const issue of allIssues.value) {
    for (const tagName of issue.tags) {
      if (tagName) seen.add(tagName)
    }
  }
  return Array.from(seen).sort().map((value) => ({ label: value, value }))
})

const columns: DataTableColumns<Issue> = [
  {
    title: 'Issue',
    key: 'jira_key',
    width: 120,
    render: (row) =>
      h(
        'a',
        {
          class: 'issue-key',
          href: `/issues/${row.jira_key}`,
          onClick: (event: MouseEvent) => {
            event.preventDefault()
            router.push(`/issues/${row.jira_key}`)
          }
        },
        row.jira_key
      )
  },
  {
    title: '简述',
    key: 'summary_md',
    minWidth: 240,
    render: (row) => row.summary_md || row.solution_md || '-'
  },
  {
    title: '地址',
    key: 'links',
    minWidth: 200,
    render: (row) => {
      const href = firstJiraLink(row)
      return href ? h('a', { class: 'ext-link', href, onClick: openExternalClick(href) }, href) : '-'
    }
  },
  {
    title: '类型',
    key: 'type',
    width: 110,
    render: (row) => h(TypeTag, { type: parseJiraMeta(row.background_md).issueType })
  },
  {
    title: '状态',
    key: 'status',
    width: 110,
    render: (row) => h(StatusTag, { status: row.status, background: row.background_md })
  },
  {
    title: '优先级',
    key: 'priority',
    width: 100,
    render: (row) => h(PriorityTag, { priority: row.priority, jiraPriority: parseJiraMeta(row.background_md).jiraPriority })
  },
  {
    title: '标签',
    key: 'tags',
    minWidth: 140,
    render: (row) =>
      row.tags.length
        ? row.tags.map((rowTag) =>
            h(
              NTag,
              {
                size: 'small',
                style: 'margin-right: 4px; cursor: pointer',
                onClick: () => showIssuesByTag(rowTag)
              },
              { default: () => rowTag }
            )
          )
        : '-'
  },
  {
    title: '导入时间',
    key: 'created_at',
    width: 150,
    render: (row) => formatDate(row.created_at)
  }
]

async function load() {
  loading.value = true
  try {
    const filters = {
      q: query.value.trim() || undefined,
      status: status.value.trim() || undefined,
      tag: tag.value.trim() || undefined
    }
    const hasActiveFilter = Boolean(filters.q || filters.status || filters.tag)
    issues.value = await api.listIssues(filters)
    if (!hasActiveFilter) {
      // The unfiltered list is already the full set — reuse it as the facet source
      // instead of firing a second identical list request.
      allIssues.value = issues.value
    } else if (allIssues.value.length === 0) {
      // Landed directly on a filtered view: fetch the full set once for dropdown options.
      await loadFacets()
    }
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    loading.value = false
  }
}

function issueRowKey(row: Issue) {
  return row.jira_key
}

function applyFilters() {
  router.push({ path: '/issues', query: filterQuery() })
}

function onStatusChange(value: unknown) {
  status.value = typeof value === 'string' ? value : ''
  applyFilters()
}

function onTagChange(value: unknown) {
  tag.value = typeof value === 'string' ? value : ''
  applyFilters()
}

async function loadFacets() {
  try {
    allIssues.value = await api.listIssues({})
  } catch {
    // Facet options are best-effort; ignore failures.
  }
}

function clearFilters() {
  router.push({ path: '/issues' })
}

function clearTag() {
  tag.value = ''
  applyFilters()
}

function showIssuesByTag(value: string) {
  router.push({ path: '/issues', query: { tag: value } })
}

function filterQuery() {
  return {
    ...(query.value.trim() ? { q: query.value.trim() } : {}),
    ...(status.value.trim() ? { status: status.value.trim() } : {}),
    ...(tag.value.trim() ? { tag: tag.value.trim() } : {})
  }
}

function syncFiltersFromRoute() {
  query.value = stringParam(route.query.q)
  status.value = stringParam(route.query.status)
  tag.value = stringParam(route.query.tag)
}

function stringParam(value: unknown) {
  return typeof value === 'string' ? value : ''
}

function firstJiraLink(row: Issue) {
  return row.links.find((link) => link.type === 'jira')?.url ?? row.links[0]?.url ?? ''
}

function formatDate(value: string) {
  return formatDateTime(value)
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
.issue-key {
  color: #2563eb;
  font-weight: 600;
  font-size: 12px;
  padding: 2px 6px;
  background: #eff6ff;
  border-radius: 4px;
}

.issue-key:hover {
  background: #dbeafe;
}

.ext-link {
  color: #0f766e;
  font-size: 12px;
  overflow-wrap: anywhere;
}

.ext-link:hover {
  color: #115e59;
  text-decoration: underline;
}
</style>
