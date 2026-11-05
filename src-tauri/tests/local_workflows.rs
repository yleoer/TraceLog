use chrono_tz::Asia::Shanghai;
use tracelog::{
    config::Config,
    database,
    models::*,
    service::{export, local, views},
    store,
};

fn issue(key: &str) -> Issue {
    Issue {
        jira_key: key.into(),
        title: "迁移验证".into(),
        jira_tags: vec!["remote".into()],
        custom_tags: vec!["personal".into()],
        ..Default::default()
    }
}

#[test]
fn first_week_includes_completion_dates_and_converts_to_the_configured_timezone() {
    let root = tempfile::tempdir().unwrap();
    let db = database::open(&Config::isolated(root.path().into())).unwrap();
    local::save_issue(&db, None, issue("TL-1001"), false).unwrap();
    db.execute("UPDATE issues SET completed_at='2026-06-07T16:30:00Z'", [])
        .unwrap();
    assert_eq!(
        views::week_bounds(&db, Shanghai).unwrap().first_week,
        "2026-W24"
    );
    assert_eq!(
        views::week_bounds(&db, chrono_tz::UTC).unwrap().first_week,
        "2026-W23"
    );

    let task = local::save_task(
        &db,
        0,
        TempTask {
            title: "历史完成的需求".into(),
            completed_at: "2026-06-01T00:00:00+08:00".into(),
            ..Default::default()
        },
    )
    .unwrap();
    assert_eq!(
        views::week_bounds(&db, Shanghai).unwrap().first_week,
        "2026-W23"
    );
    local::delete_task(&db, task.id).unwrap();
    local::create_day(
        &db,
        DayEntry {
            date: "2026-05-31".into(),
            content_md: "日期记录使用本地日历".into(),
            ..Default::default()
        },
    )
    .unwrap();
    assert_eq!(
        views::week_bounds(&db, Shanghai).unwrap().first_week,
        "2026-W22"
    );
}

#[test]
fn weekly_activity_uses_instants_for_fractional_seconds_and_timezone_offsets() {
    let root = tempfile::tempdir().unwrap();
    let db = database::open(&Config::isolated(root.path().into())).unwrap();
    for key in ["TL-1001", "TL-1002"] {
        local::save_issue(&db, None, issue(key), false).unwrap();
    }
    for (key, at) in [
        ("TL-1001", "2026-05-31T16:00:00.000Z"),
        ("TL-1001", "2026-06-01T00:00:00+08:00"),
        ("TL-1002", "2026-06-07T16:00:00.000Z"),
    ] {
        local::save_issue_event(
            &db,
            Some(key),
            0,
            IssueEvent {
                event_type: "note".into(),
                content_md: at.into(),
                happened_at: at.into(),
                ..Default::default()
            },
        )
        .unwrap();
        local::save_todo(
            &db,
            Some(key),
            0,
            IssueTodo {
                content: at.into(),
                due_at: at.into(),
                ..Default::default()
            },
        )
        .unwrap();
        local::save_task(
            &db,
            0,
            TempTask {
                title: at.into(),
                started_at: at.into(),
                ..Default::default()
            },
        )
        .unwrap();
    }
    let view = views::week(&db, "2026-W23", Shanghai).unwrap();
    assert_eq!(view.issues.len(), 1);
    assert_eq!(view.issues[0].jira_key, "TL-1001");
    assert_eq!(view.events.len(), 2);
    assert_eq!(view.todos.len(), 2);
    assert_eq!(view.temp_tasks.len(), 2);
    assert_eq!(view.days[0].comments.len(), 2);
    assert!(view.days[1..].iter().all(|day| day.comments.is_empty()));
}

