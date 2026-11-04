use super::{local, markdown, now};
use crate::{
    error::{invalid, AppError, Result},
    models::*,
    store::{self, index},
};
use chrono::{DateTime, Datelike, Duration, NaiveDate, TimeZone, Utc, Weekday};
use chrono_tz::Tz;
use rusqlite::{params, Connection};

pub fn current_week(timezone: Tz) -> String {
    Utc::now()
        .with_timezone(&timezone)
        .format("%G-W%V")
        .to_string()
}

pub fn week_start(week: &str, timezone: Tz) -> Result<NaiveDate> {
    let week = if week.is_empty() {
        current_week(timezone)
    } else {
        week.into()
    };
    if week.len() != 8 || !week.is_ascii() || &week[4..6] != "-W" {
        return Err(invalid("week must use YYYY-Www format"));
    }
    let year = week[..4].parse().map_err(|_| invalid("invalid ISO week"))?;
    let number = week[6..].parse().map_err(|_| invalid("invalid ISO week"))?;
    NaiveDate::from_isoywd_opt(year, number, Weekday::Mon)
        .ok_or_else(|| invalid("invalid ISO week"))
}

fn utc_start(date: NaiveDate, timezone: Tz) -> Result<String> {
    Ok(timezone
        .from_local_datetime(
            &date
                .and_hms_opt(0, 0, 0)
                .ok_or_else(|| invalid("invalid date"))?,
        )
        .earliest()
        .ok_or_else(|| invalid("invalid local date"))?
        .with_timezone(&Utc)
        .to_rfc3339_opts(chrono::SecondsFormat::Secs, true))
}

pub fn list_weeks(db: &Connection) -> Result<Vec<WeeklyLog>> {
    store::query(db, "SELECT * FROM weekly_logs ORDER BY week DESC", [])
}

pub fn week_bounds(db: &Connection, timezone: Tz) -> Result<WeekBounds> {
    let current = current_week(timezone);
    let first: Option<String> = db.query_row("SELECT min(value) FROM (SELECT min(created_at) AS value FROM issues UNION ALL SELECT min(started_at) FROM issues WHERE started_at<>'' UNION ALL SELECT min(happened_at) FROM issue_events UNION ALL SELECT min(due_at) FROM issue_todos WHERE due_at<>'' UNION ALL SELECT min(updated_at) FROM issue_todos WHERE done=1 UNION ALL SELECT min(created_at) FROM temp_tasks UNION ALL SELECT min(started_at) FROM temp_tasks WHERE started_at<>'' UNION ALL SELECT min(happened_at) FROM temp_task_events UNION ALL SELECT min(happened_at) FROM activity_events UNION ALL SELECT min(date) FROM day_entries)",[],|r| r.get(0))?;
    let mut first_week = first
        .and_then(|v| NaiveDate::parse_from_str(v.get(..10)?, "%Y-%m-%d").ok())
        .map(|d| d.format("%G-W%V").to_string())
        .unwrap_or_else(|| current.clone());
    for log in list_weeks(db)? {
        if week_start(&log.week, timezone).is_ok() && log.week < first_week {
            first_week = log.week;
        }
    }
    if first_week > current {
        first_week = current.clone();
    }
    Ok(WeekBounds {
        first_week,
        current_week: current,
    })
}

pub fn week(db: &Connection, requested: &str, timezone: Tz) -> Result<WeekView> {
    let start = week_start(requested, timezone)?;
    let week = start.format("%G-W%V").to_string();
    let from = utc_start(start, timezone)?;
    let to = utc_start(start + Duration::days(7), timezone)?;
    let log = match store::one(db, "SELECT * FROM weekly_logs WHERE week=?", [&week]) {
        Ok(log) => log,
        Err(AppError::NotFound) => WeeklyLog {
            week,
            created_at: now(),
            updated_at: now(),
            ..Default::default()
        },
        Err(error) => return Err(error),
    };
    let issues = issues_between(db, &from, &to, false)?;
    let events=store::query(db,"SELECT * FROM issue_events WHERE happened_at>=? AND happened_at<? ORDER BY happened_at DESC",params![from,to])?;
    let temp_tasks = tasks_between(db, &from, &to)?;
    let todos = todos_between(db, &from, &to)?;
    let (done, active) = classify(&issues, &temp_tasks, &todos);
    Ok(WeekView {
        log,
        issues,
        events,
        temp_tasks,
        todos,
        done,
        active,
        days: days(db, start, 7, timezone)?,
    })
}

