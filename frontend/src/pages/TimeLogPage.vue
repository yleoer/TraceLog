<template>
  <div class="time-log-page space-y-5 max-w-6xl mx-auto">
    <div class="page-header">
      <div>
        <div class="page-kicker">Tempo worklog</div>
        <h1>Time</h1>
        <p class="page-subtitle">{{ weekTitle }}</p>
      </div>
      <div class="page-toolbar">
        <n-button size="small" @click="shiftWeek(-1)">上周</n-button>
        <n-button size="small" @click="goCurrentWeek">本周</n-button>
        <n-button size="small" @click="shiftWeek(1)">下周</n-button>
        <n-button size="small" :loading="loadingWeek" @click="loadWeek(true)">刷新</n-button>
        <n-button type="primary" size="small" :loading="submitting" :disabled="!canSubmit || confirmOpen" @click="submitTime">提交 Time</n-button>
      </div>
    </div>

    <div class="grid grid-cols-1 xl:grid-cols-[minmax(0,1fr)_360px] gap-4">
      <section class="space-y-4">
        <div class="card">
          <n-form label-placement="top">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <n-form-item label="Work Item">
                <n-select
                  v-model:value="form.work_item_key"
                  size="small"
                  :options="workItemOptions"
                  :loading="loadingWeek"
                  placeholder="选择 Work Item"
                />
              </n-form-item>

              <n-form-item label="Logged">
                <n-input-number
                  v-model:value="form.hours"
                  size="small"
                  class="w-full"
                  :min="1"
                  :max="8"
                  :step="1"
                  button-placement="both"
                >
                  <template #suffix>小时</template>
                </n-input-number>
              </n-form-item>

              <n-form-item label="开始日期">
                <n-date-picker
                  v-model:formatted-value="form.start_date"
                  type="date"
                  value-format="yyyy-MM-dd"
                  size="small"
                  class="w-full"
                  :clearable="false"
                  :first-day-of-week="0"
                  :to="false"
                />
              </n-form-item>

              <n-form-item label="结束日期">
                <n-date-picker
                  v-model:formatted-value="form.end_date"
                  type="date"
                  value-format="yyyy-MM-dd"
                  size="small"
                  class="w-full"
                  :clearable="false"
                  :first-day-of-week="0"
                  :to="false"
                />
              </n-form-item>
            </div>

            <n-form-item label="Description">
              <n-input
                v-model:value="form.description"
                type="textarea"
                :autosize="{ minRows: 4, maxRows: 8 }"
                placeholder="Worklog description"
              />
            </n-form-item>
          </n-form>
        </div>

        <n-spin :show="loadingWeek">
          <div class="week-grid">
            <article v-for="day in weekData?.days ?? []" :key="day.date" class="day-card" :class="{ weekend: isWeekend(day.date) }">
              <div class="day-head">
                <div>
                  <h2>{{ day.weekday }}</h2>
                  <p>{{ day.date }}</p>
                </div>
                <strong>{{ formatHours(day.total_hours) }}</strong>
              </div>
              <ul class="worklog-list">
                <li v-for="worklog in day.worklogs" :key="worklog.tempo_worklog_id">
                  <div class="worklog-main">
                    <span class="time-range">{{ trimSeconds(worklog.start_time) }} - {{ trimSeconds(worklog.end_time) }}</span>
                    <strong>{{ formatHours(worklog.hours) }}</strong>
                  </div>
                  <p>{{ worklog.work_item_key }} {{ worklog.work_item_label }}</p>
                  <p v-if="worklog.description" class="description">{{ worklog.description }}</p>
                </li>
                <li v-if="day.worklogs.length === 0" class="empty">暂无 Time</li>
              </ul>
            </article>
          </div>
        </n-spin>
      </section>

      <aside class="space-y-4">
        <section class="card">
          <h2 class="card-title">提交预览</h2>
          <div class="summary-row">
            <span>Work Item</span>
            <strong>{{ selectedWorkItemLabel }}</strong>
          </div>
          <div class="summary-row">
            <span>Logged</span>
            <strong>{{ form.hours }}h / 天</strong>
          </div>
          <div class="summary-row">
            <span>周合计</span>
            <strong>{{ formatHours(weekData?.total_hours ?? 0) }}</strong>
          </div>
          <ul class="preview-list">
            <li v-for="item in previewItems" :key="item.date">
              <span>{{ item.date }}</span>
              <strong>{{ trimSeconds(item.startTime) }} - {{ trimSeconds(item.endTime) }}</strong>
            </li>
          </ul>
        </section>

        <section v-if="result" class="card">
          <h2 class="card-title">提交结果</h2>
          <div class="result-grid">
            <div>
              <span>成功</span>
              <strong class="text-green-700">{{ result.successful }}</strong>
            </div>
            <div>
              <span>失败</span>
              <strong :class="result.failed > 0 ? 'text-red-600' : 'text-gray-900'">{{ result.failed }}</strong>
            </div>
          </div>
          <ul class="result-list">
            <li v-for="entry in result.entries" :key="entry.date" :class="entry.error ? 'failed' : 'success'">
              <span>{{ entry.date }} {{ trimSeconds(entry.start_time) }} - {{ trimSeconds(entry.end_time) }}</span>
              <span>{{ entry.error || `#${entry.tempo_worklog_id}` }}</span>
            </li>
          </ul>
        </section>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { NButton, NDatePicker, NForm, NFormItem, NInput, NInputNumber, NSelect, NSpin, useDialog, useMessage } from 'naive-ui'
