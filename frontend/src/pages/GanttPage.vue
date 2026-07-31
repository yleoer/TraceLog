<template>
  <div class="gantt-page space-y-5">
    <div class="page-header">
      <div>
        <div class="page-kicker">Monthly schedule</div>
        <h1>甘特图</h1>
        <p class="page-subtitle">按月查看 Issues 和临时需求的工作周期。</p>
      </div>
      <div class="page-toolbar">
        <n-tooltip>
          <template #trigger>
            <n-button quaternary circle size="small" aria-label="上个月" @click="shiftMonth(-1)">
              <template #icon><ChevronLeft :size="16" /></template>
            </n-button>
          </template>
          上个月
        </n-tooltip>
        <n-date-picker
          :formatted-value="month"
          type="month"
          value-format="yyyy-MM"
          :clearable="false"
          size="small"
          style="width: 142px"
          @update:formatted-value="selectMonth"
        />
        <n-tooltip>
          <template #trigger>
            <n-button quaternary circle size="small" aria-label="下个月" @click="shiftMonth(1)">
              <template #icon><ChevronRight :size="16" /></template>
            </n-button>
          </template>
          下个月
        </n-tooltip>
        <n-button size="small" @click="goCurrentMonth">本月</n-button>
        <n-tooltip>
          <template #trigger>
            <n-button quaternary circle size="small" aria-label="刷新甘特图" @click="load">
              <template #icon><RefreshCw :size="15" /></template>
            </n-button>
          </template>
          刷新
        </n-tooltip>
      </div>
    </div>

    <n-spin :show="loading">
      <section class="card gantt-card">
        <div class="gantt-card-head">
          <div>
            <h2 class="card-title">{{ monthLabel }} 工作周期</h2>
            <p class="gantt-caption">每条横线代表一个开始至完成的工作周期。</p>
          </div>
          <div class="gantt-legend" aria-label="图例">
            <span><i class="legend-swatch issue" />Issue</span>
            <span><i class="legend-swatch temp-task" />临时需求</span>
            <span><i class="legend-open" />进行中</span>
          </div>
        </div>

        <div v-if="!view?.items?.length" class="empty">本月暂无有开始时间的工作项</div>
        <div v-else class="gantt-scroll app-scrollbar-hidden">
          <div class="gantt-grid gantt-header" :style="gridStyle">
            <div class="gantt-item-heading">工作项</div>
            <div v-for="day in days" :key="day" class="gantt-date" :class="{ weekend: isWeekend(day) }">
              <strong>{{ day }}</strong>
              <span>{{ weekdayLabel(day) }}</span>
            </div>
          </div>

          <router-link
            v-for="item in view.items"
            :key="`${item.source}-${item.id}`"
            :to="itemPath(item)"
            class="gantt-grid gantt-row"
            :style="rowStyle(item)"
            :aria-label="itemAriaLabel(item)"
          >
            <div class="gantt-item-info">
              <div class="gantt-item-title">
                <span class="gantt-item-key">{{ item.source === 'issue' ? item.jira_key : '临时需求' }}</span>
                <span class="truncate">{{ item.title }}</span>
              </div>
              <StatusTag :status="item.status" :label="statusLabel(item)" />
            </div>
            <div class="gantt-track" :style="trackStyle(item)">
              <n-tooltip v-for="(period, index) in item.periods" :key="`${period.started_at}-${period.completed_at}-${index}`">
                <template #trigger>
                  <span class="gantt-bar" :class="[item.source, { open: !period.completed_at }]" :style="periodStyle(period, index)" />
                </template>
                {{ formatPeriod(period) }}
              </n-tooltip>
            </div>
          </router-link>
        </div>
      </section>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NButton, NDatePicker, NSpin, NTooltip, useMessage } from 'naive-ui'
import { ChevronLeft, ChevronRight, RefreshCw } from 'lucide-vue-next'
import { api } from '../api/client'
import StatusTag from '../components/StatusTag.vue'
import { APP_TIMEZONE, formatDate } from '../utils/datetime'
import { tempStatusLabel } from '../utils/tempTaskDisplay'
import type { MonthGanttItem, MonthGanttPeriod, MonthGanttView } from '../types'

const message = useMessage()
const loading = ref(false)
const monthPartsFormatter = new Intl.DateTimeFormat('en-US', {
  timeZone: APP_TIMEZONE,
  year: 'numeric',
  month: '2-digit',
  day: '2-digit'
})
const month = ref(currentMonth())
const view = ref<MonthGanttView>()

