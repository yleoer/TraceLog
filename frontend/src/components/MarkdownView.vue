<template>
  <div class="markdown-body" v-html="html" @click="handleClick" />
</template>

<script setup lang="ts">
import MarkdownIt from 'markdown-it'
import { computed } from 'vue'
import { isExternalURL, openExternalURL } from '../utils/openExternal'

const props = defineProps<{ content?: string }>()

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true
})

const html = computed(() => md.render(props.content || ''))

function handleClick(event: MouseEvent) {
  const target = event.target as HTMLElement | null
  const link = target?.closest('a')
  const href = link?.getAttribute('href')
  if (!href || !isExternalURL(href)) return
  event.preventDefault()
  openExternalURL(href)
}
</script>
