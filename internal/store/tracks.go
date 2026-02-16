package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
)

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
		provider_id, title, artist, artists, album, album_artist, album_artists,
		track_number, disc_number, total_tracks, total_discs,
		year, genre, label, isrc, copyright, composer,
		duration, explicit, compilation, album_art_url, lyrics, subtitles,
		bpm, key_name, key_scale, replay_gain, peak, version, description, url, audio_quality, audio_modes, release_date,
		status, error, parent_job_id, file_path, file_extension,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = db.Exec(query,
		track.ProviderID, track.Title, track.Artist, string(artistsJSON), track.Album, track.AlbumArtist, string(albumArtistsJSON),
		track.TrackNumber, track.DiscNumber, track.TotalTracks, track.TotalDiscs,
		track.Year, track.Genre, track.Label, track.ISRC, track.Copyright, track.Composer,
		track.Duration, track.Explicit, track.Compilation, track.AlbumArtURL, track.Lyrics, track.Subtitles,
		track.BPM, track.Key, track.KeyScale, track.ReplayGain, track.Peak, track.Version, track.Description, track.URL, track.AudioQuality, track.AudioModes, track.ReleaseDate,
		track.Status, track.Error, track.ParentJobID, track.FilePath, track.FileExtension,
		track.CreatedAt, track.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create track: %w", err)
	}
	return nil
}

func (db *DB) GetTrackByID(id int) (*domain.Track, error) {
	query := `SELECT 
		id, provider_id, title, artist, artists, album, album_artist, album_artists,
		track_number, disc_number, total_tracks, total_discs,
		year, genre, label, isrc, copyright, composer,
		duration, explicit, compilation, album_art_url, lyrics, subtitles,
		bpm, key_name, key_scale, replay_gain, peak, version, description, url, audio_quality, audio_modes, release_date,
		status, error, parent_job_id, file_path, file_extension,
		created_at, updated_at, completed_at
	FROM tracks WHERE id = ?`

	row := db.QueryRow(query, id)
	return scanTrack(row)
}

func (db *DB) GetTrackByProviderID(providerID string) (*domain.Track, error) {
	query := `SELECT 
		id, provider_id, title, artist, artists, album, album_artist, album_artists,
		track_number, disc_number, total_tracks, total_discs,
		year, genre, label, isrc, copyright, composer,
		duration, explicit, compilation, album_art_url, lyrics, subtitles,
		bpm, key_name, key_scale, replay_gain, peak, version, description, url, audio_quality, audio_modes, release_date,
		status, error, parent_job_id, file_path, file_extension,
		created_at, updated_at, completed_at
	FROM tracks WHERE provider_id = ?`

	row := db.QueryRow(query, providerID)
	return scanTrack(row)
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
		provider_id = ?, title = ?, artist = ?, artists = ?, album = ?, album_artist = ?, album_artists = ?,
		track_number = ?, disc_number = ?, total_tracks = ?, total_discs = ?,
		year = ?, genre = ?, label = ?, isrc = ?, copyright = ?, composer = ?,
		duration = ?, explicit = ?, compilation = ?, album_art_url = ?, lyrics = ?, subtitles = ?,
		bpm = ?, key_name = ?, key_scale = ?, replay_gain = ?, peak = ?, version = ?, description = ?, url = ?, audio_quality = ?, audio_modes = ?, release_date = ?,
		status = ?, error = ?, parent_job_id = ?, file_path = ?, file_extension = ?,
		updated_at = ?
	WHERE id = ?`

	_, err = db.Exec(query,
		track.ProviderID, track.Title, track.Artist, string(artistsJSON), track.Album, track.AlbumArtist, string(albumArtistsJSON),
		track.TrackNumber, track.DiscNumber, track.TotalTracks, track.TotalDiscs,
		track.Year, track.Genre, track.Label, track.ISRC, track.Copyright, track.Composer,
		track.Duration, track.Explicit, track.Compilation, track.AlbumArtURL, track.Lyrics, track.Subtitles,
		track.BPM, track.Key, track.KeyScale, track.ReplayGain, track.Peak, track.Version, track.Description, track.URL, track.AudioQuality, track.AudioModes, track.ReleaseDate,
		track.Status, track.Error, track.ParentJobID, track.FilePath, track.FileExtension,
		time.Now(), track.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}
	return nil
}

func (db *DB) UpdateTrackStatus(id int, status domain.TrackStatus, filePath string) error {
	query := `UPDATE tracks SET status = ?, file_path = ?, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, status, filePath, time.Now(), id)
	return err
}

