<template>
  <div class="dashboard-page space-y-5">
    <section class="dashboard-hero">
      <div class="hero-glow" />
      <div class="hero-copy">
        <div class="hero-eyebrow"><Sparkles :size="13" /> 工作智能面板</div>
        <h1>{{ greeting }}，今天也保持专注。</h1>
        <p>{{ fullDate }} · 在一个视图中掌握进度、记录与待办。</p>
        <n-button type="primary" size="small" class="hero-action" @click="$router.push('/issues/new')">
          <template #icon><Plus :size="15" /></template>
          新增 Issue
        </n-button>
      </div>

      <div class="hero-stats">
        <div class="metric-card metric-blue">
          <span class="metric-icon"><Activity :size="17" /></span>
          <div><strong>{{ activeIssueCount }}</strong><span>进行中</span></div>
        </div>
        <div class="metric-card metric-purple">
          <span class="metric-icon"><Layers3 :size="17" /></span>
          <div><strong>{{ tempTaskCount }}</strong><span>临时需求</span></div>
        </div>
        <div class="metric-card metric-orange">
          <span class="metric-icon"><ListChecks :size="17" /></span>
          <div><strong>{{ todoCount }}</strong><span>待跟进</span></div>
        </div>
      </div>
    </section>

    <div class="section-heading">
      <div>
        <h2>动态摘要</h2>
        <p>最近工作状态与关键进展</p>
      </div>
      <button class="soft-link" @click="$router.push('/today')">进入今日工作流 <ArrowUpRight :size="14" /></button>
    </div>

    <n-spin :show="loading">
      <div class="dashboard-grid">
        <div class="card">
          <div class="card-head"><span class="card-symbol symbol-blue"><History :size="16" /></span><h2>最近更新 Issues</h2></div>
          <ul class="card-list">
            <li v-for="issue in data?.recent_issues ?? []" :key="issue.jira_key" class="card-list-item" @click="$router.push(`/issues/${issue.jira_key}`)">
              <span class="issue-chip">{{ issue.jira_key }}</span>
              <span class="item-copy truncate">{{ issue.title }}</span>
              <ChevronRight class="row-arrow" :size="15" />
            </li>
            <li v-if="(data?.recent_issues ?? []).length === 0" class="text-gray-400 text-sm py-3">暂无数据</li>
          </ul>
        </div>

        <div class="card">
          <div class="card-head"><span class="card-symbol symbol-green"><CircleDotDashed :size="16" /></span><h2>进行中 Issues</h2></div>
          <ul class="card-list">
            <li v-for="issue in data?.active_issues ?? []" :key="issue.jira_key" class="card-list-item" @click="$router.push(`/issues/${issue.jira_key}`)">
              <span class="issue-chip">{{ issue.jira_key }}</span>
              <span class="item-copy truncate">{{ issue.title }}</span>
              <StatusTag :status="issue.status" :background="issue.background_md" />
            </li>
            <li v-if="(data?.active_issues ?? []).length === 0" class="text-gray-400 text-sm py-3">暂无数据</li>
          </ul>
        </div>

        <div class="card">
          <div class="card-head"><span class="card-symbol symbol-purple"><Layers3 :size="16" /></span><h2>临时需求</h2></div>
          <ul class="card-list">
            <li v-for="task in data?.temp_tasks ?? []" :key="task.id" class="card-list-item" @click="$router.push(`/temp-tasks/${task.id}`)">
              <span class="item-copy strong truncate">{{ task.title }}</span>
              <span v-if="task.source" class="text-gray-500 shrink-0">{{ task.source }}</span>
              <StatusTag v-else :status="task.status" :label="tempStatusLabel(task.status)" />
            </li>
            <li v-if="(data?.temp_tasks ?? []).length === 0" class="text-gray-400 text-sm py-3">暂无数据</li>
          </ul>
        </div>

        <div class="card">
          <div class="card-head"><span class="card-symbol symbol-orange"><ListChecks :size="16" /></span><h2>待跟进 TODO</h2></div>
          <ul class="card-list">
            <li v-for="todo in data?.todos ?? []" :key="todo.id" class="card-list-item" @click="$router.push(`/issues/${todo.jira_key}`)">
              <span class="todo-dot" />
              <span class="issue-chip">{{ todo.jira_key }}</span>
              <span class="item-copy truncate">{{ todo.content }}</span>
            </li>
            <li v-if="(data?.todos ?? []).length === 0" class="text-gray-400 text-sm py-3">暂无数据</li>
          </ul>
        </div>

        <div class="card week-card">
          <div class="week-card-copy">
            <div class="card-head"><span class="card-symbol symbol-indigo"><CalendarRange :size="16" /></span><h2>本周 {{ data?.week.log.week ?? '' }}</h2></div>
            <p>回顾本周轨迹，整理成果与下一步计划。</p>
            <button class="soft-link" @click="$router.push(`/weeks/${data?.week.log.week}`)">
              查看周视图 <ArrowUpRight :size="14" />
            </button>
          </div>
          <div class="week-numbers">
            <div>
              <strong>{{ data?.week.issues.length ?? 0 }}</strong>
              <span>Issues</span>
            </div>
            <i />
            <div>
              <strong>{{ data?.week.temp_tasks.length ?? 0 }}</strong>
              <span>临时需求</span>
            </div>
          </div>
        </div>
      </div>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NButton, NSpin, useMessage } from 'naive-ui'
