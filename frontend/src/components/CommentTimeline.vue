<template>
  <div>
    <div class="border-b border-gray-100 pb-5 mb-5">
      <div class="flex items-center justify-between mb-2">
        <h3 class="text-xs font-medium text-gray-600">{{ newTitle }}</h3>
        <n-button size="tiny" type="primary" :loading="saving" @click="saveComment">保存评论</n-button>
      </div>
      <MarkdownEditor
        ref="commentEditor"
        v-model="commentDraft"
        :rows="6"
        :upload-context="`${uploadContext}-comment`"
        placeholder="写下进展、分析、结论..."
      />
    </div>

    <slot />

    <div>
      <div class="flex items-center justify-between mb-3">
        <h3 class="text-xs font-medium text-gray-600">历史评论</h3>
        <span class="text-xs text-gray-400">{{ events.length }} 条记录</span>
      </div>
      <div v-if="events.length === 0" class="text-sm text-gray-400">还没有评论。</div>
      <article v-for="event in events" :key="event.id" class="comment-item">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs text-gray-400">{{ formatDateTime(event.happened_at || event.created_at) }}</span>
          <div class="flex gap-2">
            <template v-if="editingId !== event.id">
              <n-button text size="tiny" type="primary" @click="startEdit(event)">编辑</n-button>
              <n-popconfirm @positive-click="deleteComment(event)">
                <template #trigger><n-button text size="tiny" type="error">删除</n-button></template>
                确认删除这条评论？
              </n-popconfirm>
            </template>
            <template v-else>
              <n-button text size="tiny" @click="cancelEdit">取消</n-button>
              <n-button text size="tiny" type="primary" :loading="saving" @click="updateComment(event)">保存</n-button>
            </template>
          </div>
        </div>
        <MarkdownEditor
          v-if="editingId === event.id"
          :ref="setEditingEditor"
          v-model="editingContent"
          :rows="5"
          :upload-context="`${uploadContext}-comment-${event.id}`"
          placeholder="更新这条记录..."
        />
        <MarkdownView v-else :content="event.content_md" />
      </article>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from 'vue'
import { NButton, NPopconfirm, useMessage } from 'naive-ui'
import MarkdownView from './MarkdownView.vue'
import { formatDateTime } from '../utils/datetime'
import { useDraftAutosave } from '../composables/useDraftAutosave'

const MarkdownEditor = defineAsyncComponent(() => import('./MarkdownEditor.vue'))

interface TimelineEvent {
  id: number
  event_type: string
  content_md: string
  happened_at: string
  created_at: string
}

const props = withDefaults(defineProps<{
  events: TimelineEvent[]
  newTitle?: string
  draftKey?: string
  uploadContext?: string
  load: () => Promise<void>
  create: (content: string) => Promise<void>
  update: (event: TimelineEvent, content: string) => Promise<void>
  remove: (event: TimelineEvent) => Promise<void>
}>(), {
  newTitle: '新增评论',
  draftKey: '',
  uploadContext: 'comment'
})

const message = useMessage()
const commentDraft = ref('')
const editingId = ref<number | null>(null)
const editingContent = ref('')
const commentEditor = ref<MarkdownEditorExpose | null>(null)
const editingEditor = ref<MarkdownEditorExpose | null>(null)
const saving = ref(false)
const draftKeyRef = computed(() => props.draftKey)
const { clearDraft } = useDraftAutosave(draftKeyRef, commentDraft)

async function saveComment() {
  commentEditor.value?.flush()
  const content = commentDraft.value.trim()
  if (!content) {
    message.error('评论内容不能为空')
    return
  }
  saving.value = true
  try {
    await props.create(content)
    commentDraft.value = ''
    clearDraft()
    await props.load()
    message.success('评论已保存')
  } catch (error) {
    showError(error)
  } finally {
    saving.value = false
  }
}

function startEdit(event: TimelineEvent) {
  editingId.value = event.id
  editingContent.value = event.content_md
  editingEditor.value = null
}

function cancelEdit() {
  editingId.value = null
  editingContent.value = ''
  editingEditor.value = null
}

async function updateComment(event: TimelineEvent) {
  editingEditor.value?.flush()
  const content = editingContent.value.trim()
  if (!content) {
    message.error('评论内容不能为空')
    return
  }
  saving.value = true
  try {
    await props.update(event, content)
    cancelEdit()
    await props.load()
    message.success('评论已更新')
  } catch (error) {
    showError(error)
  } finally {
    saving.value = false
  }
}

async function deleteComment(event: TimelineEvent) {
  try {
    await props.remove(event)
    await props.load()
    message.success('评论已删除')
  } catch (error) {
    showError(error)
  }
}

function showError(error: unknown) {
  message.error((error as Error).message)
}

function setEditingEditor(editor: unknown) {
  editingEditor.value = (editor as MarkdownEditorExpose | null) ?? null
}

type MarkdownEditorExpose = {
  flush: () => string
}
</script>

<style scoped>
.comment-item {
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 12px 14px;
  margin-top: 8px;
}
</style>
