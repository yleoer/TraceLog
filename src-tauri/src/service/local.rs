use super::now;
use crate::{
    error::{invalid, Result},
    models::*,
    store::{
        self, index,
        records::{self, Table},
    },
};
use rusqlite::{params, params_from_iter, types::Value as SqlValue, Connection};

pub fn valid_jira_key(value: &str) -> bool {
    static PATTERN: std::sync::OnceLock<regex::Regex> = std::sync::OnceLock::new();
    PATTERN
        .get_or_init(|| regex::Regex::new(r"^[A-Z][A-Z0-9]+-\d+$").expect("Jira key pattern"))
        .is_match(value)
}

pub fn list_issues(db: &Connection, filter: ListFilter) -> Result<Vec<Issue>> {
    let (suffix, args) = filter_sql(&filter, true);
    let mut issues: Vec<Issue> = store::query(
        db,
        &format!("SELECT i.* FROM issues i{suffix}"),
        params_from_iter(args),
    )?;
    for issue in &mut issues {
        issue.work_periods = periods(db, issue.id)?;
    }
    Ok(issues)
}

pub fn list_tasks(db: &Connection, filter: ListFilter) -> Result<Vec<TempTask>> {
    let (suffix, args) = filter_sql(&filter, false);
    store::query(
        db,
        &format!("SELECT t.* FROM temp_tasks t{suffix}"),
        params_from_iter(args),
    )
}

fn filter_sql(filter: &ListFilter, issue: bool) -> (String, Vec<SqlValue>) {
    let alias = if issue { "i" } else { "t" };
    let mut clauses: Vec<String> = Vec::new();
    let mut args = Vec::new();
    if !filter.query.trim().is_empty() {
        clauses.push(if issue { "i.jira_key IN (SELECT entity_id FROM search_index WHERE search_index MATCH ? AND entity_type='issue')".into() } else { "t.id IN (SELECT CAST(entity_id AS INTEGER) FROM search_index WHERE search_index MATCH ? AND entity_type='temp_task')".into() });
        args.push(SqlValue::Text(store::fts_query(&filter.query)));
    }
    if !filter.status.is_empty() {
        clauses.push(if issue {
            "(i.status = ? OR i.background_md LIKE ?)".into()
        } else {
            "t.status = ?".into()
        });
        args.push(SqlValue::Text(filter.status.clone()));
        if issue {
            args.push(SqlValue::Text(format!("%Jira status: {}%", filter.status)));
        }
    }
    if !filter.tag.is_empty() {
        clauses.push(if issue {
            "i.id IN (SELECT issue_id FROM issue_tags WHERE tag = ?)".into()
        } else {
            "t.id IN (SELECT temp_task_id FROM temp_task_tags WHERE tag = ?)".into()
        });
        args.push(SqlValue::Text(filter.tag.clone()));
    }
    let mut suffix = if clauses.is_empty() {
        String::new()
    } else {
        format!(" WHERE {}", clauses.join(" AND "))
    };
    suffix.push_str(&format!(
        " ORDER BY {alias}.{} DESC, {alias}.id DESC",
        if issue { "created_at" } else { "updated_at" }
    ));
    if !filter.all {
        suffix.push_str(" LIMIT ? OFFSET ?");
        args.extend([
            SqlValue::Integer(if filter.limit > 0 {
                filter.limit.min(10000)
            } else {
                50
            }),
            SqlValue::Integer(filter.offset.max(0)),
        ]);
    }
    (suffix, args)
}

pub fn get_issue(db: &Connection, key: &str) -> Result<Issue> {
    let mut issue: Issue = store::one(
        db,
        "SELECT * FROM issues WHERE jira_key = ?",
        [key.trim().to_uppercase()],
    )?;
    issue.work_periods = periods(db, issue.id)?;
    Ok(issue)
}

pub fn periods(db: &Connection, issue_id: i64) -> Result<Vec<IssueWorkPeriod>> {
    store::query(
        db,
        "SELECT * FROM issue_work_periods WHERE issue_id = ? ORDER BY started_at, id",
        [issue_id],
    )
}

pub fn custom_tags(db: &Connection) -> Result<Vec<String>> {
    Ok(db.prepare("SELECT DISTINCT trim(j.value) AS tag FROM issues i, json_each(i.custom_tags) j WHERE j.type = 'text' AND trim(j.value) <> '' ORDER BY tag COLLATE NOCASE, tag")?.query_map([], |r| r.get(0))?.collect::<std::result::Result<_, _>>()?)
}

