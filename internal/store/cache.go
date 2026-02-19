package store

import (
	"database/sql"
	"time"
)

func (db *DB) GetCache(key string) ([]byte, error) {
	type cacheRow struct {
		ExpiresAt sql.NullTime `db:"expires_at"`
		Data      []byte       `db:"data"`
	}

	var row cacheRow
	err := db.Get(&row, "SELECT data, expires_at FROM cache WHERE key = ?", key)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if row.ExpiresAt.Valid && time.Now().After(row.ExpiresAt.Time) {
		_, _ = db.Exec("DELETE FROM cache WHERE key = ?", key)
		return nil, nil
	}

	return row.Data, nil
}

func (db *DB) SetCache(key string, data []byte, ttl time.Duration) error {
	var expiresAt *time.Time
	if ttl > 0 {
		t := time.Now().Add(ttl)
		expiresAt = &t
	}

	_, err := db.Exec(`
		INSERT INTO cache (key, data, expires_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET data = excluded.data, expires_at = excluded.expires_at
	`, key, data, expiresAt)
	return err
}

func (db *DB) ClearCache() error {
	_, err := db.Exec("DELETE FROM cache")
	return err
}
