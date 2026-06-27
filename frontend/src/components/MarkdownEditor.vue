<template>
  <div class="rich-editor" :style="{ '--editor-min-height': `${rows * 24}px` }">
    <div v-if="editor" class="editor-toolbar">
      <n-tooltip v-for="action in actions" :key="action.key" trigger="hover">
        <template #trigger>
          <n-button
            size="small"
            quaternary
            :type="action.active() ? 'primary' : 'default'"
            :disabled="action.disabled?.()"
            @click="action.run"
          >
            <template #icon>
              <component :is="action.icon" :size="16" />
            </template>
          </n-button>
        </template>
        {{ action.label }}
      </n-tooltip>

      <span class="toolbar-divider" />

      <n-tooltip trigger="hover">
        <template #trigger>
          <n-button size="small" quaternary :loading="uploading" @click="pickImage">
            <template #icon>
              <ImageIcon :size="16" />
            </template>
          </n-button>
        </template>
        上传图片
      </n-tooltip>
      <input ref="fileInput" class="file-input" type="file" accept="image/*" @change="uploadImage" />
    </div>

    <EditorContent class="editor-surface" :editor="editor" />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { EditorContent, useEditor } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Image from '@tiptap/extension-image'
import Link from '@tiptap/extension-link'
import Placeholder from '@tiptap/extension-placeholder'
import MarkdownIt from 'markdown-it'
import TurndownService from 'turndown'
import {
  Bold,
  Code,
  Heading2,
  Image as ImageIcon,
  Italic,
  Link2,
  List,
  ListOrdered,
  Quote,
  Redo2,
  Undo2
} from 'lucide-vue-next'
import { useMessage } from 'naive-ui'
import { api } from '../api/client'

const props = withDefaults(defineProps<{ modelValue: string; rows?: number }>(), {
  rows: 8
})

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const message = useMessage()
const fileInput = ref<HTMLInputElement | null>(null)
const uploading = ref(false)
const syncing = ref(false)

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true
})

const turndown = new TurndownService({
  headingStyle: 'atx',
  bulletListMarker: '-',
  codeBlockStyle: 'fenced'
})

const editor = useEditor({
  content: markdownToHtml(props.modelValue),
  extensions: [
    StarterKit.configure({
      heading: { levels: [2, 3] }
    }),
    Link.configure({
      autolink: true,
      defaultProtocol: 'https',
      openOnClick: false
    }),
    Image.configure({
      allowBase64: false,
      inline: false
    }),
    Placeholder.configure({
      placeholder: '记录进展、分析、结论...'
    })
  ],
  editorProps: {
    attributes: {
      class: 'editor-content'
    },
    handlePaste: (_view, event) => {
      const files = imageFilesFromClipboard(event)
      if (files.length === 0) return false
      event.preventDefault()
      void insertImages(files)
      return true
    }
  },
  onUpdate: ({ editor }) => {
    if (syncing.value) return
    emit('update:modelValue', htmlToMarkdown(editor.getHTML()))
  }
})

const actions = computed(() => {
  const current = editor.value
  if (!current) return []
  return [
    {
      key: 'bold',
      label: '加粗',
      icon: Bold,
      active: () => current.isActive('bold'),
      run: () => current.chain().focus().toggleBold().run()
    },
    {
      key: 'italic',
      label: '斜体',
      icon: Italic,
      active: () => current.isActive('italic'),
      run: () => current.chain().focus().toggleItalic().run()
    },
    {
      key: 'heading',
      label: '标题',
      icon: Heading2,
      active: () => current.isActive('heading', { level: 2 }),
      run: () => current.chain().focus().toggleHeading({ level: 2 }).run()
    },
    {
      key: 'bullet',
      label: '无序列表',
      icon: List,
      active: () => current.isActive('bulletList'),
      run: () => current.chain().focus().toggleBulletList().run()
    },
    {
      key: 'ordered',
      label: '有序列表',
      icon: ListOrdered,
      active: () => current.isActive('orderedList'),
      run: () => current.chain().focus().toggleOrderedList().run()
    },
    {
      key: 'quote',
      label: '引用',
      icon: Quote,
      active: () => current.isActive('blockquote'),
      run: () => current.chain().focus().toggleBlockquote().run()
    },
    {
      key: 'code',
      label: '代码块',
      icon: Code,
      active: () => current.isActive('codeBlock'),
      run: () => current.chain().focus().toggleCodeBlock().run()
    },
    {
      key: 'link',
      label: '链接',
      icon: Link2,
      active: () => current.isActive('link'),
      run: setLink
    },
    {
      key: 'undo',
      label: '撤销',
      icon: Undo2,
      active: () => false,
      disabled: () => !current.can().undo(),
      run: () => current.chain().focus().undo().run()
    },
    {
      key: 'redo',
      label: '重做',
      icon: Redo2,
      active: () => false,
      disabled: () => !current.can().redo(),
      run: () => current.chain().focus().redo().run()
    }
  ]
})

watch(
  () => props.modelValue,
  (value) => {
    const current = editor.value
    if (!current || value === htmlToMarkdown(current.getHTML())) return
    syncing.value = true
    current.commands.setContent(markdownToHtml(value), { emitUpdate: false })
    syncing.value = false
  }
)

onBeforeUnmount(() => {
  editor.value?.destroy()
})

function markdownToHtml(value: string) {
  return md.render(value || '')
}

