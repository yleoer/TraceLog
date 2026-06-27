<template>
  <div class="space-y-5">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <h1 class="text-xl font-bold text-gray-900">周视图</h1>
      <div class="flex flex-wrap items-center gap-2">
        <n-button size="small" @click="shiftWeek(-1)">上一周</n-button>
        <n-select
          v-model:value="week"
          :options="weekOptions"
          filterable
          tag
          placeholder="YYYY-Www"
          size="small"
          style="width: 160px"
          @update:value="openWeek"
        />
        <n-button size="small" @click="shiftWeek(1)">下一周</n-button>
        <n-button size="small" @click="openWeek(currentWeek())">本周</n-button>
        <n-button size="small" @click="load">刷新</n-button>
        <n-button size="small" :loading="drafting" @click="generateDraft">生成草稿</n-button>
        <n-button size="small" type="primary" :loading="summarizing" @click="generateSummary">AI 总结</n-button>
        <n-button size="small" @click="downloadUrl(`/export/weeks/${week}.md`)">导出</n-button>
        <n-button size="small" type="primary" @click="save">保存</n-button>
      </div>
    </div>

    <n-spin :show="loading">
      <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        <div class="card">
          <h3 class="card-title">{{ week }} Issues</h3>
          <div v-if="!view?.issues?.length" class="empty">暂无</div>
          <div v-for="issue in view?.issues ?? []" :key="issue.jira_key" class="list-row" @click="$router.push(`/issues/${issue.jira_key}`)">
            <span class="font-medium text-gray-900">{{ issue.jira_key }}</span>
            <span class="text-gray-600 truncate">{{ issue.title }}</span>
            <span class="ml-auto shrink-0"><StatusTag :status="issue.status" :background="issue.background_md" /></span>
          </div>
        </div>

        <div class="card">
          <h3 class="card-title">临时需求</h3>
          <div v-if="!view?.temp_tasks?.length" class="empty">暂无</div>
          <div v-for="task in view?.temp_tasks ?? []" :key="task.id" class="list-row" @click="$router.push(`/temp-tasks/${task.id}`)">
            <span class="text-gray-900">{{ task.title }}</span>
            <span class="ml-auto shrink-0"><StatusTag :status="task.status" :label="tempStatusLabel(task.status)" /></span>
          </div>
        </div>

        <div class="card">
          <h3 class="card-title">时间线事件</h3>
          <div v-if="!view?.events?.length" class="empty">暂无</div>
          <div v-for="event in view?.events ?? []" :key="event.id" class="py-2 border-b border-gray-50 last:border-0">
            <div class="flex items-center gap-2 mb-1">
              <span class="text-xs text-gray-400">{{ formatDate(event.happened_at) }}</span>
              <span class="text-xs text-gray-500 bg-gray-100 px-1.5 py-0.5 rounded">{{ event.event_type }}</span>
            </div>
            <MarkdownView :content="event.content_md" />
          </div>
        </div>

        <div class="card">
          <h3 class="card-title">后续 TODO</h3>
          <div v-if="!view?.todos?.length" class="empty">暂无</div>
          <div v-for="todo in view?.todos ?? []" :key="todo.id" class="list-row" @click="$router.push(`/issues/${todo.jira_key}`)">
            <span :class="todo.done ? 'line-through text-gray-400' : 'text-gray-900'">{{ todo.jira_key }} · {{ todo.content }}</span>
          </div>
        </div>

        <div class="card">
          <h3 class="card-title">完成事项</h3>
          <div v-if="!view?.done?.length" class="empty">暂无</div>
          <div v-for="item in view?.done ?? []" :key="item" class="py-1.5 text-sm text-gray-700 border-b border-gray-50 last:border-0">{{ item }}</div>
        </div>

        <div class="card">
          <h3 class="card-title">进行中事项</h3>
          <div v-if="!view?.active?.length" class="empty">暂无</div>
          <div v-for="item in view?.active ?? []" :key="item" class="py-1.5 text-sm text-gray-700 border-b border-gray-50 last:border-0">{{ item }}</div>
        </div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 mt-4">
        <div class="card">
          <h3 class="card-title">周总结</h3>
          <MarkdownEditor v-model="summary" />
        </div>
        <div class="card">
          <h3 class="card-title">下周计划</h3>
          <MarkdownEditor v-model="nextPlan" />
        </div>
      </div>

      <div class="card mt-4">
        <h3 class="card-title">本周每天</h3>
        <div class="space-y-4">
          <DayWorkPanel v-for="d in view?.days ?? []" :key="d.date" :day="d" @changed="load" />
        </div>
      </div>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { useRoute, useRouter } from 'vue-router'
import { api, downloadUrl } from '../api/client'
import MarkdownEditor from '../components/MarkdownEditor.vue'
import MarkdownView from '../components/MarkdownView.vue'
import StatusTag from '../components/StatusTag.vue'
import DayWorkPanel from '../components/DayWorkPanel.vue'
import { tempStatusLabel } from '../utils/tempTaskDisplay'
import { formatDateTime } from '../utils/datetime'
import type { WeekView } from '../types'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const loading = ref(false)
const drafting = ref(false)
const summarizing = ref(false)
const week = ref(String(route.params.week || currentWeek()))
const weekOptions = ref<{ label: string; value: string }[]>([])
const view = ref<WeekView>()
const summary = ref('')
const nextPlan = ref('')

