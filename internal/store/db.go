package store

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type migration struct {
	version     int
	description string
	up          func(*sql.DB) error
}

// Migrations history is cleared for v2.0 refactor
// New two-table architecture: jobs + tracks
var migrations = []migration{
	{
		version:     1,
		description: "Add track lifecycle fields",
		up: func(db *sql.DB) error {
			columns := []string{
				"ALTER TABLE tracks ADD COLUMN file_hash TEXT",
				"ALTER TABLE tracks ADD COLUMN etag TEXT",
				"ALTER TABLE tracks ADD COLUMN last_verified_at DATETIME",
				"ALTER TABLE tracks ADD COLUMN album_id TEXT",
			}
			for _, q := range columns {
				if _, err := db.Exec(q); err != nil {
					// Ignore duplicate column errors if migration ran partially
					continue
				}
			}
			return nil
		},
	},
}

type DB struct {
	*sql.DB
}

func NewSQLiteDB(dsn string) (*DB, error) {
	// Append pragmas to DSN to ensure they apply to all connections in the pool
	// _pragma=busy_timeout(30000) avoids "database is locked" errors
	// _pragma=journal_mode(WAL) enables Write-Ahead Logging for better concurrency
	if !strings.Contains(dsn, "?") {
		dsn += "?"
	} else {
		dsn += "&"
	}
	dsn += "_pragma=busy_timeout(30000)&_pragma=journal_mode(WAL)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
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
