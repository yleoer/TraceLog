<template>
  <n-spin :show="loading">
    <div class="space-y-6 max-w-4xl mx-auto">
      <div class="flex items-center justify-between">
        <h1 class="text-xl font-semibold text-gray-900">{{ isNew ? '新增临时需求' : form.title }}</h1>
        <div class="flex items-center gap-2">
          <n-button v-if="!isNew" size="small" @click="downloadUrl(`/export/temp-tasks/${form.id}.md`)">导出</n-button>
          <n-popconfirm v-if="!isNew" @positive-click="removeTask">
            <template #trigger><n-button size="small" type="error" ghost>删除</n-button></template>
            删除后该临时需求的评论将一并移除，确认删除？
          </n-popconfirm>
          <n-button v-if="isNew" type="primary" size="small" :loading="saving" @click="save">创建</n-button>
        </div>
      </div>

      <div class="card">
        <n-form label-placement="top" size="small">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-x-4">
            <n-form-item label="标题"><n-input v-model:value="form.title" placeholder="临时需求标题" @update:value="saveMetaSoon" /></n-form-item>
            <n-form-item label="来源"><n-input v-model:value="form.source" placeholder="领导、同事、会议、线上问题..." @update:value="saveMetaSoon" /></n-form-item>
            <n-form-item label="状态"><n-select v-model:value="form.status" :options="statusOptions" @update:value="saveMetaSoon" /></n-form-item>
            <n-form-item label="优先级"><n-select v-model:value="form.priority" :options="priorityOptions" @update:value="saveMetaSoon" /></n-form-item>
            <n-form-item label="开始时间"><TimePresetDatePicker v-model:value="schedule.startedAt" @update:value="saveMetaSoon" /></n-form-item>
            <n-form-item label="结束时间"><TimePresetDatePicker v-model:value="schedule.completedAt" @update:value="saveMetaSoon" /></n-form-item>
          </div>
          <n-form-item label="标签"><n-dynamic-tags v-model:value="form.tags" @update:value="saveMetaSoon" /></n-form-item>
          <n-form-item label="Jira 编号"><n-input v-model:value="form.converted_jira_key" placeholder="例如 GCS-45000" @update:value="saveMetaSoon" /></n-form-item>
        </n-form>
      </div>

      <div class="card">
        <div v-if="contentEditing || isNew" class="space-y-3">
          <MarkdownEditor
            v-if="editorVisible"
            ref="contentEditor"
            v-model="contentDraft"
            :upload-context="tempTaskUploadContext('content')"
            placeholder="记录临时需求的背景、处理过程和后续事项..."
          />
          <div v-if="!isNew" class="flex justify-end gap-2">
            <n-button size="small" @click="cancelContentEdit">取消</n-button>
            <n-button size="small" type="primary" :loading="contentSaving" @click="saveContentEdit">保存</n-button>
          </div>
        </div>
        <div v-else class="content-view" title="双击编辑内容" @dblclick="startContentEdit">
          <MarkdownView v-if="form.content_md" :content="form.content_md" />
          <div v-else class="empty-content">暂无内容，双击添加</div>
        </div>
      </div>

      <section v-if="!isNew" class="card">
        <h2 class="text-sm font-semibold text-gray-900 mb-4">我的记录</h2>
        <CommentTimeline
          :events="events"
          :load="loadEvents"
          :create="createComment"
          :update="updateComment"
          :remove="deleteComment"
          :upload-context="tempTaskUploadContext('timeline')"
        />
      </section>
    </div>
  </n-spin>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import {
  NButton,
  NDynamicTags,
  NForm,
  NFormItem,
  NInput,
  NPopconfirm,
  NSelect,
  NSpin,
  useMessage
} from 'naive-ui'
import { useRoute, useRouter } from 'vue-router'
import { api, downloadUrl } from '../api/client'
import MarkdownView from '../components/MarkdownView.vue'
import CommentTimeline from '../components/CommentTimeline.vue'
import TimePresetDatePicker from '../components/TimePresetDatePicker.vue'
import type { TempTask, TempTaskEvent } from '../types'

const MarkdownEditor = defineAsyncComponent(() => import('../components/MarkdownEditor.vue'))
const route = useRoute()
const router = useRouter()
const message = useMessage()
const loading = ref(false)
const saving = ref(false)
const contentSaving = ref(false)
const isNew = computed(() => !route.params.id)
const editorVisible = computed(() => isNew.value || !loading.value)
const contentEditor = ref<MarkdownEditorExpose | null>(null)
const contentEditing = ref(false)
const contentDraft = ref('')
let loadToken = 0
let metaSaveTimer: number | undefined
let metaSaveToken = 0

const events = ref<TempTaskEvent[]>([])

interface TimelineEvent {
  id: number
  event_type: string
  happened_at: string
}

interface TempTaskForm {
  id: number
  title: string
  source: string
  status: string
  priority: string
  tags: string[]
  content_md: string
  started_at: string
  completed_at: string
  converted_to_jira: boolean
  converted_jira_key: string
  created_at: string
  updated_at: string
}

const form = reactive<TempTaskForm>({
  id: 0,
  title: '',
  source: '',
  status: 'todo',
  priority: 'medium',
  tags: [],
  content_md: '',
  started_at: '',
  completed_at: '',
  converted_to_jira: false,
  converted_jira_key: '',
  created_at: '',
  updated_at: ''
})

const schedule = reactive({
  startedAt: null as number | null,
  completedAt: null as number | null
})

