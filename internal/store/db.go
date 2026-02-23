package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type migration struct {
	up          func(*sqlx.Tx) error
	description string
	version     int
}

// Migrations history is cleared for v2.0 refactor
// New two-table architecture: jobs + tracks
var migrations = []migration{
	{
		version:     1,
		description: "Add track lifecycle fields",
		up: func(tx *sqlx.Tx) error {
			columns := []string{
				"ALTER TABLE tracks ADD COLUMN file_hash TEXT",
				"ALTER TABLE tracks ADD COLUMN etag TEXT",
				"ALTER TABLE tracks ADD COLUMN last_verified_at DATETIME",
				"ALTER TABLE tracks ADD COLUMN album_id TEXT",
			}
			for _, q := range columns {
				if _, err := tx.Exec(q); err != nil {
					if !strings.Contains(err.Error(), "duplicate column name") {
						return err
					}
				}
			}
			return nil
		},
	},
	{
		version:     2,
		description: "Add MusicBrainz metadata fields",
		up: func(tx *sqlx.Tx) error {
			columns := []string{
				"ALTER TABLE tracks ADD COLUMN barcode TEXT",
				"ALTER TABLE tracks ADD COLUMN catalog_number TEXT",
				"ALTER TABLE tracks ADD COLUMN release_type TEXT",
				"ALTER TABLE tracks ADD COLUMN release_id TEXT",
			}
			for _, q := range columns {
				if _, err := tx.Exec(q); err != nil {
					if !strings.Contains(err.Error(), "duplicate column name") {
						return err
					}
				}
			}
			return nil
		},
	},
	{
		version:     3,
		description: "Clear version field (no longer used)",
		up: func(tx *sqlx.Tx) error {
			_, err := tx.Exec("UPDATE tracks SET version = ''")
			return err
		},
	},
	{
		version:     4,
		description: "Add sub_genre for original MusicBrainz tag",
		up: func(tx *sqlx.Tx) error {
			if _, err := tx.Exec("ALTER TABLE tracks ADD COLUMN sub_genre TEXT"); err != nil {
				if !strings.Contains(err.Error(), "duplicate column name") {
					return err
				}
			}
			return nil
		},
	},
	{
		version:     5,
		description: "Add recording_id for MusicBrainz caching",
		up: func(tx *sqlx.Tx) error {
			if _, err := tx.Exec("ALTER TABLE tracks ADD COLUMN recording_id TEXT"); err != nil {
				if !strings.Contains(err.Error(), "duplicate column name") {
					return err
				}
			}
			return nil
		},
	},
	{
		version:     6,
		description: "Backfill NULL sub_genre to empty string",
		up: func(tx *sqlx.Tx) error {
			_, err := tx.Exec("UPDATE tracks SET sub_genre = '' WHERE sub_genre IS NULL")
			return err
		},
	},
	{
		version:     7,
		description: "Backfill NULL TEXT columns (added via ALTER TABLE) to empty string",
		up: func(tx *sqlx.Tx) error {
			// All columns below were added via ALTER TABLE with no DEFAULT,
			// leaving existing rows as NULL which cannot be scanned into Go string.
			_, err := tx.Exec(`UPDATE tracks SET
				album_id        = COALESCE(album_id, ''),
				file_hash       = COALESCE(file_hash, ''),
				etag            = COALESCE(etag, ''),
				barcode         = COALESCE(barcode, ''),
				catalog_number  = COALESCE(catalog_number, ''),
				release_type    = COALESCE(release_type, ''),
				release_id      = COALESCE(release_id, '')
			`)
			return err
		},
	},
}

type dbOps interface {
	Rebind(query string) string
	BindNamed(query string, arg interface{}) (string, []interface{}, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Queryx(query string, args ...interface{}) (*sqlx.Rows, error)
	QueryRowx(query string, args ...interface{}) *sqlx.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	NamedQuery(query string, arg interface{}) (*sqlx.Rows, error)
	NamedExec(query string, arg interface{}) (sql.Result, error)
}

type DB struct {
	dbOps
	root *sqlx.DB
}

func NewSQLiteDB(dsn string) (*DB, error) {
	if !strings.Contains(dsn, "?") {
		dsn += "?"
	} else {
		dsn += "&"
	}
	// Increase busy_timeout significantly and enable WAL mode for better concurrency
	dsn += "_pragma=busy_timeout(60000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"

	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	// SQLite only supports one concurrent writer. Setting MaxOpenConns to 1
	// ensures writers queue inside Go rather than failing at the SQLite level with SQLITE_BUSY.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	if _, err := db.Exec(Schema); err != nil {
		return nil, fmt.Errorf("failed to apply schema: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &DB{dbOps: db, root: db}, nil
}

// RunInTx runs the given function within a transaction.
// It yields a *DB instance that transparently executes operations
// over the active transaction instead of the connection pool.
func (db *DB) RunInTx(fn func(txDB *DB) error) error {
	if db.root == nil {
		// Already in a transaction, just run the function
		return fn(db)
	}

	tx, err := db.root.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback is best-effort; commit result is what matters

	txDB := &DB{
		dbOps: tx,
		root:  nil, // txDB is a transaction unit, cannot spawn nested tx
	}

	if err := fn(txDB); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
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

		tx, err := db.Beginx()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %w", m.version, err)
		}

		if err := m.up(tx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to apply migration %d (%s): %w", m.version, m.description, err)
		}

		if err := recordMigration(tx, m.version, m.description); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", m.version, err)
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

func recordMigration(tx *sqlx.Tx, version int, description string) error {
	_, err := tx.Exec("INSERT INTO schema_migrations (version, description) VALUES (?, ?)", version, description)
	return err
}

func (db *DB) Close() error {
	if db.root != nil {
		return db.root.Close()
	}
	return nil
}
