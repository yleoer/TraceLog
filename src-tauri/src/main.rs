#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod updates;

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_updater::Builder::new().build())
        .invoke_handler(tauri::generate_handler![
            updates::get_update_info,
            updates::install_update
        ])
        .setup(|app| {
            #[cfg(feature = "update-e2e")]
            updates::run_acceptance_probe(app.handle().clone());
            let _ = app;
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("无法启动 TraceLog");
}
