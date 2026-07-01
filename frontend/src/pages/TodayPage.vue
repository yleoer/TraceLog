<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-semibold text-gray-900">今日工作流</h1>
        <p class="text-sm text-gray-500 mt-0.5">{{ data?.date ?? '' }}</p>
      </div>
      <n-button size="small" @click="load">刷新</n-button>
    </div>

    <n-spin :show="loading">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div class="card">
          <h2 class="card-title">今日更新 Issues</h2>
          <ul class="card-list">
            <li v-for="issue in data?.issues ?? []" :key="issue.jira_key" class="card-list-item" @click="$router.push(`/issues/${issue.jira_key}`)">
              <span class="font-medium text-gray-900">{{ issue.jira_key }}</span>
              <span class="text-gray-500 truncate">{{ issue.summary_md || issue.title }}</span>
            </li>
            <li v-if="(data?.issues ?? []).length === 0" class="text-gray-400 text-sm py-3">暂无数据</li>
          </ul>
        </div>

        <div class="card">
          <h2 class="card-title">今日记录</h2>
          <DayWorkPanel v-if="data?.day" :day="data.day" @changed="load" />
        </div>

        <div class="card">
          <h2 class="card-title">临时需求</h2>
          <ul class="card-list">
            <li v-for="task in data?.temp_tasks ?? []" :key="task.id" class="card-list-item" @click="$router.push(`/temp-tasks/${task.id}`)">
              <span class="font-medium text-gray-900 truncate">{{ task.title }}</span>
              <StatusTag :status="task.status" :label="tempStatusLabel(task.status)" />
            </li>
            <li v-if="(data?.temp_tasks ?? []).length === 0" class="text-gray-400 text-sm py-3">暂无数据</li>
          </ul>
        </div>

        <div class="card">
          <h2 class="card-title">待跟进 TODO</h2>
          <ul class="card-list">
            <li v-for="todo in data?.todos ?? []" :key="todo.id" class="card-list-item" @click="$router.push(`/issues/${todo.jira_key}`)">
              <span :class="todo.done ? 'line-through text-gray-400' : 'text-gray-900'">{{ todo.jira_key }} · {{ todo.content }}</span>
            </li>
            <li v-if="(data?.todos ?? []).length === 0" class="text-gray-400 text-sm py-3">暂无数据</li>
          </ul>
        </div>
      </div>

      <div class="card mt-4">
        <h2 class="card-title">周报草稿片段</h2>
        <MarkdownEditor v-model="draft" :upload-context="`today-${data?.date || 'draft'}-weekly-draft`" placeholder="编辑今日生成的周报草稿片段..." />
      </div>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent, onMounted, ref } from 'vue'
import { NButton, NSpin, useMessage } from 'naive-ui'
import { api } from '../api/client'
import StatusTag from '../components/StatusTag.vue'
import DayWorkPanel from '../components/DayWorkPanel.vue'
import { tempStatusLabel } from '../utils/tempTaskDisplay'
import type { TodayWorkflow } from '../types'

const MarkdownEditor = defineAsyncComponent(() => import('../components/MarkdownEditor.vue'))
const message = useMessage()
const loading = ref(false)
const data = ref<TodayWorkflow>()
const draft = ref('')

async function load() {
  loading.value = true
  try {
    data.value = await api.today()
    draft.value = data.value.weekly_draft
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
}

.card-list-item:last-child {
  border-bottom: none;
}
</style>