pub fn save_week(
    db: &Connection,
    week: &str,
    mut log: WeeklyLog,
    timezone: Tz,
) -> Result<WeeklyLog> {
    week_start(week, timezone)?;
    log.week = week.into();
    log.updated_at = now();
    log.created_at =
        match store::one::<WeeklyLog>(db, "SELECT * FROM weekly_logs WHERE week=?", [week]) {
            Ok(existing) => existing.created_at,
            Err(AppError::NotFound) => log.updated_at.clone(),
            Err(error) => return Err(error),
        };
    let tx = db.unchecked_transaction()?;
    tx.execute("INSERT INTO weekly_logs(week,summary_md,next_plan_md,created_at,updated_at) VALUES (?,?,?,?,?) ON CONFLICT(week) DO UPDATE SET summary_md=excluded.summary_md,next_plan_md=excluded.next_plan_md,updated_at=excluded.updated_at",params![log.week,log.summary_md,log.next_plan_md,log.created_at,log.updated_at])?;
    index::upsert(
        &tx,
        "weekly_log",
        week,
        &format!("Weekly Log {week}"),
        &format!("{}\n{}", log.summary_md, log.next_plan_md),
        &log.updated_at,
    )?;
    tx.commit()?;
    store::one(db, "SELECT * FROM weekly_logs WHERE week=?", [week])
}

pub fn draft_week(db: &Connection, requested: &str, timezone: Tz) -> Result<WeeklyLog> {
    let view = week(db, requested, timezone)?;
    let mut log = view.log.clone();
    log.summary_md = markdown::workflow(&view);
    save_week(db, &view.log.week, log, timezone)
}

pub fn dashboard(db: &Connection, timezone: Tz) -> Result<Dashboard> {
    let issues = local::list_issues(
        db,
        ListFilter {
            limit: 50,
            ..Default::default()
        },
    )?;
    Ok(Dashboard {
        recently_imported_issues:issues.iter().take(8).cloned().collect(),
        active_issues:issues.into_iter().filter(|i| dashboard_active(&i.status)).take(8).collect(),
        temp_tasks:local::list_tasks(db,ListFilter{status:"todo".into(),limit:8,..Default::default()})?,
        todos:store::query(db,"SELECT t.*,i.jira_key FROM issue_todos t JOIN issues i ON i.id=t.issue_id WHERE t.done=0 ORDER BY CASE WHEN t.due_at='' THEN 1 ELSE 0 END,t.due_at,t.updated_at DESC LIMIT 8",[])?,
        week:week(db,"",timezone)?,
    })
}

pub fn today(db: &Connection, timezone: Tz) -> Result<TodayWorkflow> {
    let start = Utc::now().with_timezone(&timezone).date_naive();
    let from = utc_start(start, timezone)?;
    let to = utc_start(start + Duration::days(1), timezone)?;
    let issues = issues_between(db, &from, &to, true)?;
    let temp_tasks = tasks_between(db, &from, &to)?;
    let todos = todos_between(db, &from, &to)?;
    let events=store::query(db,"SELECT * FROM issue_events WHERE happened_at>=? AND happened_at<? ORDER BY happened_at DESC",params![from,to])?;
    let (done, active) = classify(&issues, &temp_tasks, &todos);
    let weekly_draft = markdown::workflow(&WeekView {
        issues: issues.clone(),
        temp_tasks: temp_tasks.clone(),
        todos: todos.clone(),
        events,
        ..Default::default()
    });
    Ok(TodayWorkflow {
        date: start.to_string(),
        issues,
        temp_tasks,
        todos,
        done,
        active,
        weekly_draft,
        day: days(db, start, 1, timezone)?.remove(0),
    })
}