pub fn save_issue(
    db: &Connection,
    key: Option<&str>,
    mut issue: Issue,
    imported: bool,
) -> Result<Issue> {
    let time = now();
    if let Some(key) = key {
        let existing = get_issue(db, key)?;
        issue.id = existing.id;
        issue.jira_key = existing.jira_key;
        issue.created_at = existing.created_at;
        if !imported {
            issue.jira_tags = existing.jira_tags;
            issue.work_periods = existing.work_periods;
        }
    } else {
        issue.id = 0;
        issue.jira_key = issue.jira_key.trim().to_uppercase();
        issue.created_at = time.clone();
        if !valid_jira_key(&issue.jira_key) {
            return Err(invalid("jira_key must look like GCS-45000"));
        }
    }
    issue.title = issue.title.trim().into();
    issue.status = issue.status.trim().into();
    if issue.status.is_empty() {
        issue.status = "analysis".into();
    }
    if issue.priority.is_empty() {
        issue.priority = "medium".into();
    }
    validate_title_priority(&issue.title, &issue.priority)?;
    issue.jira_tags = store::clean_tags(&issue.jira_tags);
    issue.custom_tags = store::clean_tags(&issue.custom_tags);
    issue.tags = store::clean_tags(&[issue.jira_tags.clone(), issue.custom_tags.clone()].concat());
    issue.updated_at = time.clone();
    let tx = db.unchecked_transaction()?;
    issue.id = records::save(&tx, Table::Issue, issue.id, &issue)?;
    replace_tags(&tx, true, issue.id, &issue.tags)?;
    if key.is_none() || imported {
        tx.execute(
            "DELETE FROM issue_work_periods WHERE issue_id = ?",
            [issue.id],
        )?;
        for period in &issue.work_periods {
            tx.execute("INSERT INTO issue_work_periods(issue_id,started_at,completed_at,reopened_at,reopened_status,created_at,updated_at) VALUES (?,?,?,?,?,?,?)", params![issue.id,period.started_at,period.completed_at,period.reopened_at,period.reopened_status,time,time])?;
        }
    }
    if key.is_none() {
        activity(
            &tx,
            "issue",
            issue.id,
            &issue.jira_key,
            &issue.title,
            "created",
            "",
        )?;
    }
    index::issue(&tx, &issue)?;
    tx.commit()?;
    get_issue(db, &issue.jira_key)
}

pub fn get_task(db: &Connection, id: i64) -> Result<TempTask> {
    store::one(db, "SELECT * FROM temp_tasks WHERE id = ?", [id])
}

pub fn save_task(db: &Connection, id: i64, mut task: TempTask) -> Result<TempTask> {
    let time = now();
    task.id = id;
    task.created_at = if id > 0 {
        get_task(db, id)?.created_at
    } else {
        time.clone()
    };
    task.updated_at = time;
    task.title = task.title.trim().into();
    if task.status.is_empty() {
        task.status = "todo".into();
    }
    if task.priority.is_empty() {
        task.priority = "medium".into();
    }
    if !["todo", "processing", "done", "suspended"].contains(&task.status.as_str()) {
        return Err(invalid("invalid temp task status"));
    }
    validate_title_priority(&task.title, &task.priority)?;
    task.tags = store::clean_tags(&task.tags);
    let tx = db.unchecked_transaction()?;
    task.id = records::save(&tx, Table::Task, id, &task)?;
    replace_tags(&tx, false, task.id, &task.tags)?;
    if id == 0 {
        activity(&tx, "temp_task", task.id, "", &task.title, "created", "")?;
    }
    index::task(&tx, &task)?;
    tx.commit()?;
    Ok(task)
}

fn validate_title_priority(title: &str, priority: &str) -> Result<()> {
    if title.is_empty() {
        return Err(invalid("title is required"));
    }
    if !["low", "medium", "high", "urgent"].contains(&priority) {
        return Err(invalid("invalid priority"));
    }
    Ok(())
}

fn replace_tags(db: &Connection, issue: bool, id: i64, tags: &[String]) -> Result<()> {
    let (table, column) = if issue {
        ("issue_tags", "issue_id")
    } else {
        ("temp_task_tags", "temp_task_id")
    };
    db.execute(&format!("DELETE FROM {table} WHERE {column} = ?"), [id])?;
    for tag in tags {
        db.execute(
            &format!("INSERT INTO {table}({column},tag) VALUES (?,?)"),
            params![id, tag],
        )?;
    }
    Ok(())
}

pub fn activity(
    db: &Connection,
    source: &str,
    id: i64,
    key: &str,
    title: &str,
    kind: &str,
    content: &str,
) -> Result<()> {
    let time = now();
    db.execute("INSERT INTO activity_events(source,ref_id,ref_key,ref_title,event_type,content_md,happened_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)", params![source,id,key,title,kind,content,time,time,time])?;
    Ok(())
}

