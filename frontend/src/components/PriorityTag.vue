<template>
  <n-tag size="small" :bordered="false" :color="color">
    <span class="inline-flex items-center gap-1">
      <component :is="visual.icon" :size="12" :stroke-width="2.4" />
      {{ displayLabel }}
    </span>
  </n-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'
import { priorityVisual, toneColor } from '../utils/jiraDisplay'

const props = defineProps<{
  priority: string
  jiraPriority?: string
  label?: string
}>()

const visual = computed(() => priorityVisual(props.priority, props.jiraPriority ?? ''))
const displayLabel = computed(() => props.label || visual.value.label)
const color = computed(() => toneColor(visual.value.tone))
</script>
