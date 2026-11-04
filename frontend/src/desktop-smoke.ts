// 仅通过 VITE_TRACELOG_DESKTOP_E2E=1 构建，配合 Rust desktop-e2e 功能验收。
import { api, desktopWindow, reportDesktopSmokeTest, waitForPendingCalls } from './api/client'
import router from './router'

const checks: string[] = []
const errors: string[] = []
const warnings: string[] = []
window.addEventListener('error', event => {
  // Chromium 的异步尺寸观察通知可在 Naive UI 页面切换时延后一个绘制周期。
  if (event.message === 'ResizeObserver loop completed with undelivered notifications.') warnings.push(event.message)
  else errors.push(event.message)
})
window.addEventListener('unhandledrejection', event => errors.push(String(event.reason)))
function assert(value: unknown, label: string): asserts value {
  if (!value) throw new Error(label)
  checks.push(label)
}
async function until(test: () => boolean, label: string) {
  const deadline = Date.now() + 20000
  while (!test()) {
    if (Date.now() > deadline) throw new Error(`等待超时：${label}`)
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  checks.push(label)
}

async function run() {
  await router.isReady()
  await until(() => !!document.querySelector('.app-shell'), '真实 Vue 界面已挂载')
  assert(desktopWindow.available(), 'Tauri IPC 已初始化')
  await desktopWindow.isMaximised()
  checks.push('原生窗口权限')
  const issue = await api.createIssue({ jira_key: 'TL-9001', title: '桌面验收事项', custom_tags: ['smoke'], background_md: '离线图片测试' })
  assert(issue.jira_key === 'TL-9001', 'Issue 创建契约')
  assert((await api.listIssues({ tag: 'smoke' })).length === 1, '列表筛选契约')
  const event = await api.createIssueEvent(issue.jira_key, { event_type: 'note', content_md: '桌面验收评论' })
  await api.updateIssueEvent(event.id, { ...event, content_md: '桌面更新评论' })
  const todo = await api.createIssueTodo(issue.jira_key, { content: '验收 TODO' })
  assert((await api.updateIssueTodo(todo.id, { ...todo, done: true })).done, 'TODO 更新契约')
  const task = await api.createTempTask({ title: '桌面临时需求', content_md: '临时内容' })
  const comment = await api.createTempTaskEvent(task.id, { event_type: 'note', content_md: '临时评论' })
  assert((await api.listTempTaskEvents(task.id))[0].id === comment.id, '临时评论参数契约')
  const day = await api.createDayEntry({ date: '2026-06-01', content_md: '桌面每日记录' })
  const view = await api.getWeek('2026-W23')
  assert(view.days.length === 7, '周视图契约')
  assert((await api.getMonthGantt('2026-06')).month === '2026-06', '甘特图契约')
  assert((await api.search('桌面更新评论')).length > 0, '原生搜索契约')
  const file = new File([Uint8Array.from(atob('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGNgYGD4DwABBAEAX+XDSwAAAABJRU5ErkJggg=='), char => char.charCodeAt(0))], 'smoke.png', { type: 'image/png' })
  const image = await api.uploadImage(file, 'smoke')
  const imageData = await api.getUploadedImageDataURL(image.url)
  assert(imageData.data_url.startsWith('data:image/png;base64,'), '原生图片读取契约')
  await api.updateIssue(issue.jira_key, { ...issue, background_md: `![图片](${image.url})` })
  await api.updateIssueEvent(event.id, { ...event, content_md: `桌面更新评论\n\n![图片](${image.url})` })
  await router.push(`/issues/${issue.jira_key}`)
  await until(() => !!document.querySelector('.vditor .vditor-reset[contenteditable="true"]'), '本地 Vditor 编辑器加载')
  await until(() => [...document.querySelectorAll<HTMLImageElement>('img[src^="blob:"], img[src^="data:image/png"]')].some(image => image.complete && image.naturalWidth > 0), 'Markdown 图片在真实 WebView 中显示')
  await router.push('/today')
  await until(() => !!document.body.textContent?.includes('今日'), '今日页面可访问')
  await router.push('/gantt')
  await until(() => !!document.body.textContent?.includes('甘特'), '甘特页面可访问')
  await router.push('/settings')
  await until(() => !!document.body.textContent?.includes('当前版本'), '设置页面可访问')
  await api.deleteDayEntry(day.id)
  await api.deleteTempTask(task.id)
  await api.deleteIssue(issue.jira_key)
  await api.deleteUploadedImage(image.url)
  await waitForPendingCalls()
  assert(errors.length === 0, '无未处理的 WebView 错误')
}

void run().then(
  () => reportDesktopSmokeTest({ passed: true, checks, errors, warnings }),
  error => reportDesktopSmokeTest({ passed: false, checks, errors, warnings, error: String(error) })
)
