<template>
  <div class="markdown-editor" :style="{ '--editor-min-height': `${rows * 24}px` }">
    <div ref="editorElement" class="vditor-host" />
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import Vditor from 'vditor'
import 'vditor/dist/index.css'
import { useMessage } from 'naive-ui'
import { api } from '../api/client'

type EditorMode = 'wysiwyg' | 'sv' | 'ir'

const props = withDefaults(defineProps<{ modelValue: string; rows?: number; placeholder?: string; uploadContext?: string }>(), {
  rows: 8,
  placeholder: '记录进展、分析、结论...',
  uploadContext: ''
})

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const message = useMessage()
const editorElement = ref<HTMLDivElement | null>(null)
const editor = shallowRef<Vditor | null>(null)
const mounted = ref(false)
const editorReady = ref(false)
const syncing = ref(false)
const settingValue = ref(false)
const pendingMarkdown = ref<string | null>(null)
const uploadBlobByURL = new Map<string, string>()
const uploadBlobPromiseByURL = new Map<string, Promise<string>>()
const uploadURLByBlob = new Map<string, string>()
const sessionUploadedURLs = new Set<string>()
const cleanupTimers = new Map<string, number>()

defineExpose({ flush })

const toolbar = [
  'headings',
  'bold',
  'italic',
  'strike',
  'link',
  '|',
  'list',
  'ordered-list',
  'check',
  '|',
  'quote',
  'code',
  'inline-code',
  'table',
  '|',
  'upload',
  '|',
  'undo',
  'redo',
  '|',
  {
    name: 'source-mode',
    tip: '切换源码模式 Ctrl + /',
    icon: '<svg><use xlink:href="#vditor-icon-code"></use></svg>',
    click: (event: Event) => {
      event.preventDefault()
      toggleEditMode()
    }
  },
  'edit-mode'
]

watch(
  () => props.modelValue,
  (value) => {
    const current = editor.value
    if (syncing.value) return
    const normalized = normalizeMarkdown(value || '')
    if (!current || !editorReady.value) {
      pendingMarkdown.value = normalized
      return
    }
    if (normalized === editorMarkdown()) return
    setEditorValue(normalized)
  }
)

onMounted(async () => {
  mounted.value = true
  await nextTick()
  if (!mounted.value) return
  editorElement.value?.addEventListener('error', handleUploadedImageError, true)
  mountEditor('ir', props.modelValue || '')
})

onBeforeUnmount(() => {
  mounted.value = false
  editorReady.value = false
  editorElement.value?.removeEventListener('error', handleUploadedImageError, true)
  clearCleanupTimers()
  void cleanupUnusedUploadedImages(normalizeMarkdown(props.modelValue || ''))
  revokeUploadBlobURLs()
  destroyEditor()
  editor.value = null
})

function mountEditor(mode: EditorMode, value: string) {
  if (!mounted.value || !editorElement.value) return
  editorReady.value = false
  pendingMarkdown.value = normalizeMarkdown(value)
  let instance: Vditor | null = null
  instance = new Vditor(editorElement.value, {
    cache: { enable: false },
    cdn: '/vditor',
    counter: { enable: false },
    height: 'auto',
    icon: 'ant',
    lang: 'zh_CN',
    minHeight: props.rows * 24,
    mode,
    placeholder: props.placeholder,
    toolbar,
    toolbarConfig: { hide: false, pin: false },
    value,
    preview: {
      actions: [],
      delay: 200,
      hljs: { enable: false, lineNumber: false, style: 'github' },
      markdown: {
        autoSpace: false,
        codeBlockPreview: true,
        fixTermTypo: false,
        footnotes: true,
        gfmAutoLink: true,
        linkBase: '',
        linkPrefix: '',
        listStyle: false,
        mark: false,
        mathBlockPreview: false,
        paragraphBeginningSpace: false,
        sanitize: true,
        toc: false
      },
      math: { engine: 'KaTeX', inlineDigit: false, macros: {} },
      mode: 'editor',
      render: { media: { enable: false } },
      theme: { current: 'light' }
    },
    upload: {
      accept: 'image/*',
      handler: uploadImages,
      max: 8 << 20,
      multiple: true
    },
    input: (changedValue) => {
      if (!mounted.value || !editorReady.value || settingValue.value) return
      const normalized = normalizeMarkdown(changedValue)
      syncing.value = true
      emit('update:modelValue', normalized)
      syncing.value = false
      scheduleUnusedUploadedImageCleanup(normalized)
    },
    keydown: (event) => {
      if (!isMarkdownSourceShortcut(event)) return
      event.preventDefault()
      toggleEditMode()
    },
    after: () => {
      if (!mounted.value || editor.value !== instance || !instance) return
      editorReady.value = true
      const nextValue = pendingMarkdown.value ?? props.modelValue ?? ''
      pendingMarkdown.value = null
      if (normalizeMarkdown(nextValue) !== editorMarkdown(instance)) {
        setEditorValue(nextValue)
      }
    }
  })
  editor.value = instance
}

