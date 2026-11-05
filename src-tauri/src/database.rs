use crate::{
    config::Config,
    error::{invalid, Result},
    logging, store,
};
use chrono::Utc;
use rusqlite::{Connection, OptionalExtension, MAIN_DB};
use std::{
    path::{Path, PathBuf},
    time::Duration,
};

const SCHEMA: &str = include_str!("../../db/migrations/001_init.sql");
const COMPAT_VERSION: &str = "1";

pub fn open(config: &Config) -> Result<Connection> {
    std::fs::create_dir_all(&config.data_dir)?;
    if let Some(parent) = config.database_path.parent() {
        std::fs::create_dir_all(parent)?;
    }
    for name in ["uploads", "exports", "logs"] {
        std::fs::create_dir_all(config.data_dir.join(name))?;
    }
    let existed = config.database_path.exists();
    let mut db = Connection::open(&config.database_path)?;
    db.busy_timeout(Duration::from_secs(5))?;
    db.pragma_update(None, "foreign_keys", true)?;
    db.pragma_update(None, "journal_mode", "WAL")?;
    let current = compat_version(&db)?;
    if current.as_deref().is_some_and(|v| v != COMPAT_VERSION) {
        return Err(invalid("数据库来自更新版本，请使用对应版本的 TraceLog"));
    }
    if current.is_none() {
        if existed {
            let path = backup(&db, config)?;
            logging::record(
                &config.data_dir,
                &format!("旧库转换前备份：{}", path.display()),
            );
        }
        let tx = db.transaction()?;
        // 此文件还包含 goose Down 段，必须显式截断，不能整文件 execute_batch。
        let up = SCHEMA
            .split("-- +goose Down")
            .next()
            .ok_or_else(|| invalid("初始化 schema 无效"))?;
        tx.execute_batch(up)?;
        let columns: Vec<String> = tx
            .prepare("PRAGMA table_info(issues)")?
            .query_map([], |r| r.get(1))?
            .collect::<std::result::Result<_, _>>()?;
        for name in ["jira_tags", "custom_tags"] {
            if !columns.iter().any(|v| v == name) {
                tx.execute_batch(&format!(
                    "ALTER TABLE issues ADD COLUMN {name} TEXT NOT NULL DEFAULT '[]'"
                ))?;
            }
        }
        store::index::rebuild(&tx)?;
        tx.execute(
            "INSERT INTO app_schema_meta(key, value) VALUES ('rust_compat_version', ?)",
            [COMPAT_VERSION],
        )?;
        let violation: Option<String> = tx
            .query_row("PRAGMA foreign_key_check", [], |r| r.get(0))
            .optional()?;
        if violation.is_some() {
            return Err(invalid("旧数据库存在无效外键，已保留备份，请先修复数据"));
        }
        tx.commit()?;
        logging::record(&config.data_dir, "Rust 数据兼容初始化完成");
    }
    Ok(db)
}

fn compat_version(db: &Connection) -> Result<Option<String>> {
    let has_meta: bool = db.query_row("SELECT EXISTS(SELECT 1 FROM sqlite_schema WHERE type = 'table' AND name = 'app_schema_meta')", [], |r| r.get(0))?;
    if !has_meta {
        return Ok(None);
    }
    Ok(db
        .query_row(
            "SELECT value FROM app_schema_meta WHERE key = 'rust_compat_version'",
            [],
            |r| r.get(0),
        )
        .optional()?)
}

pub fn backup(db: &Connection, config: &Config) -> Result<PathBuf> {
    let root = config.data_dir.join("backups").join(format!(
        "pre-rust-{}",
        Utc::now().format("%Y%m%dT%H%M%S%.9fZ")
    ));
    std::fs::create_dir_all(&root)?;
    // Online Backup API 包含尚在 WAL 中的已提交内容。
    db.backup(MAIN_DB, root.join("tracelog.db"), None)?;
    let settings = config.data_dir.join("tracelog-settings.json");
    if settings.exists() {
        std::fs::copy(settings, root.join("tracelog-settings.json"))?;
    }
    let uploads = config.data_dir.join("uploads");
    if uploads.exists() {
        copy_directory(&uploads, &root.join("uploads"))?;
    }
    std::fs::write(
        root.join("backup.json"),
        serde_json::to_vec_pretty(
            &serde_json::json!({"created_at": Utc::now().to_rfc3339(), "database": config.database_path, "complete": true}),
        )?,
    )?;
    Ok(root)
}

fn copy_directory(source: &Path, destination: &Path) -> Result<()> {
    std::fs::create_dir_all(destination)?;
    for entry in std::fs::read_dir(source)? {
        let entry = entry?;
        let kind = entry.file_type()?;
        if kind.is_symlink() {
            return Err(invalid("图片备份不支持符号链接"));
        }
        if kind.is_dir() {
            copy_directory(&entry.path(), &destination.join(entry.file_name()))?;
        } else {
            std::fs::copy(entry.path(), destination.join(entry.file_name()))?;
        }
    }
    Ok(())
}
