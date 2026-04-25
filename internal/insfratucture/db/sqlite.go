package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// Config holds SQLite connection settings.
// e.g. "file:./stunning.db?_journal=WAL&_foreign_keys=on"
type Config struct {
	DSN string
}

// New opens a SQLite connection, runs migrations, and returns the *sql.DB.
func New(cfg Config) (*sql.DB, error) {
	db, err := sql.Open("sqlite", cfg.DSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// if err := migrate(db); err != nil {
	// 	return nil, err
	// }

	return db, nil
}
