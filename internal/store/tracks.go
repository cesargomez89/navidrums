package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
)

type dbTrack struct {
	CreatedAt      time.Time      `db:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at"`
	LastVerifiedAt *time.Time     `db:"last_verified_at"`
	CompletedAt    *time.Time     `db:"completed_at"`
	Barcode        sql.NullString `db:"barcode"`
	ISRC           string         `db:"isrc"`
	AlbumID        string         `db:"album_id"`
	AlbumArtist    string         `db:"album_artist"`
	AlbumArtists   string         `db:"album_artists"`
	ProviderID     string         `db:"provider_id"`
	FileHash       string         `db:"file_hash"`
	ETag           string         `db:"etag"`
	Title          string         `db:"title"`
	Artist         string         `db:"artist"`
	Genre          string         `db:"genre"`
	Label          string         `db:"label"`
	KeyScale       string         `db:"key_scale"`
	Copyright      string         `db:"copyright"`
	Composer       string         `db:"composer"`
	Artists        string         `db:"artists"`
	FileExtension  string         `db:"file_extension"`
	FilePath       string         `db:"file_path"`
	AlbumArtURL    string         `db:"album_art_url"`
	Lyrics         string         `db:"lyrics"`
	Subtitles      string         `db:"subtitles"`
	ParentJobID    string         `db:"parent_job_id"`
	Album          string         `db:"album"`
	KeyName        string         `db:"key_name"`
	ReleaseDate    string         `db:"release_date"`
	Error          string         `db:"error"`
	Version        string         `db:"version"`
	Description    string         `db:"description"`
	URL            string         `db:"url"`
	AudioQuality   string         `db:"audio_quality"`
	AudioModes     string         `db:"audio_modes"`
	Status         string         `db:"status"`
	ReleaseID      sql.NullString `db:"release_id"`
	CatalogNumber  sql.NullString `db:"catalog_number"`
	ReleaseType    sql.NullString `db:"release_type"`
	ID             int            `db:"id"`
	ReplayGain     float64        `db:"replay_gain"`
	Peak           float64        `db:"peak"`
	BPM            int            `db:"bpm"`
	Duration       int            `db:"duration"`
	Year           int            `db:"year"`
	TotalDiscs     int            `db:"total_discs"`
	TotalTracks    int            `db:"total_tracks"`
	DiscNumber     int            `db:"disc_number"`
	TrackNumber    int            `db:"track_number"`
	Compilation    bool           `db:"compilation"`
	Explicit       bool           `db:"explicit"`
}

func (d *dbTrack) toDomain() *domain.Track {
	track := &domain.Track{
		ID:            d.ID,
		ProviderID:    d.ProviderID,
		Title:         d.Title,
		Artist:        d.Artist,
		Album:         d.Album,
		AlbumID:       d.AlbumID,
		AlbumArtist:   d.AlbumArtist,
		Genre:         d.Genre,
		Label:         d.Label,
		ISRC:          d.ISRC,
		Copyright:     d.Copyright,
		Composer:      d.Composer,
		AlbumArtURL:   d.AlbumArtURL,
		Status:        domain.TrackStatus(d.Status),
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
		TrackNumber:   d.TrackNumber,
		DiscNumber:    d.DiscNumber,
		TotalTracks:   d.TotalTracks,
		TotalDiscs:    d.TotalDiscs,
		Year:          d.Year,
		Duration:      d.Duration,
		Explicit:      d.Explicit,
		Compilation:   d.Compilation,
		BPM:           d.BPM,
		ReplayGain:    d.ReplayGain,
		Peak:          d.Peak,
		Key:           d.KeyName,
		KeyScale:      d.KeyScale,
		Version:       d.Version,
		Description:   d.Description,
		URL:           d.URL,
		AudioQuality:  d.AudioQuality,
		AudioModes:    d.AudioModes,
		ReleaseDate:   d.ReleaseDate,
		Barcode:       d.Barcode.String,
		CatalogNumber: d.CatalogNumber.String,
		ReleaseType:   d.ReleaseType.String,
		ReleaseID:     d.ReleaseID.String,
		Error:         d.Error,
		ParentJobID:   d.ParentJobID,
		FilePath:      d.FilePath,
		FileExtension: d.FileExtension,
		Lyrics:        d.Lyrics,
		Subtitles:     d.Subtitles,
		ETag:          d.ETag,
		FileHash:      d.FileHash,
	}

	if d.Artists != "" {
		_ = json.Unmarshal([]byte(d.Artists), &track.Artists)
	}
	if d.AlbumArtists != "" {
		_ = json.Unmarshal([]byte(d.AlbumArtists), &track.AlbumArtists)
	}
	if d.CompletedAt != nil {
		track.CompletedAt = d.CompletedAt
	}
	if d.LastVerifiedAt != nil {
		track.LastVerifiedAt = d.LastVerifiedAt
	}

	return track
}