fn issues_between(
    db: &Connection,
    start: &str,
    end: &str,
    include_import: bool,
) -> Result<Vec<Issue>> {
    let mut issues:Vec<Issue>=store::query(db,"SELECT i.* FROM issues i WHERE EXISTS(SELECT 1 FROM issue_events e WHERE e.issue_id=i.id AND e.happened_at>=?1 AND e.happened_at<?2) OR (i.started_at>=?1 AND i.started_at<?2) OR (i.completed_at>=?1 AND i.completed_at<?2) OR (?3 AND ((i.created_at>=?1 AND i.created_at<?2) OR (i.updated_at>=?1 AND i.updated_at<?2))) ORDER BY i.updated_at DESC",params![start,end,include_import])?;
    for issue in &mut issues {
        issue.work_periods = local::periods(db, issue.id)?;
    }
    Ok(issues)
}

fn tasks_between(db: &Connection, start: &str, end: &str) -> Result<Vec<TempTask>> {
    store::query(db,"SELECT * FROM temp_tasks WHERE (created_at>=?1 AND created_at<?2) OR (updated_at>=?1 AND updated_at<?2) OR (started_at>=?1 AND started_at<?2) OR (completed_at>=?1 AND completed_at<?2) ORDER BY updated_at DESC",params![start,end])
}

fn todos_between(db: &Connection, start: &str, end: &str) -> Result<Vec<IssueTodo>> {
    store::query(db,"SELECT t.*,i.jira_key FROM issue_todos t JOIN issues i ON i.id=t.issue_id WHERE (t.due_at>=?1 AND t.due_at<?2) OR (t.done=1 AND t.updated_at>=?1 AND t.updated_at<?2) ORDER BY t.done,CASE WHEN t.due_at='' THEN 1 ELSE 0 END,t.due_at,t.updated_at DESC",params![start,end])
}

pub fn is_active(status: &str) -> bool {
    let s = status.trim().to_lowercase();
    s == "processing"
        || s.contains("progress")
        || s.contains("indeterminate")
        || s.contains("处理中")
}
pub fn is_done(status: &str) -> bool {
    let s = status.trim().to_lowercase();
    s == "done"
        || s == "closed"
        || s.contains("resolved")
        || s.contains("完成")
        || s.contains("关闭")
}
fn dashboard_active(status: &str) -> bool {
    let s = status.trim().to_lowercase();
    is_active(&s) || ["in review", "todo", "to do"].contains(&s.as_str()) || s.contains("待办")
}

fn classify(
    issues: &[Issue],
    tasks: &[TempTask],
    todos: &[IssueTodo],
) -> (Vec<String>, Vec<String>) {
    let mut done = Vec::new();
    let mut active = Vec::new();
    for (label, completed) in issues
        .iter()
        .map(|i| (format!("{} {}", i.jira_key, i.title), is_done(&i.status)))
        .chain(tasks.iter().map(|t| (t.title.clone(), t.status == "done")))
        .chain(
            todos
                .iter()
                .map(|t| (format!("{} TODO: {}", t.jira_key, t.content), t.done)),
        )
    {
        if completed {
            done.push(label);
        } else {
            active.push(label);
        }
    }
    (done, active)
}