const days = computed(() => {
  const [year, monthNumber] = month.value.split('-').map(Number)
  const count = new Date(year, monthNumber, 0).getDate()
  return Array.from({ length: count }, (_, index) => index + 1)
})
const monthLabel = computed(() => {
  const [year, monthNumber] = month.value.split('-').map(Number)
  return new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'long', timeZone: APP_TIMEZONE }).format(new Date(year, monthNumber - 1, 1))
})
const gridStyle = computed(() => ({
  '--day-count': String(days.value.length),
  minWidth: `${260 + days.value.length * 30}px`
}))

async function load() {
  loading.value = true
  try {
    view.value = await api.getMonthGantt(month.value)
    month.value = view.value.month
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    loading.value = false
  }
}

function selectMonth(value: string | null) {
  if (!value || !/^\d{4}-\d{2}$/.test(value)) return
  month.value = value
  load()
}

function shiftMonth(offset: number) {
  const [year, monthNumber] = month.value.split('-').map(Number)
  const target = new Date(year, monthNumber - 1 + offset, 1)
  month.value = `${target.getFullYear()}-${String(target.getMonth() + 1).padStart(2, '0')}`
  load()
}

function goCurrentMonth() {
  month.value = currentMonth()
  load()
}

function currentMonth() {
  const parts = zonedParts(new Date().toISOString())
  return parts ? `${parts.year}-${String(parts.month).padStart(2, '0')}` : new Date().toISOString().slice(0, 7)
}

function itemPath(item: MonthGanttItem) {
  return item.source === 'issue' ? `/issues/${item.jira_key}` : `/temp-tasks/${item.id}`
}

function statusLabel(item: MonthGanttItem) {
  return item.source === 'temp_task' ? tempStatusLabel(item.status) : item.status
}

function rowStyle(item: MonthGanttItem) {
  return {
    ...gridStyle.value,
    minHeight: `${Math.max(48, item.periods.length * 20 + 16)}px`
  }
}

function trackStyle(item: MonthGanttItem) {
  return {
    gridColumn: `2 / span ${days.value.length}`,
    minHeight: `${Math.max(48, item.periods.length * 20 + 16)}px`
  }
}

function periodStyle(period: MonthGanttPeriod, index: number) {
  const total = days.value.length
  const startedAt = Math.max(0, Math.min(total - 1, dayOffset(period.started_at)))
  const completedAt = period.completed_at ? dayOffset(period.completed_at) + 1 : openPeriodEnd()
  const endedAt = Math.max(startedAt + 1, Math.min(total, completedAt))
  return {
    left: `${(startedAt / total) * 100}%`,
    width: `${((endedAt - startedAt) / total) * 100}%`,
    top: `${8 + index * 20}px`
  }
}

function openPeriodEnd() {
  if (month.value !== currentMonth()) return days.value.length
  return Math.max(1, Math.min(days.value.length, dayOffset(new Date().toISOString()) + 1))
}

function dayOffset(value: string) {
  const date = zonedParts(value)
  if (!date) return 0
  const [year, monthNumber] = month.value.split('-').map(Number)
  return Math.floor((Date.UTC(date.year, date.month - 1, date.day) - Date.UTC(year, monthNumber - 1, 1)) / 86400000)
}

function zonedParts(value: string) {
  if (!value || Number.isNaN(Date.parse(value))) return null
  const parts = monthPartsFormatter.formatToParts(new Date(value))
  const valueOf = (type: Intl.DateTimeFormatPartTypes) => Number(parts.find((part) => part.type === type)?.value)
  const year = valueOf('year')
  const monthNumber = valueOf('month')
  const day = valueOf('day')
  return Number.isFinite(year) && Number.isFinite(monthNumber) && Number.isFinite(day) ? { year, month: monthNumber, day } : null
}

function weekdayLabel(day: number) {
  const [year, monthNumber] = month.value.split('-').map(Number)
  return ['日', '一', '二', '三', '四', '五', '六'][new Date(year, monthNumber - 1, day).getDay()]
}

function isWeekend(day: number) {
  const weekday = weekdayLabel(day)
  return weekday === '日' || weekday === '六'
}

