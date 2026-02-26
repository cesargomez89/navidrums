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
		description: "Consolidated beta schema updates and full backfill",
		up: func(tx *sqlx.Tx) error {
			columns := []string{
				"ALTER TABLE tracks ADD COLUMN barcode TEXT",
				"ALTER TABLE tracks ADD COLUMN catalog_number TEXT",
				"ALTER TABLE tracks ADD COLUMN release_type TEXT",
				"ALTER TABLE tracks ADD COLUMN release_id TEXT",
				"ALTER TABLE tracks ADD COLUMN sub_genre TEXT",
				"ALTER TABLE tracks ADD COLUMN recording_id TEXT",
				"ALTER TABLE tracks ADD COLUMN tags TEXT",
				"ALTER TABLE tracks ADD COLUMN artist_ids TEXT",
				"ALTER TABLE tracks ADD COLUMN album_artist_ids TEXT",
			}
			for _, q := range columns {
				if _, err := tx.Exec(q); err != nil {
					if !strings.Contains(err.Error(), "duplicate column name") {
						return err
					}
				}
			}

			// Clear version field (no longer used)
			if _, err := tx.Exec("UPDATE tracks SET version = ''"); err != nil {
				return err
			}

			// Comprehensive backfill to avoid NULL scan panics
			_, err := tx.Exec(`UPDATE tracks SET
				artist = COALESCE(artist, ''),
				album = COALESCE(album, ''),
				album_id = COALESCE(album_id, ''),
				album_artist = COALESCE(album_artist, ''),
				genre = COALESCE(genre, ''),
				sub_genre = COALESCE(sub_genre, ''),
				label = COALESCE(label, ''),
				isrc = COALESCE(isrc, ''),
				copyright = COALESCE(copyright, ''),
				composer = COALESCE(composer, ''),
				album_art_url = COALESCE(album_art_url, ''),
				lyrics = COALESCE(lyrics, ''),
				subtitles = COALESCE(subtitles, ''),
				key_name = COALESCE(key_name, ''),
				key_scale = COALESCE(key_scale, ''),
				version = COALESCE(version, ''),
				description = COALESCE(description, ''),
				url = COALESCE(url, ''),
				audio_quality = COALESCE(audio_quality, ''),
				audio_modes = COALESCE(audio_modes, ''),
				release_date = COALESCE(release_date, ''),
				barcode = COALESCE(barcode, ''),
				catalog_number = COALESCE(catalog_number, ''),
				release_type = COALESCE(release_type, ''),
				release_id = COALESCE(release_id, ''),
				recording_id = COALESCE(recording_id, ''),
				tags = COALESCE(tags, '[]'),
				artist_ids = COALESCE(artist_ids, '[]'),
				album_artist_ids = COALESCE(album_artist_ids, '[]'),
				error = COALESCE(error, ''),
				parent_job_id = COALESCE(parent_job_id, ''),
				file_path = COALESCE(file_path, ''),
				file_extension = COALESCE(file_extension, ''),
				file_hash = COALESCE(file_hash, ''),
				etag = COALESCE(etag, ''),
				track_number = COALESCE(track_number, 0),
				disc_number = COALESCE(disc_number, 0),
				total_tracks = COALESCE(total_tracks, 0),
				total_discs = COALESCE(total_discs, 0),
				year = COALESCE(year, 0),
				duration = COALESCE(duration, 0),
				bpm = COALESCE(bpm, 0),
				replay_gain = COALESCE(replay_gain, 0.0),
				peak = COALESCE(peak, 0.0),
				explicit = COALESCE(explicit, 0),
				compilation = COALESCE(compilation, 0)
			`)
			return err
		},
	},
	{
		version:     3,
		description: "Merge sub_genre into genre as 'genre; subgenre'",
		up: func(tx *sqlx.Tx) error {
			// Merge sub_genre into genre for tracks that have a non-empty sub_genre
			// that is different from the genre (avoid "Genre; Genre" duplication).
			// The sub_genre column is kept in the DB but no longer used by the app.
			_, err := tx.Exec(`
				UPDATE tracks
				SET genre = genre || '; ' || sub_genre
				WHERE sub_genre IS NOT NULL
				  AND TRIM(sub_genre) != ''
				  AND LOWER(REPLACE(REPLACE(REPLACE(genre, ' ', ''), '-', ''), '_', ''))
				    != LOWER(REPLACE(REPLACE(REPLACE(sub_genre, ' ', ''), '-', ''), '_', ''))
			`)
			return err
		},
	},
	{
		version:     4,
		description: "Clean up carriage returns and duplicate newlines in lyrics and subtitles",
		up: func(tx *sqlx.Tx) error {
			// Subtitles should not have any empty lines, collapse \n\n to \n
			// Subtitles should not have any empty lines, collapse \n\n to \n
			_, err := tx.Exec(`
				UPDATE tracks 
				SET subtitles = REPLACE(REPLACE(REPLACE(subtitles, '\n', CHAR(10)), CHAR(13), ''), CHAR(10) || CHAR(10), CHAR(10))
				WHERE subtitles IS NOT NULL AND subtitles != ''
			`)
			if err != nil {
				return err
			}

			// Lyrics can have paragraphs, but we should clean up double carriage returns which resulted in \n\n\n\n
			// We'll replace literal \n first, then \r, then compress > 2 newlines into 2.
			_, err = tx.Exec(`
				UPDATE tracks 
				SET lyrics = REPLACE(
								REPLACE(
									REPLACE(
										REPLACE(lyrics, '\n', CHAR(10)),
									CHAR(13), ''), 
								CHAR(10) || CHAR(10) || CHAR(10) || CHAR(10), CHAR(10) || CHAR(10)),
							 CHAR(10) || CHAR(10) || CHAR(10), CHAR(10) || CHAR(10))
				WHERE lyrics IS NOT NULL AND lyrics != ''
			`)
			return err
		},
	},
	{
		version:     5,
		description: "Add indexes for track album_id, track created_at, and job status",
		up: func(tx *sqlx.Tx) error {
			queries := []string{
				"CREATE INDEX IF NOT EXISTS idx_tracks_album_id ON tracks(album_id);",
				"CREATE INDEX IF NOT EXISTS idx_tracks_created_at ON tracks(created_at DESC);",
				"CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);",
			}
			for _, q := range queries {
				if _, err := tx.Exec(q); err != nil {
					return err
				}
			}
			return nil
		},
	},
	{
		version:     6,
		description: "Fill year field from release_date",
		up: func(tx *sqlx.Tx) error {
			_, err := tx.Exec(`
				UPDATE tracks
				SET year = CAST(SUBSTR(release_date, 1, 4) AS INTEGER)
				WHERE release_date IS NOT NULL
				  AND LENGTH(release_date) >= 4
				  AND SUBSTR(release_date, 1, 4) GLOB '[0-9][0-9][0-9][0-9]'
			`)
			return err
		},
	},
	{
		version:     7,
		description: "Remove subgenre: extract genre from 'genre; subgenre' and drop column",
		up: func(tx *sqlx.Tx) error {
			_, err := tx.Exec(`
				UPDATE tracks
				SET genre = TRIM(SUBSTR(genre, 1, INSTR(genre || ';', ';') - 1))
				WHERE genre LIKE '%;%'
			`)
			if err != nil {
				return err
			}
			_, err = tx.Exec("ALTER TABLE tracks DROP COLUMN sub_genre")
			if err != nil && !strings.Contains(err.Error(), "no such column") {
				return err
			}
			return nil
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

// checkRowsAffected ensures that an UPDATE or DELETE affected at least one row
func checkRowsAffected(result sql.Result, entity string, id interface{}) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("%s with id %v not found", entity, id)
	}
	return nil
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
