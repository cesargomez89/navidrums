package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/cesargomez89/navidrums/internal/domain"
)

func (db *DB) CreateTrack(track *domain.Track) error {
	track.Normalize()

	query := `INSERT INTO tracks (
		provider_id, title, artist, artists, album, album_id, album_artist, album_artists, artist_ids, album_artist_ids,
		track_number, disc_number, total_tracks, total_discs,
		year, genre, label, isrc, copyright, composer,
		duration, explicit, compilation, album_art_url, lyrics, subtitles,
		bpm, key_name, key_scale, replay_gain, peak, version, description, url, audio_quality, audio_modes, release_date,
		barcode, catalog_number, release_type, release_id, recording_id, tags,
		status, error, parent_job_id, file_path, file_extension,
		created_at, updated_at, etag, file_hash, last_verified_at
	) VALUES (
		:provider_id, :title, :artist, :artists, :album, :album_id, :album_artist, :album_artists, :artist_ids, :album_artist_ids,
		:track_number, :disc_number, :total_tracks, :total_discs,
		:year, :genre, :label, :isrc, :copyright, :composer,
		:duration, :explicit, :compilation, :album_art_url, :lyrics, :subtitles,
		:bpm, :key_name, :key_scale, :replay_gain, :peak, :version, :description, :url, :audio_quality, :audio_modes, :release_date,
		:barcode, :catalog_number, :release_type, :release_id, :recording_id, :tags,
		:status, :error, :parent_job_id, :file_path, :file_extension,
		:created_at, :updated_at, :etag, :file_hash, :last_verified_at
	) RETURNING id`

	rows, err := db.NamedQuery(query, track)
	if err != nil {
		return fmt.Errorf("failed to create track (named query): %w", err)
	}
	defer rows.Close() //nolint:errcheck // deferred cleanup

	if rows.Next() {
		if err := rows.Scan(&track.ID); err != nil {
			return fmt.Errorf("failed to scan track id: %w", err)
		}
	} else if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating returning rows: %w", err)
	}

	return nil
}

func (db *DB) GetTrackByID(id int) (*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE id = ?`

	var track domain.Track
	err := db.Get(&track, query, id)
	if err != nil {
		return nil, err
	}
	return &track, nil
}

func (db *DB) GetTrackByProviderID(providerID string) (*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE provider_id = ?`

	var track domain.Track
	err := db.Get(&track, query, providerID)
	if err != nil {
		return nil, err
	}
	return &track, nil
}

func (db *DB) UpdateTrack(track *domain.Track) error {
	track.Normalize()

	query := `UPDATE tracks SET
		provider_id = :provider_id, title = :title, artist = :artist, artists = :artists,
		album = :album, album_id = :album_id, album_artist = :album_artist, album_artists = :album_artists,
		artist_ids = :artist_ids, album_artist_ids = :album_artist_ids,
		track_number = :track_number, disc_number = :disc_number, total_tracks = :total_tracks, total_discs = :total_discs,
		year = :year, genre = :genre, label = :label, isrc = :isrc, copyright = :copyright, composer = :composer,
		duration = :duration, explicit = :explicit, compilation = :compilation, album_art_url = :album_art_url, lyrics = :lyrics, subtitles = :subtitles,
		bpm = :bpm, key_name = :key_name, key_scale = :key_scale, replay_gain = :replay_gain, peak = :peak,
		version = :version, description = :description, url = :url, audio_quality = :audio_quality, audio_modes = :audio_modes, release_date = :release_date,
		barcode = :barcode, catalog_number = :catalog_number, release_type = :release_type, release_id = :release_id, recording_id = :recording_id, tags = :tags,
		status = :status, error = :error, parent_job_id = :parent_job_id, file_path = :file_path, file_extension = :file_extension,
		updated_at = :updated_at, etag = :etag, file_hash = :file_hash, completed_at = :completed_at, last_verified_at = :last_verified_at
	WHERE id = :id`

	track.UpdatedAt = time.Now()

	result, err := db.NamedExec(query, track)
	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("track with id %d not found", track.ID)
	}
	return nil
}

func (db *DB) UpdateTrackStatus(id int, status domain.TrackStatus, filePath string) error {
	query := `UPDATE tracks SET status = ?, file_path = ?, updated_at = ? WHERE id = ?`
	result, err := db.Exec(query, status, filePath, time.Now(), id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("track with id %d not found", id)
	}
	return nil
}

