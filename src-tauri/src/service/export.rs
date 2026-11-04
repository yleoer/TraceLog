use super::{local, markdown, now, views};
use crate::{
    error::{AppError, Result},
    models::*,
    store,
};
use chrono_tz::Tz;
use rusqlite::Connection;
use std::{
    collections::BTreeMap,
    io::{Cursor, Write},
};

pub fn json(db: &Connection) -> Result<Vec<u8>> {
    let issues = local::list_issues(
        db,
        ListFilter {
            all: true,
            ..Default::default()
        },
    )?;
    let mut events = BTreeMap::new();
    let mut todos = BTreeMap::new();
    for issue in &issues {
        events.insert(
            &issue.jira_key,
            local::list_issue_events(db, &issue.jira_key)?,
        );
        todos.insert(
            &issue.jira_key,
            local::list_todos(db, &issue.jira_key, true)?,
        );
    }
    Ok(serde_json::to_vec_pretty(
        &serde_json::json!({"schema_version":1,"exported_at":now(),"issues":issues,"issue_events":events,"issue_todos":todos,"temp_tasks":local::list_tasks(db,ListFilter{all:true,..Default::default()})?,"weekly_logs":views::list_weeks(db)?,"temp_task_events":store::query::<TempTaskEvent>(db,"SELECT * FROM temp_task_events ORDER BY id",[])?,"day_entries":store::query::<DayEntry>(db,"SELECT * FROM day_entries ORDER BY date,created_at",[])?}),
    )?)
}

pub fn issue(db: &Connection, key: &str) -> Result<(String, Vec<u8>)> {
    let item = local::get_issue(db, key)?;
    Ok((
        format!("{}.md", item.jira_key),
        markdown::issue(
            &item,
            &local::list_issue_events(db, key)?,
            &local::list_todos(db, key, true)?,
        )
        .into_bytes(),
    ))
}

pub fn task(db: &Connection, id: i64) -> Result<(String, Vec<u8>)> {
    Ok((
        format!("temp-task-{id}.md"),
        markdown::task(&local::get_task(db, id)?).into_bytes(),
    ))
}
pub fn week(db: &Connection, week: &str, timezone: Tz) -> Result<(String, Vec<u8>)> {
    let view = views::week(db, week, timezone)?;
    Ok((
        format!("{}.md", view.log.week),
        markdown::week(&view).into_bytes(),
    ))
}

pub fn archive(db: &Connection, timezone: Tz, upload_dir: &std::path::Path) -> Result<Vec<u8>> {
    let mut writer = zip::ZipWriter::new(Cursor::new(Vec::new()));
    fn add(writer: &mut zip::ZipWriter<Cursor<Vec<u8>>>, name: &str, data: &[u8]) -> Result<()> {
        writer
            .start_file(name, zip::write::SimpleFileOptions::default())
            .map_err(|e| AppError::Remote(format!("创建导出压缩包失败：{e}")))?;
        if name.ends_with(".md") && name.contains('/') {
            let content = String::from_utf8_lossy(data).replace("](/uploads/", "](../uploads/");
            writer.write_all(content.as_bytes())?;
        } else {
            writer.write_all(data)?;
        }
        Ok(())
    }
    add(&mut writer, "data.json", &json(db)?)?;
    let issues = local::list_issues(
        db,
        ListFilter {
            all: true,
            ..Default::default()
        },
    )?;
    for item in &issues {
        let (name, bytes) = issue(db, &item.jira_key)?;
        add(&mut writer, &format!("issues/{name}"), &bytes)?;
    }
    let tasks = local::list_tasks(
        db,
        ListFilter {
            all: true,
            ..Default::default()
        },
    )?;
    for item in &tasks {
        add(
            &mut writer,
            &format!("temp-tasks/{}.md", item.id),
            markdown::task(item).as_bytes(),
        )?;
    }
    let weeks = views::list_weeks(db)?;
    for log in &weeks {
        let (name, bytes) = week(db, &log.week, timezone)?;
        add(&mut writer, &format!("weekly-logs/{name}"), &bytes)?;
    }
    for entry in std::fs::read_dir(upload_dir)? {
        let entry = entry?;
        if entry.file_type()?.is_file() {
            add(
                &mut writer,
                &format!("uploads/{}", entry.file_name().to_string_lossy()),
                &std::fs::read(entry.path())?,
            )?;
        }
    }
    add(&mut writer,"index.md",format!("# TraceLog Export\n\nExported at: {}\n\n- Issues: {}\n- Temp tasks: {}\n- Weekly logs: {}\n",now(),issues.len(),tasks.len(),weeks.len()).as_bytes())?;
    Ok(writer
        .finish()
        .map_err(|e| AppError::Remote(format!("完成导出压缩包失败：{e}")))?
        .into_inner())
}