pub fn days(db: &Connection, start: NaiveDate, count: i64, timezone: Tz) -> Result<Vec<DayWork>> {
    let from = utc_start(start, timezone)?;
    let to = utc_start(start + Duration::days(count), timezone)?;
    let sql="SELECT * FROM (
      SELECT 'issue' AS source,e.id AS event_id,e.event_type,e.content_md,e.happened_at,i.id AS ref_id,i.jira_key AS ref_key,i.title AS ref_title FROM issue_events e JOIN issues i ON i.id=e.issue_id
      UNION ALL SELECT 'temp_task',e.id,e.event_type,e.content_md,e.happened_at,t.id,'',t.title FROM temp_task_events e JOIN temp_tasks t ON t.id=e.temp_task_id
      UNION ALL SELECT source,id,event_type,content_md,happened_at,ref_id,ref_key,ref_title FROM activity_events
      UNION ALL SELECT 'issue',t.id,'todo_done','完成 TODO：'||t.content,t.updated_at,i.id,i.jira_key,i.title FROM issue_todos t JOIN issues i ON i.id=t.issue_id WHERE t.done=1
      UNION ALL SELECT 'issue',0,'created','',i.created_at,i.id,i.jira_key,i.title FROM issues i WHERE NOT EXISTS(SELECT 1 FROM activity_events a WHERE a.source='issue' AND a.ref_id=i.id AND a.event_type='created')
      UNION ALL SELECT 'temp_task',0,'created','',t.created_at,t.id,'',t.title FROM temp_tasks t WHERE NOT EXISTS(SELECT 1 FROM activity_events a WHERE a.source='temp_task' AND a.ref_id=t.id AND a.event_type='created')
    ) WHERE happened_at>=? AND happened_at<? ORDER BY happened_at";
    let comments: Vec<DayComment> = store::query(db, sql, params![from, to])?;
    let entries: Vec<DayEntry> = store::query(
        db,
        "SELECT * FROM day_entries WHERE date>=? AND date<? ORDER BY created_at",
        params![
            start.to_string(),
            (start + Duration::days(count)).to_string()
        ],
    )?;
    let mut result: Vec<DayWork> = (0..count)
        .map(|i| {
            let date = start + Duration::days(i);
            DayWork {
                date: date.to_string(),
                weekday: weekday(date).into(),
                ..Default::default()
            }
        })
        .collect();
    for mut comment in comments {
        let Ok(timestamp) = DateTime::parse_from_rfc3339(&comment.happened_at) else {
            continue;
        };
        let index = (timestamp.with_timezone(&timezone).date_naive() - start).num_days();
        if index < 0 || index >= count {
            continue;
        }
        comment.url = if comment.source == "temp_task" {
            format!("/temp-tasks/{}", comment.ref_id)
        } else {
            format!("/issues/{}", comment.ref_key)
        };
        let day = &mut result[index as usize];
        let group = day.activities.iter().position(|g| {
            g.source == comment.source && g.ref_id == comment.ref_id && g.ref_key == comment.ref_key
        });
        let group = match group {
            Some(i) => i,
            None => {
                day.activities.push(DayActivity {
                    source: comment.source.clone(),
                    ref_id: comment.ref_id,
                    ref_key: comment.ref_key.clone(),
                    ref_title: comment.ref_title.clone(),
                    url: comment.url.clone(),
                    started_at: comment.happened_at.clone(),
                    comments: Vec::new(),
                });
                day.activities.len() - 1
            }
        };
        day.activities[group].comments.push(comment.clone());
        day.comments.push(comment);
    }
    for entry in entries {
        if let Some(day) = result.iter_mut().find(|d| d.date == entry.date) {
            day.entries.push(entry);
        }
    }
    Ok(result)
}

pub fn weekday(date: NaiveDate) -> &'static str {
    ["周一", "周二", "周三", "周四", "周五", "周六", "周日"]
        [date.weekday().num_days_from_monday() as usize]
}

