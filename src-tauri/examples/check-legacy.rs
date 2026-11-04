use rusqlite::{Connection, OpenFlags, MAIN_DB};
use tracelog::{config::Config, database};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let source = std::env::args_os()
        .nth(1)
        .ok_or("用法：cargo run --example check-legacy -- <旧版数据库文件>")?;
    let source = Connection::open_with_flags(source, OpenFlags::SQLITE_OPEN_READ_ONLY)?;
    let temporary = tempfile::tempdir()?;
    let config = Config::isolated(temporary.path().into());
    source.backup(MAIN_DB, &config.database_path, None)?;
    let migrated = database::open(&config)?;
    let mut counts = serde_json::Map::new();
    for table in [
        "issues",
        "issue_events",
        "issue_todos",
        "issue_work_periods",
        "temp_tasks",
        "temp_task_events",
        "day_entries",
        "weekly_logs",
    ] {
        let sql = format!("SELECT count(*) FROM {table}");
        let before: i64 = source.query_row(&sql, [], |r| r.get(0))?;
        let after: i64 = migrated.query_row(&sql, [], |r| r.get(0))?;
        if before != after {
            return Err(format!("{table} 数据数量不一致").into());
        }
        counts.insert(table.into(), before.into());
    }
    let integrity: String = migrated.query_row("PRAGMA integrity_check", [], |r| r.get(0))?;
    if integrity != "ok" {
        return Err(integrity.into());
    }
    println!(
        "{}",
        serde_json::to_string_pretty(
            &serde_json::json!({"result": "旧库副本转换通过，源数据库以只读方式打开", "counts": counts})
        )?
    );
    Ok(())
}
