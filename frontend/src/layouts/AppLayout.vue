<template>
  <div class="app-shell" :class="{ 'sidebar-collapsed': collapsed }">
    <header class="window-titlebar" @dblclick="handleTitlebarDoubleClick">
      <div class="titlebar-brand">
        <span class="titlebar-app-icon"><Activity :size="14" :stroke-width="2.35" /></span>
        <strong>TraceLog</strong>
        <span class="titlebar-divider" />
        <span class="titlebar-section">{{ activeItem.label }}</span>
      </div>

      <div class="titlebar-drag-space" />

      <div class="titlebar-actions" @dblclick.stop>
        <n-popover v-model:show="themeMenuOpen" trigger="click" placement="bottom-end" :show-arrow="false" :overlap="false">
          <template #trigger>
            <button class="theme-trigger" type="button" aria-label="切换界面主题" title="切换界面主题">
              <Palette :size="14" />
              <span>{{ activeTheme.label }}</span>
              <ChevronDown :size="12" />
            </button>
          </template>
          <div class="theme-menu">
            <div class="theme-menu-head">
              <strong>工作主题</strong>
              <span>选择更舒适的专注色</span>
            </div>
            <button
              v-for="theme in themes"
              :key="theme.key"
              type="button"
              class="theme-menu-option"
              :class="{ active: selectedThemeKey === theme.key }"
              @click="selectTheme(theme.key)"
            >
              <span class="theme-swatch" :style="{ background: `linear-gradient(145deg, ${theme.highlight}, ${theme.primary})` }" />
              <span class="theme-option-copy">
                <strong>{{ theme.label }}</strong>
                <small>{{ theme.description }}</small>
              </span>
              <Check v-if="selectedThemeKey === theme.key" :size="15" />
            </button>
          </div>
        </n-popover>

        <span class="window-controls-divider" />
        <button class="window-control" type="button" aria-label="最小化" title="最小化" @click="minimiseWindow">
          <Minus :size="15" :stroke-width="1.8" />
        </button>
        <button
          class="window-control"
          type="button"
          :aria-label="isMaximised ? '还原' : '最大化'"
          :title="isMaximised ? '还原' : '最大化'"
          @click="toggleMaximise"
        >
          <Copy v-if="isMaximised" class="restore-icon" :size="12" :stroke-width="1.6" />
          <Square v-else :size="12" :stroke-width="1.6" />
        </button>
        <button class="window-control close-control" type="button" aria-label="关闭" title="关闭" @click="closeWindow">
          <X :size="15" :stroke-width="1.8" />
        </button>
      </div>
    </header>

    <div class="app-workspace">
    <aside
      class="app-sidebar"
      :class="collapsed ? 'is-collapsed' : ''"
    >
      <div class="sidebar-brand">
        <button
          class="sidebar-toggle"
          :aria-label="collapsed ? '展开导航' : '收起导航'"
          @click="collapsed = !collapsed"
        >
          <PanelLeftClose v-if="!collapsed" :size="15" />
          <PanelLeftOpen v-else :size="15" />
        </button>
      </div>

      <nav class="sidebar-nav app-scrollbar-hidden">
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          :class="{ active: isActive(item.path) }"
          :title="collapsed ? item.label : undefined"
        >
          <span class="nav-icon"><component :is="item.icon" :size="18" :stroke-width="1.85" /></span>
          <span v-if="!collapsed" class="truncate">{{ item.label }}</span>
          <span v-if="!collapsed" class="nav-indicator" />
        </router-link>
      </nav>
    </aside>

    <main class="app-main">
      <div class="page-scroll app-scrollbar-hidden">
        <div class="page-content">
          <router-view v-slot="{ Component, route: currentRoute }">
            <transition name="page" mode="out-in">
              <component :is="Component" :key="currentRoute.path" />
            </transition>
          </router-view>
        </div>
      </div>
    </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { NPopover } from 'naive-ui'
import { Quit, WindowIsMaximised, WindowMinimise, WindowToggleMaximise } from '../wailsjs/runtime/runtime'
import { useWorkTheme } from '../composables/useWorkTheme'
import type { WorkThemeKey } from '../composables/useWorkTheme'
import {
  Activity,
  CalendarDays,
  Check,
  ChevronDown,
  ClipboardList,
  Copy,
  FileText,
  Home,
  Minus,
  Palette,
  PanelLeftClose,
  PanelLeftOpen,
  Search,
  Settings,
  Square,
  Timer,
  TimerReset,
  X
} from 'lucide-vue-next'