function formatPeriod(period: MonthGanttPeriod) {
  return `${formatDate(period.started_at)} 至 ${period.completed_at ? formatDate(period.completed_at) : '进行中'}`
}

function itemAriaLabel(item: MonthGanttItem) {
  const reference = item.source === 'issue' ? item.jira_key : '临时需求'
  return `${reference} ${item.title}，${item.periods.map(formatPeriod).join('；')}`
}

onMounted(load)
</script>

<style scoped>
.gantt-card {
  padding: 0;
  overflow: hidden;
}

.gantt-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  padding: 16px 18px 12px;
}

.card-title {
  margin: 0;
}

.gantt-caption {
  margin: 5px 0 0;
  color: #8590a2;
  font-size: 12px;
}

.gantt-legend {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 12px;
  color: #667085;
  font-size: 12px;
}

.gantt-legend span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  white-space: nowrap;
}

.legend-swatch {
  width: 11px;
  height: 8px;
  border-radius: 3px;
  background: var(--accent);
}

.legend-swatch.temp-task {
  background: #5e8b78;
}

.legend-open {
  width: 11px;
  height: 8px;
  border: 1px dashed #667085;
  border-radius: 3px;
}

.empty {
  padding: 24px 18px;
  color: #98a2b3;
  font-size: 13px;
}

.gantt-scroll {
  overflow-x: auto;
  border-top: 1px solid rgba(76, 92, 118, 0.1);
}

.gantt-grid {
  display: grid;
  grid-template-columns: minmax(260px, 1fr) repeat(var(--day-count), minmax(30px, 1fr));
}

.gantt-header {
  position: sticky;
  top: 0;
  z-index: 3;
  min-height: 46px;
  color: #667085;
  background: rgba(247, 250, 253, 0.9);
  border-bottom: 1px solid rgba(76, 92, 118, 0.1);
}

.gantt-item-heading,
.gantt-date {
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-width: 0;
  border-right: 1px solid rgba(76, 92, 118, 0.08);
}

.gantt-item-heading {
  position: sticky;
  left: 0;
  z-index: 4;
  padding: 0 14px;
  background: rgba(247, 250, 253, 0.96);
  font-size: 12px;
  font-weight: 600;
}

.gantt-date {
  align-items: center;
  gap: 1px;
  font-size: 10px;
}

.gantt-date strong {
  color: #475467;
  font-size: 11px;
  font-weight: 600;
}

.gantt-date span {
  color: #98a2b3;
}

.gantt-date.weekend {
  background: rgba(var(--accent-rgb), 0.045);
}

.gantt-row {
  color: inherit;
  min-width: 100%;
  border-bottom: 1px solid rgba(76, 92, 118, 0.07);
}

.gantt-row:last-child {
  border-bottom: 0;
}

.gantt-row:hover .gantt-item-info {
  background: rgba(var(--accent-rgb), 0.08);
}

.gantt-row:hover .gantt-bar {
  filter: brightness(0.96);
}

.gantt-item-info {
  position: sticky;
  left: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-width: 0;
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.72);
  box-shadow: 1px 0 0 rgba(76, 92, 118, 0.08);
}

.gantt-item-title {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
  color: #344054;
  font-size: 13px;
}

.gantt-item-key {
  flex: 0 0 auto;
  color: var(--accent);
  font-size: 11px;
  font-weight: 650;
}

.gantt-track {
  position: relative;
  background-image: linear-gradient(to right, rgba(76, 92, 118, 0.08) 1px, transparent 1px);
  background-size: calc(100% / var(--day-count)) 100%;
}

.gantt-bar {
  position: absolute;
  display: block;
  height: 12px;
  min-width: 5px;
  border-radius: 3px;
  background: var(--accent);
  box-shadow: 0 1px 2px rgba(var(--accent-rgb), 0.24);
}

.gantt-bar.temp_task {
  background: #5e8b78;
  box-shadow: 0 1px 2px rgba(61, 103, 81, 0.24);
}

.gantt-bar.open {
  border: 1px dashed rgba(255, 255, 255, 0.95);
  opacity: 0.8;
}

@media (max-width: 720px) {
  .gantt-card-head {
    flex-direction: column;
  }

  .gantt-legend {
    justify-content: flex-start;
  }

  .gantt-grid {
    grid-template-columns: minmax(210px, 1fr) repeat(var(--day-count), minmax(30px, 1fr));
  }
}
</style>