func (db *DB) MarkTrackCompleted(id int, filePath string) error {
	query := `UPDATE tracks SET status = ?, file_path = ?, completed_at = ?, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, domain.TrackStatusCompleted, filePath, time.Now(), time.Now(), id)
	return err
}

func (db *DB) MarkTrackFailed(id int, errorMsg string) error {
	query := `UPDATE tracks SET status = ?, error = ?, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, domain.TrackStatusFailed, errorMsg, time.Now(), id)
	return err
}

func (db *DB) ListTracks(limit int) ([]*domain.Track, error) {
	query := `SELECT 
		id, provider_id, title, artist, artists, album, album_artist, album_artists,
		track_number, disc_number, total_tracks, total_discs,
		year, genre, label, isrc, copyright, composer,
		duration, explicit, compilation, album_art_url, lyrics, subtitles,
		bpm, key_name, key_scale, replay_gain, peak, version, description, url, audio_quality, audio_modes, release_date,
		status, error, parent_job_id, file_path, file_extension,
		created_at, updated_at, completed_at
	FROM tracks ORDER BY created_at DESC LIMIT ?`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTracks(rows)
}

func (db *DB) ListTracksByStatus(status domain.TrackStatus, limit int) ([]*domain.Track, error) {
	query := `SELECT 
		id, provider_id, title, artist, artists, album, album_artist, album_artists,
		track_number, disc_number, total_tracks, total_discs,
		year, genre, label, isrc, copyright, composer,
		duration, explicit, compilation, album_art_url, lyrics, subtitles,
		bpm, key_name, key_scale, replay_gain, peak, version, description, url, audio_quality, audio_modes, release_date,
		status, error, parent_job_id, file_path, file_extension,
		created_at, updated_at, completed_at
	FROM tracks WHERE status = ? ORDER BY created_at DESC LIMIT ?`

	rows, err := db.Query(query, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTracks(rows)
}

func (db *DB) ListTracksByParentJobID(parentJobID string) ([]*domain.Track, error) {
	query := `SELECT 
		id, provider_id, title, artist, artists, album, album_artist, album_artists,
		track_number, disc_number, total_tracks, total_discs,
		year, genre, label, isrc, copyright, composer,
		duration, explicit, compilation, album_art_url, lyrics, subtitles,
		bpm, key_name, key_scale, replay_gain, peak, version, description, url, audio_quality, audio_modes, release_date,
		status, error, parent_job_id, file_path, file_extension,
		created_at, updated_at, completed_at
	FROM tracks WHERE parent_job_id = ? ORDER BY track_number ASC`

	rows, err := db.Query(query, parentJobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTracks(rows)
}

func (db *DB) ListCompletedTracks(limit int) ([]*domain.Track, error) {
	return db.ListTracksByStatus(domain.TrackStatusCompleted, limit)
}

func (db *DB) SearchTracks(query string, limit int) ([]*domain.Track, error) {
	sqlQuery := `SELECT 
		id, provider_id, title, artist, artists, album, album_artist, album_artists,
		track_number, disc_number, total_tracks, total_discs,
		year, genre, label, isrc, copyright, composer,
		duration, explicit, compilation, album_art_url, lyrics, subtitles,
		bpm, key_name, key_scale, replay_gain, peak, version, description, url, audio_quality, audio_modes, release_date,
		status, error, parent_job_id, file_path, file_extension,
		created_at, updated_at, completed_at
	FROM tracks 
	WHERE title LIKE ? OR artist LIKE ? OR album LIKE ?
	ORDER BY created_at DESC LIMIT ?`
	searchTerm := "%" + query + "%"

	rows, err := db.Query(sqlQuery, searchTerm, searchTerm, searchTerm, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTracks(rows)
}

func (db *DB) DeleteTrack(id int) error {
	_, err := db.Exec("DELETE FROM tracks WHERE id = ?", id)
	return err
}

func (db *DB) IsTrackDownloaded(providerID string) (bool, error) {
	query := `SELECT COUNT(*) FROM tracks WHERE provider_id = ? AND status = 'completed'`
	var count int
	err := db.QueryRow(query, providerID).Scan(&count)
	return count > 0, err
}

func (db *DB) GetDownloadedTrack(providerID string) (*domain.Track, error) {
	query := `SELECT 
		id, provider_id, title, artist, artists, album, album_artist, album_artists,
		track_number, disc_number, total_tracks, total_discs,
		year, genre, label, isrc, copyright, composer,
		duration, explicit, compilation, album_art_url, lyrics, subtitles,
		bpm, key_name, key_scale, replay_gain, peak, version, description, url, audio_quality, audio_modes, release_date,
		status, error, parent_job_id, file_path, file_extension,
		created_at, updated_at, completed_at
	FROM tracks WHERE provider_id = ? AND status = 'completed' LIMIT 1`

	row := db.QueryRow(query, providerID)
	return scanTrack(row)
}

// scanTrack scans a single track from a row
func scanTrack(row *sql.Row) (*domain.Track, error) {
	track, err := scanTrackFromScanner(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return track, err
}

// rowsScanner wraps sql.Rows to implement trackScanner
type rowsScanner struct {
	rows *sql.Rows
}

func (r *rowsScanner) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func scanTracks(rows *sql.Rows) ([]*domain.Track, error) {
	var tracks []*domain.Track
	for rows.Next() {
		track, err := scanTrackFromScanner(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, track)
	}
	return tracks, rows.Err()
}

// scanTrackFromScanner scans a track from any scanner (Row or Rows)
func scanTrackFromScanner(scanner interface {
	Scan(dest ...interface{}) error
}) (*domain.Track, error) {
	track := &domain.Track{}
	var artistsJSON, albumArtistsJSON string
	var errMsg, filePath, fileExt, lyrics, subtitles, parentJobID sql.NullString
	var trackNum, discNum, totalTracks, totalDiscs, year, duration, bpm sql.NullInt64
	var replayGain, peak sql.NullFloat64
	var explicit, compilation sql.NullBool
	var completedAt sql.NullTime
	var key, keyScale, version, description, url, audioQuality, audioModes, releaseDate sql.NullString

	err := scanner.Scan(
		&track.ID, &track.ProviderID, &track.Title, &track.Artist, &artistsJSON, &track.Album, &track.AlbumArtist, &albumArtistsJSON,
		&trackNum, &discNum, &totalTracks, &totalDiscs,
		&year, &track.Genre, &track.Label, &track.ISRC, &track.Copyright, &track.Composer,
		&duration, &explicit, &compilation, &track.AlbumArtURL, &lyrics, &subtitles,
		&bpm, &key, &keyScale, &replayGain, &peak, &version, &description, &url, &audioQuality, &audioModes, &releaseDate,
		&track.Status, &errMsg, &parentJobID, &filePath, &fileExt,
		&track.CreatedAt, &track.UpdatedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON arrays
	if artistsJSON != "" {
		if err := json.Unmarshal([]byte(artistsJSON), &track.Artists); err != nil {
			return nil, fmt.Errorf("failed to unmarshal artists: %w", err)
		}
	}
	if albumArtistsJSON != "" {
		if err := json.Unmarshal([]byte(albumArtistsJSON), &track.AlbumArtists); err != nil {
			return nil, fmt.Errorf("failed to unmarshal album artists: %w", err)
		}
	}

	// Set nullable fields
	if trackNum.Valid {
		track.TrackNumber = int(trackNum.Int64)
	}
	if discNum.Valid {
		track.DiscNumber = int(discNum.Int64)
	}
	if totalTracks.Valid {
		track.TotalTracks = int(totalTracks.Int64)
	}
	if totalDiscs.Valid {
		track.TotalDiscs = int(totalDiscs.Int64)
	}
	if year.Valid {
		track.Year = int(year.Int64)
	}
	if duration.Valid {
		track.Duration = int(duration.Int64)
	}
	if bpm.Valid {
		track.BPM = int(bpm.Int64)
	}
	if replayGain.Valid {
		track.ReplayGain = replayGain.Float64
	}
	if peak.Valid {
		track.Peak = peak.Float64
	}
	if explicit.Valid {
		track.Explicit = explicit.Bool
	}
	if compilation.Valid {
		track.Compilation = compilation.Bool
	}
	if key.Valid {
		track.Key = key.String
	}
	if keyScale.Valid {
		track.KeyScale = keyScale.String
	}
	if version.Valid {
		track.Version = version.String
	}
	if description.Valid {
		track.Description = description.String
	}
	if url.Valid {
		track.URL = url.String
	}
	if audioQuality.Valid {
		track.AudioQuality = audioQuality.String
	}
	if audioModes.Valid {
		track.AudioModes = audioModes.String
	}
	if releaseDate.Valid {
		track.ReleaseDate = releaseDate.String
	}
	if errMsg.Valid {
		track.Error = errMsg.String
	}
	if parentJobID.Valid {
		track.ParentJobID = parentJobID.String
	}
	if filePath.Valid {
		track.FilePath = filePath.String
	}
	if fileExt.Valid {
		track.FileExtension = fileExt.String
	}
	if lyrics.Valid {
		track.Lyrics = lyrics.String
	}
	if subtitles.Valid {
		track.Subtitles = subtitles.String
	}
	if completedAt.Valid {
		track.CompletedAt = completedAt.Time
	}

	return track, nil
}