pub fn delete_issue(db: &Connection, key: &str) -> Result<()> {
    let issue = get_issue(db, key)?;
    let tx = db.unchecked_transaction()?;
    activity(
        &tx,
        "issue",
        issue.id,
        &issue.jira_key,
        &issue.title,
        "deleted",
        "删除 Issue",
    )?;
    for event in list_issue_events(&tx, &issue.jira_key)? {
        index::remove(&tx, "issue_event", &event.id.to_string())?;
    }
    for todo in list_todos(&tx, &issue.jira_key, true)? {
        index::remove(
            &tx,
            "issue_todo",
            &format!("{}:{}", issue.jira_key, todo.id),
        )?;
    }
    index::remove(&tx, "issue", &issue.jira_key)?;
    tx.execute("DELETE FROM issues WHERE id = ?", [issue.id])?;
    tx.commit()?;
    Ok(())
}

pub fn delete_task(db: &Connection, id: i64) -> Result<()> {
    let task = get_task(db, id)?;
    let tx = db.unchecked_transaction()?;
    activity(
        &tx,
        "temp_task",
        id,
        "",
        &task.title,
        "deleted",
        "删除临时需求",
    )?;
    for event in list_task_events(&tx, id)? {
        index::remove(&tx, "temp_task_event", &event.id.to_string())?;
    }
    index::remove(&tx, "temp_task", &id.to_string())?;
    tx.execute("DELETE FROM temp_tasks WHERE id = ?", [id])?;
    tx.commit()?;
    Ok(())
}

pub fn list_issue_events(db: &Connection, key: &str) -> Result<Vec<IssueEvent>> {
    let issue = get_issue(db, key)?;
    store::query(
        db,
        "SELECT * FROM issue_events WHERE issue_id = ? ORDER BY happened_at DESC, created_at DESC",
        [issue.id],
    )
}

pub fn list_task_events(db: &Connection, id: i64) -> Result<Vec<TempTaskEvent>> {
    get_task(db, id)?;
    store::query(db,"SELECT * FROM temp_task_events WHERE temp_task_id = ? ORDER BY happened_at DESC, created_at DESC",[id])
}

pub fn save_issue_event(
    db: &Connection,
    key: Option<&str>,
    id: i64,
    mut event: IssueEvent,
) -> Result<IssueEvent> {
    let time = now();
    if let Some(key) = key {
        event.issue_id = get_issue(db, key)?.id;
        event.created_at = time.clone();
    } else {
        let existing: IssueEvent = store::one(db, "SELECT * FROM issue_events WHERE id=?", [id])?;
        event.issue_id = existing.issue_id;
        event.created_at = existing.created_at;
        if event.happened_at.is_empty() {
            event.happened_at = existing.happened_at;
        }
    }
    event.updated_at = time.clone();
    if event.happened_at.is_empty() {
        event.happened_at = time;
    }
    normalize_event(
        &mut event.event_type,
        &mut event.content_md,
        &event.happened_at,
    )?;
    let tx = db.unchecked_transaction()?;
    event.id = records::save(&tx, Table::IssueEvent, id, &event)?;
    let key: String = tx.query_row(
        "SELECT jira_key FROM issues WHERE id=?",
        [event.issue_id],
        |r| r.get(0),
    )?;
    index::upsert(
        &tx,
        "issue_event",
        &event.id.to_string(),
        &format!("{key} {}", event.event_type),
        &event.content_md,
        &event.updated_at,
    )?;
    tx.commit()?;
    Ok(event)
}

pub fn save_task_event(
    db: &Connection,
    parent: Option<i64>,
    id: i64,
    mut event: TempTaskEvent,
) -> Result<TempTaskEvent> {
    let time = now();
    if let Some(parent) = parent {
        get_task(db, parent)?;
        event.temp_task_id = parent;
        event.created_at = time.clone();
    } else {
        let existing: TempTaskEvent =
            store::one(db, "SELECT * FROM temp_task_events WHERE id=?", [id])?;
        event.temp_task_id = existing.temp_task_id;
        event.created_at = existing.created_at;
        if event.happened_at.is_empty() {
            event.happened_at = existing.happened_at;
        }
    }
    event.updated_at = time.clone();
    if event.happened_at.is_empty() {
        event.happened_at = time;
    }
    normalize_event(
        &mut event.event_type,
        &mut event.content_md,
        &event.happened_at,
    )?;
    let tx = db.unchecked_transaction()?;
    event.id = records::save(&tx, Table::TaskEvent, id, &event)?;
    let title: String = tx.query_row(
        "SELECT title FROM temp_tasks WHERE id=?",
        [event.temp_task_id],
        |r| r.get(0),
    )?;
    index::upsert(
        &tx,
        "temp_task_event",
        &event.id.to_string(),
        &format!("{title} {}", event.event_type),
        &event.content_md,
        &event.updated_at,
    )?;
    tx.commit()?;
    Ok(event)
}

