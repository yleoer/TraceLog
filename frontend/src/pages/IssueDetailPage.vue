<template>
  <div class="space-y-5 max-w-4xl mx-auto">
    <div class="flex items-center justify-between">
      <div>
        <p class="text-xs text-gray-400 uppercase tracking-wide">{{ isNew ? '新增工作记录' : '工作记录' }}</p>
        <h1 class="text-2xl font-bold text-gray-900 mt-0.5">{{ form.jira_key || 'GCS-' }}</h1>
      </div>
      <div class="flex items-center gap-2">
        <n-button v-if="!isNew" size="small" @click="downloadUrl(`/export/issues/${form.jira_key}.md`)">导出</n-button>
        <n-popconfirm v-if="!isNew" @positive-click="removeIssue">
          <template #trigger><n-button size="small" type="error" ghost>删除</n-button></template>
          删除后该 Issue 的评论、TODO 等将一并移除，确认删除？
        </n-popconfirm>
        <n-button v-if="isNew && hasJiraData" type="primary" size="small" :loading="saving" @click="save">创建</n-button>
      </div>
    </div>

    <!-- Import panel -->
    <section v-if="isNew" class="card bg-gray-50">
      <div class="flex items-center justify-between mb-3">
        <div>
          <h2 class="text-sm font-semibold text-gray-900">从 Jira 导入</h2>
          <p class="text-xs text-gray-500 mt-0.5">输入 GCS 开头的 issue key，系统自动获取数据。</p>
        </div>
        <n-button size="small" :loading="importing" @click="importFromJira">获取 Jira 数据</n-button>
      </div>
      <n-input-group>
        <n-input-group-label>Issue</n-input-group-label>
        <n-input v-model:value="jiraInput" placeholder="GCS-45000" size="small" @keyup.enter="importFromJira" />
      </n-input-group>
    </section>

    <!-- Jira info -->
    <section v-if="hasJiraData" class="card">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-sm font-semibold text-gray-900">Jira 信息</h2>
        <StatusTag :status="form.status" :background="form.background_md" />
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div class="col-span-full border-b border-gray-100 pb-3">
          <label class="field-label">标题</label>
          <div class="text-base font-semibold text-gray-900">{{ form.title || '-' }}</div>
        </div>
        <div class="col-span-full border-b border-gray-100 pb-3">
          <label class="field-label">地址</label>
          <a v-if="jiraAddress" href="#" class="text-sm text-teal-700 hover:underline break-all" @click.prevent="openExternalURL(jiraAddress)">{{ jiraAddress }}</a>
          <span v-else class="text-sm text-gray-400">-</span>
        </div>
        <div class="border-b border-gray-100 pb-3">
          <label class="field-label">Issue Key</label>
          <div class="text-sm font-semibold">{{ form.jira_key || '-' }}</div>
        </div>
        <div class="border-b border-gray-100 pb-3">
          <label class="field-label">类型</label>
          <div class="text-sm">
            <TypeTag :type="jiraMeta.issueType" />
          </div>
        </div>
        <div class="border-b border-gray-100 pb-3">
          <label class="field-label">状态</label>
          <div class="text-sm">
            <StatusTag :status="form.status" :background="form.background_md" />
          </div>
        </div>
        <div class="border-b border-gray-100 pb-3">
          <label class="field-label">发布请求</label>
          <div class="text-sm">{{ jiraMeta.releaseRequested || '-' }}</div>
        </div>
        <div class="border-b border-gray-100 pb-3">
          <label class="field-label">优先级</label>
          <div class="text-sm">
            <PriorityTag :priority="form.priority" :jira-priority="jiraMeta.jiraPriority" />
          </div>
        </div>
        <div class="border-b border-gray-100 pb-3">
          <label class="field-label">导入时间</label>
          <div class="text-sm">{{ importedAtText }}</div>
        </div>
        <div class="col-span-full border-b border-gray-100 pb-3">
          <label class="field-label">标签</label>
          <div class="flex flex-wrap gap-1.5 mt-1">
            <n-tag v-for="tagItem in form.tags" :key="tagItem" size="small" class="cursor-pointer" @click="showIssuesByTag(tagItem)">{{ tagItem }}</n-tag>
            <span v-if="form.tags.length === 0" class="text-sm text-gray-400">-</span>
          </div>
        </div>
        <div v-if="!isNew" class="border-b border-gray-100 pb-3">
          <label class="field-label">开始时间</label>
          <n-date-picker v-model:value="manual.startedAt" type="datetime" clearable size="small" class="w-full" @update:value="saveManualFieldsSoon" />
        </div>
        <div v-if="!isNew" class="border-b border-gray-100 pb-3">
          <label class="field-label">完成时间</label>
          <n-date-picker v-model:value="manual.completedAt" type="datetime" clearable size="small" class="w-full" @update:value="saveManualFieldsSoon" />
        </div>
      </div>

      <div v-if="!isNew" class="mt-5">
        <div class="flex items-center justify-between mb-2">
          <label class="field-label">简述</label>
          <n-button size="tiny" type="primary" :loading="summaryGenerating" @click.stop="generateSummary">AI 总结</n-button>
        </div>
        <n-input v-model:value="manual.summary" type="textarea" size="small" :autosize="{ minRows: 3 }" placeholder="点击 AI 总结自动生成" @update:value="saveManualFieldsSoon" />
      </div>

      <n-collapse v-if="jiraMeta.description" class="mt-4 border-t border-gray-100 pt-3" :default-expanded-names="[]">
        <n-collapse-item title="Description" name="description">
          <MarkdownView :content="jiraMeta.description" />
        </n-collapse-item>
      </n-collapse>
    </section>

    <!-- My records -->
    <section v-if="!isNew" class="card">
      <h2 class="text-sm font-semibold text-gray-900 mb-4">我的记录</h2>

      <CommentTimeline
        :events="events"
        new-title="新增阶段评论"
        :draft-key="commentDraftKey"
        :load="loadComments"
        :create="createComment"
        :update="updateComment"
        :remove="deleteComment"
        :upload-context="issueUploadContext('timeline')"
      >
        <div class="border-b border-gray-100 pb-5 mb-5">
          <div class="flex items-center justify-between mb-2">
            <div>
              <h3 class="text-xs font-medium text-gray-600">后续 TODO</h3>
              <p class="text-xs text-gray-400">{{ openTodos.length }} 项未完成</p>
            </div>
            <n-button size="tiny" type="primary" :loading="todoSaving" @click="saveTodo">添加 TODO</n-button>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-[1fr_220px] gap-2 mb-3">
            <n-input v-model:value="todoDraft.content" size="small" placeholder="需要跟进的事项" @keyup.enter="saveTodo" />
            <n-date-picker v-model:value="todoDraft.dueAt" type="datetime" clearable size="small" />
          </div>
          <div v-if="todos.length === 0" class="text-sm text-gray-400">还没有 TODO。</div>
          <div v-for="todo in todos" :key="todo.id" class="todo-item" :class="{ done: todo.done }">
            <n-checkbox :checked="todo.done" @update:checked="toggleTodo(todo, $event)" />
            <div class="flex-1 min-w-0">
              <div class="text-sm">{{ todo.content }}</div>
              <span v-if="todo.due_at" class="text-xs text-gray-400">截止 {{ formatDateTime(todo.due_at) }}</span>
            </div>
            <n-button text size="tiny" type="error" @click="deleteTodo(todo)">删除</n-button>
          </div>
        </div>
      </CommentTimeline>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import {
  NButton,
  NCheckbox,
  NCollapse,
  NCollapseItem,
  NDatePicker,
  NInput,
  NInputGroup,
  NInputGroupLabel,
  NPopconfirm,
  NTag,
  useMessage
} from 'naive-ui'
import { useRoute, useRouter } from 'vue-router'
import { api, downloadUrl } from '../api/client'
import MarkdownView from '../components/MarkdownView.vue'
import CommentTimeline from '../components/CommentTimeline.vue'
import StatusTag from '../components/StatusTag.vue'
import TypeTag from '../components/TypeTag.vue'
import PriorityTag from '../components/PriorityTag.vue'
import type { Issue, IssueEvent, IssueTodo, Link } from '../types'
import { parseJiraMeta } from '../utils/jiraDisplay'
import { formatDateTime as sharedFormatDateTime } from '../utils/datetime'
import { openExternalURL } from '../utils/openExternal'