const route = useRoute()
const collapsed = ref(false)
const isMaximised = ref(false)
const themeMenuOpen = ref(false)
const { themes, selectedThemeKey, activeTheme, setTheme } = useWorkTheme()

const navItems = [
  { path: '/', label: 'Dashboard', icon: Home },
  { path: '/today', label: '今日', icon: TimerReset },
  { path: '/time', label: 'Time', icon: Timer },
  { path: '/issues', label: 'Issues', icon: ClipboardList },
  { path: '/temp-tasks', label: '临时需求', icon: FileText },
  { path: '/weeks', label: '周视图', icon: CalendarDays },
  { path: '/search', label: '全局搜索', icon: Search },
  { path: '/settings', label: '设置', icon: Settings }
]

const activeKey = computed(() => {
  if (route.path.startsWith('/issues')) return '/issues'
  if (route.path.startsWith('/today')) return '/today'
  if (route.path.startsWith('/time')) return '/time'
  if (route.path.startsWith('/temp-tasks')) return '/temp-tasks'
  if (route.path.startsWith('/weeks')) return '/weeks'
  if (route.path.startsWith('/search')) return '/search'
  if (route.path.startsWith('/settings')) return '/settings'
  return '/'
})

const activeItem = computed(() => navItems.find((item) => item.path === activeKey.value) ?? navItems[0])

function isActive(path: string) {
  return activeKey.value === path
}

function hasWailsRuntime() {
  return typeof window !== 'undefined' && Boolean((window as Window & { runtime?: unknown }).runtime)
}

function minimiseWindow() {
  if (hasWailsRuntime()) WindowMinimise()
}

function closeWindow() {
  if (hasWailsRuntime()) Quit()
}

function toggleMaximise() {
  if (!hasWailsRuntime()) return
  WindowToggleMaximise()
  window.setTimeout(syncMaximisedState, 100)
}

function handleTitlebarDoubleClick() {
  toggleMaximise()
}

async function syncMaximisedState() {
  if (!hasWailsRuntime()) return
  try {
    isMaximised.value = await WindowIsMaximised()
  } catch {
    // Ignore transient runtime errors while the native window is resizing.
  }
}

function selectTheme(key: WorkThemeKey) {
  setTheme(key)
  themeMenuOpen.value = false
}

onMounted(() => {
  syncMaximisedState()
  window.addEventListener('resize', syncMaximisedState)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', syncMaximisedState)
})
</script>

