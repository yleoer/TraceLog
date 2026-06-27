package db

import (
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"tracelog/internal/config"
)

func Open(cfg config.Config) (*sql.DB, error) {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(cfg.DataDir, "exports"), 0o755); err != nil {
		return nil, err
	}

	database, err := sql.Open("sqlite", cfg.DatabasePath)
	if err != nil {
		return nil, err
	}

	database.SetMaxOpenConns(1)
	if _, err := database.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;"); err != nil {
		database.Close()
		return nil, err
	}

	return database, nil
}

func Migrate(database *sql.DB, dir string) error {
	goose.SetBaseFS(nil)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.Up(database, dir)
}

func MigrateFS(database *sql.DB, migrations fs.FS, dir string) error {
	goose.SetBaseFS(migrations)
	defer goose.SetBaseFS(nil)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.Up(database, dir)
}
