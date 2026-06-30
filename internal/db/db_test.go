package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigrateInitializesTempTaskWithoutResultMD(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})

	if err := Migrate(database, filepath.Join("..", "..", "db", "migrations")); err != nil {
		t.Fatal(err)
	}

	rows, err := database.Query(`PRAGMA table_info(temp_tasks)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	hasContent := false
	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatal(err)
		}
		if name == "result_md" {
			t.Fatal("temp_tasks should not include result_md after migrations")
		}
		if name == "content_md" {
			hasContent = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !hasContent {
		t.Fatal("temp_tasks should include content_md after migrations")
	}
}
