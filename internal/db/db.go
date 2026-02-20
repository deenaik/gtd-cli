package db

import (
	"database/sql"
	"sync"

	"github.com/deenaik/gtd-cli/internal/config"
	_ "modernc.org/sqlite"
)

var (
	database *sql.DB
	once     sync.Once
)

func Get() *sql.DB {
	once.Do(func() {
		database = mustOpen()
	})
	return database
}

func mustOpen() *sql.DB {
	cfg := config.Get()
	db, err := sql.Open("sqlite", cfg.DBPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		panic("failed to open database: " + err.Error())
	}
	db.SetMaxOpenConns(1) // SQLite single-writer
	return db
}

func Migrate() error {
	_, err := Get().Exec(schema)
	return err
}

func Close() {
	if database != nil {
		database.Close()
	}
}
