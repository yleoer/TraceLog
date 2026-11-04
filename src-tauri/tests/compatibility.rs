use rusqlite::{params, Connection};
use tracelog::{
    config::Config,
    database,
    models::*,
    settings::{self, SettingsStore},
    store,
};

#[test]
fn legacy_database_is_backed_up_migrated_and_reopened_without_data_loss() {
    let directory = tempfile::tempdir().unwrap();
    let config = Config::isolated(directory.path().into());
    let db = Connection::open(&config.database_path).unwrap();
    let schema = include_str!("../../db/migrations/001_init.sql")
        .split("-- +goose Down")
        .next()
        .unwrap();
    db.execute_batch(schema).unwrap();
    db.execute_batch("ALTER TABLE issues DROP COLUMN jira_tags; ALTER TABLE issues DROP COLUMN custom_tags; CREATE TABLE goose_db_version(id INTEGER PRIMARY KEY, version_id INTEGER, is_applied INTEGER, tstamp TEXT); INSERT INTO goose_db_version VALUES(1,1,1,'2026-01-01');").unwrap();
    db.pragma_update(None, "journal_mode", "WAL").unwrap();
    db.execute("INSERT INTO issues(id,jira_key,title,status,priority,tags,created_at,updated_at) VALUES (42,'TL-1001','兼容数据','In Progress','high',?,'2026-11-01T00:00:00Z','2026-11-02T00:00:00Z')", [r#"[" jira ","personal","personal"]"#]).unwrap();
    db.execute("INSERT INTO issue_events(issue_id,event_type,content_md,happened_at,created_at,updated_at) VALUES (42,'note','旧评论','2026-11-02T00:00:00Z','2026-11-02T00:00:00Z','2026-11-02T00:00:00Z')", []).unwrap();
    std::fs::create_dir_all(directory.path().join("uploads")).unwrap();
    std::fs::write(directory.path().join("uploads/image.png"), [1, 2, 3]).unwrap();
    std::fs::write(
        directory.path().join("tracelog-settings.json"),
        r#"{"jira":{"api_token":"keep-secret"}}"#,
    )
    .unwrap();
    let migrated = database::open(&config).unwrap();
    let issue: Issue = store::one(&migrated, "SELECT * FROM issues WHERE id=42", []).unwrap();
    assert_eq!(issue.custom_tags, ["jira", "personal"]);
    assert_eq!(issue.jira_tags, Vec::<String>::new());
    assert_eq!(issue.title, "兼容数据");
    assert_eq!(
        migrated
            .query_row::<i64, _, _>("SELECT count(*) FROM goose_db_version", [], |r| r.get(0))
            .unwrap(),
        1
    );
    assert_eq!(
        migrated
            .query_row::<i64, _, _>(
                "SELECT count(*) FROM search_index WHERE search_index MATCH ?",
                ["旧评论"],
                |r| r.get(0)
            )
            .unwrap(),
        1
    );
    let backups: Vec<_> = std::fs::read_dir(directory.path().join("backups"))
        .unwrap()
        .map(|e| e.unwrap().path())
        .collect();
    assert_eq!(backups.len(), 1);
    let backup = Connection::open(backups[0].join("tracelog.db")).unwrap();
    assert_eq!(
        backup
            .query_row::<String, _, _>("SELECT title FROM issues WHERE id=42", [], |r| r.get(0))
            .unwrap(),
        "兼容数据"
    );
    assert_eq!(
        std::fs::read(backups[0].join("uploads/image.png")).unwrap(),
        [1, 2, 3]
    );
    assert!(backups[0].join("backup.json").exists());
    drop(migrated);
    let reopened = database::open(&config).unwrap();
    assert_eq!(
        reopened
            .query_row::<i64, _, _>("SELECT count(*) FROM issue_events", [], |r| r.get(0))
            .unwrap(),
        1
    );
    assert_eq!(
        std::fs::read_dir(directory.path().join("backups"))
            .unwrap()
            .count(),
        1
    );
    drop(db);
}

#[test]
fn invalid_legacy_data_rolls_back_migration_and_preserves_backup() {
    let directory = tempfile::tempdir().unwrap();
    let config = Config::isolated(directory.path().into());
    let db = Connection::open(&config.database_path).unwrap();
    db.execute_batch(
        include_str!("../../db/migrations/001_init.sql")
            .split("-- +goose Down")
            .next()
            .unwrap(),
    )
    .unwrap();
    db.execute("INSERT INTO issues(jira_key,title,status,priority,tags,created_at,updated_at) VALUES ('TL-1','bad','todo','low','invalid-json','','')", []).unwrap();
    assert!(database::open(&config).is_err());
    assert_eq!(
        db.query_row::<String, _, _>("SELECT tags FROM issues", [], |r| r.get(0))
            .unwrap(),
        "invalid-json"
    );
    assert_eq!(
        db.query_row::<i64, _, _>(
            "SELECT count(*) FROM app_schema_meta WHERE key='rust_compat_version'",
            [],
            |r| r.get(0)
        )
        .unwrap(),
        0
    );
    assert_eq!(
        std::fs::read_dir(directory.path().join("backups"))
            .unwrap()
            .count(),
        1
    );
}

#[test]
fn settings_preserve_masked_credentials_and_replace_existing_file_atomically() {
    let directory = tempfile::tempdir().unwrap();
    let store = SettingsStore::new(directory.path());
    let mut settings = store.load().unwrap();
    settings.jira.api_token = "jira-secret".into();
    settings.tempo.api_token = "tempo-secret".into();
    settings.tempo.author_account_id = "author".into();
    settings.openai.api_key = "ai-secret".into();
    store.save(&settings).unwrap();
    let mut input = settings::public(settings);
    assert!(input.jira.has_api_token);
    assert!(input.jira.api_token.is_empty());
    input.jira.email = "new@example.com".into();
    input.tempo.author_account_id.clear();
    let merged = store.merge_input(input).unwrap();
    store.save(&merged).unwrap();
    let saved = store.load().unwrap();
    assert_eq!(saved.jira.api_token, "jira-secret");
    assert_eq!(saved.tempo.author_account_id, "author");
    assert_eq!(saved.openai.api_key, "ai-secret");
    assert_eq!(saved.jira.email, "new@example.com");
    assert_eq!(std::fs::read_dir(directory.path()).unwrap().count(), 1);
}

#[test]
fn search_index_updates_keep_one_reference_and_delete_both_rows() {
    let directory = tempfile::tempdir().unwrap();
    let db = database::open(&Config::isolated(directory.path().into())).unwrap();
    for body in ["first", "second"] {
        store::index::upsert(&db, "issue", "TL-1", "title", body, "now").unwrap();
    }
    assert_eq!(
        db.query_row::<i64, _, _>("SELECT count(*) FROM search_index", [], |r| r.get(0))
            .unwrap(),
        1
    );
    assert_eq!(
        db.query_row::<String, _, _>(
            "SELECT body FROM search_index WHERE entity_type = ? AND entity_id = ?",
            params!["issue", "TL-1"],
            |r| r.get(0)
        )
        .unwrap(),
        "second"
    );
    store::index::remove(&db, "issue", "TL-1").unwrap();
    assert_eq!(
        db.query_row::<i64, _, _>("SELECT count(*) FROM search_index_refs", [], |r| r.get(0))
            .unwrap(),
        0
    );
}