fn normalize_event(kind: &mut String, content: &mut String, happened: &str) -> Result<()> {
    *kind = kind.trim().into();
    *content = content.trim().into();
    if content.is_empty() {
        return Err(invalid("content_md is required"));
    }
    if ![
        "note", "analysis", "action", "decision", "blocker", "result",
    ]
    .contains(&kind.as_str())
    {
        return Err(invalid("invalid event_type"));
    }
    validate_timestamp(happened)
}

pub fn delete_event(db: &Connection, issue: bool, id: i64) -> Result<()> {
    let (table, kind) = if issue {
        ("issue_events", "issue_event")
    } else {
        ("temp_task_events", "temp_task_event")
    };
    let tx = db.unchecked_transaction()?;
    if tx.execute(&format!("DELETE FROM {table} WHERE id=?"), [id])? == 0 {
        return Err(crate::error::AppError::NotFound);
    }
    index::remove(&tx, kind, &id.to_string())?;
    tx.commit()?;
    Ok(())
}

pub fn list_todos(db: &Connection, key: &str, include_done: bool) -> Result<Vec<IssueTodo>> {
    let issue = get_issue(db, key)?;
    store::query(db,"SELECT t.*, i.jira_key FROM issue_todos t JOIN issues i ON i.id=t.issue_id WHERE t.issue_id=? AND (? OR t.done=0) ORDER BY t.done, CASE WHEN t.due_at='' THEN 1 ELSE 0 END,t.due_at,t.created_at",params![issue.id,include_done])
}

pub fn get_todo(db: &Connection, id: i64) -> Result<IssueTodo> {
    store::one(
        db,
        "SELECT t.*,i.jira_key FROM issue_todos t JOIN issues i ON i.id=t.issue_id WHERE t.id=?",
        [id],
    )
}

pub fn save_todo(
    db: &Connection,
    key: Option<&str>,
    id: i64,
    mut todo: IssueTodo,
) -> Result<IssueTodo> {
    let time = now();
    if let Some(key) = key {
        let issue = get_issue(db, key)?;
        todo.issue_id = issue.id;
        todo.jira_key = issue.jira_key;
        todo.created_at = time.clone();
    } else {
        let existing = get_todo(db, id)?;
        todo.issue_id = existing.issue_id;
        todo.jira_key = existing.jira_key;
        todo.created_at = existing.created_at;
    }
    todo.content = todo.content.trim().into();
    todo.due_at = todo.due_at.trim().into();
    if todo.content.is_empty() {
        return Err(invalid("content is required"));
    }
    validate_timestamp(&todo.due_at)?;
    todo.updated_at = time;
    let tx = db.unchecked_transaction()?;
    todo.id = records::save(&tx, Table::Todo, id, &todo)?;
    index::upsert(
        &tx,
        "issue_todo",
        &format!("{}:{}", todo.jira_key, todo.id),
        &format!("{} TODO", todo.jira_key),
        &format!("{}\n{}", todo.content, todo.due_at),
        &todo.updated_at,
    )?;
    tx.commit()?;
    Ok(todo)
}

pub fn delete_todo(db: &Connection, id: i64) -> Result<()> {
    let todo = get_todo(db, id)?;
    let tx = db.unchecked_transaction()?;
    tx.execute("DELETE FROM issue_todos WHERE id=?", [id])?;
    index::remove(&tx, "issue_todo", &format!("{}:{}", todo.jira_key, id))?;
    tx.commit()?;
    Ok(())
}

pub fn create_day(db: &Connection, mut day: DayEntry) -> Result<DayEntry> {
    day.date = day.date.trim().into();
    day.content_md = day.content_md.trim().into();
    if chrono::NaiveDate::parse_from_str(&day.date, "%Y-%m-%d").is_err() || day.date.len() != 10 {
        return Err(invalid("date must use YYYY-MM-DD format"));
    }
    if day.content_md.is_empty() {
        return Err(invalid("content_md is required"));
    }
    day.created_at = now();
    day.updated_at = day.created_at.clone();
    day.id = records::save(db, Table::Day, 0, &day)?;
    Ok(day)
}

pub fn delete_day(db: &Connection, id: i64) -> Result<()> {
    if db.execute("DELETE FROM day_entries WHERE id=?", [id])? == 0 {
        return Err(crate::error::AppError::NotFound);
    }
    Ok(())
}

pub fn validate_timestamp(value: &str) -> Result<()> {
    if !value.is_empty() && chrono::DateTime::parse_from_rfc3339(value).is_err() {
        return Err(invalid("time must be RFC3339"));
    }
    Ok(())
}
