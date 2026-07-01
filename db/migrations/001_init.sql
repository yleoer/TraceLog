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

CREATE INDEX IF NOT EXISTS idx_issues_updated_at ON issues(updated_at);
CREATE INDEX IF NOT EXISTS idx_issues_status ON issues(status);
CREATE INDEX IF NOT EXISTS idx_issues_started_at ON issues(started_at);
CREATE INDEX IF NOT EXISTS idx_issues_completed_at ON issues(completed_at);
CREATE INDEX IF NOT EXISTS idx_issue_events_issue_id ON issue_events(issue_id);
CREATE INDEX IF NOT EXISTS idx_issue_events_happened_at ON issue_events(happened_at);
CREATE INDEX IF NOT EXISTS idx_issue_todos_issue_id ON issue_todos(issue_id);
CREATE INDEX IF NOT EXISTS idx_issue_todos_due_at ON issue_todos(due_at);
CREATE INDEX IF NOT EXISTS idx_issue_todos_done ON issue_todos(done);
CREATE INDEX IF NOT EXISTS idx_temp_tasks_updated_at ON temp_tasks(updated_at);
CREATE INDEX IF NOT EXISTS idx_temp_tasks_status ON temp_tasks(status);
CREATE INDEX IF NOT EXISTS idx_temp_task_events_task ON temp_task_events(temp_task_id);
CREATE INDEX IF NOT EXISTS idx_weekly_logs_week ON weekly_logs(week);
CREATE INDEX IF NOT EXISTS idx_day_entries_date ON day_entries(date);
CREATE INDEX IF NOT EXISTS idx_activity_events_happened_at ON activity_events(happened_at);
CREATE INDEX IF NOT EXISTS idx_activity_events_ref ON activity_events(source, ref_id);

-- +goose Down
DROP TABLE IF EXISTS search_index;
DROP TABLE IF EXISTS activity_events;
DROP TABLE IF EXISTS day_entries;
DROP TABLE IF EXISTS weekly_logs;
DROP TABLE IF EXISTS temp_task_events;
DROP TABLE IF EXISTS temp_tasks;
DROP TABLE IF EXISTS issue_todos;
DROP TABLE IF EXISTS issue_events;
DROP TABLE IF EXISTS issues;