<style scoped>
.app-shell {
  position: relative;
  display: flex;
  height: 100vh;
  flex-direction: column;
  overflow: hidden;
  isolation: isolate;
  background:
    radial-gradient(circle at 8% 10%, rgba(90, 200, 250, 0.12), transparent 30%),
    radial-gradient(circle at 95% 92%, rgba(175, 82, 222, 0.09), transparent 34%),
    linear-gradient(145deg, #f6f9fd 0%, #eef3fa 52%, #f8f6fc 100%);
}

.window-titlebar {
  position: relative;
  z-index: 20;
  display: flex;
  height: 44px;
  flex: 0 0 44px;
  align-items: center;
  border-bottom: 1px solid rgba(255, 255, 255, 0.54);
  background: rgba(247, 250, 254, 0.48);
  box-shadow: inset 0 -1px 0 rgba(83, 98, 123, 0.045);
  backdrop-filter: blur(28px) saturate(160%);
  -webkit-backdrop-filter: blur(28px) saturate(160%);
  --wails-draggable: drag;
  user-select: none;
}

.titlebar-brand {
  display: flex;
  height: 100%;
  align-items: center;
  gap: 9px;
  padding-left: 14px;
  color: #435168;
}

.titlebar-app-icon {
  display: grid;
  width: 25px;
  height: 25px;
  place-items: center;
  border: 1px solid rgba(255, 255, 255, 0.7);
  border-radius: 8px;
  color: white;
  background: linear-gradient(145deg, var(--accent-highlight), var(--accent) 58%, var(--accent-deep));
  box-shadow: 0 5px 13px rgba(var(--accent-rgb), 0.22), inset 0 1px 0 rgba(255, 255, 255, 0.34);
  transition: background 0.3s ease, box-shadow 0.3s ease;
}

.titlebar-brand strong {
  color: #2f3c52;
  font-size: 11px;
  font-weight: 720;
  letter-spacing: -0.01em;
}

.titlebar-divider,
.window-controls-divider {
  width: 1px;
  background: rgba(79, 94, 117, 0.12);
}

.titlebar-divider {
  height: 13px;
  margin: 0 1px;
}

.titlebar-section {
  color: #9099a9;
  font-size: 10px;
  font-weight: 560;
}

.titlebar-drag-space {
  min-width: 30px;
  flex: 1;
  align-self: stretch;
}

.titlebar-actions {
  display: flex;
  height: 100%;
  align-items: center;
  padding-right: 6px;
  --wails-draggable: no-drag;
}

.theme-trigger {
  display: flex;
  height: 29px;
  align-items: center;
  gap: 6px;
  margin-right: 7px;
  padding: 0 9px;
  border: 1px solid rgba(255, 255, 255, 0.7);
  border-radius: 10px;
  color: #667286;
  background: rgba(255, 255, 255, 0.42);
  box-shadow: 0 4px 12px rgba(45, 63, 90, 0.055), inset 0 1px 0 rgba(255, 255, 255, 0.8);
  font-size: 10px;
  font-weight: 620;
  transition: color 0.2s ease, background 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
}

.theme-trigger:hover {
  color: var(--accent);
  background: rgba(255, 255, 255, 0.78);
  box-shadow: 0 6px 16px rgba(var(--accent-rgb), 0.1);
  transform: translateY(-1px);
}

.window-controls-divider {
  height: 18px;
  margin-right: 3px;
}

.window-control {
  display: grid;
  width: 40px;
  height: 32px;
  place-items: center;
  border-radius: 9px;
  color: #657084;
  transition: color 0.16s ease, background 0.16s ease, transform 0.16s ease;
}

.window-control:hover {
  color: #2f3b50;
  background: rgba(73, 89, 112, 0.085);
}

.window-control:active {
  transform: scale(0.94);
}

.close-control:hover {
  color: white;
  background: linear-gradient(145deg, #ff6961, #e94a54);
  box-shadow: 0 5px 14px rgba(233, 74, 84, 0.22);
}

.restore-icon {
  transform: rotate(180deg);
}

.theme-menu {
  width: 292px;
  padding: 5px;
}

.theme-menu-head {
  display: flex;
  flex-direction: column;
  padding: 6px 8px 10px;
}

.theme-menu-head strong {
  color: #344054;
  font-size: 12px;
  font-weight: 700;
}

.theme-menu-head span {
  margin-top: 3px;
  color: #98a2b3;
  font-size: 10px;
}

.theme-menu-option {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 10px;
  padding: 9px 8px;
  border-radius: 11px;
  color: #8490a2;
  text-align: left;
  transition: color 0.18s ease, background 0.18s ease, transform 0.18s ease;
}

.theme-menu-option:hover {
  color: #526074;
  background: rgba(88, 103, 126, 0.06);
  transform: translateX(2px);
}

.theme-menu-option.active {
  color: var(--accent);
  background: rgba(var(--accent-rgb), 0.08);
}

.theme-swatch {
  width: 28px;
  height: 28px;
  flex: 0 0 28px;
  border: 2px solid rgba(255, 255, 255, 0.85);
  border-radius: 9px;
  box-shadow: 0 4px 10px rgba(43, 58, 82, 0.14);
}

.theme-option-copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.theme-option-copy strong {
  color: #3f4b5f;
  font-size: 11px;
  font-weight: 650;
}

.theme-option-copy small {
  margin-top: 2px;
  overflow: hidden;
  color: #98a2b3;
  font-size: 9px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.app-workspace {
  display: flex;
  min-height: 0;
  flex: 1;
  overflow: hidden;
}

.ambient {
  position: absolute;
  z-index: -1;
  border-radius: 999px;
  filter: blur(20px);
  opacity: 0.62;
  pointer-events: none;
  animation: ambient-float 18s ease-in-out infinite alternate;
}

.ambient-one {
  width: 360px;
  height: 360px;
  top: -190px;
  left: 18%;
  background: radial-gradient(circle, rgba(var(--accent-rgb), 0.22), rgba(var(--accent-rgb), 0.015) 70%);
}

.ambient-two {
  width: 420px;
  height: 420px;
  right: -210px;
  bottom: -190px;
  background: radial-gradient(circle, rgba(191, 90, 242, 0.15), rgba(255, 55, 95, 0.02) 70%);
  animation-delay: -7s;
}

.ambient-three {
  width: 240px;
  height: 240px;
  left: 44%;
  bottom: -160px;
  background: rgba(48, 209, 88, 0.08);
  animation-delay: -12s;
}

.app-sidebar {
  position: relative;
  z-index: 4;
  display: flex;
  flex: 0 0 248px;
  width: 248px;
  flex-direction: column;
  margin: 12px 0 12px 12px;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.74);
  border-radius: 24px;
  background: rgba(248, 251, 255, 0.64);
  box-shadow: 0 18px 50px rgba(52, 72, 103, 0.11), inset 0 1px 0 rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(28px) saturate(160%);
  -webkit-backdrop-filter: blur(28px) saturate(160%);
  transition: width 0.42s cubic-bezier(0.22, 1, 0.36, 1), flex-basis 0.42s cubic-bezier(0.22, 1, 0.36, 1);
}

.app-sidebar.is-collapsed {
  flex-basis: 76px;
  width: 76px;
}

.sidebar-brand {
  display: flex;
  min-height: 76px;
  align-items: center;
  gap: 12px;
  padding: 14px 14px 12px;
}

.brand-mark {
  display: grid;
  width: 42px;
  height: 42px;
  flex: 0 0 42px;
  place-items: center;
  border: 1px solid rgba(255, 255, 255, 0.58);
  border-radius: 14px;
  color: white;
  background: linear-gradient(145deg, var(--accent-highlight) 0%, var(--accent) 56%, var(--accent-deep) 100%);
  box-shadow: 0 9px 22px rgba(var(--accent-rgb), 0.25), inset 0 1px 1px rgba(255, 255, 255, 0.38);
  transition: background 0.3s ease, box-shadow 0.3s ease;
}

.brand-copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  line-height: 1.2;
  animation: copy-in 0.24s ease both;
}

.brand-copy strong {
  color: #172033;
  font-size: 15px;
  font-weight: 720;
  letter-spacing: -0.02em;
}

.brand-copy span {
  margin-top: 4px;
  color: #929bad;
  font-size: 10px;
  font-weight: 560;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.nav-caption {
  padding: 11px 22px 6px;
  color: #9aa3b4;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.sidebar-nav {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 4px;
  overflow-y: auto;
  padding: 7px 10px;
}

.nav-item {
  position: relative;
  display: flex;
  align-items: center;
  min-height: 43px;
  gap: 11px;
  padding: 7px 11px;
  overflow: hidden;
  border: 1px solid transparent;
  border-radius: 14px;
  color: #687386;
  font-size: 13px;
  font-weight: 590;
  transition: color 0.22s ease, background 0.22s ease, border-color 0.22s ease, transform 0.22s ease;
}

.nav-item:hover {
  border-color: rgba(255, 255, 255, 0.62);
  color: #22314a;
  background: rgba(255, 255, 255, 0.58);
  transform: translateX(2px);
}

.nav-item.active {
  border-color: rgba(255, 255, 255, 0.82);
  color: var(--accent);
  background: linear-gradient(115deg, rgba(255, 255, 255, 0.92), rgba(238, 247, 255, 0.7));
  box-shadow: 0 8px 24px rgba(43, 77, 113, 0.1), inset 0 1px 0 white;
}

.nav-icon {
  display: grid;
  width: 28px;
  height: 28px;
  flex: 0 0 28px;
  place-items: center;
  border-radius: 9px;
  transition: color 0.22s ease, background 0.22s ease, transform 0.22s ease;
}

.nav-item.active .nav-icon {
  color: white;
  background: linear-gradient(145deg, var(--accent-highlight), var(--accent));
  box-shadow: 0 5px 12px rgba(var(--accent-rgb), 0.25);
  transform: scale(1.02);
}

.nav-indicator {
  width: 5px;
  height: 5px;
  margin-left: auto;
  border-radius: 999px;
  opacity: 0;
  background: var(--accent);
  box-shadow: 0 0 0 4px rgba(var(--accent-rgb), 0.1);
  transition: opacity 0.2s ease;
}

.nav-item.active .nav-indicator {
  opacity: 1;
}

.sidebar-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  flex: 0 0 30px;
  border: 1px solid rgba(81, 96, 121, 0.09);
  border-radius: 10px;
  color: #7d8798;
  background: rgba(255, 255, 255, 0.5);
  transition: all 0.2s ease;
}

.sidebar-toggle:hover {
  color: var(--accent);
  background: rgba(255, 255, 255, 0.9);
  box-shadow: 0 5px 14px rgba(42, 60, 86, 0.1);
  transform: translateY(-1px);
}

.is-collapsed .sidebar-brand {
  flex-direction: column;
  gap: 7px;
  padding: 13px 10px 9px;
}

.is-collapsed .sidebar-toggle {
  width: 28px;
  height: 25px;
  flex-basis: 25px;
}

.is-collapsed .sidebar-nav {
  align-items: center;
  padding-inline: 9px;
}

.is-collapsed .nav-item {
  width: 48px;
  justify-content: center;
  padding-inline: 9px;
}

.is-collapsed .nav-item:hover {
  transform: translateY(-1px);
}

.sidebar-footer {
  display: flex;
  min-height: 62px;
  align-items: center;
  gap: 10px;
  margin: 9px 10px 10px;
  padding: 11px 12px;
  border: 1px solid rgba(255, 255, 255, 0.62);
  border-radius: 15px;
  background: rgba(255, 255, 255, 0.36);
}

.sync-dot {
  width: 8px;
  height: 8px;
  flex: 0 0 8px;
  border-radius: 50%;
  background: #30d158;
  box-shadow: 0 0 0 5px rgba(48, 209, 88, 0.1), 0 0 14px rgba(48, 209, 88, 0.42);
}

.footer-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  line-height: 1.25;
}