func (db *DB) UpdateTrackPartial(id int, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	allowedColumns := map[string]bool{
		"title":            true,
		"artist":           true,
		"album":            true,
		"album_artist":     true,
		"artist_ids":       true,
		"album_artist_ids": true,
		"genre":            true,
		"tags":             true,
		"label":            true,
		"composer":         true,
		"copyright":        true,
		"isrc":             true,
		"version":          true,
		"description":      true,
		"url":              true,
		"audio_quality":    true,
		"audio_modes":      true,
		"lyrics":           true,
		"subtitles":        true,
		"barcode":          true,
		"catalog_number":   true,
		"release_type":     true,
		"release_date":     true,
		"key_name":         true,
		"key_scale":        true,
		"track_number":     true,
		"disc_number":      true,
		"total_tracks":     true,
		"total_discs":      true,
		"year":             true,
		"bpm":              true,
		"replay_gain":      true,
		"peak":             true,
		"compilation":      true,
		"explicit":         true,
	}

	setClauses := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+2)

	for col, val := range updates {
		if !allowedColumns[col] {
			return fmt.Errorf("invalid column name: %s", col)
		}
		setClauses = append(setClauses, col+" = ?")
		args = append(args, val)
	}

	args = append(args, time.Now(), id)

	query := fmt.Sprintf("UPDATE tracks SET %s, updated_at = ? WHERE id = ?", strings.Join(setClauses, ", "))

	result, err := db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("track with id %d not found", id)
	}
	return nil
}

func (db *DB) MarkTrackCompleted(id int, filePath, fileHash string) error {
	query := `UPDATE tracks SET status = ?, file_path = ?, completed_at = ?, file_hash = ?, last_verified_at = ?, updated_at = ? WHERE id = ?`
	now := time.Now()
	result, err := db.Exec(query, domain.TrackStatusCompleted, filePath, now, fileHash, now, now, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("track with id %d not found", id)
	}
	return nil
}

func (db *DB) MarkTrackFailed(id int, errorMsg string) error {
	query := `UPDATE tracks SET status = ?, error = ?, updated_at = ? WHERE id = ?`
	result, err := db.Exec(query, domain.TrackStatusFailed, errorMsg, time.Now(), id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("track with id %d not found", id)
	}
	return nil
}

func (db *DB) ListTracks(limit int) ([]*domain.Track, error) {
	query := `SELECT * FROM tracks ORDER BY created_at DESC LIMIT ?`
	return selectTracks(db, query, limit)
}

func (db *DB) ListTracksByStatus(status domain.TrackStatus, limit int) ([]*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE status = ? ORDER BY created_at DESC LIMIT ?`
	return selectTracks(db, query, status, limit)
}

func (db *DB) ListTracksByParentJobID(parentJobID string) ([]*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE parent_job_id = ? ORDER BY track_number ASC`
	return selectTracks(db, query, parentJobID)
}

func (db *DB) ListCompletedTracks(limit int) ([]*domain.Track, error) {
	return db.ListTracksByStatus(domain.TrackStatusCompleted, limit)
}

func (db *DB) SearchTracks(q string, limit int) ([]*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE title LIKE ? OR artist LIKE ? OR album LIKE ? ORDER BY created_at DESC LIMIT ?`
	searchTerm := "%" + q + "%"
	return selectTracks(db, query, searchTerm, searchTerm, searchTerm, limit)
}

func (db *DB) ListCompletedTracksNoGenre(limit int) ([]*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE status = 'completed' AND (genre IS NULL OR TRIM(genre) = '') ORDER BY created_at DESC LIMIT ?`
	return selectTracks(db, query, limit)
}

func (db *DB) DeleteTrack(id int) error {
	_, err := db.Exec("DELETE FROM tracks WHERE id = ?", id)
	return err
}

func (db *DB) IsTrackDownloaded(providerID string) (bool, error) {
	query := `SELECT COUNT(*) FROM tracks WHERE provider_id = ? AND status = 'completed' AND file_path IS NOT NULL`
	var count int
	err := db.Get(&count, query, providerID)
	return count > 0, err
}

func (db *DB) GetDownloadedTrack(providerID string) (*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE provider_id = ? AND status = 'completed' AND file_path IS NOT NULL LIMIT 1`

	var track domain.Track
	err := db.Get(&track, query, providerID)
	if err != nil {
		return nil, err
	}
	return &track, nil
}

func (db *DB) RecomputeAlbumState(albumID string) (string, error) {
	query := `SELECT 
		COUNT(*) as total, 
		SUM(CASE WHEN status = 'completed' AND file_path IS NOT NULL THEN 1 ELSE 0 END) as completed 
	FROM tracks WHERE album_id = ?`

	type result struct {
		Total     int `db:"total"`
		Completed int `db:"completed"`
	}
	var r result
	if err := db.Get(&r, query, albumID); err != nil {
		return "", err
	}

	if r.Completed == 0 {
		return "missing", nil
	}
	if r.Completed < r.Total {
		return "partial", nil
	}
	return "completed", nil
}

func (db *DB) FindInterruptedTracks() ([]*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE status IN ('downloading', 'processing')`
	return selectTracks(db, query)
}

func (db *DB) ListCompletedTracksWithISRC() ([]*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE status = 'completed' AND isrc != '' ORDER BY created_at DESC`
	return selectTracks(db, query)
}

func (db *DB) ListAllCompletedTracks() ([]*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE status = ? ORDER BY created_at DESC`
	return selectTracks(db, query, domain.TrackStatusCompleted)
}

func selectTracks(q sqlx.Queryer, query string, args ...interface{}) ([]*domain.Track, error) {
	var tracks []*domain.Track
	err := sqlx.Select(q, &tracks, query, args...)
	return tracks, err
}
