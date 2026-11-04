// 与 frontend/src/types/index.ts 保持一致；JSON 字段名为原生 IPC 契约。
use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct Link {
    pub title: String,
    pub url: String,
    pub r#type: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct Issue {
    pub id: i64,
    pub jira_key: String,
    pub title: String,
    pub status: String,
    pub priority: String,
    pub tags: Vec<String>,
    pub jira_tags: Vec<String>,
    pub custom_tags: Vec<String>,
    pub summary_md: String,
    pub background_md: String,
    pub analysis_md: String,
    pub solution_md: String,
    pub actions_md: String,
    pub result_md: String,
    pub todo_md: String,
    pub links: Vec<Link>,
    pub started_at: String,
    pub completed_at: String,
    pub created_at: String,
    pub updated_at: String,
    pub work_periods: Vec<IssueWorkPeriod>,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct IssueWorkPeriod {
    pub id: i64,
    pub issue_id: i64,
    pub started_at: String,
    pub completed_at: String,
    pub reopened_at: String,
    pub reopened_status: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct IssueTodo {
    pub id: i64,
    pub issue_id: i64,
    pub jira_key: String,
    pub content: String,
    pub due_at: String,
    pub done: bool,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct IssueEvent {
    pub id: i64,
    pub issue_id: i64,
    pub event_type: String,
    pub content_md: String,
    pub happened_at: String,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct TempTaskEvent {
    pub id: i64,
    pub temp_task_id: i64,
    pub event_type: String,
    pub content_md: String,
    pub happened_at: String,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct TempTask {
    pub id: i64,
    pub title: String,
    pub source: String,
    pub status: String,
    pub priority: String,
    pub tags: Vec<String>,
    pub content_md: String,
    pub started_at: String,
    pub completed_at: String,
    pub converted_to_jira: bool,
    pub converted_jira_key: String,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct MonthGanttView {
    pub month: String,
    pub items: Vec<MonthGanttItem>,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct MonthGanttItem {
    pub id: i64,
    pub source: String,
    pub jira_key: String,
    pub title: String,
    pub status: String,
    pub periods: Vec<MonthGanttPeriod>,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct MonthGanttPeriod {
    pub started_at: String,
    pub completed_at: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct WeeklyLog {
    pub id: i64,
    pub week: String,
    pub summary_md: String,
    pub next_plan_md: String,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct WeekBounds {
    pub first_week: String,
    pub current_week: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct DayComment {
    pub source: String,
    pub event_id: i64,
    pub event_type: String,
    pub content_md: String,
    pub happened_at: String,
    pub ref_key: String,
    pub ref_id: i64,
    pub ref_title: String,
    pub url: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct DayActivity {
    pub source: String,
    pub ref_id: i64,
    pub ref_key: String,
    pub ref_title: String,
    pub url: String,
    pub started_at: String,
    pub comments: Vec<DayComment>,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct DayEntry {
    pub id: i64,
    pub date: String,
    pub content_md: String,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct DayWork {
    pub date: String,
    pub weekday: String,
    pub comments: Vec<DayComment>,
    pub activities: Vec<DayActivity>,
    pub entries: Vec<DayEntry>,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct WeekView {
    pub log: WeeklyLog,
    pub issues: Vec<Issue>,
    pub events: Vec<IssueEvent>,
    pub temp_tasks: Vec<TempTask>,
    pub todos: Vec<IssueTodo>,
    pub done: Vec<String>,
    pub active: Vec<String>,
    pub days: Vec<DayWork>,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct SearchResult {
    pub r#type: String,
    pub id: String,
    pub title: String,
    pub snippet: String,
    pub url: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct Dashboard {
    pub recently_imported_issues: Vec<Issue>,
    pub active_issues: Vec<Issue>,
    pub temp_tasks: Vec<TempTask>,
    pub todos: Vec<IssueTodo>,
    pub week: WeekView,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct TodayWorkflow {
    pub date: String,
    pub issues: Vec<Issue>,
    pub temp_tasks: Vec<TempTask>,
    pub todos: Vec<IssueTodo>,
    pub done: Vec<String>,
    pub active: Vec<String>,
    pub weekly_draft: String,
    pub day: DayWork,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct AppSettings {
    pub jira: JiraSettings,
    pub tempo: TempoSettings,
    pub ai: AISettings,
    pub openai: ProviderSettings,
    pub deepseek: ProviderSettings,
    pub prompts: PromptSettings,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct JiraSettings {
    pub base_url: String,
    pub email: String,
    pub api_token: String,
    pub has_api_token: bool,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct TempoSettings {
    pub base_url: String,
    pub api_token: String,
    pub has_api_token: bool,
    pub author_account_id: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct AISettings {
    pub provider: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct ProviderSettings {
    pub base_url: String,
    pub model: String,
    pub api_key: String,
    pub has_api_key: bool,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct PromptSettings {
    pub issue_summary: String,
    pub weekly_summary: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct IssueSummaryResponse {
    pub summary: String,
    pub issue: Issue,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct UploadedImage {
    pub url: String,
    pub filename: String,
    pub content_type: String,
    pub size: i64,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct UploadedImageData {
    pub url: String,
    pub data_url: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct UploadedImageCleanup {
    pub scanned: i64,
    pub deleted: i64,
    pub kept: i64,
    pub failed: i64,
    pub freed_bytes: i64,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct UpdateInfo {
    pub current_version: String,
    pub latest_version: String,
    pub has_update: bool,
    pub skipped: bool,
    pub message: String,
    pub checked_at: String,
    pub release_url: String,
    pub asset_name: String,
    pub asset_url: String,
    pub asset_digest: String,
    pub published_at: String,
    pub release_notes: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct UpdateInstallResult {
    pub started: bool,
    pub path: String,
    pub message: String,
    pub will_quit: bool,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct TimeWorkItem {
    pub key: String,
    pub label: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct LogTimeRequest {
    pub work_item_key: String,
    pub description: String,
    pub hours: f64,
    pub start_date: String,
    pub end_date: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct LogTimeResult {
    pub work_item_key: String,
    pub description: String,
    pub hours: f64,
    pub start_date: String,
    pub end_date: String,
    pub total: i64,
    pub successful: i64,
    pub failed: i64,
    pub entries: Vec<LogTimeEntry>,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct LogTimeEntry {
    pub date: String,
    pub tempo_worklog_id: i64,
    #[serde(rename = "self")]
    pub self_url: String,
    pub start_time: String,
    pub end_time: String,
    pub error: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct TimeWeekView {
    pub week: String,
    pub start_date: String,
    pub end_date: String,
    pub worklogs: Vec<TimeWorklog>,
    pub days: Vec<TimeDay>,
    pub total_hours: f64,
    pub work_items: Vec<TimeWorkItem>,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct TimeDay {
    pub date: String,
    pub weekday: String,
    pub worklogs: Vec<TimeWorklog>,
    pub total_hours: f64,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default)]
pub struct TimeWorklog {
    pub tempo_worklog_id: i64,
    pub work_item_key: String,
    pub work_item_label: String,
    pub description: String,
    pub start_date: String,
    pub start_time: String,
    pub end_time: String,
    pub time_spent_seconds: i64,
    pub hours: f64,
    #[serde(rename = "self")]
    pub self_url: String,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
#[serde(default, rename_all = "PascalCase")]
pub struct ListFilter {
    pub query: String,
    pub status: String,
    pub tag: String,
    pub limit: i64,
    pub offset: i64,
    pub all: bool,
}
