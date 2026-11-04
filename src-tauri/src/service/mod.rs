use crate::{
    config::Config,
    database,
    error::{AppError, Result},
    settings::SettingsStore,
};
use rusqlite::Connection;
use std::sync::Mutex;

pub mod export;
pub mod local;
pub mod markdown;
pub mod views;

pub struct Service {
    pub config: Config,
    pub database: Mutex<Connection>,
    pub settings: Mutex<SettingsStore>,
}

impl Service {
    pub fn new(config: Config) -> Result<Self> {
        let database = database::open(&config)?;
        let settings = SettingsStore::new(&config.data_dir);
        settings.load()?;
        Ok(Self {
            database: Mutex::new(database),
            settings: Mutex::new(settings),
            config,
        })
    }

    pub fn with_db<T>(&self, call: impl FnOnce(&Connection) -> Result<T>) -> Result<T> {
        let db = self.database.lock().map_err(|_| AppError::State)?;
        call(&db)
    }
}

pub fn now() -> String {
    chrono::Utc::now().to_rfc3339_opts(chrono::SecondsFormat::Secs, true)
}
