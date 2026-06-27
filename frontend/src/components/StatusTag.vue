<template>
  <n-tag v-if="displayLabel && displayLabel !== '-'" size="small" :bordered="false" :color="color">{{ displayLabel }}</n-tag>
  <span v-else class="text-gray-400">-</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'
import { statusVisual, toneColor } from '../utils/jiraDisplay'

const props = defineProps<{
  status: string
  background?: string
  label?: string
}>()

const visual = computed(() => statusVisual(props.status, props.background ?? ''))
const displayLabel = computed(() => props.label || visual.value.label)
const color = computed(() => toneColor(visual.value.tone))
</script>
