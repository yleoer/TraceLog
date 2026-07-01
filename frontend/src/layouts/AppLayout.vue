<template>
  <div class="flex h-screen overflow-hidden bg-[#f8f9fb]">
    <aside
      class="flex flex-col border-r border-gray-200 bg-white transition-all duration-200"
      :class="collapsed ? 'w-16' : 'w-56'"
    >
      <div class="flex h-10 items-center justify-center border-b border-gray-100">
        <button
          class="sidebar-toggle"
          :aria-label="collapsed ? '展开导航' : '收起导航'"
          @click="collapsed = !collapsed"
        >
          <PanelLeftClose v-if="!collapsed" :size="16" />
          <PanelLeftOpen v-else :size="16" />
        </button>
      </div>

      <nav class="flex-1 overflow-y-auto py-3 px-2 space-y-0.5 app-scrollbar-hidden">
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          :class="{ active: isActive(item.path) }"
        >
          <component :is="item.icon" :size="18" :stroke-width="1.8" />
          <span v-if="!collapsed" class="truncate">{{ item.label }}</span>
        </router-link>
      </nav>

    </aside>

    <main class="flex-1 flex flex-col overflow-hidden">
      <div class="flex-1 overflow-y-auto p-6 app-scrollbar-hidden">
        <router-view />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import {
  CalendarDays,
  ClipboardList,
  FileText,
  Home,
  PanelLeftClose,
  PanelLeftOpen,
  Search,
  Settings,
  TimerReset
} from 'lucide-vue-next'

const route = useRoute()
const collapsed = ref(false)

const navItems = [
  { path: '/', label: 'Dashboard', icon: Home },
  { path: '/today', label: '今日', icon: TimerReset },
  { path: '/issues', label: 'Issues', icon: ClipboardList },
  { path: '/temp-tasks', label: '临时需求', icon: FileText },
  { path: '/weeks', label: '周视图', icon: CalendarDays },
  { path: '/search', label: '全局搜索', icon: Search },
  { path: '/settings', label: '设置', icon: Settings }
]

const activeKey = computed(() => {
  if (route.path.startsWith('/issues')) return '/issues'
  if (route.path.startsWith('/today')) return '/today'
  if (route.path.startsWith('/temp-tasks')) return '/temp-tasks'
  if (route.path.startsWith('/weeks')) return '/weeks'
  if (route.path.startsWith('/search')) return '/search'
  if (route.path.startsWith('/settings')) return '/settings'
  return '/'
})

function isActive(path: string) {
  return activeKey.value === path
}
</script>

<style scoped>
.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 6px;
  color: #4b5563;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.15s ease;
}

.nav-item:hover {
  background: #f3f4f6;
  color: #1f2937;
}

.nav-item.active {
  background: #eff6ff;
  color: #2563eb;
}

.sidebar-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  color: #6b7280;
  transition: all 0.15s ease;
}

.sidebar-toggle:hover {
  background: #f3f4f6;
  color: #374151;
}
</style>
