package store

import (
	"database/sql"
	"time"
)

func (db *DB) GetCache(key string) ([]byte, error) {
	var data []byte
	var expiresAt sql.NullTime

	err := db.QueryRow("SELECT data, expires_at FROM cache WHERE key = ?", key).Scan(&data, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		db.Exec("DELETE FROM cache WHERE key = ?", key)
		return nil, nil
	}

	return data, nil
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
