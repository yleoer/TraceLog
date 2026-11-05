import type {
  AppSettings,
  Dashboard,
  DayEntry,
  Issue,
  IssueEvent,
  IssueSummaryResponse,
  IssueTodo,
  LogTimeRequest,
  LogTimeResult,
  MonthGanttView,
  SearchResult,
  TempTask,
  TempTaskEvent,
  TimeWeekView,
  TimeWorkItem,
  TodayWorkflow,
  UpdateInfo,
  UploadedImageData,
  UploadedImage,
  UploadedImageCleanup,
  WeekBounds,
  WeekView,
  WeeklyLog
} from '../types'
import { invoke, isTauri } from '@tauri-apps/api/core'
import { getCurrentWindow } from '@tauri-apps/api/window'
import { listen } from '@tauri-apps/api/event'

type QueryParams = Record<string, string | number | undefined>
type OkResponse = { ok: boolean }
type SaveResult = { path: string; canceled: boolean }

type DesktopIssueFilter = {
  Query?: string
  Status?: string
  Tag?: string
  Limit?: number
  Offset?: number
  All?: boolean
}

type DesktopTempTaskFilter = DesktopIssueFilter
type DayEntryInput = { date: string; content_md: string }

function issueFilter(params: QueryParams): DesktopIssueFilter {
  return {
    Query: stringParam(params.q),
    Status: stringParam(params.status),
    Tag: stringParam(params.tag),
    Limit: numberParam(params.limit),
    Offset: numberParam(params.offset)
  }
}

function tempTaskFilter(params: QueryParams): DesktopTempTaskFilter {
  return {
    Query: stringParam(params.q),
    Status: stringParam(params.status),
    Tag: stringParam(params.tag),
    Limit: numberParam(params.limit),
    Offset: numberParam(params.offset)
  }
}

function stringParam(value: string | number | undefined) {
  return value === undefined ? undefined : String(value)
}

function numberParam(value: string | number | undefined) {
  if (value === undefined || value === '') return undefined
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : undefined
}

function numberID(value: string | number) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) throw new Error('id must be a number')
  return parsed
}

function errorMessage(error: unknown) {
  if (typeof error === 'string') return error
  if (error instanceof Error) return error.message
  if (typeof error === 'object' && error && 'message' in error) {
    return String((error as { message?: unknown }).message)
  }
  return 'Request failed'
}

async function nativeCall<T>(call: () => Promise<T>) {
  try {
    return await call()
  } catch (error) {
    throw new Error(errorMessage(error))
  }
}

export const desktopWindow = {
  available: isTauri,
  minimise: () => getCurrentWindow().minimize(),
  toggleMaximise: () => getCurrentWindow().toggleMaximize(),
  isMaximised: () => getCurrentWindow().isMaximized(),
  startDragging: () => getCurrentWindow().startDragging(),
  close: () => getCurrentWindow().close()
}

async function fileToDataURL(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error ?? new Error('failed to read file'))
    reader.readAsDataURL(file)
  })
}

async function uploadImage(file: File, context = '') {
  const data = await fileToDataURL(file)
  return nativeCall(() => invoke('upload_image', { file: { name: file.name, data, context } })) as Promise<UploadedImage>
}