func (db *DB) CreateTrack(track *domain.Track) error {
	artistsJSON, err := json.Marshal(track.Artists)
	if err != nil {
		return fmt.Errorf("failed to marshal artists: %w", err)
	}
	albumArtistsJSON, err := json.Marshal(track.AlbumArtists)
	if err != nil {
		return fmt.Errorf("failed to marshal album artists: %w", err)
	}

	query := `INSERT INTO tracks (
		provider_id, title, artist, artists, album, album_id, album_artist, album_artists,
		track_number, disc_number, total_tracks, total_discs,
		year, genre, label, isrc, copyright, composer,
		duration, explicit, compilation, album_art_url, lyrics, subtitles,
		bpm, key_name, key_scale, replay_gain, peak, version, description, url, audio_quality, audio_modes, release_date,
		barcode, catalog_number, release_type, release_id,
		status, error, parent_job_id, file_path, file_extension,
		created_at, updated_at, etag, file_hash, last_verified_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := db.Exec(query,
		track.ProviderID, track.Title, track.Artist, string(artistsJSON), track.Album, track.AlbumID, track.AlbumArtist, string(albumArtistsJSON),
		track.TrackNumber, track.DiscNumber, track.TotalTracks, track.TotalDiscs,
		track.Year, track.Genre, track.Label, track.ISRC, track.Copyright, track.Composer,
		track.Duration, track.Explicit, track.Compilation, track.AlbumArtURL, track.Lyrics, track.Subtitles,
		track.BPM, track.Key, track.KeyScale, track.ReplayGain, track.Peak, track.Version, track.Description, track.URL, track.AudioQuality, track.AudioModes, track.ReleaseDate,
		track.Barcode, track.CatalogNumber, track.ReleaseType, track.ReleaseID,
		track.Status, track.Error, track.ParentJobID, track.FilePath, track.FileExtension,
		track.CreatedAt, track.UpdatedAt, track.ETag, track.FileHash, track.LastVerifiedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create track: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	track.ID = int(id)

	return nil
}

func (db *DB) GetTrackByID(id int) (*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE id = ?`

	var t dbTrack
	err := db.Get(&t, query, id)
	if err != nil {
		return nil, err
	}
	return t.toDomain(), nil
}

