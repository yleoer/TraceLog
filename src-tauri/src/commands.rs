use std::sync::Arc;
use tauri::{AppHandle, State};
use tauri_plugin_clipboard_manager::ClipboardExt;
use tauri_plugin_dialog::DialogExt;
use tauri_plugin_opener::OpenerExt;
use tracelog::{
    error::{AppError, Result},
    logging,
    models::*,
    service::{export, local, time, uploads, views, Service},
    settings,
};

macro_rules! local_command {
    ($name:ident($($arg:ident:$kind:ty),*) -> $output:ty, |$db:ident,$service:ident| $body:expr) => {
        #[tauri::command]
        pub async fn $name(state:State<'_,Arc<Service>>,$($arg:$kind),*) -> Result<$output> {
            let $service=state.inner().clone();
            tauri::async_runtime::spawn_blocking(move|| {
                let result=$service.with_db(|$db| {let _ = &$service; $body});
                if let Err(error)=&result {logging::record(&$service.config.data_dir,&format!("{}：{error}",stringify!($name)));}
                result
            }).await.map_err(|_|AppError::State)?
        }
    }
}

local_command!(dashboard()->Dashboard,|db,svc| views::dashboard(db,svc.config.timezone));
local_command!(today()->TodayWorkflow,|db,svc| views::today(db,svc.config.timezone));
local_command!(
    get_month_gantt(month: String) -> MonthGanttView,
    |db, svc| views::gantt(db, &month, svc.config.timezone)
);
local_command!(get_settings()->AppSettings,|_db,svc| Ok(settings::public(svc.load_settings()?)));
macro_rules! remote_command {
    ($name:ident($($arg:ident:$kind:ty),*) -> $output:ty, |$svc:ident| $body:expr) => {
        #[tauri::command]
        pub async fn $name(state:State<'_,Arc<Service>>,$($arg:$kind),*) -> Result<$output> {
            let $svc=state.inner().clone();
            let result=$body.await;
            if let Err(error)=&result {logging::record(&$svc.config.data_dir,&format!("{}：{error}",stringify!($name)));}
            result
        }
    }
}
remote_command!(
    update_settings(settings: AppSettings) -> AppSettings,
    |svc| svc.update_settings(settings)
);
remote_command!(import_jira_issue(jira_key: String) -> Issue, |svc| svc
    .import_jira(&jira_key));
remote_command!(refresh_jira_issue(jira_key: String) -> Issue, |svc| svc
    .refresh_jira(&jira_key));
remote_command!(
    generate_issue_summary(jira_key: String) -> IssueSummaryResponse,
    |svc| svc.issue_summary(&jira_key)
);
remote_command!(generate_week_summary(week: String) -> WeeklyLog, |svc| svc
    .week_summary(&week));
remote_command!(get_time_week(week: String) -> TimeWeekView, |svc| svc
    .time_week(&week, false));
remote_command!(refresh_time_week(week: String) -> TimeWeekView, |svc| svc
    .time_week(&week, true));
remote_command!(
    log_tempo_time(request: LogTimeRequest) -> LogTimeResult,
    |svc| svc.log_time(request)
);
local_command!(list_time_work_items()->Vec<TimeWorkItem>,|_db,svc|Ok(time::work_items()));
local_command!(
    upload_image(file: uploads::UploadFile) -> UploadedImage,
    |_db, svc| uploads::save(&svc.config.data_dir.join("uploads"), file)
);
local_command!(
    get_uploaded_image_data_url(url: String) -> UploadedImageData,
    |_db, svc| uploads::data_url(&svc.config.data_dir.join("uploads"), &url)
);
local_command!(
    delete_uploaded_image(url: String) -> serde_json::Value,
    |db, svc| Ok(
        serde_json::json!({"ok":uploads::delete(db,&svc.config.data_dir.join("uploads"),&url)?})
    )
);
local_command!(cleanup_unused_uploaded_images()->UploadedImageCleanup,|db,svc|uploads::cleanup(db,&svc.config.data_dir.join("uploads")));

