export interface Link {
  title: string
  url: string
  type: string
}

export interface Issue {
  id: number
  jira_key: string
  title: string
  status: string
  priority: string
  tags: string[]
  summary_md: string
  background_md: string
  analysis_md: string
  solution_md: string
  actions_md: string
  result_md: string
  todo_md: string
  links: Link[]
  started_at: string
  completed_at: string
  created_at: string
  updated_at: string
}

export interface IssueTodo {
  id: number
  issue_id: number
  jira_key?: string
  content: string
  due_at: string
  done: boolean
  created_at: string
  updated_at: string
}

export interface IssueEvent {
  id: number
  issue_id: number
  event_type: string
  content_md: string
  happened_at: string
  created_at: string
  updated_at: string
}

export interface TempTaskEvent {
  id: number
  temp_task_id: number
  event_type: string
  content_md: string
  happened_at: string
  created_at: string
  updated_at: string
}

export interface TempTask {
  id: number
  title: string
  source: string
  status: string
  priority: string
  tags: string[]
  content_md: string
  started_at: string
  completed_at: string
  converted_to_jira: boolean
  converted_jira_key: string
  created_at: string
  updated_at: string
}

export interface WeeklyLog {
  id: number
  week: string
  summary_md: string
  next_plan_md: string
  created_at: string
  updated_at: string
}

export interface WeekBounds {
  first_week: string
  current_week: string
}

export interface DayComment {
  source: string
  event_id: number
  event_type: string
  content_md: string
  happened_at: string
  ref_key: string
  ref_id: number
  ref_title: string
  url: string
}

export interface DayActivity {
  source: string
  ref_id: number
  ref_key: string
  ref_title: string
  url: string
  started_at: string
  comments: DayComment[]
}

export interface DayEntry {
  id: number
  date: string
  content_md: string
  created_at: string
  updated_at: string
}

export interface DayWork {
  date: string
  weekday: string
  comments: DayComment[]
  activities: DayActivity[]
  entries: DayEntry[]
}

export interface WeekView {
  log: WeeklyLog
  issues: Issue[]
  events: IssueEvent[]
  temp_tasks: TempTask[]
  todos: IssueTodo[]
  done: string[]
  active: string[]
  days: DayWork[]
}

export interface SearchResult {
  type: string
  id: string
  title: string
  snippet: string
  url: string
  updated_at: string
}

export interface Dashboard {
  recent_issues: Issue[]
  active_issues: Issue[]
  temp_tasks: TempTask[]
  todos: IssueTodo[]
  week: WeekView
}

export interface TodayWorkflow {
  date: string
  issues: Issue[]
  temp_tasks: TempTask[]
  todos: IssueTodo[]
  done: string[]
  active: string[]
  weekly_draft: string
  day: DayWork
}

export interface AppSettings {
  jira: JiraSettings
  ai: AISettings
  openai: ProviderSettings
  deepseek: ProviderSettings
  prompts: PromptSettings
}

export interface JiraSettings {
  base_url: string
  email: string
  api_token?: string
  has_api_token: boolean
}

export interface AISettings {
  provider: string
}

export interface ProviderSettings {
  base_url: string
  model: string
  api_key?: string
  has_api_key: boolean
}

export interface PromptSettings {
  issue_summary: string
  weekly_summary: string
}

export interface IssueSummaryResponse {
  summary: string
  issue: Issue
}

export interface UploadedImage {
  url: string
  filename: string
  content_type: string
  size: number
}

export interface UploadedImageData {
  url: string
  data_url: string
}

export interface UploadedImageCleanup {
  scanned: number
  deleted: number
  kept: number
  failed: number
  freed_bytes: number
}
