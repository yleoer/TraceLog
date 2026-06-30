import type {
  AppSettings,
  Dashboard,
  DayEntry,
  Issue,
  IssueEvent,
  IssueSummaryResponse,
  IssueTodo,
  SearchResult,
  TempTask,
  TempTaskEvent,
  TodayWorkflow,
  UploadedImageData,
  UploadedImage,
  UploadedImageCleanup,
  WeekView,
  WeeklyLog
} from '../types'
import * as DesktopApp from '../wailsjs/go/desktop/App'
import { desktop, service } from '../wailsjs/go/models'

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
  return nativeCall(() => DesktopApp.UploadImage(desktop.FileUpload.createFrom({ name: file.name, data, context }))) as Promise<UploadedImage>
}

export const api = {
  dashboard: () => nativeCall(() => DesktopApp.Dashboard()) as Promise<Dashboard>,
  today: () => nativeCall(() => DesktopApp.Today()) as Promise<TodayWorkflow>,
  getSettings: () => nativeCall(() => DesktopApp.GetSettings()) as Promise<AppSettings>,
  updateSettings: (settings: Partial<AppSettings>) =>
    nativeCall(() => DesktopApp.UpdateSettings(service.AppSettings.createFrom(settings))) as Promise<AppSettings>,
  listIssues: (params: QueryParams) => nativeCall(() => DesktopApp.ListIssues(service.IssueFilter.createFrom(issueFilter(params)))) as Promise<Issue[]>,
  importJiraIssue: (jiraKey: string) => nativeCall(() => DesktopApp.ImportJiraIssue(jiraKey)) as Promise<Issue>,
  createIssue: (issue: Partial<Issue>) => nativeCall(() => DesktopApp.CreateIssue(service.Issue.createFrom(issue))) as Promise<Issue>,
  getIssue: (jiraKey: string) => nativeCall(() => DesktopApp.GetIssue(jiraKey)) as Promise<Issue>,
  updateIssue: (jiraKey: string, issue: Partial<Issue>) =>
    nativeCall(() => DesktopApp.UpdateIssue(jiraKey, service.Issue.createFrom(issue))) as Promise<Issue>,
  generateIssueSummary: (jiraKey: string) => nativeCall(() => DesktopApp.GenerateIssueSummary(jiraKey)) as Promise<IssueSummaryResponse>,
  deleteIssue: (jiraKey: string) => nativeCall(() => DesktopApp.DeleteIssue(jiraKey)) as Promise<OkResponse>,
  listIssueEvents: (jiraKey: string) => nativeCall(() => DesktopApp.ListIssueEvents(jiraKey)) as Promise<IssueEvent[]>,
  createIssueEvent: (jiraKey: string, event: Partial<IssueEvent>) =>
    nativeCall(() => DesktopApp.CreateIssueEvent(jiraKey, service.IssueEvent.createFrom(event))) as Promise<IssueEvent>,
  updateIssueEvent: (id: number, event: Partial<IssueEvent>) =>
    nativeCall(() => DesktopApp.UpdateIssueEvent(id, service.IssueEvent.createFrom(event))) as Promise<IssueEvent>,
  deleteIssueEvent: (id: number) => nativeCall(() => DesktopApp.DeleteIssueEvent(id)) as Promise<OkResponse>,
  listIssueTodos: (jiraKey: string, includeDone = true) => nativeCall(() => DesktopApp.ListIssueTodos(jiraKey, includeDone)) as Promise<IssueTodo[]>,
  createIssueTodo: (jiraKey: string, todo: Partial<IssueTodo>) =>
    nativeCall(() => DesktopApp.CreateIssueTodo(jiraKey, service.IssueTodo.createFrom(todo))) as Promise<IssueTodo>,
  updateIssueTodo: (id: number, todo: Partial<IssueTodo>) =>
    nativeCall(() => DesktopApp.UpdateIssueTodo(id, service.IssueTodo.createFrom(todo))) as Promise<IssueTodo>,
  deleteIssueTodo: (id: number) => nativeCall(() => DesktopApp.DeleteIssueTodo(id)) as Promise<OkResponse>,
  uploadImage,
  getUploadedImageDataURL: (url: string) => nativeCall(() => DesktopApp.GetUploadedImageDataURL(url)) as Promise<UploadedImageData>,
  deleteUploadedImage: (url: string) => nativeCall(() => DesktopApp.DeleteUploadedImage(url)) as Promise<OkResponse>,
  cleanupUnusedUploadedImages: () => nativeCall(() => DesktopApp.CleanupUnusedUploadedImages()) as Promise<UploadedImageCleanup>,
  listTempTasks: (params: QueryParams) =>
    nativeCall(() => DesktopApp.ListTempTasks(service.TempTaskFilter.createFrom(tempTaskFilter(params)))) as Promise<TempTask[]>,
  createTempTask: (task: Partial<TempTask>) => nativeCall(() => DesktopApp.CreateTempTask(service.TempTask.createFrom(task))) as Promise<TempTask>,
  getTempTask: (id: string | number) => nativeCall(() => DesktopApp.GetTempTask(numberID(id))) as Promise<TempTask>,
  updateTempTask: (id: string | number, task: Partial<TempTask>) =>
    nativeCall(() => DesktopApp.UpdateTempTask(numberID(id), service.TempTask.createFrom(task))) as Promise<TempTask>,
  deleteTempTask: (id: string | number) => nativeCall(() => DesktopApp.DeleteTempTask(numberID(id))) as Promise<OkResponse>,
  listTempTaskEvents: (id: string | number) => nativeCall(() => DesktopApp.ListTempTaskEvents(numberID(id))) as Promise<TempTaskEvent[]>,
  createTempTaskEvent: (id: string | number, event: Partial<TempTaskEvent>) =>
    nativeCall(() => DesktopApp.CreateTempTaskEvent(numberID(id), service.TempTaskEvent.createFrom(event))) as Promise<TempTaskEvent>,
  updateTempTaskEvent: (id: number, event: Partial<TempTaskEvent>) =>
    nativeCall(() => DesktopApp.UpdateTempTaskEvent(id, service.TempTaskEvent.createFrom(event))) as Promise<TempTaskEvent>,
  deleteTempTaskEvent: (id: number) => nativeCall(() => DesktopApp.DeleteTempTaskEvent(id)) as Promise<OkResponse>,
  createDayEntry: (entry: DayEntryInput) => nativeCall(() => DesktopApp.CreateDayEntry(service.DayEntry.createFrom(entry))) as Promise<DayEntry>,
  deleteDayEntry: (id: number) => nativeCall(() => DesktopApp.DeleteDayEntry(id)) as Promise<OkResponse>,
  listWeeks: () => nativeCall(() => DesktopApp.ListWeeks()) as Promise<WeeklyLog[]>,
  getWeek: (week: string) => nativeCall(() => DesktopApp.GetWeek(week)) as Promise<WeekView>,
  updateWeek: (week: string, log: Partial<WeeklyLog>) =>
    nativeCall(() => DesktopApp.UpdateWeek(week, service.WeeklyLog.createFrom(log))) as Promise<WeeklyLog>,
  generateWeekDraft: (week: string) => nativeCall(() => DesktopApp.GenerateWeekDraft(week)) as Promise<WeeklyLog>,
  generateWeekSummary: (week: string) => nativeCall(() => DesktopApp.GenerateWeekSummary(week)) as Promise<WeeklyLog>,
  search: (q: string) => nativeCall(() => DesktopApp.Search(q, '', 50, 0)) as Promise<SearchResult[]>
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
  if (path === '/export/json') return () => DesktopApp.ExportJSON()
  if (path === '/export/markdown.zip') return () => DesktopApp.ExportMarkdownZip()

  const issue = path.match(/^\/export\/issues\/(.+?)(?:\.md)?$/)
  if (issue) return () => DesktopApp.ExportIssueMarkdown(decodeURIComponent(issue[1]))

  const week = path.match(/^\/export\/weeks\/(.+?)(?:\.md)?$/)
  if (week) return () => DesktopApp.ExportWeekMarkdown(decodeURIComponent(week[1]))

  const task = path.match(/^\/export\/temp-tasks\/(\d+)(?:\.md)?$/)
  if (task) return () => DesktopApp.ExportTempTaskMarkdown(Number(task[1]))

  return null
}