import {
  Activity,
  ArrowUpRight,
  CalendarRange,
  ChevronRight,
  CircleDotDashed,
  History,
  Layers3,
  ListChecks,
  Plus,
  Sparkles
} from 'lucide-vue-next'
import { api } from '../api/client'
import StatusTag from '../components/StatusTag.vue'
import { tempStatusLabel } from '../utils/tempTaskDisplay'
import type { Dashboard } from '../types'

const message = useMessage()
const loading = ref(false)
const data = ref<Dashboard>()
const activeIssueCount = computed(() => data.value?.active_issues.length ?? 0)
const tempTaskCount = computed(() => data.value?.temp_tasks.length ?? 0)
const todoCount = computed(() => data.value?.todos.length ?? 0)
const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 11) return '早上好'
  if (hour < 14) return '中午好'
  if (hour < 18) return '下午好'
  return '晚上好'
})
const fullDate = new Intl.DateTimeFormat('zh-CN', {
  year: 'numeric',
  month: 'long',
  day: 'numeric',
  weekday: 'long'
}).format(new Date())

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
.dashboard-hero {
  position: relative;
  display: grid;
  min-height: 220px;
  grid-template-columns: minmax(0, 1.25fr) minmax(340px, 0.75fr);
  gap: 34px;
  align-items: center;
  overflow: hidden;
  padding: 30px 32px;
  border: 1px solid rgba(255, 255, 255, 0.82);
  border-radius: 24px;
  background:
    radial-gradient(circle at 86% 0%, rgba(128, 198, 255, 0.3), transparent 34%),
    linear-gradient(125deg, rgba(255, 255, 255, 0.92), rgba(235, 246, 255, 0.76) 53%, rgba(244, 239, 255, 0.7));
  box-shadow: 0 22px 56px rgba(47, 70, 103, 0.12), inset 0 1px 0 white;
  isolation: isolate;
}

.hero-glow {
  position: absolute;
  z-index: -1;
  width: 280px;
  height: 280px;
  right: -70px;
  top: -120px;
  border-radius: 50%;
  background: linear-gradient(135deg, rgba(90, 200, 250, 0.32), rgba(175, 82, 222, 0.12));
  filter: blur(12px);
  animation: hero-pulse 8s ease-in-out infinite alternate;
}

.hero-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
  color: var(--accent);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.075em;
  text-transform: uppercase;
}

.hero-copy h1 {
  max-width: 620px;
  font-size: clamp(27px, 2.5vw, 38px) !important;
  letter-spacing: -0.045em;
}

.hero-copy p {
  margin: 10px 0 19px;
  color: #718096;
  font-size: 13px;
}

.hero-action {
  min-width: 116px;
}

