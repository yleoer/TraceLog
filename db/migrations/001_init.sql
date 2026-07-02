-- +goose Up
CREATE TABLE IF NOT EXISTS issues (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  jira_key TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  priority TEXT NOT NULL,
  tags TEXT NOT NULL DEFAULT '[]',
  summary_md TEXT NOT NULL DEFAULT '',
  background_md TEXT NOT NULL DEFAULT '',
  analysis_md TEXT NOT NULL DEFAULT '',
  solution_md TEXT NOT NULL DEFAULT '',
  actions_md TEXT NOT NULL DEFAULT '',
  result_md TEXT NOT NULL DEFAULT '',
  todo_md TEXT NOT NULL DEFAULT '',
  links_json TEXT NOT NULL DEFAULT '[]',
  started_at TEXT NOT NULL DEFAULT '',
  completed_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS issue_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  issue_id INTEGER NOT NULL,
  event_type TEXT NOT NULL,
  content_md TEXT NOT NULL,
  happened_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS issue_todos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  issue_id INTEGER NOT NULL,
  content TEXT NOT NULL,
  due_at TEXT NOT NULL DEFAULT '',
  done INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS issue_tags (
  issue_id INTEGER NOT NULL,
  tag TEXT NOT NULL,
  PRIMARY KEY (issue_id, tag),
  FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS temp_tasks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  priority TEXT NOT NULL,
  tags TEXT NOT NULL DEFAULT '[]',
  content_md TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL DEFAULT '',
  completed_at TEXT NOT NULL DEFAULT '',
  converted_to_jira INTEGER NOT NULL DEFAULT 0,
  converted_jira_key TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS temp_task_tags (
  temp_task_id INTEGER NOT NULL,
  tag TEXT NOT NULL,
  PRIMARY KEY (temp_task_id, tag),
  FOREIGN KEY (temp_task_id) REFERENCES temp_tasks(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS temp_task_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  temp_task_id INTEGER NOT NULL,
  event_type TEXT NOT NULL,
  content_md TEXT NOT NULL,
  happened_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (temp_task_id) REFERENCES temp_tasks(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS weekly_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  week TEXT NOT NULL UNIQUE,
  summary_md TEXT NOT NULL DEFAULT '',
  next_plan_md TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS day_entries (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date TEXT NOT NULL,
  content_md TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tempo_time_worklogs (
  account_id TEXT NOT NULL,
  tempo_worklog_id INTEGER NOT NULL,
  work_item_key TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  start_date TEXT NOT NULL,
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  time_spent_seconds INTEGER NOT NULL DEFAULT 0,
  self TEXT NOT NULL DEFAULT '',
  cached_at TEXT NOT NULL,
  PRIMARY KEY (account_id, tempo_worklog_id)
);

CREATE TABLE IF NOT EXISTS tempo_time_cache_ranges (
  account_id TEXT NOT NULL,
  start_date TEXT NOT NULL,
  end_date TEXT NOT NULL,
  refreshed_at TEXT NOT NULL,
  PRIMARY KEY (account_id, start_date, end_date)
);

CREATE TABLE IF NOT EXISTS activity_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source TEXT NOT NULL,
  ref_id INTEGER NOT NULL,
  ref_key TEXT NOT NULL DEFAULT '',
  ref_title TEXT NOT NULL,
  event_type TEXT NOT NULL,
  content_md TEXT NOT NULL DEFAULT '',
  happened_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS search_index USING fts5(
  entity_type UNINDEXED,
  entity_id UNINDEXED,
  title,
  body,
  updated_at UNINDEXED,
  tokenize = 'unicode61'
);

CREATE TABLE IF NOT EXISTS search_index_refs (
  entity_type TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  search_rowid INTEGER NOT NULL UNIQUE,
  PRIMARY KEY (entity_type, entity_id)
);

CREATE TABLE IF NOT EXISTS app_schema_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_issues_created_at ON issues(created_at);
CREATE INDEX IF NOT EXISTS idx_issues_updated_at ON issues(updated_at);
CREATE INDEX IF NOT EXISTS idx_issues_status ON issues(status);
CREATE INDEX IF NOT EXISTS idx_issues_status_updated_at ON issues(status, updated_at);
CREATE INDEX IF NOT EXISTS idx_issues_started_at ON issues(started_at);
CREATE INDEX IF NOT EXISTS idx_issues_completed_at ON issues(completed_at);
CREATE INDEX IF NOT EXISTS idx_issue_events_issue_id ON issue_events(issue_id);
CREATE INDEX IF NOT EXISTS idx_issue_events_happened_at ON issue_events(happened_at);
CREATE INDEX IF NOT EXISTS idx_issue_events_issue_happened_at ON issue_events(issue_id, happened_at, created_at);
CREATE INDEX IF NOT EXISTS idx_issue_todos_issue_id ON issue_todos(issue_id);
CREATE INDEX IF NOT EXISTS idx_issue_todos_due_at ON issue_todos(due_at);
CREATE INDEX IF NOT EXISTS idx_issue_todos_done ON issue_todos(done);
CREATE INDEX IF NOT EXISTS idx_issue_todos_updated_at ON issue_todos(updated_at);
CREATE INDEX IF NOT EXISTS idx_issue_todos_done_updated_at ON issue_todos(done, updated_at);
CREATE INDEX IF NOT EXISTS idx_issue_todos_done_due_updated_at ON issue_todos(done, due_at, updated_at);
CREATE INDEX IF NOT EXISTS idx_issue_todos_issue_done_due_updated_at ON issue_todos(issue_id, done, due_at, updated_at);
CREATE INDEX IF NOT EXISTS idx_issue_tags_tag_issue ON issue_tags(tag, issue_id);
CREATE INDEX IF NOT EXISTS idx_temp_tasks_created_at ON temp_tasks(created_at);
CREATE INDEX IF NOT EXISTS idx_temp_tasks_updated_at ON temp_tasks(updated_at);
CREATE INDEX IF NOT EXISTS idx_temp_tasks_status ON temp_tasks(status);
CREATE INDEX IF NOT EXISTS idx_temp_tasks_status_updated_at ON temp_tasks(status, updated_at);
CREATE INDEX IF NOT EXISTS idx_temp_tasks_started_at ON temp_tasks(started_at);
CREATE INDEX IF NOT EXISTS idx_temp_tasks_completed_at ON temp_tasks(completed_at);
CREATE INDEX IF NOT EXISTS idx_temp_task_tags_tag_task ON temp_task_tags(tag, temp_task_id);
CREATE INDEX IF NOT EXISTS idx_temp_task_events_task ON temp_task_events(temp_task_id);
CREATE INDEX IF NOT EXISTS idx_temp_task_events_happened_at ON temp_task_events(happened_at);
CREATE INDEX IF NOT EXISTS idx_temp_task_events_task_happened_at ON temp_task_events(temp_task_id, happened_at, created_at);
CREATE INDEX IF NOT EXISTS idx_weekly_logs_week ON weekly_logs(week);
CREATE INDEX IF NOT EXISTS idx_day_entries_date ON day_entries(date);
CREATE INDEX IF NOT EXISTS idx_day_entries_date_created_at ON day_entries(date, created_at);
CREATE INDEX IF NOT EXISTS idx_tempo_time_worklogs_account_date ON tempo_time_worklogs(account_id, start_date, start_time);
CREATE INDEX IF NOT EXISTS idx_tempo_time_cache_ranges_account_range ON tempo_time_cache_ranges(account_id, start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_activity_events_happened_at ON activity_events(happened_at);
CREATE INDEX IF NOT EXISTS idx_activity_events_ref ON activity_events(source, ref_id);
CREATE INDEX IF NOT EXISTS idx_activity_events_ref_type ON activity_events(source, ref_id, event_type);

-- +goose Down
DROP TABLE IF EXISTS search_index_refs;
DROP TABLE IF EXISTS search_index;
DROP TABLE IF EXISTS app_schema_meta;
DROP TABLE IF EXISTS activity_events;
DROP TABLE IF EXISTS tempo_time_cache_ranges;
DROP TABLE IF EXISTS tempo_time_worklogs;
DROP TABLE IF EXISTS day_entries;
DROP TABLE IF EXISTS weekly_logs;
DROP TABLE IF EXISTS temp_task_events;
DROP TABLE IF EXISTS temp_task_tags;
DROP TABLE IF EXISTS temp_tasks;
DROP TABLE IF EXISTS issue_tags;
DROP TABLE IF EXISTS issue_todos;
DROP TABLE IF EXISTS issue_events;
DROP TABLE IF EXISTS issues;