#[tauri::command]
pub fn copy_text(app: AppHandle, text: String) -> Result<bool> {
    app.clipboard()
        .write_text(text)
        .map_err(|_| AppError::Remote("复制到剪贴板失败".into()))?;
    Ok(true)
}
#[tauri::command]
pub fn open_external(app: AppHandle, url: String) -> Result<()> {
    let parsed = url::Url::parse(&url).map_err(|_| tracelog::error::invalid("链接无效"))?;
    if !matches!(parsed.scheme(), "https" | "http" | "mailto") {
        return Err(tracelog::error::invalid("只允许打开网页或邮件链接"));
    }
    app.opener()
        .open_url(url, None::<&str>)
        .map_err(|_| AppError::Remote("打开链接失败".into()))
}

local_command!(list_issues(filter: ListFilter) -> Vec<Issue>, |db, svc| {
    local::list_issues(db, filter)
});
local_command!(list_custom_issue_tags()->Vec<String>,|db,svc| local::custom_tags(db));
local_command!(get_issue(jira_key: String) -> Issue, |db, svc| {
    local::get_issue(db, &jira_key)
});
local_command!(create_issue(issue: Issue) -> Issue, |db, svc| {
    local::save_issue(db, None, issue, false)
});
local_command!(
    update_issue(jira_key: String, issue: Issue) -> Issue,
    |db, svc| local::save_issue(db, Some(&jira_key), issue, false)
);
local_command!(
    delete_issue(jira_key: String) -> serde_json::Value,
    |db, svc| {
        local::delete_issue(db, &jira_key)?;
        Ok(serde_json::json!({"ok":true}))
    }
);
local_command!(
    list_issue_events(jira_key: String) -> Vec<IssueEvent>,
    |db, svc| local::list_issue_events(db, &jira_key)
);
local_command!(
    create_issue_event(jira_key: String, event: IssueEvent) -> IssueEvent,
    |db, svc| local::save_issue_event(db, Some(&jira_key), 0, event)
);
local_command!(
    update_issue_event(id: i64, event: IssueEvent) -> IssueEvent,
    |db, svc| local::save_issue_event(db, None, id, event)
);
local_command!(
    delete_issue_event(id: i64) -> serde_json::Value,
    |db, svc| {
        local::delete_event(db, true, id)?;
        Ok(serde_json::json!({"ok":true}))
    }
);
local_command!(
    list_issue_todos(jira_key: String, include_done: bool) -> Vec<IssueTodo>,
    |db, svc| local::list_todos(db, &jira_key, include_done)
);
local_command!(
    create_issue_todo(jira_key: String, todo: IssueTodo) -> IssueTodo,
    |db, svc| local::save_todo(db, Some(&jira_key), 0, todo)
);
local_command!(
    update_issue_todo(id: i64, todo: IssueTodo) -> IssueTodo,
    |db, svc| local::save_todo(db, None, id, todo)
);
local_command!(
    delete_issue_todo(id: i64) -> serde_json::Value,
    |db, svc| {
        local::delete_todo(db, id)?;
        Ok(serde_json::json!({"ok":true}))
    }
);
local_command!(
    list_temp_tasks(filter: ListFilter) -> Vec<TempTask>,
    |db, svc| local::list_tasks(db, filter)
);
local_command!(get_temp_task(id: i64) -> TempTask, |db, svc| {
    local::get_task(db, id)
});
local_command!(create_temp_task(task: TempTask) -> TempTask, |db, svc| {
    local::save_task(db, 0, task)
});
local_command!(
    update_temp_task(id: i64, task: TempTask) -> TempTask,
    |db, svc| local::save_task(db, id, task)
);
local_command!(delete_temp_task(id: i64) -> serde_json::Value, |db, svc| {
    local::delete_task(db, id)?;
    Ok(serde_json::json!({"ok":true}))
});
local_command!(
    list_temp_task_events(task_id: i64) -> Vec<TempTaskEvent>,
    |db, svc| local::list_task_events(db, task_id)
);
local_command!(
    create_temp_task_event(task_id: i64, event: TempTaskEvent) -> TempTaskEvent,
    |db, svc| local::save_task_event(db, Some(task_id), 0, event)
);
local_command!(
    update_temp_task_event(id: i64, event: TempTaskEvent) -> TempTaskEvent,
    |db, svc| local::save_task_event(db, None, id, event)
);
local_command!(
    delete_temp_task_event(id: i64) -> serde_json::Value,
    |db, svc| {
        local::delete_event(db, false, id)?;
        Ok(serde_json::json!({"ok":true}))
    }
);
local_command!(create_day_entry(entry: DayEntry) -> DayEntry, |db, svc| {
    local::create_day(db, entry)
});
local_command!(delete_day_entry(id: i64) -> serde_json::Value, |db, svc| {
    local::delete_day(db, id)?;
    Ok(serde_json::json!({"ok":true}))
});
local_command!(list_weeks()->Vec<WeeklyLog>,|db,svc| views::list_weeks(db));
local_command!(get_week_bounds()->WeekBounds,|db,svc| views::week_bounds(db,svc.config.timezone));
local_command!(get_week(week: String) -> WeekView, |db, svc| views::week(
    db,
    &week,
    svc.config.timezone
));
local_command!(
    update_week(week: String, log: WeeklyLog) -> WeeklyLog,
    |db, svc| views::save_week(db, &week, log, svc.config.timezone)
);
local_command!(generate_week_draft(week: String) -> WeeklyLog, |db, svc| {
    views::draft_week(db, &week, svc.config.timezone)
});
local_command!(
    search(query: String, entity_type: String, limit: i64, offset: i64) -> Vec<SearchResult>,
    |db, svc| views::search(db, &query, &entity_type, limit, offset)
);