.footer-copy span {
  color: #4f5c70;
  font-size: 11px;
  font-weight: 650;
}

.footer-copy small {
  margin-top: 2px;
  color: #98a1b1;
  font-size: 9px;
}

.is-collapsed .sidebar-footer {
  justify-content: center;
  min-height: 46px;
  padding: 8px;
}

.app-main {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  overflow: hidden;
}

.app-topbar {
  display: flex;
  height: 70px;
  flex: 0 0 70px;
  align-items: center;
  justify-content: space-between;
  padding: 0 30px 0 28px;
}

.current-location,
.topbar-meta,
.status-pill {
  display: flex;
  align-items: center;
}

.current-location {
  gap: 9px;
  color: #5f6c80;
  font-size: 12px;
  font-weight: 650;
  letter-spacing: 0.01em;
}

.location-icon {
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  border: 1px solid rgba(255, 255, 255, 0.72);
  border-radius: 10px;
  color: var(--accent);
  background: rgba(255, 255, 255, 0.52);
  box-shadow: 0 5px 16px rgba(46, 67, 96, 0.07);
  backdrop-filter: blur(14px);
}

.topbar-meta {
  gap: 13px;
  color: #8791a2;
  font-size: 11px;
  font-weight: 540;
}

.status-pill {
  gap: 7px;
  padding: 6px 10px;
  border: 1px solid rgba(255, 255, 255, 0.68);
  border-radius: 999px;
  color: #657186;
  background: rgba(255, 255, 255, 0.48);
  box-shadow: 0 4px 14px rgba(48, 66, 90, 0.06);
}

