-- +goose Up
ALTER TABLE issues ADD COLUMN summary_md TEXT NOT NULL DEFAULT '';
ALTER TABLE issues ADD COLUMN started_at TEXT NOT NULL DEFAULT '';
ALTER TABLE issues ADD COLUMN completed_at TEXT NOT NULL DEFAULT '';

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

CREATE INDEX IF NOT EXISTS idx_issue_todos_issue_id ON issue_todos(issue_id);
CREATE INDEX IF NOT EXISTS idx_issue_todos_due_at ON issue_todos(due_at);
CREATE INDEX IF NOT EXISTS idx_issue_todos_done ON issue_todos(done);
CREATE INDEX IF NOT EXISTS idx_issues_started_at ON issues(started_at);
CREATE INDEX IF NOT EXISTS idx_issues_completed_at ON issues(completed_at);

-- +goose Down
DROP TABLE IF EXISTS issue_todos;
DROP INDEX IF EXISTS idx_issues_completed_at;
DROP INDEX IF EXISTS idx_issues_started_at;
ALTER TABLE issues DROP COLUMN completed_at;
ALTER TABLE issues DROP COLUMN started_at;
ALTER TABLE issues DROP COLUMN summary_md;
