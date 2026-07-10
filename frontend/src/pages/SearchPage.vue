<template>
  <div class="search-page space-y-5">
    <div class="page-header">
      <div>
        <div class="page-kicker">Universal search</div>
        <h1>全局搜索</h1>
        <p class="page-subtitle">跨 Issues、事件、临时需求和周记录快速定位信息。</p>
      </div>
    </div>

    <div class="search-surface">
      <div class="search-orb" />
      <div class="search-mark"><Search :size="23" /></div>
      <div class="search-copy">
        <strong>查找任何工作记录</strong>
        <span>输入编号、关键词或内容片段</span>
      </div>
      <n-input v-model:value="query" placeholder="搜索 Issues、事件、临时需求、周记录…" clearable size="small" class="search-input" @keyup.enter="load">
        <template #prefix><Search :size="15" /></template>
      </n-input>
      <n-button type="primary" size="small" @click="load">搜索</n-button>
    </div>

    <div v-if="results.length" class="result-caption">找到 {{ results.length }} 条结果</div>
    <div class="result-list space-y-3">
      <div
        v-for="result in results"
        :key="`${result.type}-${result.id}`"
        class="card result-card cursor-pointer"
        @click="$router.push(result.url)"
      >
        <div class="flex items-center justify-between mb-1">
          <h3 class="text-sm font-semibold text-gray-900">{{ result.title }}</h3>
          <span class="result-meta">{{ result.type }} · {{ result.updated_at }}</span>
        </div>
        <div class="text-sm text-gray-600 markdown-body" v-html="result.snippet" />
      </div>
      <div v-if="results.length === 0 && query" class="empty-search">
        <SearchX :size="25" />
        <strong>没有找到匹配结果</strong>
        <span>试试更短的关键词，或检查拼写。</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { NButton, NInput, useMessage } from 'naive-ui'
import { Search, SearchX } from 'lucide-vue-next'
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
  padding: 14px 18px;
}

.search-surface {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  overflow: hidden;
  padding: 20px;
  border: 1px solid rgba(255, 255, 255, 0.82);
  border-radius: 21px;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.84), rgba(237, 247, 255, 0.67));
  box-shadow: 0 18px 46px rgba(48, 69, 99, 0.1), inset 0 1px 0 white;
}

.search-orb {
  position: absolute;
  width: 180px;
  height: 180px;
  top: -110px;
  right: -50px;
  border-radius: 50%;
  background: rgba(90, 200, 250, 0.18);
  filter: blur(7px);
  pointer-events: none;
}

.search-mark {
  display: grid;
  width: 45px;
  height: 45px;
  flex: 0 0 45px;
  place-items: center;
  border-radius: 14px;
  color: white;
  background: linear-gradient(145deg, var(--accent-highlight), var(--accent));
  box-shadow: 0 9px 20px rgba(var(--accent-rgb), 0.24), inset 0 1px 0 rgba(255, 255, 255, 0.35);
}

.search-copy {
  display: flex;
  min-width: 190px;
  flex-direction: column;
}

.search-copy strong {
  color: #28364e;
  font-size: 12px;
  font-weight: 680;
}

.search-copy span {
  margin-top: 3px;
  color: #8a95a7;
  font-size: 10px;
}

.search-input {
  min-width: 240px;
  flex: 1;
}

.result-caption {
  color: #8b95a6;
  font-size: 10px;
  font-weight: 650;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.result-card {
  position: relative;
}

.result-card::before {
  position: absolute;
  width: 3px;
  top: 13px;
  bottom: 13px;
  left: 0;
  border-radius: 0 5px 5px 0;
  background: linear-gradient(var(--accent-highlight), var(--accent));
  content: '';
  opacity: 0;
  transform: scaleY(0.4);
  transition: opacity 0.2s ease, transform 0.25s ease;
}

.result-card:hover {
  transform: translateY(-2px);
}

.result-card:hover::before {
  opacity: 1;
  transform: scaleY(1);
}

.result-meta {
  padding: 3px 7px;
  border-radius: 999px;
  color: #8b95a7;
  background: rgba(103, 117, 139, 0.07);
  font-size: 9px;
}

.empty-search {
  display: flex;
  min-height: 190px;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  border: 1px dashed rgba(107, 122, 146, 0.2);
  border-radius: 18px;
  color: #a0a9b8;
  background: rgba(255, 255, 255, 0.28);
}

.empty-search strong {
  margin-top: 10px;
  color: #6d788b;
  font-size: 12px;
}

.empty-search span {
  margin-top: 4px;
  font-size: 10px;
}

@media (max-width: 850px) {
  .search-surface {
    align-items: stretch;
    flex-direction: column;
  }

  .search-mark,
  .search-copy {
    display: none;
  }
}

.search-surface {
  gap: 10px;
  padding: 15px;
  border-color: #e1e6ec;
  border-radius: 12px;
  background: #fff;
  box-shadow: none;
}

.search-orb {
  display: none;
}

.search-mark {
  width: 38px;
  height: 38px;
  flex-basis: 38px;
  border-radius: 9px;
  background: var(--accent);
  box-shadow: none;
}

.result-card:hover {
  transform: none;
}

.result-card::before {
  width: 2px;
}

.empty-search {
  min-height: 160px;
  border-radius: 12px;
  background: transparent;
}
</style>
