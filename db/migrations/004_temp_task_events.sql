-- +goose Up
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
CREATE INDEX IF NOT EXISTS idx_temp_task_events_task ON temp_task_events(temp_task_id);

-- +goose Down
DROP TABLE IF EXISTS temp_task_events;
