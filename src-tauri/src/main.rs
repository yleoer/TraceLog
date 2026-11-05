#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod commands;
mod updates;

use std::sync::Arc;
use tauri::Manager;
use tracelog::{config::Config, logging, service::Service};

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_clipboard_manager::init())
        .plugin(tauri_plugin_opener::init())
        .invoke_handler(tauri::generate_handler![
            updates::get_update_info,
            updates::install_update,
            commands::dashboard,
            commands::today,
            commands::get_month_gantt,
            commands::get_settings,
            commands::update_settings,
            commands::list_issues,
            commands::list_custom_issue_tags,
            commands::get_issue,
            commands::create_issue,
            commands::update_issue,
            commands::delete_issue,
            commands::list_issue_events,
            commands::create_issue_event,
            commands::update_issue_event,
            commands::delete_issue_event,
            commands::list_issue_todos,
            commands::create_issue_todo,
            commands::update_issue_todo,
            commands::delete_issue_todo,
            commands::list_temp_tasks,
            commands::get_temp_task,
            commands::create_temp_task,
            commands::update_temp_task,
            commands::delete_temp_task,
            commands::list_temp_task_events,
            commands::create_temp_task_event,
            commands::update_temp_task_event,
            commands::delete_temp_task_event,
            commands::create_day_entry,
            commands::delete_day_entry,
            commands::list_weeks,
            commands::get_week_bounds,
            commands::get_week,
            commands::update_week,
            commands::generate_week_draft,
            commands::search,
            commands::export_file,
            commands::import_jira_issue,
            commands::refresh_jira_issue,
            commands::generate_issue_summary,
            commands::generate_week_summary,
            commands::list_time_work_items,
            commands::get_time_week,
            commands::refresh_time_week,
            commands::log_tempo_time,
            commands::upload_image,
            commands::get_uploaded_image_data_url,
            commands::delete_uploaded_image,
            commands::cleanup_unused_uploaded_images,
            commands::copy_text,
            commands::open_external
        ])
        .setup(|app| {
            let config = Config::load();
            let service = Service::new(config.clone()).map_err(|error| {
                logging::record(&config.data_dir, &format!("启动失败：{error}"));
                error
            })?;
            app.manage(Arc::new(service));
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("无法启动 TraceLog");
}
