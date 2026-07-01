<template>
  <div class="day-panel">
    <div class="flex items-center gap-2 mb-2">
      <span class="text-sm font-semibold text-gray-800">{{ day.date }}</span>
      <span class="text-xs px-1.5 py-0.5 rounded bg-gray-100 text-gray-500">{{ day.weekday }}</span>
      <span class="text-xs text-gray-300 ml-auto">{{ eventCount }} 事件 · {{ day.entries.length }} 手记</span>
    </div>

    <div v-if="eventCount === 0 && day.entries.length === 0" class="text-sm text-gray-400 py-2">这一天还没有记录</div>

    <article v-for="activity in activities" :key="activityKey(activity)" class="day-item">
      <div class="flex items-center gap-2 mb-1">
        <a class="ref-chip" :class="activity.source" @click="goTo(activity.url)">{{ activityLabel(activity) }}</a>
        <span class="text-xs text-gray-400">{{ activity.comments.length }} 条</span>
      </div>
      <div class="space-y-2">
        <div v-for="c in activity.comments" :key="commentKey(c)" class="event-row">
          <div class="flex items-center gap-2">
            <span class="event-chip" :class="c.event_type">{{ eventTypeLabel(c.event_type) }}</span>
            <span class="text-xs text-gray-400">{{ formatTime(c.happened_at) }}</span>
          </div>
          <MarkdownView v-if="c.content_md" :content="c.content_md" />
        </div>
      </div>
    </article>

    <div v-for="e in day.entries" :key="`e-${e.id}`" class="day-item flex items-start gap-2">
      <span class="entry-chip">手记</span>
      <div class="flex-1 min-w-0"><MarkdownView :content="e.content_md" /></div>
      <n-popconfirm @positive-click="removeEntry(e)">
        <template #trigger><n-button text size="tiny" type="error">删除</n-button></template>
        删除这条手记？
      </n-popconfirm>
    </div>

    <div v-if="entryEditorVisible" class="entry-editor mt-3">
      <MarkdownEditor ref="entryEditor" v-model="draft" :rows="4" :upload-context="`day-${day.date}-entry`" placeholder="给这一天加一条手记..." />
      <div class="flex justify-end gap-2 mt-2">
        <n-button size="small" @click="cancelEntry">取消</n-button>
        <n-button size="small" type="primary" :loading="saving" @click="addEntry">添加</n-button>
      </div>
    </div>
    <div v-else class="flex justify-end mt-3">
      <n-button size="small" @click="showEntryEditor">添加手记</n-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from 'vue'
import { NButton, NPopconfirm, useMessage } from 'naive-ui'
import { useRouter } from 'vue-router'
import MarkdownView from './MarkdownView.vue'
import { api } from '../api/client'
import { formatTime } from '../utils/datetime'
import type { DayWork, DayEntry, DayActivity, DayComment } from '../types'

const MarkdownEditor = defineAsyncComponent(() => import('./MarkdownEditor.vue'))
const props = defineProps<{ day: DayWork }>()
const emit = defineEmits<{ (e: 'changed'): void }>()
const router = useRouter()
const message = useMessage()
const draft = ref('')
const saving = ref(false)
const entryEditorVisible = ref(false)
const entryEditor = ref<MarkdownEditorExpose | null>(null)
const activities = computed(() => props.day.activities?.length ? props.day.activities : groupComments(props.day.comments))
const eventCount = computed(() => activities.value.reduce((total, activity) => total + activity.comments.length, 0))

function goTo(url: string) {
  router.push(url)
}

function groupComments(comments: DayComment[]) {
  const grouped: DayActivity[] = []
  const index = new Map<string, DayActivity>()
  for (const comment of comments) {
    const key = `${comment.source}:${comment.ref_id}:${comment.ref_key}`
    let activity = index.get(key)
    if (!activity) {
      activity = {
        source: comment.source,
        ref_id: comment.ref_id,
        ref_key: comment.ref_key,
        ref_title: comment.ref_title,
        url: comment.url,
        started_at: comment.happened_at,
        comments: []
      }
      grouped.push(activity)
      index.set(key, activity)
    }
    activity.comments.push(comment)
  }
  return grouped
}

function activityKey(activity: DayActivity) {
  return `${activity.source}-${activity.ref_id}-${activity.ref_key}`
}

function commentKey(comment: DayComment) {
  return `${comment.source}-${comment.event_type}-${comment.event_id}-${comment.happened_at}`
}

function activityLabel(activity: DayActivity) {
  return activity.source === 'issue' ? activity.ref_key : activity.ref_title
}

function eventTypeLabel(type: string) {
  if (type === 'created') return '添加'
  if (type === 'deleted') return '删除'
  return type || '事件'
}

async function addEntry() {
  entryEditor.value?.flush()
  const content = draft.value.trim()
  if (!content) return
  saving.value = true
  try {
    await api.createDayEntry({ date: props.day.date, content_md: content })
    draft.value = ''
    entryEditorVisible.value = false
    emit('changed')
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    saving.value = false
  }
}

function showEntryEditor() {
  entryEditorVisible.value = true
}

function cancelEntry() {
  draft.value = ''
  entryEditorVisible.value = false
}

async function removeEntry(entry: DayEntry) {
  try {
    await api.deleteDayEntry(entry.id)
    emit('changed')
  } catch (error) {
    message.error((error as Error).message)
  }
}

type MarkdownEditorExpose = {
  flush: () => string
}
</script>

<style scoped>
.day-panel {
  padding: 14px 16px;
  border: 1px solid #eef0f3;
  border-radius: 8px;
  background: #fcfcfd;
}

.day-item {
  padding: 8px 0;
  border-bottom: 1px dashed #eef0f3;
}

.day-item:last-of-type {
  border-bottom: none;
}

.ref-chip {
  display: inline-flex;
  align-items: center;
  font-size: 12px;
  font-weight: 600;
  padding: 1px 7px;
  border-radius: 4px;
  cursor: pointer;
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ref-chip.issue {
  background: #eff6ff;
  color: #2563eb;
}

.ref-chip.issue:hover {
  background: #dbeafe;
}

.ref-chip.temp_task {
  background: #f0fdf4;
  color: #16a34a;
}

.ref-chip.temp_task:hover {
  background: #dcfce7;
}

.entry-chip {
  flex-shrink: 0;
  font-size: 11px;
  padding: 1px 7px;
  border-radius: 4px;
  background: #fff7ed;
  color: #ea580c;
  margin-top: 2px;
}

.event-row {
  display: grid;
  gap: 4px;
  padding: 6px 0 6px 10px;
  border-left: 2px solid #eef0f3;
}

.event-chip {
  flex-shrink: 0;
  font-size: 11px;
  padding: 1px 7px;
  border-radius: 4px;
  background: #f3f4f6;
  color: #4b5563;
}

.event-chip.created {
  background: #ecfdf5;
  color: #059669;
}

.event-chip.deleted {
  background: #fef2f2;
  color: #dc2626;
}

.entry-editor {
  border-top: 1px solid #eef0f3;
  padding-top: 12px;
}
</style>