const ATLASSIAN_BASE_URL = 'https://axyomcore.atlassian.net'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const saving = ref(false)
const importing = ref(false)
const summaryGenerating = ref(false)
const jiraInput = ref('')
const events = ref<IssueEvent[]>([])
const todos = ref<IssueTodo[]>([])
const isNew = computed(() => !route.params.jiraKey)
let manualSaveTimer: number | undefined
let manualSaveToken = 0

interface TimelineEvent {
  id: number
  event_type: string
  happened_at: string
}

interface IssueForm {
  id: number
  jira_key: string
  title: string
  status: string
  priority: string
  tags: string[]
  summary_md: string
  background_md: string
  analysis_md: string
  solution_md: string
  actions_md: string
  result_md: string
  todo_md: string
  links: Link[]
  started_at: string
  completed_at: string
  created_at: string
  updated_at: string
}

const form = reactive<IssueForm>({
  id: 0,
  jira_key: '',
  title: '',
  status: 'analysis',
  priority: 'medium',
  tags: [],
  summary_md: '',
  background_md: '',
  analysis_md: '',
  solution_md: '',
  actions_md: '',
  result_md: '',
  todo_md: '',
  links: [],
  started_at: '',
  completed_at: '',
  created_at: '',
  updated_at: ''
})

const manual = reactive({
  startedAt: null as number | null,
  completedAt: null as number | null,
  summary: ''
})