import { api } from '../api/client'
import type { LogTimeResult, TimeWeekView, TimeWorkItem } from '../types'

const message = useMessage()
const dialog = useDialog()
const loadingWeek = ref(false)
const submitting = ref(false)
const confirmOpen = ref(false)
const weekData = ref<TimeWeekView | null>(null)
const result = ref<LogTimeResult | null>(null)
const currentWeek = ref(currentISOWeek(new Date()))

const form = reactive({
  work_item_key: 'CORETIME-80',
  description: '',
  hours: 8,
  start_date: todayString(),
  end_date: todayString()
})

const workItems = computed<TimeWorkItem[]>(() => weekData.value?.work_items ?? [])
const workItemOptions = computed(() =>
  workItems.value.map((item) => ({
    label: `${item.key} ${item.label}`,
    value: item.key
  }))
)

const selectedWorkItemLabel = computed(() => {
  const item = workItems.value.find((value) => value.key === form.work_item_key)
  return item ? `${item.key} ${item.label}` : form.work_item_key || '-'
})

const weekTitle = computed(() => {
  if (!weekData.value) return currentWeek.value
  return `${weekData.value.week} · ${weekData.value.start_date} 至 ${weekData.value.end_date}`
})

const previewItems = computed(() => {
  const dates = buildDates(form.start_date, form.end_date)
  return dates.map((date) => {
    const loggedSeconds = weekData.value?.days.find((day) => day.date === date)?.worklogs.reduce((sum, item) => sum + item.time_spent_seconds, 0) ?? 0
    const startTime = addSeconds('08:00:00', loggedSeconds)
    return {
      date,
      startTime,
      endTime: addSeconds(startTime, form.hours * 3600)
    }
  })
})

const canSubmit = computed(() =>
  Boolean(form.work_item_key && form.description.trim() && form.hours >= 1 && form.hours <= 8 && previewItems.value.length > 0)
)

async function loadWeek(forceRefresh = false) {
  loadingWeek.value = true
  try {
    weekData.value = forceRefresh ? await api.refreshTimeWeek(currentWeek.value) : await api.getTimeWeek(currentWeek.value)
    if (!workItems.value.some((item) => item.key === form.work_item_key) && workItems.value[0]) {
      form.work_item_key = workItems.value[0].key
    }
    if (isDateInRange(todayString(), weekData.value.start_date, weekData.value.end_date)) {
      form.start_date = todayString()
      form.end_date = todayString()
    } else {
      form.start_date = weekData.value.start_date
      form.end_date = weekData.value.start_date
    }
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    loadingWeek.value = false
  }
}