#[derive(serde::Serialize)]
pub struct SaveResult {
    path: String,
    canceled: bool,
}

#[derive(serde::Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ExportKind {
    Json,
    MarkdownZip,
    Issue,
    Task,
    Week,
}

#[tauri::command]
pub async fn export_file(
    app: AppHandle,
    state: State<'_, Arc<Service>>,
    kind: ExportKind,
    key: String,
    id: i64,
) -> Result<SaveResult> {
    let service = state.inner().clone();
    tauri::async_runtime::spawn_blocking(move || {
        let (name, bytes) = service.with_db(|db| match kind {
            ExportKind::Json => Ok(("tracelog.json".into(), export::json(db)?)),
            ExportKind::MarkdownZip => Ok((
                "tracelog-markdown.zip".into(),
                export::archive(
                    db,
                    service.config.timezone,
                    &service.config.data_dir.join("uploads"),
                )?,
            )),
            ExportKind::Issue => export::issue(db, &key),
            ExportKind::Task => export::task(db, id),
            ExportKind::Week => export::week(db, &key, service.config.timezone),
        })?;
        let Some(path) = app.dialog().file().set_file_name(name).blocking_save_file() else {
            return Ok(SaveResult {
                path: String::new(),
                canceled: true,
            });
        };
        let path = path
            .into_path()
            .map_err(|_| tracelog::error::invalid("导出路径无效"))?;
        use std::io::Write;
        let mut file = tempfile::NamedTempFile::new_in(
            path.parent()
                .ok_or_else(|| tracelog::error::invalid("导出路径无效"))?,
        )?;
        file.write_all(&bytes)?;
        file.as_file().sync_all()?;
        file.persist(&path).map_err(|e| e.error)?;
        Ok(SaveResult {
            path: path.to_string_lossy().into_owned(),
            canceled: false,
        })
    })
    .await
    .map_err(|_| AppError::State)?
}
