<template>
  <div class="space-y-5 max-w-3xl mx-auto">
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-bold text-gray-900">设置 / 导出</h1>
      <n-button type="primary" size="small" :loading="saving" @click="saveSettings">保存设置</n-button>
    </div>

    <div class="card">
      <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 class="card-title mb-2">应用更新</h2>
          <div class="space-y-1 text-sm text-gray-600">
            <div>当前版本：<span class="font-medium text-gray-900">{{ updateInfo?.current_version || '-' }}</span></div>
            <div>最新版本：<span class="font-medium text-gray-900">{{ updateInfo?.latest_version || '-' }}</span></div>
            <div v-if="updateInfo?.checked_at" class="text-xs text-gray-400">上次检查：{{ formatCheckedAt(updateInfo.checked_at) }}</div>
            <div v-if="updateInfo?.asset_name" class="text-xs text-gray-400">{{ updateInfo.asset_name }}</div>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <n-button size="small" :loading="checkingUpdate" @click="checkUpdate({ force: true })">检查更新</n-button>
          <n-button
            size="small"
            type="primary"
            :disabled="!updateInfo?.has_update || !updateInfo?.asset_url"
            :loading="installingUpdate"
            @click="installUpdate"
          >
            升级
          </n-button>
          <a v-if="updateInfo?.release_url" href="#" class="text-xs text-blue-600 hover:text-blue-700" @click.prevent="openExternalURL(updateInfo.release_url)">Release</a>
        </div>
      </div>
      <p class="text-xs mt-3" :class="updateStatusClass">{{ updateStatusText }}</p>
    </div>

    <div class="card">
      <h2 class="card-title">Jira</h2>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <n-form-item label="Base URL" label-placement="top">
          <n-input v-model:value="form.jira.base_url" size="small" placeholder="https://axyomcore.atlassian.net" />
        </n-form-item>
        <n-form-item label="Email" label-placement="top">
          <n-input v-model:value="form.jira.email" size="small" placeholder="name@example.com" />
        </n-form-item>
        <n-form-item :label="form.jira.has_api_token ? 'API Token（已配置）' : 'API Token'" label-placement="top">
          <n-input v-model:value="form.jira.api_token" size="small" type="password" show-password-on="click" placeholder="Atlassian API token" />
        </n-form-item>
      </div>
    </div>

    <div class="card">
      <h2 class="card-title">Tempo</h2>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <n-form-item label="Base URL" label-placement="top">
          <n-input v-model:value="form.tempo.base_url" size="small" placeholder="https://api.tempo.io" />
        </n-form-item>
        <n-form-item :label="form.tempo.has_api_token ? 'API Token（已配置）' : 'API Token'" label-placement="top">
          <n-input v-model:value="form.tempo.api_token" size="small" type="password" show-password-on="click" placeholder="Tempo API token" />
        </n-form-item>
        <n-form-item label="Author Account ID" label-placement="top">
          <n-input v-model:value="form.tempo.author_account_id" size="small" placeholder="Atlassian accountId" />
        </n-form-item>
      </div>
    </div>

    <div class="card">
      <h2 class="card-title">AI 总结</h2>
      <n-form-item label="默认服务" label-placement="top">
        <n-radio-group v-model:value="form.ai.provider" size="small">
          <n-radio-button value="openai">OpenAI</n-radio-button>
          <n-radio-button value="deepseek">DeepSeek</n-radio-button>
        </n-radio-group>
      </n-form-item>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
        <div class="provider-panel">
          <h3 class="text-xs font-semibold text-gray-600 mb-3">OpenAI</h3>
          <div class="space-y-3">
            <n-form-item label="Base URL" label-placement="top">
              <n-input v-model:value="form.openai.base_url" size="small" placeholder="https://api.openai.com/v1" />
            </n-form-item>
            <n-form-item label="Model" label-placement="top">
              <n-input v-model:value="form.openai.model" size="small" placeholder="gpt-4.1-mini" />
            </n-form-item>
            <n-form-item :label="form.openai.has_api_key ? 'API Key（已配置）' : 'API Key'" label-placement="top">
              <n-input v-model:value="form.openai.api_key" size="small" type="password" show-password-on="click" placeholder="OpenAI API key" />
            </n-form-item>
          </div>
        </div>

        <div class="provider-panel">
          <h3 class="text-xs font-semibold text-gray-600 mb-3">DeepSeek</h3>
          <div class="space-y-3">
            <n-form-item label="Base URL" label-placement="top">
              <n-input v-model:value="form.deepseek.base_url" size="small" placeholder="https://api.deepseek.com" />
            </n-form-item>
            <n-form-item label="Model" label-placement="top">
              <n-input v-model:value="form.deepseek.model" size="small" placeholder="deepseek-v4-flash" />
            </n-form-item>
            <n-form-item :label="form.deepseek.has_api_key ? 'API Key（已配置）' : 'API Key'" label-placement="top">
              <n-input v-model:value="form.deepseek.api_key" size="small" type="password" show-password-on="click" placeholder="DeepSeek API key" />
            </n-form-item>
          </div>
        </div>
      </div>
    </div>

    <div class="card">
      <h2 class="card-title">AI 提示词</h2>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <n-form-item label="Issue 简述提示词" label-placement="top">
          <n-input v-model:value="form.prompts.issue_summary" size="small" type="textarea" :autosize="{ minRows: 4 }" placeholder="用于 issue 详情页的 AI 总结" />
        </n-form-item>
        <n-form-item label="周总结提示词" label-placement="top">
          <n-input v-model:value="form.prompts.weekly_summary" size="small" type="textarea" :autosize="{ minRows: 5 }" placeholder="用于周视图的 AI 总结" />
        </n-form-item>
      </div>
    </div>

    <div class="card">
      <h2 class="card-title">数据导出</h2>
      <div class="flex items-center gap-2 mb-3">
        <n-button size="small" type="primary" @click="downloadUrl('/export/json')">导出 JSON</n-button>
        <n-button size="small" @click="downloadUrl('/export/markdown.zip')">导出 Markdown ZIP</n-button>
      </div>
      <p class="text-xs text-gray-400">桌面版数据默认保存在系统用户配置目录的 TraceLog 文件夹中；备份该目录即可保留 SQLite 数据库、设置和上传图片。</p>
    </div>

    <div class="card">
      <div class="flex items-center justify-between gap-4">
        <div>
          <h2 class="card-title mb-1">图片清理</h2>
          <p class="text-xs text-gray-400">删除上传目录中未被任何记录引用的图片，已引用图片会保留。</p>
        </div>
        <n-popconfirm @positive-click="cleanupImages">
          <template #trigger>
            <n-button size="small" :loading="cleaningImages">清理多余图片</n-button>
          </template>
          清理只会删除未被 Markdown 引用的上传图片，确认继续？
        </n-popconfirm>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { NButton, NFormItem, NInput, NPopconfirm, NRadioButton, NRadioGroup, useDialog, useMessage } from 'naive-ui'
