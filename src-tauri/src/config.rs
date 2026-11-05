use chrono_tz::Tz;
use std::path::PathBuf;

#[derive(Clone)]
pub struct Config {
    pub data_dir: PathBuf,
    pub database_path: PathBuf,
    pub timezone: Tz,
}

impl Config {
    pub fn load() -> Self {
        let default = std::env::var_os("APPDATA")
            .map(PathBuf::from)
            .unwrap_or_else(|| PathBuf::from("."))
            .join("TraceLog");
        let data_dir = std::env::var_os("APP_DATA_DIR")
            .filter(|v| !v.is_empty())
            .map(PathBuf::from)
            .unwrap_or(default);
        let database_path = std::env::var_os("DATABASE_PATH")
            .filter(|v| !v.is_empty())
            .map(PathBuf::from)
            .unwrap_or_else(|| data_dir.join("tracelog.db"));
        Self {
            data_dir,
            database_path,
            timezone: env("APP_TIMEZONE", "Asia/Shanghai")
                .parse()
                .unwrap_or(chrono_tz::UTC),
        }
    }

    pub fn isolated(data_dir: PathBuf) -> Self {
        Self {
            database_path: data_dir.join("tracelog.db"),
            data_dir,
            timezone: chrono_tz::Asia::Shanghai,
        }
    }
}

pub fn env(name: &str, fallback: &str) -> String {
    std::env::var(name)
        .ok()
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| fallback.into())
}
