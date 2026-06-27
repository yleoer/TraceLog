-- +goose Up
CREATE TABLE IF NOT EXISTS day_entries (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date TEXT NOT NULL,
  content_md TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_day_entries_date ON day_entries(date);

-- +goose Down
DROP TABLE IF EXISTS day_entries;
