<template>
  <div class="space-y-5">
    <h1 class="text-xl font-semibold text-gray-900">全局搜索</h1>

    <div class="flex items-center gap-2">
      <n-input v-model:value="query" placeholder="Search issues, events, temp tasks, weekly logs" clearable size="small" class="!w-80" @keyup.enter="load" />
      <n-button type="primary" size="small" @click="load">Search</n-button>
    </div>

    <div class="space-y-3">
      <div
        v-for="result in results"
        :key="`${result.type}-${result.id}`"
        class="card cursor-pointer hover:border-blue-200 transition-colors"
        @click="$router.push(result.url)"
      >
        <div class="flex items-center justify-between mb-1">
          <h3 class="text-sm font-semibold text-gray-900">{{ result.title }}</h3>
          <span class="text-xs text-gray-400">{{ result.type }} · {{ result.updated_at }}</span>
        </div>
        <div class="text-sm text-gray-600 markdown-body" v-html="result.snippet" />
      </div>
      <div v-if="results.length === 0 && query" class="text-center text-gray-400 py-12 text-sm">
        没有找到匹配结果
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useMessage } from 'naive-ui'
import { api } from '../api/client'
import type { SearchResult } from '../types'

const message = useMessage()
const query = ref('')
const results = ref<SearchResult[]>([])

async function load() {
  try {
    results.value = await api.search(query.value)
  } catch (error) {
    message.error((error as Error).message)
  }
}
</script>

<style scoped>
.card {
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  padding: 14px 18px;
}
</style>
