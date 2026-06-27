-- +goose Up
ALTER TABLE temp_tasks ADD COLUMN started_at TEXT NOT NULL DEFAULT '';
ALTER TABLE temp_tasks ADD COLUMN completed_at TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE temp_tasks DROP COLUMN completed_at;
ALTER TABLE temp_tasks DROP COLUMN started_at;