async function load() {
  const normalized = normalizeWeek(week.value)
  if (!normalized) {
    message.error('周号格式应为 YYYY-Www，例如 2026-W26')
    return
  }
  week.value = normalized
  loading.value = true
  try {
    view.value = await api.getWeek(week.value)
    summary.value = view.value.log.summary_md
    nextPlan.value = view.value.log.next_plan_md
    if (route.params.week !== week.value) router.replace(`/weeks/${week.value}`)
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    loading.value = false
  }
}

async function loadWeekOptions() {
  try {
    const logs = await api.listWeeks()
    const values = new Set([currentWeek(), week.value, ...logs.map((log) => log.week)])
    weekOptions.value = Array.from(values)
      .sort()
      .reverse()
      .map((value) => ({ label: value, value }))
  } catch (error) {
    message.error((error as Error).message)
  }
}

async function save() {
  try {
    await api.updateWeek(week.value, { summary_md: summary.value, next_plan_md: nextPlan.value })
    message.success('周记录已保存')
    await loadWeekOptions()
    await load()
  } catch (error) {
    message.error((error as Error).message)
  }
}

async function generateDraft() {
  drafting.value = true
  try {
    const log = await api.generateWeekDraft(week.value)
    summary.value = log.summary_md
    message.success('周报草稿已生成')
    await loadWeekOptions()
    await load()
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    drafting.value = false
  }
}

async function generateSummary() {
  summarizing.value = true
  try {
    const log = await api.generateWeekSummary(week.value)
    summary.value = log.summary_md
    message.success('周总结已生成')
    await loadWeekOptions()
    await load()
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    summarizing.value = false
  }
}

function openWeek(value: string) {
  const normalized = normalizeWeek(value)
  if (!normalized) {
    message.error('周号格式应为 YYYY-Www，例如 2026-W26')
    return
  }
  week.value = normalized
  router.push(`/weeks/${normalized}`)
}

function shiftWeek(delta: number) {
  openWeek(addWeeks(week.value, delta))
}

function currentWeek() {
  const now = new Date()
  const date = new Date(Date.UTC(now.getFullYear(), now.getMonth(), now.getDate()))
  const day = date.getUTCDay() || 7
  date.setUTCDate(date.getUTCDate() + 4 - day)
  const yearStart = new Date(Date.UTC(date.getUTCFullYear(), 0, 1))
  const weekNo = Math.ceil((((date.getTime() - yearStart.getTime()) / 86400000) + 1) / 7)
  return `${date.getUTCFullYear()}-W${String(weekNo).padStart(2, '0')}`
}

function normalizeWeek(value: string) {
  const match = String(value || '').trim().toUpperCase().match(/^(\d{4})-W(\d{1,2})$/)
  if (!match) return ''
  const weekNumber = Number(match[2])
  if (weekNumber < 1 || weekNumber > 53) return ''
  return `${match[1]}-W${String(weekNumber).padStart(2, '0')}`
}

function addWeeks(value: string, delta: number) {
  const normalized = normalizeWeek(value) || currentWeek()
  const [yearText, weekText] = normalized.split('-W')
  const date = isoWeekToDate(Number(yearText), Number(weekText))
  date.setUTCDate(date.getUTCDate() + delta * 7)
  return weekFromDate(date)
}

function isoWeekToDate(year: number, weekNumber: number) {
  const fourthOfJanuary = new Date(Date.UTC(year, 0, 4))
  const day = fourthOfJanuary.getUTCDay() || 7
  const monday = new Date(fourthOfJanuary)
  monday.setUTCDate(fourthOfJanuary.getUTCDate() - day + 1 + (weekNumber - 1) * 7)
  return monday
}

function weekFromDate(input: Date) {
  const date = new Date(Date.UTC(input.getUTCFullYear(), input.getUTCMonth(), input.getUTCDate()))
  const day = date.getUTCDay() || 7
  date.setUTCDate(date.getUTCDate() + 4 - day)
  const yearStart = new Date(Date.UTC(date.getUTCFullYear(), 0, 1))
  const weekNo = Math.ceil((((date.getTime() - yearStart.getTime()) / 86400000) + 1) / 7)
  return `${date.getUTCFullYear()}-W${String(weekNo).padStart(2, '0')}`
}

function formatDate(value: string) {
  return formatDateTime(value)
}

watch(() => route.params.week, (value) => {
  if (value) {
    week.value = String(value)
    load()
  }
})
onMounted(async () => {
  await loadWeekOptions()
  await load()
})
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
  color: #374151;
  margin: 0 0 12px;
}

.list-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  margin: 0 -10px;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.15s;
}

.list-row:hover {
  background: #f3f4f6;
}

.empty {
  color: #9ca3af;
  font-size: 13px;
  padding: 8px 0;
}
</style>
