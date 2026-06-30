<template>
  <div class="day-panel">
    <div class="flex items-center gap-2 mb-2">
      <span class="text-sm font-semibold text-gray-800">{{ day.date }}</span>
      <span class="text-xs px-1.5 py-0.5 rounded bg-gray-100 text-gray-500">{{ day.weekday }}</span>
      <span class="text-xs text-gray-300 ml-auto">{{ day.comments.length }} 评论 · {{ day.entries.length }} 手记</span>
    </div>

    <div v-if="day.comments.length === 0 && day.entries.length === 0" class="text-sm text-gray-400 py-2">这一天还没有记录</div>

    <article v-for="c in day.comments" :key="`c-${c.event_id}`" class="day-item">
      <div class="flex items-center gap-2 mb-1">
        <a class="ref-chip" :class="c.source" @click="goTo(c.url)">{{ c.source === 'issue' ? c.ref_key : c.ref_title }}</a>
        <span class="text-xs text-gray-400">{{ formatTime(c.happened_at) }}</span>
      </div>
      <MarkdownView :content="c.content_md" />
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
import { ref } from 'vue'
import { NButton, NPopconfirm, useMessage } from 'naive-ui'
import { useRouter } from 'vue-router'
import MarkdownEditor from './MarkdownEditor.vue'
import MarkdownView from './MarkdownView.vue'
import { api } from '../api/client'
import { formatTime } from '../utils/datetime'
import type { DayWork, DayEntry } from '../types'

const props = defineProps<{ day: DayWork }>()
const emit = defineEmits<{ (e: 'changed'): void }>()
const router = useRouter()
const message = useMessage()
const draft = ref('')
const saving = ref(false)
const entryEditorVisible = ref(false)
const entryEditor = ref<MarkdownEditorExpose | null>(null)

function goTo(url: string) {
  router.push(url)
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

.entry-editor {
  border-top: 1px solid #eef0f3;
  padding-top: 12px;
}
</style>