export const api = {
  dashboard: () => nativeCall(() => invoke('dashboard')) as Promise<Dashboard>,
  today: () => nativeCall(() => invoke('today')) as Promise<TodayWorkflow>,
  getMonthGantt: (month = '') => nativeCall(() => invoke('get_month_gantt', { month: month })) as Promise<MonthGanttView>,
  getSettings: () => nativeCall(() => invoke('get_settings')) as Promise<AppSettings>,
  updateSettings: (settings: Partial<AppSettings>) =>
    nativeCall(() => invoke('update_settings', { settings: settings })) as Promise<AppSettings>,
  listTimeWorkItems: () => nativeCall(() => invoke('list_time_work_items')) as Promise<TimeWorkItem[]>,
  getTimeWeek: (week = '') => nativeCall(() => invoke('get_time_week', { week: week })) as Promise<TimeWeekView>,
  refreshTimeWeek: (week = '') => nativeCall(() => invoke('refresh_time_week', { week: week })) as Promise<TimeWeekView>,
  logTempoTime: (request: LogTimeRequest) =>
    nativeCall(() => invoke('log_tempo_time', { request: request })) as Promise<LogTimeResult>,
  getUpdateInfo: () => nativeCall(() => invoke('get_update_info')) as Promise<UpdateInfo>,
  installUpdate: () => nativeCall(() => invoke<void>('install_update')),
  copyText: (text: string) => nativeCall(() => invoke<boolean>('copy_text', { text })),
  openExternal: (url: string) => nativeCall(() => invoke<void>('open_external', { url })),
  onUpdateProgress: (callback: (progress: { chunk: number; total?: number }) => void) => listen<{ chunk: number; total?: number }>('update-progress', event => callback(event.payload)),
  listIssues: (params: QueryParams) => nativeCall(() => invoke('list_issues', { filter: issueFilter(params) })) as Promise<Issue[]>,
  listCustomIssueTags: () => nativeCall(() => invoke('list_custom_issue_tags')) as Promise<string[]>,
  importJiraIssue: (jiraKey: string) => nativeCall(() => invoke('import_jira_issue', { jiraKey: jiraKey })) as Promise<Issue>,
  refreshJiraIssue: (jiraKey: string) => nativeCall(() => invoke('refresh_jira_issue', { jiraKey: jiraKey })) as Promise<Issue>,
  createIssue: (issue: Partial<Issue>) => nativeCall(() => invoke('create_issue', { issue: issue })) as Promise<Issue>,
  getIssue: (jiraKey: string) => nativeCall(() => invoke('get_issue', { jiraKey: jiraKey })) as Promise<Issue>,
  updateIssue: (jiraKey: string, issue: Partial<Issue>) =>
    nativeCall(() => invoke('update_issue', { jiraKey: jiraKey, issue: issue })) as Promise<Issue>,
  generateIssueSummary: (jiraKey: string) => nativeCall(() => invoke('generate_issue_summary', { jiraKey: jiraKey })) as Promise<IssueSummaryResponse>,
  deleteIssue: (jiraKey: string) => nativeCall(() => invoke('delete_issue', { jiraKey: jiraKey })) as Promise<OkResponse>,
  listIssueEvents: (jiraKey: string) => nativeCall(() => invoke('list_issue_events', { jiraKey: jiraKey })) as Promise<IssueEvent[]>,
  createIssueEvent: (jiraKey: string, event: Partial<IssueEvent>) =>
    nativeCall(() => invoke('create_issue_event', { jiraKey: jiraKey, event: event })) as Promise<IssueEvent>,
  updateIssueEvent: (id: number, event: Partial<IssueEvent>) =>
    nativeCall(() => invoke('update_issue_event', { id: id, event: event })) as Promise<IssueEvent>,
  deleteIssueEvent: (id: number) => nativeCall(() => invoke('delete_issue_event', { id: id })) as Promise<OkResponse>,
  listIssueTodos: (jiraKey: string, includeDone = true) => nativeCall(() => invoke('list_issue_todos', { jiraKey: jiraKey, includeDone: includeDone })) as Promise<IssueTodo[]>,
  createIssueTodo: (jiraKey: string, todo: Partial<IssueTodo>) =>
    nativeCall(() => invoke('create_issue_todo', { jiraKey: jiraKey, todo: todo })) as Promise<IssueTodo>,
  updateIssueTodo: (id: number, todo: Partial<IssueTodo>) =>
    nativeCall(() => invoke('update_issue_todo', { id: id, todo: todo })) as Promise<IssueTodo>,
  deleteIssueTodo: (id: number) => nativeCall(() => invoke('delete_issue_todo', { id: id })) as Promise<OkResponse>,
  uploadImage,
  getUploadedImageDataURL: (url: string) => nativeCall(() => invoke('get_uploaded_image_data_url', { url: url })) as Promise<UploadedImageData>,
  deleteUploadedImage: (url: string) => nativeCall(() => invoke('delete_uploaded_image', { url: url })) as Promise<OkResponse>,
  cleanupUnusedUploadedImages: () => nativeCall(() => invoke('cleanup_unused_uploaded_images')) as Promise<UploadedImageCleanup>,
  listTempTasks: (params: QueryParams) =>
    nativeCall(() => invoke('list_temp_tasks', { filter: tempTaskFilter(params) })) as Promise<TempTask[]>,
  createTempTask: (task: Partial<TempTask>) => nativeCall(() => invoke('create_temp_task', { task: task })) as Promise<TempTask>,
  getTempTask: (id: string | number) => nativeCall(() => invoke('get_temp_task', { id: numberID(id) })) as Promise<TempTask>,
  updateTempTask: (id: string | number, task: Partial<TempTask>) =>
    nativeCall(() => invoke('update_temp_task', { id: numberID(id), task: task })) as Promise<TempTask>,
  deleteTempTask: (id: string | number) => nativeCall(() => invoke('delete_temp_task', { id: numberID(id) })) as Promise<OkResponse>,
  listTempTaskEvents: (id: string | number) => nativeCall(() => invoke('list_temp_task_events', { taskId: numberID(id) })) as Promise<TempTaskEvent[]>,
  createTempTaskEvent: (id: string | number, event: Partial<TempTaskEvent>) =>
    nativeCall(() => invoke('create_temp_task_event', { taskId: numberID(id), event: event })) as Promise<TempTaskEvent>,
  updateTempTaskEvent: (id: number, event: Partial<TempTaskEvent>) =>
    nativeCall(() => invoke('update_temp_task_event', { id: id, event: event })) as Promise<TempTaskEvent>,
  deleteTempTaskEvent: (id: number) => nativeCall(() => invoke('delete_temp_task_event', { id: id })) as Promise<OkResponse>,
  createDayEntry: (entry: DayEntryInput) => nativeCall(() => invoke('create_day_entry', { entry: entry })) as Promise<DayEntry>,
  deleteDayEntry: (id: number) => nativeCall(() => invoke('delete_day_entry', { id: id })) as Promise<OkResponse>,
  listWeeks: () => nativeCall(() => invoke('list_weeks')) as Promise<WeeklyLog[]>,
  getWeekBounds: () => nativeCall(() => invoke('get_week_bounds')) as Promise<WeekBounds>,
  getWeek: (week: string) => nativeCall(() => invoke('get_week', { week: week })) as Promise<WeekView>,
  updateWeek: (week: string, log: Partial<WeeklyLog>) =>
    nativeCall(() => invoke('update_week', { week: week, log: log })) as Promise<WeeklyLog>,
  generateWeekDraft: (week: string) => nativeCall(() => invoke('generate_week_draft', { week: week })) as Promise<WeeklyLog>,
  generateWeekSummary: (week: string) => nativeCall(() => invoke('generate_week_summary', { week: week })) as Promise<WeeklyLog>,
  search: (q: string) => nativeCall(() => invoke('search', { query: q, entityType: '', limit: 50, offset: 0 })) as Promise<SearchResult[]>
}