#[test]
fn issue_edits_preserve_remote_tags_periods_and_allow_clearing_custom_tags() {
    let root = tempfile::tempdir().unwrap();
    let db = database::open(&Config::isolated(root.path().into())).unwrap();
    let mut input = issue("tl-1001");
    input.work_periods = vec![IssueWorkPeriod {
        started_at: "2026-06-01T00:00:00Z".into(),
        ..Default::default()
    }];
    let created = local::save_issue(&db, None, input, false).unwrap();
    assert_eq!(created.jira_key, "TL-1001");
    let mut edited = created.clone();
    edited.title = "保存后的标题".into();
    edited.jira_tags = vec!["forged".into()];
    edited.custom_tags.clear();
    edited.work_periods.clear();
    let updated = local::save_issue(&db, Some("tl-1001"), edited, false).unwrap();
    assert_eq!(updated.id, created.id);
    assert_eq!(updated.created_at, created.created_at);
    assert_eq!(updated.tags, ["remote"]);
    assert!(updated.custom_tags.is_empty());
    assert_eq!(updated.work_periods.len(), 1);
    assert_eq!(
        views::search(&db, "保存后的标题", "", 50, 0).unwrap().len(),
        1
    );
    assert!(local::custom_tags(&db).unwrap().is_empty());
}

#[test]
fn deletion_cascades_children_and_search_but_keeps_activity_history() {
    let root = tempfile::tempdir().unwrap();
    let db = database::open(&Config::isolated(root.path().into())).unwrap();
    local::save_issue(&db, None, issue("TL-1001"), false).unwrap();
    local::save_issue_event(
        &db,
        Some("TL-1001"),
        0,
        IssueEvent {
            event_type: "note".into(),
            content_md: "commentneedle".into(),
            ..Default::default()
        },
    )
    .unwrap();
    local::save_todo(
        &db,
        Some("TL-1001"),
        0,
        IssueTodo {
            content: "todoneedle".into(),
            ..Default::default()
        },
    )
    .unwrap();
    assert_eq!(
        views::search(&db, "commentneedle", "", 50, 0).unwrap()[0].url,
        "/issues/TL-1001"
    );
    local::delete_issue(&db, "TL-1001").unwrap();
    for table in [
        "issues",
        "issue_events",
        "issue_todos",
        "issue_tags",
        "search_index",
        "search_index_refs",
    ] {
        assert_eq!(
            db.query_row::<i64, _, _>(&format!("SELECT count(*) FROM {table}"), [], |r| r.get(0))
                .unwrap(),
            0
        );
    }
    assert_eq!(
        db.query_row::<i64, _, _>(
            "SELECT count(*) FROM activity_events WHERE event_type='deleted'",
            [],
            |r| r.get(0)
        )
        .unwrap(),
        1
    );
}

#[test]
fn week_uses_work_activity_and_buckets_comments_in_configured_timezone() {
    let root = tempfile::tempdir().unwrap();
    let db = database::open(&Config::isolated(root.path().into())).unwrap();
    local::save_issue(&db, None, issue("TL-1001"), false).unwrap();
    local::save_issue(&db, None, issue("TL-1002"), false).unwrap();
    db.execute(
        "UPDATE issues SET created_at='2026-06-03T00:00:00Z',updated_at='2026-06-03T00:00:00Z'",
        [],
    )
    .unwrap();
    local::save_issue_event(
        &db,
        Some("TL-1002"),
        0,
        IssueEvent {
            event_type: "note".into(),
            content_md: "工作进展".into(),
            happened_at: "2026-06-01T16:00:00Z".into(),
            ..Default::default()
        },
    )
    .unwrap();
    local::create_day(
        &db,
        DayEntry {
            date: "2026-06-02".into(),
            content_md: "每日记录".into(),
            ..Default::default()
        },
    )
    .unwrap();
    let view = views::week(&db, "2026-W23", Shanghai).unwrap();
    assert_eq!(view.issues.len(), 1);
    assert_eq!(view.issues[0].jira_key, "TL-1002");
    assert_eq!(view.days.len(), 7);
    assert_eq!(view.days[1].date, "2026-06-02");
    assert!(view.days[1]
        .comments
        .iter()
        .any(|c| c.content_md == "工作进展"));
    assert_eq!(view.days[1].entries.len(), 1);
    assert!(views::week_start("2020-W53", Shanghai).is_ok());
    assert!(views::week_start("2021-W53", Shanghai).is_err());
    assert!(views::week_start("ééé-W01", Shanghai).is_err());
}