.status-pill i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #30d158;
  box-shadow: 0 0 8px rgba(48, 209, 88, 0.52);
}

.page-scroll {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  padding: 0 18px 18px 0;
}

.page-content {
  min-height: 100%;
  padding: 26px 28px 34px;
  border: 1px solid rgba(255, 255, 255, 0.72);
  border-radius: 26px;
  background: rgba(255, 255, 255, 0.38);
  box-shadow: 0 24px 70px rgba(49, 68, 96, 0.1), inset 0 1px 0 rgba(255, 255, 255, 0.76);
  backdrop-filter: blur(24px) saturate(145%);
  -webkit-backdrop-filter: blur(24px) saturate(145%);
}

.page-enter-active,
.page-leave-active {
  transition: opacity 0.18s ease, transform 0.24s cubic-bezier(0.22, 1, 0.36, 1), filter 0.2s ease;
}

.page-enter-from {
  opacity: 0;
  filter: blur(3px);
  transform: translateY(7px) scale(0.995);
}

.page-leave-to {
  opacity: 0;
  transform: translateY(-3px);
}

@keyframes ambient-float {
  from { transform: translate3d(-2%, -2%, 0) scale(0.96); }
  to { transform: translate3d(6%, 5%, 0) scale(1.08); }
}

@keyframes copy-in {
  from { opacity: 0; transform: translateX(-4px); }
  to { opacity: 1; transform: translateX(0); }
}

