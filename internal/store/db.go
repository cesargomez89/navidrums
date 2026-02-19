package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type migration struct {
	version     int
	description string
	up          func(*sqlx.DB) error
}

// Migrations history is cleared for v2.0 refactor
// New two-table architecture: jobs + tracks
var migrations = []migration{
	{
		version:     1,
		description: "Add track lifecycle fields",
		up: func(db *sqlx.DB) error {
			columns := []string{
				"ALTER TABLE tracks ADD COLUMN file_hash TEXT",
				"ALTER TABLE tracks ADD COLUMN etag TEXT",
				"ALTER TABLE tracks ADD COLUMN last_verified_at DATETIME",
				"ALTER TABLE tracks ADD COLUMN album_id TEXT",
			}
			for _, q := range columns {
				if _, err := db.Exec(q); err != nil {
					continue
				}
			}
			return nil
		},
	},
	{
		version:     2,
		description: "Add MusicBrainz metadata fields",
		up: func(db *sqlx.DB) error {
			columns := []string{
				"ALTER TABLE tracks ADD COLUMN barcode TEXT",
				"ALTER TABLE tracks ADD COLUMN catalog_number TEXT",
				"ALTER TABLE tracks ADD COLUMN release_type TEXT",
				"ALTER TABLE tracks ADD COLUMN release_id TEXT",
			}
			for _, q := range columns {
				if _, err := db.Exec(q); err != nil {
					continue
				}
			}
			return nil
		},
	},
}

type DB struct {
	*sqlx.DB
}

func NewSQLiteDB(dsn string) (*DB, error) {
	if !strings.Contains(dsn, "?") {
		dsn += "?"
	} else {
		dsn += "&"
	}
	dsn += "_pragma=busy_timeout(30000)&_pragma=journal_mode(WAL)"

	db, err := sqlx.Open("sqlite", dsn)
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

func runMigrations(db *sqlx.DB) error {
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

func isMigrationApplied(db *sqlx.DB, version int) (bool, error) {
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

func recordMigration(db *sqlx.DB, version int, description string) error {
	_, err := db.Exec("INSERT INTO schema_migrations (version, description) VALUES (?, ?)", version, description)
	return err
}

func (db *DB) Close() error {
	return db.DB.Close()
}