function shiftWeek(delta: number) {
  currentWeek.value = shiftISOWeek(currentWeek.value, delta)
  result.value = null
  loadWeek()
}

function goCurrentWeek() {
  currentWeek.value = currentISOWeek(new Date())
  result.value = null
  loadWeek()
}

function submitTime() {
  if (submitting.value || confirmOpen.value) return
  if (!canSubmit.value) {
    message.warning('请填写 Work Item、Description、Logged 和日期')
    return
  }
  confirmOpen.value = true
  const confirmDialog = dialog.warning({
    title: '确认提交 Time',
    content: `将向 Tempo 提交 ${previewItems.value.length} 天，每天 ${form.hours} 小时，Work Item 为 ${selectedWorkItemLabel.value}。`,
    positiveText: '提交',
    negativeText: '取消',
    positiveButtonProps: {
      loading: false
    },
    onClose: () => {
      confirmOpen.value = false
    },
    onNegativeClick: () => {
      confirmOpen.value = false
    },
    onPositiveClick: () => {
      if (submitting.value) return false
      confirmDialog.loading = true
      confirmDialog.positiveButtonProps = { loading: true, disabled: true }
      confirmDialog.negativeButtonProps = { disabled: true }
      return doSubmitTime().finally(() => {
        confirmOpen.value = false
      })
    }
  })
}

async function doSubmitTime() {
  if (submitting.value) return
  submitting.value = true
  result.value = null
  try {
    const response = await api.logTempoTime({
      work_item_key: form.work_item_key,
      description: form.description,
      hours: form.hours,
      start_date: form.start_date,
      end_date: form.end_date
    })
    result.value = response
    await loadWeek()
    if (response.failed > 0 && response.successful > 0) {
      message.warning(`已成功 ${response.successful} 天，失败 ${response.failed} 天`)
      return
    }
    if (response.failed > 0) {
      message.error(`提交失败 ${response.failed} 天`)
      return
    }
    message.success(`已提交 ${response.successful} 天 Time`)
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    submitting.value = false
  }
}

function todayString() {
  return formatDate(new Date())
}

function buildDates(startDate: string, endDate: string) {
  if (!datePattern.test(startDate) || !datePattern.test(endDate)) return []
  const start = parseDate(startDate)
  const end = parseDate(endDate)
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime()) || end < start) return []
  const dates: string[] = []
  for (let cursor = new Date(start); cursor <= end; cursor.setDate(cursor.getDate() + 1)) {
    dates.push(formatDate(cursor))
    if (dates.length > 31) break
  }
  return dates
}

function parseDate(value: string) {
  const [year, month, day] = value.split('-').map(Number)
  return new Date(year, month - 1, day)
}

