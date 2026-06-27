<template>
  <n-tag v-if="visual" size="small" :bordered="false" :color="color">
    <span class="inline-flex items-center gap-1">
      <component :is="visual.icon" :size="12" :stroke-width="2.2" />
      {{ visual.label }}
    </span>
  </n-tag>
  <span v-else class="text-gray-400">-</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'
import { issueTypeVisual, toneColor } from '../utils/jiraDisplay'

const props = defineProps<{ type: string }>()

const visual = computed(() => issueTypeVisual(props.type))
const color = computed(() => (visual.value ? toneColor(visual.value.tone) : undefined))
</script>
