package store

import (
	"database/sql"
	"time"
)

type SettingsRepo struct {
	db *DB
}

func NewSettingsRepo(db *DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) Get(key string) (string, error) {
	var value string
	err := r.db.Get(&value, "SELECT value FROM settings WHERE key = ?", key)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (r *SettingsRepo) Set(key, value string) error {
	_, err := r.db.Exec(`
		INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, time.Now())
	return err
}

func (r *SettingsRepo) Delete(key string) error {
	_, err := r.db.Exec("DELETE FROM settings WHERE key = ?", key)
	return err
}

const (
	SettingActiveProvider  = "active_provider"
	SettingCustomProviders = "custom_providers"
	SettingGenreMap        = "genre_map"
	SettingGenreSeparator  = "genre_separator"
)