const emptyForm: TempTaskForm = {
  id: 0,
  title: '',
  source: '',
  status: 'todo',
  priority: 'medium',
  tags: [],
  content_md: '',
  started_at: '',
  completed_at: '',
  converted_to_jira: false,
  converted_jira_key: '',
  created_at: '',
  updated_at: ''
}

const statusOptions = [
  { label: '待处理', value: 'todo' },
  { label: '处理中', value: 'processing' },
  { label: '已完成', value: 'done' },
  { label: '挂起', value: 'suspended' }
]
const priorityOptions = [
  { label: '低', value: 'low' },
  { label: '中', value: 'medium' },
  { label: '高', value: 'high' },
  { label: '紧急', value: 'urgent' }
]

async function load() {
  await flushMeta()
  const token = ++loadToken
  const id = route.params.id
  if (!id) {
    resetForm()
    return
  }
  loading.value = true
  try {
    const task = await api.getTempTask(String(id))
    if (token !== loadToken) return
    Object.assign(form, task)
    contentDraft.value = form.content_md
    contentEditing.value = false
    hydrateSchedule()
    await loadEvents(String(id))
  } catch (error) {
    if (token !== loadToken) return
    message.error((error as Error).message)
  } finally {
    if (token === loadToken) loading.value = false
  }
}

async function loadEvents(id = String(route.params.id || '')) {
  if (!id) return
  events.value = await api.listTempTaskEvents(id)
}

async function createComment(content: string) {
  await api.createTempTaskEvent(String(route.params.id), { event_type: 'note', content_md: content })
}

async function updateComment(event: TimelineEvent, content: string) {
  await api.updateTempTaskEvent(event.id, { event_type: event.event_type || 'note', content_md: content, happened_at: event.happened_at })
}

async function deleteComment(event: TimelineEvent) {
  await api.deleteTempTaskEvent(event.id)
}

async function removeTask() {
  try {
    await api.deleteTempTask(String(route.params.id))
    message.success('临时需求已删除')
    router.push('/temp-tasks')
  } catch (error) {
    message.error((error as Error).message)
  }
}

async function save() {
  contentEditor.value?.flush()
  if (isNew.value) {
    form.content_md = contentDraft.value
  }
  if (!form.title.trim()) {
    message.error('标题不能为空')
    return
  }
  serializeSchedule()
  saving.value = true
  try {
    if (isNew.value) {
      const created = await api.createTempTask(form)
      message.success('临时需求已创建')
      router.push(`/temp-tasks/${created.id}`)
    } else {
      Object.assign(form, await api.updateTempTask(String(route.params.id), form))
      hydrateSchedule()
      message.success('临时需求已保存')
    }
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    saving.value = false
  }
}

function startContentEdit() {
  contentDraft.value = form.content_md
  contentEditing.value = true
}

function cancelContentEdit() {
  contentDraft.value = form.content_md
  contentEditing.value = false
}

async function saveContentEdit() {
  contentEditor.value?.flush()
  contentSaving.value = true
  try {
    serializeSchedule()
    const updated = await api.updateTempTask(String(route.params.id), { ...form, content_md: contentDraft.value })
    Object.assign(form, updated)
    hydrateSchedule()
    contentDraft.value = form.content_md
    contentEditing.value = false
    message.success('内容已保存')
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    contentSaving.value = false
  }
}

function saveMetaSoon() {
  serializeSchedule()
  if (isNew.value) return
  window.clearTimeout(metaSaveTimer)
  metaSaveTimer = window.setTimeout(() => {
    metaSaveTimer = undefined
    void saveMeta()
  }, 450)
}

async function saveMeta() {
  if (isNew.value || !route.params.id) return
  if (!form.title.trim()) {
    message.error('标题不能为空')
    return
  }
  const token = ++metaSaveToken
  try {
    const updated = await api.updateTempTask(String(route.params.id), form)
    if (token !== metaSaveToken) return
    Object.assign(form, updated)
    hydrateSchedule()
  } catch (error) {
    message.error((error as Error).message)
  }
}

async function flushMeta() {
  if (metaSaveTimer === undefined) return
  window.clearTimeout(metaSaveTimer)
  metaSaveTimer = undefined
  await saveMeta()
}

function hydrateSchedule() {
  schedule.startedAt = parseMillis(form.started_at)
  schedule.completedAt = parseMillis(form.completed_at)
}

function serializeSchedule() {
  form.started_at = schedule.startedAt ? new Date(schedule.startedAt).toISOString() : ''
  form.completed_at = schedule.completedAt ? new Date(schedule.completedAt).toISOString() : ''
}

function parseMillis(value: string) {
  if (!value) return null
  const timestamp = Date.parse(value)
  return Number.isNaN(timestamp) ? null : timestamp
}

function tempTaskUploadContext(part: string) {
  return `temp-task-${form.id || route.params.id || 'new'}-${part}`
}

function resetForm() {
  Object.assign(form, { ...emptyForm, tags: [] })
  contentDraft.value = ''
  contentEditing.value = false
  events.value = []
  schedule.startedAt = null
  schedule.completedAt = null
}

watch(() => route.params.id, load)
onMounted(load)
onBeforeUnmount(() => {
  void flushMeta()
})

type MarkdownEditorExpose = {
  flush: () => string
}
</script>

<style scoped>
.card {
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  padding: 18px 20px;
}

.content-view {
  min-height: 120px;
  cursor: text;
}

.empty-content {
  color: #9ca3af;
  font-size: 13px;
  padding: 8px 0;
}
</style>