function htmlToMarkdown(html: string) {
  return turndown
    .turndown(html)
    .replace(/\n{3,}/g, '\n\n')
    .trim()
}

function setLink() {
  const current = editor.value
  if (!current) return
  const previousUrl = current.getAttributes('link').href as string | undefined
  const url = window.prompt('链接地址', previousUrl ?? '')
  if (url === null) return
  if (url.trim() === '') {
    current.chain().focus().extendMarkRange('link').unsetLink().run()
    return
  }
  current.chain().focus().extendMarkRange('link').setLink({ href: url.trim() }).run()
}

function pickImage() {
  fileInput.value?.click()
}

async function uploadImage(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (!file.type.startsWith('image/')) {
    message.error('只能上传图片')
    input.value = ''
    return
  }
  await insertImages([file])
  input.value = ''
}

function imageFilesFromClipboard(event: ClipboardEvent) {
  const clipboard = event.clipboardData
  if (!clipboard) return []

  const transferFiles = Array.from(clipboard.files ?? []).filter((file) => file.type.startsWith('image/'))
  const itemFiles = Array.from(clipboard.items ?? [])
    .filter((item) => item.kind === 'file' && item.type.startsWith('image/'))
    .map((item) => item.getAsFile())
    .filter((file): file is File => Boolean(file))
  const htmlDataImages = dataImageFilesFromHtml(clipboard.getData('text/html'))
  const plainDataImages = dataImageFilesFromText(clipboard.getData('text/plain'))

  return dedupeFiles([...transferFiles, ...itemFiles, ...htmlDataImages, ...plainDataImages])
}

function dataImageFilesFromHtml(html: string) {
  if (!html) return []
  const doc = new DOMParser().parseFromString(html, 'text/html')
  return Array.from(doc.querySelectorAll('img'))
    .map((img) => img.getAttribute('src') ?? '')
    .filter((src) => src.startsWith('data:image/'))
    .map((src, index) => fileFromDataUrl(src, `pasted-image-${index + 1}`))
    .filter((file): file is File => Boolean(file))
}

function dataImageFilesFromText(text: string) {
  if (!text.startsWith('data:image/')) return []
  const file = fileFromDataUrl(text, 'pasted-image')
  return file ? [file] : []
}

function fileFromDataUrl(dataUrl: string, baseName: string) {
  const match = dataUrl.match(/^data:(image\/[a-zA-Z0-9.+-]+);base64,(.+)$/)
  if (!match) return null
  const [, mimeType, base64] = match
  const binary = window.atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }
  return new File([bytes], `${baseName}.${extensionForMimeType(mimeType)}`, { type: mimeType })
}

function extensionForMimeType(mimeType: string) {
  switch (mimeType) {
    case 'image/jpeg':
      return 'jpg'
    case 'image/gif':
      return 'gif'
    case 'image/webp':
      return 'webp'
    default:
      return 'png'
  }
}

function dedupeFiles(files: File[]) {
  const seen = new Set<string>()
  return files.filter((file) => {
    const key = `${file.name}:${file.type}:${file.size}:${file.lastModified}`
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
}

async function insertImages(files: File[]) {
  const imageFiles = files.filter((file) => file.type.startsWith('image/'))
  if (imageFiles.length === 0) return
  uploading.value = true
  let inserted = 0
  try {
    for (const file of imageFiles) {
      const uploaded = await api.uploadImage(file)
      editor.value?.chain().focus().setImage({ src: uploaded.url, alt: file.name || 'pasted image' }).run()
      inserted += 1
    }
    message.success(inserted === 1 ? '图片已插入' : `${inserted} 张图片已插入`)
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    uploading.value = false
  }
}
</script>

<style scoped>
.rich-editor {
  border: 1px solid #d9dee8;
  border-radius: 8px;
  overflow: hidden;
}

.editor-toolbar {
  align-items: center;
  background: #f8fafc;
  border-bottom: 1px solid #e5e7eb;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 6px;
}

.toolbar-divider {
  background: #d9dee8;
  height: 22px;
  margin: 0 4px;
  width: 1px;
}

.file-input {
  display: none;
}

.editor-surface :deep(.editor-content) {
  background: #fff;
  min-height: var(--editor-min-height);
  outline: none;
  padding: 12px;
}

.editor-surface :deep(.editor-content > *:first-child) {
  margin-top: 0;
}

.editor-surface :deep(.editor-content > *:last-child) {
  margin-bottom: 0;
}

.editor-surface :deep(.editor-content p) {
  line-height: 1.7;
  margin: 0 0 10px;
}

.editor-surface :deep(.editor-content h2),
.editor-surface :deep(.editor-content h3) {
  line-height: 1.35;
  margin: 14px 0 8px;
}

.editor-surface :deep(.editor-content blockquote) {
  border-left: 3px solid #94a3b8;
  color: #475569;
  margin: 10px 0;
  padding-left: 10px;
}

.editor-surface :deep(.editor-content pre) {
  background: #111827;
  border-radius: 6px;
  color: #f8fafc;
  overflow-x: auto;
  padding: 10px;
}

.editor-surface :deep(.editor-content img) {
  border-radius: 6px;
  display: block;
  height: auto;
  margin: 10px 0;
  max-width: 100%;
}

.editor-surface :deep(.is-editor-empty:first-child::before) {
  color: #94a3b8;
  content: attr(data-placeholder);
  float: left;
  height: 0;
  pointer-events: none;
}
</style>