const todoDraft = reactive({
  content: '',
  dueAt: null as number | null
})
const todoSaving = ref(false)
const openTodos = computed(() => todos.value.filter((todo) => !todo.done))
const commentDraftKey = computed(() => form.jira_key ? `tracelog:draft:issue-comment:${form.jira_key}` : '')

const jiraMeta = computed(() => parseJiraMeta(form.background_md))
const hasJiraData = computed(() => Boolean(form.jira_key && form.title))
const jiraAddress = computed(() => firstJiraLink(form.links) || (form.jira_key ? `${ATLASSIAN_BASE_URL}/browse/${form.jira_key}` : ''))
const importedAtText = computed(() => formatDateTime(form.created_at || form.updated_at))

async function load() {
  await flushManualFields()
  if (isNew.value) return
  const jiraKey = String(route.params.jiraKey)
  try {
    Object.assign(form, await api.getIssue(jiraKey))
    hydrateManualFields()
    await loadComments(jiraKey)
    await loadTodos(jiraKey)
    jiraInput.value = form.jira_key
  } catch (error) {
    message.error((error as Error).message)
  }
}

async function loadComments(jiraKey = form.jira_key) {
  if (!jiraKey) return
  events.value = await api.listIssueEvents(jiraKey)
}

async function loadTodos(jiraKey = form.jira_key) {
  if (!jiraKey) return
  todos.value = await api.listIssueTodos(jiraKey, true)
}

async function importFromJira() {
  const jiraKey = parseJiraKey(jiraInput.value || form.jira_key)
  if (!jiraKey) {
    message.error('请输入 Jira URL 或 GCS-45000 这样的 issue key')
    return
  }
  importing.value = true
  try {
    const imported = await api.importJiraIssue(jiraKey)
    Object.assign(form, {
      ...form,
      jira_key: imported.jira_key,
      title: imported.title,
      status: imported.status,
      priority: imported.priority,
      tags: imported.tags ?? [],
      background_md: imported.background_md,
      links: imported.links?.length ? imported.links : [{ title: 'Jira', url: `${ATLASSIAN_BASE_URL}/browse/${imported.jira_key}`, type: 'jira' }]
    })
    if (!form.created_at) {
      form.created_at = new Date().toISOString()
    }
    jiraInput.value = imported.jira_key
    message.success('已获取 Jira 数据，请确认后保存')
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    importing.value = false
  }
}

async function save() {
  saving.value = true
  try {
    if (!isNew.value) {
      serializeManualFields()
    }
    if (isNew.value) {
      const created = await api.createIssue(form)
      message.success('已创建')
      router.push(`/issues/${created.jira_key}`)
    } else {
      Object.assign(form, await api.updateIssue(String(route.params.jiraKey), form))
      hydrateManualFields()
      message.success('已保存')
    }
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    saving.value = false
  }
}

function saveManualFieldsSoon() {
  serializeManualFields()
  if (isNew.value) return
  window.clearTimeout(manualSaveTimer)
  manualSaveTimer = window.setTimeout(() => {
    manualSaveTimer = undefined
    void saveManualFields()
  }, 450)
}