async function uploadImages(files: File[]): Promise<null> {
  if (!mounted.value || !editorReady.value) return null
  const imageFiles = files.filter((file): file is File => file instanceof File && file.type.startsWith('image/'))
  if (imageFiles.length === 0) {
    message.error('只能上传图片')
    return null
  }

  try {
    const markdown: string[] = []
    for (const file of imageFiles) {
      const uploaded = await api.uploadImage(file, props.uploadContext)
      if (!mounted.value) return null
      sessionUploadedURLs.add(uploaded.url)
      markdown.push(`![${escapeMarkdownImageAlt(file.name || 'pasted image')}](${uploaded.url})`)
    }
    editor.value?.insertMD(markdown.join('\n') + '\n')
    emitCurrentValue()
    scheduleUnusedUploadedImageCleanup(editorMarkdown())
    message.success(imageFiles.length === 1 ? '图片已插入' : `${imageFiles.length} 张图片已插入`)
  } catch (error) {
    message.error((error as Error).message)
  }
  return null
}

function toggleEditMode() {
  const current = editor.value
  if (!current || !editorReady.value) return

  const nextMode: EditorMode = current.getCurrentMode() === 'sv' ? 'ir' : 'sv'
  const value = editorMarkdown(current)
  destroyEditor()
  editor.value = null
  editorReady.value = false

  void nextTick(() => {
    if (!mounted.value) return
    mountEditor(nextMode, value)
    editor.value?.focus()
  })
}

function handleUploadedImageError(event: Event) {
  const image = event.target
  if (!(image instanceof HTMLImageElement)) return
  const uploadSrc = image.getAttribute('src')
  if (!uploadSrc?.startsWith('/uploads/')) return
  void showUploadedImageFallback(image, uploadSrc)
}

async function showUploadedImageFallback(image: HTMLImageElement, uploadSrc: string) {
  const objectURL = await objectURLForUploadedImage(uploadSrc)
  if (mounted.value && objectURL !== uploadSrc) image.setAttribute('src', objectURL)
}

async function objectURLForUploadedImage(url: string) {
  const cached = uploadBlobByURL.get(url)
  if (cached) return cached
  const pending = uploadBlobPromiseByURL.get(url)
  if (pending) return pending

  const pendingObjectURL = createUploadedImageObjectURL(url)
  uploadBlobPromiseByURL.set(url, pendingObjectURL)
  try {
    return await pendingObjectURL
  } finally {
    uploadBlobPromiseByURL.delete(url)
  }
}

async function createUploadedImageObjectURL(url: string) {
  try {
    const image = await api.getUploadedImageDataURL(url)
    const blob = blobFromDataURL(image.data_url)
    const objectURL = URL.createObjectURL(blob)
    uploadBlobByURL.set(url, objectURL)
    uploadURLByBlob.set(objectURL, image.url)
    return objectURL
  } catch {
    return url
  }
}

function normalizeMarkdown(value: string) {
  let normalized = value || ''
  for (const [objectURL, uploadURL] of uploadURLByBlob.entries()) {
    normalized = normalized.split(objectURL).join(uploadURL)
  }
  return normalized.trim()
}

function emitCurrentValue() {
  const current = editor.value
  if (!current || !mounted.value || !editorReady.value) return
  const normalized = editorMarkdown(current)
  syncing.value = true
  emit('update:modelValue', normalized)
  syncing.value = false
  scheduleUnusedUploadedImageCleanup(normalized)
}

function flush() {
  emitCurrentValue()
  return editorReady.value ? editorMarkdown() : normalizeMarkdown(props.modelValue || '')
}

function revokeUploadBlobURLs() {
  for (const objectURL of uploadBlobByURL.values()) {
    URL.revokeObjectURL(objectURL)
  }
  uploadBlobByURL.clear()
  uploadBlobPromiseByURL.clear()
  uploadURLByBlob.clear()
}