import { api, downloadUrl } from '../api/client'
import type { AppSettings, UpdateInfo } from '../types'
import { openExternalURL } from '../utils/openExternal'

const message = useMessage()
const dialog = useDialog()
const saving = ref(false)
const cleaningImages = ref(false)
const checkingUpdate = ref(false)
const installingUpdate = ref(false)
const updateInfo = ref<UpdateInfo | null>(null)
const updateCacheKey = 'tracelog:update-info:v1'
const updateAutoCheckIntervalMs = 12 * 60 * 60 * 1000

interface CachedUpdateInfo {
  checkedAt: number
  info: UpdateInfo
}

const form = reactive<AppSettings>({
  jira: { base_url: '', email: '', api_token: '', has_api_token: false },
  tempo: { base_url: 'https://api.tempo.io', api_token: '', has_api_token: false, author_account_id: '' },
  ai: { provider: 'openai' },
  openai: { base_url: 'https://api.openai.com/v1', model: 'gpt-4.1-mini', api_key: '', has_api_key: false },
  deepseek: { base_url: 'https://api.deepseek.com', model: 'deepseek-v4-flash', api_key: '', has_api_key: false },
  prompts: { issue_summary: '', weekly_summary: '' }
})

const updateStatusText = computed(() => {
  if (!updateInfo.value) return '自动检查最多每 12 小时执行一次，也可手动检查更新。'
  if (updateInfo.value.skipped) return updateInfo.value.message || '开发版本不检查更新。'
  if (updateInfo.value.has_update && updateInfo.value.asset_url) return '发现新版本，升级时会关闭 TraceLog，并交给更新助手安装后重启。'
  if (updateInfo.value.has_update) return '发现新版本，但当前平台没有可用安装包。'
  return '当前已是最新版本。'
})