function formatDate(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function currentISOWeek(date: Date) {
  const target = new Date(date)
  target.setHours(0, 0, 0, 0)
  target.setDate(target.getDate() + 3 - ((target.getDay() + 6) % 7))
  const week1 = new Date(target.getFullYear(), 0, 4)
  const weekNumber = 1 + Math.round(((target.getTime() - week1.getTime()) / 86400000 - 3 + ((week1.getDay() + 6) % 7)) / 7)
  return `${target.getFullYear()}-W${String(weekNumber).padStart(2, '0')}`
}

function shiftISOWeek(week: string, delta: number) {
  const [yearPart, weekPart] = week.split('-W')
  const year = Number(yearPart)
  const weekNumber = Number(weekPart)
  const jan4 = new Date(year, 0, 4)
  const monday = new Date(jan4)
  monday.setDate(jan4.getDate() - ((jan4.getDay() + 6) % 7) + (weekNumber - 1) * 7 + delta * 7)
  return currentISOWeek(monday)
}

function isDateInRange(date: string, start: string, end: string) {
  return date >= start && date <= end
}

function isWeekend(date: string) {
  const parsed = parseDate(date)
  const day = parsed.getDay()
  return day === 0 || day === 6
}

function addSeconds(time: string, seconds: number) {
  const [hour, minute, second] = time.split(':').map(Number)
  const date = new Date(2000, 0, 1, hour || 0, minute || 0, second || 0)
  date.setSeconds(date.getSeconds() + seconds)
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`
}

function trimSeconds(time: string) {
  return time?.slice(0, 5) || ''
}

function formatHours(value: number) {
  if (!Number.isFinite(value) || value === 0) return '0h'
  if (Number.isInteger(value)) return `${value}h`
  return `${value.toFixed(1)}h`
}

const datePattern = /^\d{4}-\d{2}-\d{2}$/

onMounted(loadWeek)
</script>

<style scoped>
.card,
.day-card {
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 18px 20px;
}

.card-title {
  font-size: 13px;
  font-weight: 600;
  color: #6b7280;
  text-transform: uppercase;
  letter-spacing: 0.02em;
  margin: 0 0 12px;
}

.week-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(230px, 1fr));
  gap: 12px;
}

.day-card {
  padding: 14px;
}

.day-card.weekend {
  border-color: rgba(var(--accent-rgb), 0.18);
  box-shadow: inset 3px 0 0 rgba(var(--accent-rgb), 0.2);
}

.day-card.weekend .day-head h2,
.day-card.weekend .day-head strong {
  color: var(--accent);
}

.day-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  border-bottom: 1px solid #f3f4f6;
  padding-bottom: 10px;
}

.day-head h2 {
  color: #111827;
  font-size: 14px;
  font-weight: 700;
  margin: 0;
}

.day-head p {
  color: #6b7280;
  font-size: 12px;
  margin: 2px 0 0;
}

.day-head strong {
  color: var(--accent);
  font-size: 14px;
}

.worklog-list,
.preview-list,
.result-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.worklog-list li {
  border-bottom: 1px solid #f3f4f6;
  padding: 10px 0;
}

.worklog-list li:last-child {
  border-bottom: 0;
}

.worklog-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.worklog-main strong {
  color: #111827;
}

.time-range {
  color: var(--accent);
  font-size: 12px;
  font-weight: 700;
}

.worklog-list p {
  color: #4b5563;
  font-size: 12px;
  margin: 5px 0 0;
  overflow-wrap: anywhere;
}

.worklog-list .description {
  color: #6b7280;
}

.worklog-list .empty {
  color: #9ca3af;
  font-size: 12px;
}

.summary-row,
.preview-list li,
.result-list li {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid #f3f4f6;
  color: #6b7280;
  font-size: 13px;
  padding: 8px 0;
}

.summary-row strong,
.preview-list strong {
  color: #111827;
  font-weight: 600;
  text-align: right;
}

.result-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.result-grid div {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 10px;
}

.result-grid span {
  display: block;
  color: #6b7280;
  font-size: 12px;
}

.result-grid strong {
  display: block;
  font-size: 20px;
  margin-top: 2px;
}

.result-list {
  margin-top: 12px;
}

.result-list li {
  font-size: 12px;
}

.result-list li:last-child,
.preview-list li:last-child {
  border-bottom: 0;
}

.result-list .success span:last-child {
  color: #15803d;
  font-weight: 600;
}

.result-list .failed span:last-child {
  color: #dc2626;
  overflow-wrap: anywhere;
  text-align: right;
}

.time-log-page :deep(.n-date-panel-calendar .n-date-panel-weekdays__day:nth-child(6)),
.time-log-page :deep(.n-date-panel-calendar .n-date-panel-weekdays__day:nth-child(7)) {
  color: #94a3b8;
}

.time-log-page :deep(.n-date-panel-calendar .n-date-panel-dates .n-date-panel-date:nth-child(7n + 6):not(.n-date-panel-date--selected)),
.time-log-page :deep(.n-date-panel-calendar .n-date-panel-dates .n-date-panel-date:nth-child(7n):not(.n-date-panel-date--selected)) {
  color: #64748b;
}

.time-log-page :deep(.n-date-panel-month__fast-prev),
.time-log-page :deep(.n-date-panel-month__fast-next) {
  pointer-events: none;
  visibility: hidden;
}
</style>
