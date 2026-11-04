<template>
  <div ref="root" class="markdown-body" v-html="html" @click="handleClick" />
</template>

<script setup lang="ts">
import MarkdownIt from 'markdown-it'
import { computed, ref, watch, nextTick } from 'vue'
import { api } from '../api/client'
import { isExternalURL, openExternalURL } from '../utils/openExternal'

const props = defineProps<{ content?: string }>()

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true
})

const html = computed(() => md.render(props.content || ''))
const root = ref<HTMLElement>()
let imageGeneration = 0
watch(html, async () => {
  const generation = ++imageGeneration
  await nextTick()
  const images = root.value?.querySelectorAll<HTMLImageElement>('img[src^="/uploads/"]') ?? []
  await Promise.allSettled([...images].map(async image => {
    const url = image.getAttribute('src')!
    const result = await api.getUploadedImageDataURL(url)
    if (generation === imageGeneration && image.isConnected) image.src = result.data_url
  }))
}, { immediate: true, flush: 'post' })

function handleClick(event: MouseEvent) {
  const target = event.target as HTMLElement | null
  const link = target?.closest('a')
  const href = link?.getAttribute('href')
  if (!href || !isExternalURL(href)) return
  event.preventDefault()
  openExternalURL(href)
}
</script>