pub fn gantt(db: &Connection, month: &str, timezone: Tz) -> Result<MonthGanttView> {
    let month = if month.trim().is_empty() {
        Utc::now()
            .with_timezone(&timezone)
            .format("%Y-%m")
            .to_string()
    } else {
        month.trim().into()
    };
    if month.len() != 7 {
        return Err(invalid("month must use YYYY-MM format"));
    }
    let start = NaiveDate::parse_from_str(&format!("{month}-01"), "%Y-%m-%d")
        .map_err(|_| invalid("month must use YYYY-MM format"))?;
    let next = if start.month() == 12 {
        NaiveDate::from_ymd_opt(start.year() + 1, 1, 1)
    } else {
        NaiveDate::from_ymd_opt(start.year(), start.month() + 1, 1)
    }
    .ok_or_else(|| invalid("invalid month"))?;
    let from = DateTime::parse_from_rfc3339(&utc_start(start, timezone)?)
        .map_err(|_| invalid("invalid month"))?;
    let to = DateTime::parse_from_rfc3339(&utc_start(next, timezone)?)
        .map_err(|_| invalid("invalid month"))?;
    let intersects = |p: &MonthGanttPeriod| {
        DateTime::parse_from_rfc3339(&p.started_at).is_ok_and(|s| s < to)
            && (p.completed_at.is_empty()
                || DateTime::parse_from_rfc3339(&p.completed_at).is_ok_and(|e| e > from))
    };
    let mut items = Vec::new();
    for issue in local::list_issues(
        db,
        ListFilter {
            all: true,
            ..Default::default()
        },
    )? {
        let periods = if issue.work_periods.is_empty() {
            vec![MonthGanttPeriod {
                started_at: issue.started_at,
                completed_at: issue.completed_at,
            }]
        } else {
            issue
                .work_periods
                .into_iter()
                .map(|p| MonthGanttPeriod {
                    started_at: p.started_at,
                    completed_at: p.completed_at,
                })
                .collect()
        };
        let periods: Vec<_> = periods.into_iter().filter(&intersects).collect();
        if !periods.is_empty() {
            items.push(MonthGanttItem {
                id: issue.id,
                source: "issue".into(),
                jira_key: issue.jira_key,
                title: issue.title,
                status: issue.status,
                periods,
            });
        }
    }
    for task in local::list_tasks(
        db,
        ListFilter {
            all: true,
            ..Default::default()
        },
    )? {
        let period = MonthGanttPeriod {
            started_at: task.started_at,
            completed_at: task.completed_at,
        };
        if intersects(&period) {
            items.push(MonthGanttItem {
                id: task.id,
                source: "temp_task".into(),
                title: task.title,
                status: task.status,
                periods: vec![period],
                ..Default::default()
            });
        }
    }
    items.sort_by(|a, b| {
        a.periods[0]
            .started_at
            .cmp(&b.periods[0].started_at)
            .then(a.source.cmp(&b.source))
            .then(a.title.to_lowercase().cmp(&b.title.to_lowercase()))
    });
    Ok(MonthGanttView { month, items })
}

pub fn search(
    db: &Connection,
    query: &str,
    kind: &str,
    limit: i64,
    offset: i64,
) -> Result<Vec<SearchResult>> {
    if query.trim().is_empty() {
        return Ok(Vec::new());
    }
    let sql="SELECT entity_type AS type,entity_id AS id,title,snippet(search_index,3,'[[[HL]]]','[[[/HL]]]','...',12) AS snippet,updated_at,
    CASE entity_type WHEN 'issue' THEN '/issues/'||entity_id WHEN 'issue_event' THEN coalesce((SELECT '/issues/'||i.jira_key FROM issue_events e JOIN issues i ON i.id=e.issue_id WHERE CAST(e.id AS TEXT)=search_index.entity_id),'/issues') WHEN 'issue_todo' THEN '/issues/'||substr(entity_id,1,instr(entity_id,':')-1) WHEN 'temp_task' THEN '/temp-tasks/'||entity_id WHEN 'temp_task_event' THEN coalesce((SELECT '/temp-tasks/'||e.temp_task_id FROM temp_task_events e WHERE CAST(e.id AS TEXT)=search_index.entity_id),'/temp-tasks') WHEN 'weekly_log' THEN '/weeks/'||entity_id ELSE '/search' END AS url
    FROM search_index WHERE search_index MATCH ? AND (?='' OR entity_type=?) ORDER BY rank LIMIT ? OFFSET ?";
    let mut rows: Vec<SearchResult> = store::query(
        db,
        sql,
        params![
            store::fts_query(query),
            kind,
            kind,
            if limit > 0 { limit.min(1000) } else { 50 },
            offset.max(0)
        ],
    )?;
    for row in &mut rows {
        row.snippet = row
            .snippet
            .replace('&', "&amp;")
            .replace('<', "&lt;")
            .replace('>', "&gt;")
            .replace('"', "&quot;")
            .replace('\'', "&#39;")
            .replace("[[[HL]]]", "<mark>")
            .replace("[[[/HL]]]", "</mark>");
    }
    Ok(rows)
}