func (db *DB) GetTrackByProviderID(providerID string) (*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE provider_id = ?`

	var t dbTrack
	err := db.Get(&t, query, providerID)
	if err != nil {
		return nil, err
	}
	return t.toDomain(), nil
}

func (db *DB) UpdateTrack(track *domain.Track) error {
	artistsJSON, err := json.Marshal(track.Artists)
	if err != nil {
		return fmt.Errorf("failed to marshal artists: %w", err)
	}
	albumArtistsJSON, err := json.Marshal(track.AlbumArtists)
	if err != nil {
		return fmt.Errorf("failed to marshal album artists: %w", err)
	}

	query := `UPDATE tracks SET
		provider_id = ?, title = ?, artist = ?, artists = ?, album = ?, album_id = ?, album_artist = ?, album_artists = ?,
		track_number = ?, disc_number = ?, total_tracks = ?, total_discs = ?,
		year = ?, genre = ?, label = ?, isrc = ?, copyright = ?, composer = ?,
		duration = ?, explicit = ?, compilation = ?, album_art_url = ?, lyrics = ?, subtitles = ?,
		bpm = ?, key_name = ?, key_scale = ?, replay_gain = ?, peak = ?, version = ?, description = ?, url = ?, audio_quality = ?, audio_modes = ?, release_date = ?,
		barcode = ?, catalog_number = ?, release_type = ?, release_id = ?,
		status = ?, error = ?, parent_job_id = ?, file_path = ?, file_extension = ?,
		updated_at = ?, etag = ?, file_hash = ?, completed_at = ?, last_verified_at = ?
	WHERE id = ?`

	var completedAt, lastVerifiedAt interface{}
	if track.CompletedAt != nil {
		completedAt = *track.CompletedAt
	}
	if track.LastVerifiedAt != nil {
		lastVerifiedAt = *track.LastVerifiedAt
	}

	result, err := db.Exec(query,
		track.ProviderID, track.Title, track.Artist, string(artistsJSON), track.Album, track.AlbumID, track.AlbumArtist, string(albumArtistsJSON),
		track.TrackNumber, track.DiscNumber, track.TotalTracks, track.TotalDiscs,
		track.Year, track.Genre, track.Label, track.ISRC, track.Copyright, track.Composer,
		track.Duration, track.Explicit, track.Compilation, track.AlbumArtURL, track.Lyrics, track.Subtitles,
		track.BPM, track.Key, track.KeyScale, track.ReplayGain, track.Peak, track.Version, track.Description, track.URL, track.AudioQuality, track.AudioModes, track.ReleaseDate,
		track.Barcode, track.CatalogNumber, track.ReleaseType, track.ReleaseID,
		track.Status, track.Error, track.ParentJobID, track.FilePath, track.FileExtension,
		time.Now(), track.ETag, track.FileHash, completedAt, lastVerifiedAt, track.ID,
	)
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
		"title":          true,
		"artist":         true,
		"album":          true,
		"album_artist":   true,
		"genre":          true,
		"label":          true,
		"composer":       true,
		"copyright":      true,
		"isrc":           true,
		"version":        true,
		"description":    true,
		"url":            true,
		"audio_quality":  true,
		"audio_modes":    true,
		"lyrics":         true,
		"subtitles":      true,
		"barcode":        true,
		"catalog_number": true,
		"release_type":   true,
		"release_date":   true,
		"key_name":       true,
		"key_scale":      true,
		"track_number":   true,
		"disc_number":    true,
		"total_tracks":   true,
		"total_discs":    true,
		"year":           true,
		"bpm":            true,
		"replay_gain":    true,
		"peak":           true,
		"compilation":    true,
		"explicit":       true,
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
	result, err := db.Exec(query, domain.TrackStatusCompleted, filePath, time.Now(), fileHash, time.Now(), time.Now(), id)
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

	var tracks []*domain.Track
	var rows []dbTrack
	err := db.Select(&rows, query, limit)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		tracks = append(tracks, rows[i].toDomain())
	}
	return tracks, nil
}

func (db *DB) ListTracksByStatus(status domain.TrackStatus, limit int) ([]*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE status = ? ORDER BY created_at DESC LIMIT ?`

	var tracks []*domain.Track
	var rows []dbTrack
	err := db.Select(&rows, query, status, limit)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		tracks = append(tracks, rows[i].toDomain())
	}
	return tracks, nil
}

func (db *DB) ListTracksByParentJobID(parentJobID string) ([]*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE parent_job_id = ? ORDER BY track_number ASC`

	var tracks []*domain.Track
	var rows []dbTrack
	err := db.Select(&rows, query, parentJobID)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		tracks = append(tracks, rows[i].toDomain())
	}
	return tracks, nil
}

func (db *DB) ListCompletedTracks(limit int) ([]*domain.Track, error) {
	return db.ListTracksByStatus(domain.TrackStatusCompleted, limit)
}

func (db *DB) SearchTracks(q string, limit int) ([]*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE title LIKE ? OR artist LIKE ? OR album LIKE ? ORDER BY created_at DESC LIMIT ?`
	searchTerm := "%" + q + "%"

	var tracks []*domain.Track
	var rows []dbTrack
	err := db.Select(&rows, query, searchTerm, searchTerm, searchTerm, limit)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		tracks = append(tracks, rows[i].toDomain())
	}
	return tracks, nil
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

	var t dbTrack
	err := db.Get(&t, query, providerID)
	if err != nil {
		return nil, err
	}
	return t.toDomain(), nil
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

	var tracks []*domain.Track
	var rows []dbTrack
	err := db.Select(&rows, query)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		tracks = append(tracks, rows[i].toDomain())
	}
	return tracks, nil
}

func (db *DB) ListCompletedTracksWithISRC() ([]*domain.Track, error) {
	query := `SELECT * FROM tracks WHERE status = 'completed' AND isrc != '' ORDER BY created_at DESC`

	var tracks []*domain.Track
	var rows []dbTrack
	err := db.Select(&rows, query)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		tracks = append(tracks, rows[i].toDomain())
	}
	return tracks, nil
}
