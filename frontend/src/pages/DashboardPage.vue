<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-semibold text-gray-900">Dashboard</h1>
      <n-button type="primary" size="small" @click="$router.push('/issues/new')">New Issue</n-button>
    </div>

    <n-spin :show="loading">
      <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        <div class="card">
          <h2 class="card-title">最近更新 Issues</h2>
          <ul class="card-list">
            <li v-for="issue in data?.recent_issues ?? []" :key="issue.jira_key" class="card-list-item" @click="$router.push(`/issues/${issue.jira_key}`)">
              <span class="font-medium text-gray-900">{{ issue.jira_key }}</span>
              <span class="text-gray-500 truncate">{{ issue.title }}</span>
            </li>
            <li v-if="(data?.recent_issues ?? []).length === 0" class="text-gray-400 text-sm py-3">暂无数据</li>
          </ul>
        </div>

        <div class="card">
          <h2 class="card-title">进行中 Issues</h2>
          <ul class="card-list">
            <li v-for="issue in data?.active_issues ?? []" :key="issue.jira_key" class="card-list-item" @click="$router.push(`/issues/${issue.jira_key}`)">
              <span class="font-medium text-gray-900">{{ issue.jira_key }}</span>
              <StatusTag :status="issue.status" :background="issue.background_md" />
            </li>
            <li v-if="(data?.active_issues ?? []).length === 0" class="text-gray-400 text-sm py-3">暂无数据</li>
          </ul>
        </div>

        <div class="card">
          <h2 class="card-title">临时需求</h2>
          <ul class="card-list">
            <li v-for="task in data?.temp_tasks ?? []" :key="task.id" class="card-list-item" @click="$router.push(`/temp-tasks/${task.id}`)">
              <span class="font-medium text-gray-900 truncate">{{ task.title }}</span>
              <span v-if="task.source" class="text-gray-500 shrink-0">{{ task.source }}</span>
              <StatusTag v-else :status="task.status" :label="tempStatusLabel(task.status)" />
            </li>
            <li v-if="(data?.temp_tasks ?? []).length === 0" class="text-gray-400 text-sm py-3">暂无数据</li>
          </ul>
        </div>

        <div class="card">
          <h2 class="card-title">待跟进 TODO</h2>
          <ul class="card-list">
            <li v-for="todo in data?.todos ?? []" :key="todo.id" class="card-list-item" @click="$router.push(`/issues/${todo.jira_key}`)">
              <span class="font-medium text-gray-900">{{ todo.jira_key }}</span>
              <span class="text-gray-500 truncate">{{ todo.content }}</span>
            </li>
            <li v-if="(data?.todos ?? []).length === 0" class="text-gray-400 text-sm py-3">暂无数据</li>
          </ul>
        </div>

        <div class="card">
          <h2 class="card-title">本周 {{ data?.week.log.week ?? '' }}</h2>
          <div class="flex gap-6 py-3">
            <div class="text-center">
              <div class="text-2xl font-semibold text-gray-900">{{ data?.week.issues.length ?? 0 }}</div>
              <div class="text-xs text-gray-500 mt-1">Issues</div>
            </div>
            <div class="text-center">
              <div class="text-2xl font-semibold text-gray-900">{{ data?.week.temp_tasks.length ?? 0 }}</div>
              <div class="text-xs text-gray-500 mt-1">临时需求</div>
            </div>
          </div>
          <button class="text-sm text-blue-600 hover:text-blue-700 font-medium" @click="$router.push(`/weeks/${data?.week.log.week}`)">
            查看周视图 &rarr;
          </button>
        </div>
      </div>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { NButton, NSpin, useMessage } from 'naive-ui'
import { api } from '../api/client'
import StatusTag from '../components/StatusTag.vue'
import { tempStatusLabel } from '../utils/tempTaskDisplay'
import type { Dashboard } from '../types'

const message = useMessage()
const loading = ref(false)
const data = ref<Dashboard>()

async function load() {
  loading.value = true
  try {
    data.value = await api.dashboard()
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.card {
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  padding: 16px 18px;
}

.card-title {
  font-size: 13px;
  font-weight: 600;
  color: #6b7280;
  text-transform: uppercase;
  letter-spacing: 0.02em;
  margin: 0 0 8px;
}

.card-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.card-list-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
  border-bottom: 1px solid #f3f4f6;
  cursor: pointer;
  font-size: 13px;
  transition: background 0.1s;
}

.card-list-item:last-child {
  border-bottom: none;
}

.card-list-item:hover {
  background: #f9fafb;
  margin: 0 -18px;
  padding-left: 18px;
  padding-right: 18px;
}
</style>