.hero-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.metric-card {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 17px;
  padding: 15px;
  border: 1px solid rgba(255, 255, 255, 0.78);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.58);
  box-shadow: 0 10px 28px rgba(46, 67, 98, 0.09), inset 0 1px 0 rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(18px);
  transition: transform 0.25s cubic-bezier(0.22, 1, 0.36, 1), box-shadow 0.25s ease;
}

.metric-card:hover {
  box-shadow: 0 15px 34px rgba(46, 67, 98, 0.13), inset 0 1px 0 white;
  transform: translateY(-3px);
}

.metric-icon,
.card-symbol {
  display: grid;
  place-items: center;
}

.metric-icon {
  width: 31px;
  height: 31px;
  border-radius: 10px;
}

.metric-card div {
  display: flex;
  flex-direction: column;
}

.metric-card strong {
  color: #182338;
  font-size: 23px;
  font-weight: 720;
  letter-spacing: -0.04em;
  line-height: 1;
}

.metric-card div span {
  margin-top: 6px;
  color: #8791a3;
  font-size: 10px;
  font-weight: 620;
  white-space: nowrap;
}

.metric-blue .metric-icon { color: var(--accent); background: rgba(var(--accent-rgb), 0.11); }
.metric-purple .metric-icon { color: #9b51d0; background: rgba(175, 82, 222, 0.11); }
.metric-orange .metric-icon { color: #ed7c17; background: rgba(255, 149, 0, 0.12); }

.section-heading {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 18px;
  padding: 4px 2px 0;
}

.section-heading h2 {
  margin: 0;
  color: #253149;
  font-size: 15px;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.section-heading p {
  margin: 3px 0 0;
  color: #98a2b3;
  font-size: 11px;
}

.soft-link {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--accent);
  font-size: 11px;
  font-weight: 650;
  transition: gap 0.2s ease, color 0.2s ease;
}

.soft-link:hover {
  gap: 8px;
  color: var(--accent-hover);
}

.dashboard-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 15px;
}

.card {
  min-height: 180px;
  padding: 19px 20px;
}

.card-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.card-head h2 {
  margin: 0;
  color: #354159;
  font-size: 12px;
  font-weight: 680;
  letter-spacing: -0.01em;
}

.card-symbol {
  width: 31px;
  height: 31px;
  border-radius: 10px;
}

.symbol-blue { color: var(--accent); background: rgba(var(--accent-rgb), 0.1); }
.symbol-green { color: #1faf4b; background: rgba(48, 209, 88, 0.1); }
.symbol-purple { color: #9b51d0; background: rgba(175, 82, 222, 0.1); }
.symbol-orange { color: #e77c19; background: rgba(255, 149, 0, 0.11); }
.symbol-indigo { color: #675cf5; background: rgba(88, 86, 214, 0.1); }

.card-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.card-list-item {
  display: flex;
  align-items: center;
  min-height: 38px;
  gap: 9px;
  padding: 7px 6px;
  border-bottom: 1px solid rgba(67, 84, 111, 0.07);
  border-radius: 9px;
  cursor: pointer;
  font-size: 12px;
  transition: background 0.18s ease, transform 0.2s ease;
}

.card-list-item:last-child {
  border-bottom: none;
}

.card-list-item:hover {
  background: rgba(var(--accent-rgb), 0.055);
  transform: translateX(2px);
}

.item-copy {
  min-width: 0;
  flex: 1;
  color: #667085;
}

.item-copy.strong {
  color: #344054;
  font-weight: 620;
}

.issue-chip {
  flex: 0 0 auto;
  color: var(--accent);
  font-size: 10px;
  font-weight: 720;
  letter-spacing: 0.01em;
}

.row-arrow {
  flex: 0 0 auto;
  color: #b1b8c4;
  transition: color 0.18s ease, transform 0.18s ease;
}

.card-list-item:hover .row-arrow {
  color: var(--accent);
  transform: translateX(2px);
}

.todo-dot {
  width: 7px;
  height: 7px;
  flex: 0 0 7px;
  border: 1.5px solid #ff9f0a;
  border-radius: 50%;
  box-shadow: 0 0 0 3px rgba(255, 159, 10, 0.08);
}

.week-card {
  grid-column: 1 / -1;
  display: flex;
  min-height: 150px;
  align-items: center;
  justify-content: space-between;
  gap: 28px;
  overflow: hidden;
  background:
    radial-gradient(circle at 90% 0, rgba(88, 86, 214, 0.12), transparent 36%),
    linear-gradient(145deg, rgba(255, 255, 255, 0.86), rgba(245, 245, 255, 0.67)) !important;
}

.week-card-copy p {
  margin: -2px 0 13px 41px;
  color: #7c8799;
  font-size: 11px;
}

.week-card-copy .soft-link {
  margin-left: 41px;
}

.week-numbers {
  display: flex;
  min-width: 250px;
  align-items: center;
  justify-content: space-around;
  padding: 17px 22px;
  border: 1px solid rgba(255, 255, 255, 0.76);
  border-radius: 17px;
  background: rgba(255, 255, 255, 0.48);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.85);
}

.week-numbers div {
  display: flex;
  min-width: 72px;
  flex-direction: column;
  align-items: center;
}

.week-numbers strong {
  color: #2b3650;
  font-size: 27px;
  font-weight: 720;
  letter-spacing: -0.05em;
  line-height: 1;
}

.week-numbers span {
  margin-top: 6px;
  color: #8993a4;
  font-size: 10px;
}

.week-numbers i {
  width: 1px;
  height: 35px;
  background: rgba(75, 91, 116, 0.1);
}

@keyframes hero-pulse {
  from { transform: scale(0.92) translate3d(-8px, -5px, 0); opacity: 0.7; }
  to { transform: scale(1.08) translate3d(6px, 8px, 0); opacity: 1; }
}

@media (max-width: 1100px) {
  .dashboard-hero {
    grid-template-columns: 1fr;
  }

  .hero-stats {
    max-width: 520px;
  }
}

@media (max-width: 720px) {
  .dashboard-hero {
    padding: 24px;
  }

  .dashboard-grid {
    grid-template-columns: 1fr;
  }

  .week-card {
    grid-column: auto;
    align-items: stretch;
    flex-direction: column;
  }

  .week-numbers {
    min-width: 0;
  }
}

/* Minimal dashboard */
.dashboard-hero {
  min-height: 170px;
  grid-template-columns: minmax(0, 1fr) minmax(300px, 0.55fr);
  gap: 24px;
  padding: 24px;
  border-color: #e1e6ec;
  border-radius: 14px;
  background: #fff;
  box-shadow: none;
}

.hero-glow {
  display: none;
}

.hero-eyebrow {
  margin-bottom: 8px;
  font-size: 9px;
  letter-spacing: 0.06em;
}

.hero-copy h1 {
  font-size: clamp(24px, 2.2vw, 31px) !important;
  letter-spacing: -0.035em;
}

.hero-copy p {
  margin: 8px 0 16px;
  font-size: 11px;
}

.hero-stats {
  gap: 8px;
}

.metric-card {
  gap: 12px;
  padding: 12px;
  border-color: #e6eaf0;
  border-radius: 10px;
  background: #f8f9fb;
  box-shadow: none;
  backdrop-filter: none;
}

.metric-card:hover {
  box-shadow: none;
  transform: none;
}

.metric-icon {
  width: 28px;
  height: 28px;
  border-radius: 8px;
}

.metric-card strong {
  font-size: 20px;
}

.dashboard-grid {
  gap: 12px;
}

.card {
  min-height: 160px;
  padding: 16px;
}

.card-symbol {
  width: 28px;
  height: 28px;
  border-radius: 8px;
}

.week-card {
  min-height: 130px;
  background: #fff !important;
}

.week-numbers {
  padding: 14px 18px;
  border-color: #e5e9ee;
  border-radius: 10px;
  background: #f8f9fb;
  box-shadow: none;
}

@media (max-width: 1100px) {
  .dashboard-hero {
    grid-template-columns: 1fr;
  }
}
</style>
