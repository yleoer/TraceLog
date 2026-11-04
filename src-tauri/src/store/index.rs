use super::{clean_tags, query};
use crate::{error::Result, models::*};
use rusqlite::{params, Connection, OptionalExtension};

pub fn upsert(
    db: &Connection,
    kind: &str,
    id: &str,
    title: &str,
    body: &str,
    updated: &str,
) -> Result<()> {
    let row_id: Option<i64> = db
        .query_row(
            "SELECT search_rowid FROM search_index_refs WHERE entity_type = ? AND entity_id = ?",
            params![kind, id],
            |row| row.get(0),
        )
        .optional()?;
    if let Some(row_id) = row_id {
        db.execute(
            "UPDATE search_index SET title = ?, body = ?, updated_at = ? WHERE rowid = ?",
            params![title, body, updated, row_id],
        )?;
    } else {
        db.execute("INSERT INTO search_index(entity_type, entity_id, title, body, updated_at) VALUES (?, ?, ?, ?, ?)", params![kind, id, title, body, updated])?;
        db.execute(
            "INSERT INTO search_index_refs(entity_type, entity_id, search_rowid) VALUES (?, ?, ?)",
            params![kind, id, db.last_insert_rowid()],
        )?;
    }
    Ok(())
}

pub fn remove(db: &Connection, kind: &str, id: &str) -> Result<()> {
    db.execute("DELETE FROM search_index WHERE rowid IN (SELECT search_rowid FROM search_index_refs WHERE entity_type = ? AND entity_id = ?)", params![kind, id])?;
    db.execute(
        "DELETE FROM search_index_refs WHERE entity_type = ? AND entity_id = ?",
        params![kind, id],
    )?;
    Ok(())
}

pub fn issue(db: &Connection, issue: &Issue) -> Result<()> {
    let body = [
        &issue.jira_key,
        &issue.summary_md,
        &issue.background_md,
        &issue.analysis_md,
        &issue.solution_md,
        &issue.actions_md,
        &issue.result_md,
        &issue.todo_md,
        &issue.started_at,
        &issue.completed_at,
    ]
    .map(|v| v.as_str())
    .join("\n");
    upsert(
        db,
        "issue",
        &issue.jira_key,
        &format!("{} {}", issue.jira_key, issue.title),
        &body,
        &issue.updated_at,
    )
}

pub fn task(db: &Connection, task: &TempTask) -> Result<()> {
    let body = [
        &task.source,
        &task.content_md,
        &task.started_at,
        &task.completed_at,
        &task.converted_jira_key,
    ]
    .map(|v| v.as_str())
    .join("\n");
    upsert(
        db,
        "temp_task",
        &task.id.to_string(),
        &task.title,
        &body,
        &task.updated_at,
    )
}

pub fn rebuild(db: &Connection) -> Result<()> {
    db.execute_batch("DELETE FROM search_index; DELETE FROM search_index_refs; DELETE FROM issue_tags; DELETE FROM temp_task_tags;")?;
    let issues: Vec<Issue> = query(db, "SELECT * FROM issues", [])?;
    for mut item in issues {
        if item.jira_tags.is_empty() && item.custom_tags.is_empty() {
            item.custom_tags = item.tags;
        }
        item.jira_tags = clean_tags(&item.jira_tags);
        item.custom_tags = clean_tags(&item.custom_tags);
        item.tags = clean_tags(&[item.jira_tags.clone(), item.custom_tags.clone()].concat());
        db.execute(
            "UPDATE issues SET tags = ?, jira_tags = ?, custom_tags = ? WHERE id = ?",
            params![
                serde_json::to_string(&item.tags)?,
                serde_json::to_string(&item.jira_tags)?,
                serde_json::to_string(&item.custom_tags)?,
                item.id
            ],
        )?;
        for tag in &item.tags {
            db.execute(
                "INSERT INTO issue_tags(issue_id, tag) VALUES (?, ?)",
                params![item.id, tag],
            )?;
        }
        issue(db, &item)?;
    }
    for item in query::<TempTask>(db, "SELECT * FROM temp_tasks", [])? {
        for tag in clean_tags(&item.tags) {
            db.execute(
                "INSERT INTO temp_task_tags(temp_task_id, tag) VALUES (?, ?)",
                params![item.id, tag],
            )?;
        }
        task(db, &item)?;
    }
    for event in query::<IssueEvent>(db, "SELECT * FROM issue_events", [])? {
        let key: String = db.query_row(
            "SELECT jira_key FROM issues WHERE id = ?",
            [event.issue_id],
            |r| r.get(0),
        )?;
        upsert(
            db,
            "issue_event",
            &event.id.to_string(),
            &format!("{key} {}", event.event_type),
            &event.content_md,
            &event.updated_at,
        )?;
    }
    for event in query::<TempTaskEvent>(db, "SELECT * FROM temp_task_events", [])? {
        let title: String = db.query_row(
            "SELECT title FROM temp_tasks WHERE id = ?",
            [event.temp_task_id],
            |r| r.get(0),
        )?;
        upsert(
            db,
            "temp_task_event",
            &event.id.to_string(),
            &format!("{title} {}", event.event_type),
            &event.content_md,
            &event.updated_at,
        )?;
    }
    for todo in query::<IssueTodo>(
        db,
        "SELECT t.*, i.jira_key FROM issue_todos t JOIN issues i ON i.id = t.issue_id",
        [],
    )? {
        upsert(
            db,
            "issue_todo",
            &format!("{}:{}", todo.jira_key, todo.id),
            &format!("{} TODO", todo.jira_key),
            &format!("{}\n{}", todo.content, todo.due_at),
            &todo.updated_at,
        )?;
    }
    for week in query::<WeeklyLog>(db, "SELECT * FROM weekly_logs", [])? {
        upsert(
            db,
            "weekly_log",
            &week.week,
            &format!("Weekly Log {}", week.week),
            &format!("{}\n{}", week.summary_md, week.next_plan_md),
            &week.updated_at,
        )?;
    }
    for key in [
        "issue_tag_sources_version",
        "tag_backfill_version",
        "search_index_backfill_version",
    ] {
        db.execute("INSERT INTO app_schema_meta(key, value) VALUES (?, '1') ON CONFLICT(key) DO UPDATE SET value = excluded.value", [key])?;
    }
    Ok(())
}