@media (max-width: 900px) {
  .app-sidebar {
    flex-basis: 76px;
    width: 76px;
  }

  .sidebar-brand {
    flex-direction: column;
  }

  .brand-copy,
  .nav-caption,
  .nav-item > span:not(.nav-icon),
  .footer-copy {
    display: none;
  }

  .sidebar-nav {
    align-items: center;
  }

  .nav-item {
    width: 48px;
    justify-content: center;
    padding-inline: 9px;
  }

  .page-content {
    padding: 22px 20px 28px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .ambient,
  .page-enter-active,
  .page-leave-active {
    animation: none;
    transition-duration: 0.01ms;
  }
}

/* Minimal workspace treatment */
.app-shell {
  background: #f4f6f8;
}

.window-titlebar {
  height: 42px;
  flex-basis: 42px;
  border-bottom-color: #e3e7ed;
  background: rgba(250, 251, 252, 0.92);
  box-shadow: none;
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
}

.titlebar-brand {
  gap: 8px;
  padding-left: 12px;
}

.titlebar-app-icon {
  width: 23px;
  height: 23px;
  border: 0;
  border-radius: 7px;
  background: var(--accent);
  box-shadow: none;
}

.titlebar-brand strong {
  font-size: 11px;
  font-weight: 680;
}

.theme-trigger {
  height: 28px;
  border-color: #e1e6ec;
  border-radius: 8px;
  background: #fff;
  box-shadow: none;
}

.theme-trigger:hover {
  background: var(--accent-soft);
  box-shadow: none;
  transform: none;
}

.window-control {
  width: 38px;
  height: 30px;
  border-radius: 7px;
}

.close-control:hover {
  background: #e5484d;
  box-shadow: none;
}

.app-sidebar {
  flex-basis: 220px;
  width: 220px;
  margin: 0;
  border: 0;
  border-right: 1px solid #e3e7ed;
  border-radius: 0;
  background: rgba(249, 250, 251, 0.86);
  box-shadow: none;
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
}

.app-sidebar.is-collapsed {
  flex-basis: 64px;
  width: 64px;
}

.sidebar-brand {
  justify-content: flex-end;
  min-height: 48px;
  padding: 8px 12px 4px;
}

.sidebar-toggle,
.is-collapsed .sidebar-toggle {
  width: 30px;
  height: 30px;
  flex-basis: 30px;
  border: 0;
  border-radius: 8px;
  background: transparent;
}

.sidebar-toggle:hover {
  background: #eceff3;
  box-shadow: none;
  transform: none;
}

.sidebar-nav {
  gap: 2px;
  padding: 6px 10px 12px;
}

.nav-item {
  min-height: 38px;
  gap: 9px;
  padding: 5px 9px;
  border: 0;
  border-radius: 9px;
  font-size: 12px;
  font-weight: 560;
}

.nav-item:hover {
  border-color: transparent;
  background: #eceff3;
  transform: none;
}

.nav-item.active {
  border-color: transparent;
  background: rgba(var(--accent-rgb), 0.1);
  box-shadow: none;
}

.nav-icon {
  width: 26px;
  height: 26px;
  flex-basis: 26px;
  border-radius: 7px;
}

.nav-item.active .nav-icon {
  color: var(--accent);
  background: transparent;
  box-shadow: none;
  transform: none;
}

.nav-indicator {
  width: 4px;
  height: 16px;
  border-radius: 4px;
  box-shadow: none;
}

.app-main {
  background: #f7f8fa;
}

.page-scroll {
  padding: 0;
}

.page-content {
  min-height: 100%;
  padding: 24px 28px 32px;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  backdrop-filter: none;
  -webkit-backdrop-filter: none;
}

.page-enter-active,
.page-leave-active {
  transition: opacity 0.14s ease;
}

.page-enter-from,
.page-leave-to {
  filter: none;
  transform: none;
}

.theme-menu {
  width: 280px;
}

.theme-menu-option:hover {
  transform: none;
}

.theme-swatch {
  border-width: 1px;
  box-shadow: none;
}

@media (max-width: 900px) {
  .app-sidebar {
    flex-basis: 64px;
    width: 64px;
  }

  .page-content {
    padding: 20px;
  }
}
</style>