async function saveManualFields() {
  if (isNew.value || !form.jira_key) return
  const token = ++manualSaveToken
  const payload = { ...form }
  try {
    const updated = await api.updateIssue(form.jira_key, payload)
    if (token !== manualSaveToken) return
    Object.assign(form, updated)
    hydrateManualFields()
  } catch (error) {
    message.error((error as Error).message)
  }
}

async function flushManualFields() {
  if (manualSaveTimer === undefined) return
  window.clearTimeout(manualSaveTimer)
  manualSaveTimer = undefined
  await saveManualFields()
}

async function generateSummary() {
  if (!form.jira_key) return
  summaryGenerating.value = true
  try {
    const result = await api.generateIssueSummary(form.jira_key)
    Object.assign(form, result.issue)
    hydrateManualFields()
    message.success('简述已生成')
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    summaryGenerating.value = false
  }
}

async function createComment(content: string) {
  await api.createIssueEvent(form.jira_key, {
    event_type: 'note',
    content_md: content,
    happened_at: new Date().toISOString()
  })
}

async function updateComment(event: TimelineEvent, content: string) {
  await api.updateIssueEvent(event.id, {
    event_type: event.event_type || 'note',
    content_md: content,
    happened_at: event.happened_at
  })
}

async function deleteComment(event: TimelineEvent) {
  await api.deleteIssueEvent(event.id)
}

async function removeIssue() {
  try {
    await api.deleteIssue(form.jira_key)
    message.success('Issue 已删除')
    router.push('/issues')
  } catch (error) {
    message.error((error as Error).message)
  }
}

async function saveTodo() {
  const content = todoDraft.content.trim()
  if (!content) {
    message.error('TODO 内容不能为空')
    return
  }
  todoSaving.value = true
  try {
    await api.createIssueTodo(form.jira_key, {
      content,
      due_at: todoDraft.dueAt ? new Date(todoDraft.dueAt).toISOString() : ''
    })
    todoDraft.content = ''
    todoDraft.dueAt = null
    await loadTodos()
    message.success('TODO 已添加')
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    todoSaving.value = false
  }
}

async function toggleTodo(todo: IssueTodo, checked: boolean) {
  try {
    await api.updateIssueTodo(todo.id, { ...todo, done: checked })
    await loadTodos()
  } catch (error) {
    message.error((error as Error).message)
  }
}

async function deleteTodo(todo: IssueTodo) {
  try {
    await api.deleteIssueTodo(todo.id)
    await loadTodos()
    message.success('TODO 已删除')
  } catch (error) {
    message.error((error as Error).message)
  }
}

function hydrateManualFields() {
  manual.startedAt = parseMillis(form.started_at)
  manual.completedAt = parseMillis(form.completed_at)
  manual.summary = form.summary_md || ''
}

function serializeManualFields() {
  form.started_at = manual.startedAt ? new Date(manual.startedAt).toISOString() : ''
  form.completed_at = manual.completedAt ? new Date(manual.completedAt).toISOString() : ''
  form.summary_md = manual.summary
}

function parseJiraKey(value: string) {
  const match = value.toUpperCase().match(/[A-Z][A-Z0-9]+-\d+/)
  return match?.[0] ?? ''
}

function firstJiraLink(links: Link[]) {
  return links.find((link) => link.type === 'jira')?.url ?? links[0]?.url ?? ''
}

function showIssuesByTag(tagValue: string) {
  router.push({ path: '/issues', query: { tag: tagValue } })
}

function parseMillis(value: string) {
  if (!value) return null
  const timestamp = Date.parse(value)
  return Number.isNaN(timestamp) ? null : timestamp
}

function formatDateTime(value: string) {
  if (!value) return '保存后生成'
  return sharedFormatDateTime(value)
}

function issueUploadContext(part: string) {
  return `issue-${form.jira_key || route.params.jiraKey || 'new'}-${part}`
}

watch(() => route.params.jiraKey, load)
onMounted(load)
onBeforeUnmount(() => {
  void flushManualFields()
})
</script>

<style scoped>
.card {
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  padding: 18px 20px;
}

.field-label {
  color: #6b7280;
  font-size: 11px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.025em;
  display: block;
  margin-bottom: 4px;
}

.todo-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 12px;
  margin-top: 6px;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
}

.todo-item.done {
  opacity: 0.6;
  text-decoration: line-through;
}

</style>