#[test]
fn gantt_keeps_multiple_periods_and_excludes_exact_month_boundary() {
    let root = tempfile::tempdir().unwrap();
    let db = database::open(&Config::isolated(root.path().into())).unwrap();
    let mut input = issue("TL-1001");
    input.work_periods = vec![
        IssueWorkPeriod {
            started_at: "2026-06-01T00:00:00Z".into(),
            completed_at: "2026-06-07T00:00:00Z".into(),
            ..Default::default()
        },
        IssueWorkPeriod {
            started_at: "2026-06-10T00:00:00Z".into(),
            ..Default::default()
        },
    ];
    local::save_issue(&db, None, input, false).unwrap();
    let mut ended = issue("TL-1002");
    ended.started_at = "2026-05-01T00:00:00Z".into();
    ended.completed_at = "2026-05-31T16:00:00Z".into();
    local::save_issue(&db, None, ended, false).unwrap();
    let view = views::gantt(&db, "2026-06", Shanghai).unwrap();
    assert_eq!(view.items.len(), 1);
    assert_eq!(view.items[0].periods.len(), 2);
    assert!(views::gantt(&db, "2026-13", Shanghai).is_err());
}

#[test]
fn task_todos_search_and_exports_preserve_user_content_and_escape_snippets() {
    let root = tempfile::tempdir().unwrap();
    let db = database::open(&Config::isolated(root.path().into())).unwrap();
    local::save_issue(&db, None, issue("TL-1001"), false).unwrap();
    let event = local::save_issue_event(
        &db,
        Some("TL-1001"),
        0,
        IssueEvent {
            event_type: "note".into(),
            content_md: "needle <script>alert(1)</script>".into(),
            ..Default::default()
        },
    )
    .unwrap();
    let snippets = views::search(&db, "needle", "", 50, 0).unwrap();
    assert!(snippets[0].snippet.contains("<mark>"));
    assert!(!snippets[0].snippet.contains("<script>"));
    let todo = local::save_todo(
        &db,
        Some("TL-1001"),
        0,
        IssueTodo {
            content: "完成迁移".into(),
            ..Default::default()
        },
    )
    .unwrap();
    assert!(local::save_todo(
        &db,
        None,
        todo.id,
        IssueTodo {
            content: "bad".into(),
            due_at: "not-a-date".into(),
            ..todo.clone()
        }
    )
    .is_err());
    let task = local::save_task(
        &db,
        0,
        TempTask {
            title: "临时验证".into(),
            tags: vec!["rust".into()],
            content_md: "![image](/uploads/test.png)".into(),
            ..Default::default()
        },
    )
    .unwrap();
    local::save_task_event(
        &db,
        Some(task.id),
        0,
        TempTaskEvent {
            event_type: "result".into(),
            content_md: "保留评论".into(),
            ..Default::default()
        },
    )
    .unwrap();
    local::create_day(
        &db,
        DayEntry {
            date: "2026-06-02".into(),
            content_md: "保留日记录".into(),
            ..Default::default()
        },
    )
    .unwrap();
    std::fs::write(root.path().join("uploads/test.png"), [1, 2, 3]).unwrap();
    let payload: serde_json::Value = serde_json::from_slice(&export::json(&db).unwrap()).unwrap();
    assert_eq!(payload["day_entries"].as_array().unwrap().len(), 1);
    assert_eq!(payload["temp_task_events"].as_array().unwrap().len(), 1);
    let zip = export::archive(&db, Shanghai, &root.path().join("uploads")).unwrap();
    let mut archive = zip::ZipArchive::new(std::io::Cursor::new(zip)).unwrap();
    assert!(archive.by_name("uploads/test.png").is_ok());
    assert!(archive.by_name("issues/TL-1001.md").is_ok());
    let mut markdown = String::new();
    std::io::Read::read_to_string(
        &mut archive
            .by_name(&format!("temp-tasks/{}.md", task.id))
            .unwrap(),
        &mut markdown,
    )
    .unwrap();
    assert!(markdown.contains("](../uploads/test.png)"));
    assert!(String::from_utf8(export::issue(&db, "TL-1001").unwrap().1)
        .unwrap()
        .contains("needle <script>"));
    local::delete_event(&db, true, event.id).unwrap();
    assert!(views::search(&db, "needle", "", 50, 0).unwrap().is_empty());
    let filtered = local::list_tasks(
        &db,
        ListFilter {
            tag: "rust".into(),
            ..Default::default()
        },
    )
    .unwrap();
    assert_eq!(filtered.len(), 1);
    let _: Issue = store::one(&db, "SELECT * FROM issues WHERE jira_key='TL-1001'", []).unwrap();
}