function scheduleUnusedUploadedImageCleanup(markdown: string, delay = 1500) {
  for (const url of sessionUploadedURLs) {
    if (markdown.includes(url) || cleanupTimers.has(url)) continue
    const timer = window.setTimeout(() => {
      cleanupTimers.delete(url)
      void cleanupUploadedImage(url)
    }, delay)
    cleanupTimers.set(url, timer)
  }
}

async function cleanupUnusedUploadedImages(markdown: string) {
  await Promise.all(Array.from(sessionUploadedURLs)
    .filter((url) => !markdown.includes(url))
    .map((url) => cleanupUploadedImage(url)))
}

async function cleanupUploadedImage(url: string) {
  const currentMarkdown = editorReady.value ? editorMarkdown() : normalizeMarkdown(props.modelValue || '')
  if (currentMarkdown.includes(url)) return
  try {
    await api.deleteUploadedImage(url)
    sessionUploadedURLs.delete(url)
    revokeUploadBlobURL(url)
  } catch {
    // 清理失败不影响编辑，后续打开编辑器时仍可继续引用原文件。
  }
}

function clearCleanupTimers() {
  for (const timer of cleanupTimers.values()) {
    window.clearTimeout(timer)
  }
  cleanupTimers.clear()
}

function revokeUploadBlobURL(url: string) {
  const objectURL = uploadBlobByURL.get(url)
  if (!objectURL) return
  URL.revokeObjectURL(objectURL)
  uploadBlobByURL.delete(url)
  uploadURLByBlob.delete(objectURL)
}

function isMarkdownSourceShortcut(event: KeyboardEvent) {
  return !event.isComposing && event.ctrlKey && !event.altKey && !event.shiftKey && (event.key === '/' || event.code === 'Slash')
}

function editorMarkdown(current = editor.value) {
  if (!current || !editorReady.value) return normalizeMarkdown(props.modelValue || '')
  return normalizeMarkdown(current.getValue())
}

function setEditorValue(value: string) {
  const current = editor.value
  if (!current || !editorReady.value) {
    pendingMarkdown.value = normalizeMarkdown(value)
    return
  }
  settingValue.value = true
  try {
    current.setValue(value || '', true)
  } finally {
    void nextTick(() => {
      settingValue.value = false
    })
  }
}

function destroyEditor() {
  const current = editor.value
  if (!current) return
  if (!editorReady.value) {
    markEditorDestroyed(current)
    return
  }
  current.destroy()
}

function markEditorDestroyed(current: object) {
  ;(current as { isDestroyed?: boolean }).isDestroyed = true
}

function escapeMarkdownImageAlt(value: string) {
  return value.replace(/\\/g, '\\\\').replace(/\]/g, '\\]')
}

function blobFromDataURL(dataURL: string) {
  const [metadata, payload = ''] = dataURL.split(',', 2)
  const mimeType = metadata.match(/^data:([^;]+)/)?.[1] || 'application/octet-stream'
  const binary = metadata.includes(';base64') ? window.atob(payload) : window.decodeURIComponent(payload)
  const bytes = new Uint8Array(binary.length)
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }
  return new Blob([bytes], { type: mimeType })
}
</script>

<style scoped>
.markdown-editor {
  border: 1px solid #d9dee8;
  border-radius: 8px;
  overflow: hidden;
}

.vditor-host :deep(.vditor) {
  border: 0;
  border-radius: 0;
  min-height: var(--editor-min-height);
}

.vditor-host :deep(.vditor-toolbar) {
  background: #f8fafc;
  border-bottom: 1px solid #e5e7eb;
}

.vditor-host :deep(.vditor-reset),
.vditor-host :deep(.vditor-ir),
.vditor-host :deep(.vditor-sv),
.vditor-host :deep(.vditor-wysiwyg) {
  color: #111827;
  font-size: 14px;
}

.vditor-host :deep(.vditor-reset h2),
.vditor-host :deep(.vditor-wysiwyg h2) {
  color: #0f172a;
  font-size: 20px;
  font-weight: 700;
}

.vditor-host :deep(.vditor-reset h3),
.vditor-host :deep(.vditor-wysiwyg h3) {
  color: #1f2937;
  font-size: 16px;
  font-weight: 700;
}

.vditor-host :deep(.vditor-reset img),
.vditor-host :deep(.vditor-wysiwyg img) {
  border-radius: 6px;
  max-width: 100%;
}
</style>