const updateStatusClass = computed(() => updateInfo.value?.has_update ? 'text-blue-600' : 'text-gray-400')

async function loadSettings() {
  try {
    const settings = await api.getSettings()
    Object.assign(form, settings)
    clearSecrets()
  } catch (error) {
    message.error((error as Error).message)
  }
}

async function checkUpdate(options: { force?: boolean; silent?: boolean } = {}) {
  if (!options.force && loadCachedUpdateInfo()) return
  checkingUpdate.value = true
  try {
    updateInfo.value = await api.getUpdateInfo()
    saveUpdateInfoCache(updateInfo.value)
    if (options.silent) return
    if (updateInfo.value.skipped) {
      message.info(updateInfo.value.message || '开发版本不检查更新')
      return
    }
    if (updateInfo.value.has_update) {
      message.success(`发现新版本 ${updateInfo.value.latest_version}`)
      return
    }
    message.success('当前已是最新版本')
  } catch (error) {
    if (!options.silent) message.error((error as Error).message)
  } finally {
    checkingUpdate.value = false
  }
}

function loadCachedUpdateInfo() {
  try {
    const raw = localStorage.getItem(updateCacheKey)
    if (!raw) return false
    const cache = JSON.parse(raw) as CachedUpdateInfo
    if (!cache.info || typeof cache.checkedAt !== 'number' || Date.now() - cache.checkedAt > updateAutoCheckIntervalMs) return false
    updateInfo.value = cache.info
    return true
  } catch {
    localStorage.removeItem(updateCacheKey)
    return false
  }
}

function saveUpdateInfoCache(info: UpdateInfo) {
  try {
    localStorage.setItem(updateCacheKey, JSON.stringify({ checkedAt: Date.now(), info }))
  } catch {
    // localStorage may be unavailable in some embedded/webview modes.
  }
}

async function installUpdate() {
  if (!updateInfo.value?.has_update || !updateInfo.value.asset_url) return
  dialog.warning({
    title: '确认升级',
    content: `将由更新助手下载 ${updateInfo.value.latest_version} 安装包，随后关闭 TraceLog、完成安装并尝试重启。请先保存当前编辑内容。`,
    positiveText: '升级并关闭',
    negativeText: '取消',
    onPositiveClick: startInstallUpdate
  })
}

async function startInstallUpdate() {
  installingUpdate.value = true
  try {
    const result = await api.installUpdate()
    if (result.message) {
      const type = result.will_quit ? 'warning' : 'success'
      message[type](result.message)
    }
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    installingUpdate.value = false
  }
}

async function saveSettings() {
  saving.value = true
  try {
    const settings = await api.updateSettings(form)
    Object.assign(form, settings)
    clearSecrets()
    message.success('设置已保存')
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    saving.value = false
  }
}

function clearSecrets() {
  form.jira.api_token = ''
  form.tempo.api_token = ''
  form.openai.api_key = ''
  form.deepseek.api_key = ''
}

async function cleanupImages() {
  cleaningImages.value = true
  try {
    const result = await api.cleanupUnusedUploadedImages()
    const freed = formatBytes(result.freed_bytes)
    if (result.failed > 0) {
      message.warning(`已删除 ${result.deleted} 张多余图片，释放 ${freed}；${result.failed} 个文件清理失败`)
      return
    }
    message.success(`已删除 ${result.deleted} 张多余图片，释放 ${freed}；保留 ${result.kept} 张仍在使用的图片`)
  } catch (error) {
    message.error((error as Error).message)
  } finally {
    cleaningImages.value = false
  }
}

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function formatCheckedAt(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

onMounted(async () => {
  await loadSettings()
  await checkUpdate({ silent: true })
})
</script>

<style scoped>
.card {
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  padding: 18px 20px;
}

.card-title {
  font-size: 14px;
  font-weight: 600;
  color: #374151;
  margin: 0 0 14px;
}

.provider-panel {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 14px;
}
</style>
