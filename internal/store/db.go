package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type migration struct {
	version     int
	description string
	up          func(*sql.DB) error
}

var migrations = []migration{
	{
		version:     1,
		description: "add_file_extension_to_downloads",
		up: func(db *sql.DB) error {
			_, err := db.Exec("ALTER TABLE downloads ADD COLUMN file_extension TEXT DEFAULT '.flac'")
			return err
		},
	},
}

type DB struct {
	*sql.DB
}

func NewSQLiteDB(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=30000"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	if _, err := db.Exec(Schema); err != nil {
		return nil, fmt.Errorf("failed to apply schema: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &DB{db}, nil
}

func runMigrations(db *sql.DB) error {
	for _, m := range migrations {
		applied, err := isMigrationApplied(db, m.version)
		if err != nil {
			return fmt.Errorf("failed to check migration %d: %w", m.version, err)
		}

		if applied {
			continue
		}

		if err := m.up(db); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", m.version, m.description, err)
		}

		if err := recordMigration(db, m.version, m.description); err != nil {
			return fmt.Errorf("failed to record migration %d: %w", m.version, err)
		}
	}

	return nil
}

func isMigrationApplied(db *sql.DB, version int) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func recordMigration(db *sql.DB, version int, description string) error {
	_, err := db.Exec("INSERT INTO schema_migrations (version, description) VALUES (?, ?)", version, description)
	return err
}

func (db *DB) Close() error {
	return db.DB.Close()
}
