<template>
  <div class="space-y-5 max-w-3xl mx-auto">
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-bold text-gray-900">设置 / 导出</h1>
      <n-button type="primary" size="small" :loading="saving" @click="saveSettings">保存设置</n-button>
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
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useMessage } from 'naive-ui'
import { api, downloadUrl } from '../api/client'
import type { AppSettings } from '../types'

const message = useMessage()
const saving = ref(false)

const form = reactive<AppSettings>({
  jira: { base_url: '', email: '', api_token: '', has_api_token: false },
  ai: { provider: 'openai' },
  openai: { base_url: 'https://api.openai.com/v1', model: 'gpt-4.1-mini', api_key: '', has_api_key: false },
  deepseek: { base_url: 'https://api.deepseek.com', model: 'deepseek-v4-flash', api_key: '', has_api_key: false },
  prompts: { issue_summary: '', weekly_summary: '' }
})

async function loadSettings() {
  try {
    const settings = await api.getSettings()
    Object.assign(form, settings)
    clearSecrets()
  } catch (error) {
    message.error((error as Error).message)
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
  form.openai.api_key = ''
  form.deepseek.api_key = ''
}

onMounted(loadSettings)
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
