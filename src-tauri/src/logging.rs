use chrono::Utc;
use std::{fs::OpenOptions, io::Write, path::Path};

pub fn record(data_dir: &Path, message: &str) {
    let directory = data_dir.join("logs");
    if std::fs::create_dir_all(&directory).is_err() {
        return;
    }
    if let Ok(mut file) = OpenOptions::new()
        .create(true)
        .append(true)
        .open(directory.join("app.log"))
    {
        let _ = writeln!(file, "{} {}", Utc::now().to_rfc3339(), message);
    }
}
