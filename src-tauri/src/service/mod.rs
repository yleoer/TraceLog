use crate::{
    config::Config,
    database,
    error::{AppError, Result},
    settings::SettingsStore,
};
use rusqlite::Connection;
use std::sync::Mutex;

pub mod export;
pub mod integrations;
pub mod jira;
pub mod local;
pub mod markdown;
pub mod remote;
pub mod time;
pub mod uploads;
pub mod views;

pub struct Service {
    pub config: Config,
    pub database: Mutex<Connection>,
    pub settings: SettingsStore,
    pub http: reqwest::Client,
}

impl Service {
    pub fn new(config: Config) -> Result<Self> {
        let database = database::open(&config)?;
        let settings = SettingsStore::new(&config.data_dir);
        settings.load()?;
        Ok(Self {
            database: Mutex::new(database),
            settings,
            config,
            http: reqwest::Client::builder()
                .timeout(std::time::Duration::from_secs(60))
                .connect_timeout(std::time::Duration::from_secs(10))
                .redirect(reqwest::redirect::Policy::none())
                .build()
                .map_err(|_| AppError::Remote("无法初始化网络客户端".into()))?,
        })
    }

    pub fn with_db<T>(&self, call: impl FnOnce(&Connection) -> Result<T>) -> Result<T> {
        let db = self.database.lock().map_err(|_| AppError::State)?;
        call(&db)
    }

    pub fn load_settings(&self) -> Result<crate::models::AppSettings> {
        self.settings.load()
    }
}

pub fn now() -> String {
    chrono::Utc::now().to_rfc3339_opts(chrono::SecondsFormat::Secs, true)
}