export async function downloadUrl(path: string) {
  const save = exportActionForPath(path)
  if (!save) throw new Error('unsupported export path')
  const result = await nativeCall(save)
  if (!result.canceled && result.path) {
    console.info(`Export saved to ${result.path}`)
  }
}

function exportActionForPath(path: string): (() => Promise<SaveResult>) | null {
  if (path === '/export/json') return () => invoke<SaveResult>('export_file', { kind: 'json', key: '', id: 0 })
  if (path === '/export/markdown.zip') return () => invoke<SaveResult>('export_file', { kind: 'markdown_zip', key: '', id: 0 })

  const issue = path.match(/^\/export\/issues\/(.+?)(?:\.md)?$/)
  if (issue) return () => invoke<SaveResult>('export_file', { kind: 'issue', key: decodeURIComponent(issue[1]), id: 0 })

  const week = path.match(/^\/export\/weeks\/(.+?)(?:\.md)?$/)
  if (week) return () => invoke<SaveResult>('export_file', { kind: 'week', key: decodeURIComponent(week[1]), id: 0 })

  const task = path.match(/^\/export\/temp-tasks\/(\d+)(?:\.md)?$/)
  if (task) return () => invoke<SaveResult>('export_file', { kind: 'task', key: '', id: Number(task[1]) })

  return null
}
