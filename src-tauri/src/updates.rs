use chrono::Utc;
use std::{fs::OpenOptions, io::Write, path::PathBuf};
use tauri::{AppHandle, Emitter};
use tauri_plugin_updater::UpdaterExt;
use tracelog::models::UpdateInfo;

fn log_path() -> PathBuf {
    std::env::var_os("APP_DATA_DIR")
        .map(PathBuf::from)
        .unwrap_or_else(|| {
            PathBuf::from(std::env::var_os("APPDATA").unwrap_or_default()).join("TraceLog")
        })
        .join("logs/update.log")
}

pub fn log(message: &str) {
    let path = log_path();
    if let Some(parent) = path.parent() {
        let _ = std::fs::create_dir_all(parent);
    }
    if let Ok(mut file) = OpenOptions::new().create(true).append(true).open(path) {
        let _ = writeln!(file, "{} {}", Utc::now().to_rfc3339(), message);
    }
}

fn updater(app: &AppHandle) -> Result<tauri_plugin_updater::Updater, String> {
    let builder = app
        .updater_builder()
        .on_before_exit(|| log("安装器即将接管，应用退出"));
    builder.build().map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn get_update_info(app: AppHandle) -> Result<UpdateInfo, String> {
    if cfg!(debug_assertions) {
        return Ok(UpdateInfo {
            current_version: app.package_info().version.to_string(),
            skipped: true,
            message: "开发版本不检查更新".into(),
            ..Default::default()
        });
    }
    log("开始检查更新");
    let update = updater(&app)?.check().await.map_err(|e| {
        log(&format!("检查更新失败：{e}"));
        e.to_string()
    })?;
    let version = app.package_info().version.to_string();
    Ok(UpdateInfo {
        current_version: version.clone(),
        latest_version: update
            .as_ref()
            .map(|u| u.version.clone())
            .unwrap_or(version),
        has_update: update.is_some(),
        skipped: false,
        message: if update.is_some() {
            "发现新版本"
        } else {
            "当前已是最新版本"
        }
        .into(),
        checked_at: Utc::now().to_rfc3339(),
        release_url: "https://github.com/yleoer/TraceLog/releases".into(),
        asset_name: update
            .as_ref()
            .map(|u| {
                u.download_url
                    .path()
                    .rsplit('/')
                    .next()
                    .unwrap_or("")
                    .into()
            })
            .unwrap_or_default(),
        asset_url: update
            .as_ref()
            .map(|u| u.download_url.to_string())
            .unwrap_or_default(),
        asset_digest: String::new(),
        published_at: update
            .as_ref()
            .and_then(|u| u.date)
            .map(|d| d.to_string())
            .unwrap_or_default(),
        release_notes: update.and_then(|u| u.body).unwrap_or_default(),
    })
}

#[tauri::command]
pub async fn install_update(app: AppHandle) -> Result<(), String> {
    let update = updater(&app)?
        .check()
        .await
        .map_err(|e| e.to_string())?
        .ok_or_else(|| "当前已是最新版本".to_string())?;
    log(&format!("开始下载并校验版本 {}", update.version));
    let progress_app = app.clone();
    update
        .download_and_install(
            move |chunk, total| {
                let _ = progress_app.emit(
                    "update-progress",
                    serde_json::json!({"chunk": chunk, "total": total}),
                );
            },
            || log("下载结束，正在校验签名并启动安装器"),
        )
        .await
        .map_err(|error| {
            log(&format!("更新失败：{error}"));
            error.to_string()
        })
}
