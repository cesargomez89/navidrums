package store

import (
	"fmt"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
)

func (db *DB) CreatePlaylist(playlist *domain.Playlist) error {
	query := `INSERT INTO playlists (provider_id, title, description, image_url, created_at, updated_at)
		VALUES (:provider_id, :title, :description, :image_url, :created_at, :updated_at)
		RETURNING id`

	playlist.CreatedAt = time.Now()
	playlist.UpdatedAt = time.Now()

	row, err := db.NamedQuery(query, playlist)
	if err != nil {
		return fmt.Errorf("failed to create playlist: %w", err)
	}
	defer func() { _ = row.Close() }()

	if row.Next() {
		if err := row.Scan(&playlist.ID); err != nil {
			return fmt.Errorf("failed to scan playlist id: %w", err)
		}
	}
	return row.Err()
}

func (db *DB) GetPlaylistByID(id int64) (*domain.Playlist, error) {
	query := `SELECT * FROM playlists WHERE id = ?`
	var playlist domain.Playlist
	err := db.Get(&playlist, query, id)
	if err != nil {
		return nil, err
	}
	return &playlist, nil
}

func (db *DB) GetPlaylistByProviderID(providerID string) (*domain.Playlist, error) {
	query := `SELECT * FROM playlists WHERE provider_id = ?`
	var playlist domain.Playlist
	err := db.Get(&playlist, query, providerID)
	if err != nil {
		return nil, err
	}
	return &playlist, nil
}

func (db *DB) UpdatePlaylist(playlist *domain.Playlist) error {
	playlist.UpdatedAt = time.Now()
	query := `UPDATE playlists SET title = ?, description = ?, image_url = ?, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, playlist.Title, playlist.Description, playlist.ImageURL, playlist.UpdatedAt, playlist.ID)
	return err
}

func (db *DB) ListPlaylists(limit, offset int) ([]*domain.Playlist, error) {
	query := `SELECT * FROM playlists ORDER BY updated_at DESC LIMIT ? OFFSET ?`
	var playlists []*domain.Playlist
	err := db.Select(&playlists, query, limit, offset)
	return playlists, err
}

func (db *DB) DeletePlaylist(id int64) error {
	_, err := db.Exec(`DELETE FROM playlists WHERE id = ?`, id)
	return err
}

func (db *DB) AddTrackToPlaylist(playlistID int64, trackID int, position int) error {
	query := `INSERT OR IGNORE INTO playlist_tracks (playlist_id, track_id, position, added_at) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, playlistID, trackID, position, time.Now())
	return err
}

func (db *DB) RemoveTrackFromPlaylist(playlistID int64, trackID int) error {
	_, err := db.Exec(`DELETE FROM playlist_tracks WHERE playlist_id = ? AND track_id = ?`, playlistID, trackID)
	return err
}

func (db *DB) GetTracksByPlaylistID(playlistID int64) ([]*domain.Track, error) {
	query := `
		SELECT t.* FROM tracks t
		INNER JOIN playlist_tracks pt ON t.id = pt.track_id
		WHERE pt.playlist_id = ?
		ORDER BY pt.position ASC`
	return selectTracks(db, query, playlistID)
}

func (db *DB) GetPlaylistsByTrackID(trackID int) ([]*domain.Playlist, error) {
	query := `
		SELECT p.* FROM playlists p
		INNER JOIN playlist_tracks pt ON p.id = pt.playlist_id
		WHERE pt.track_id = ?`
	var playlists []*domain.Playlist
	err := db.Select(&playlists, query, trackID)
	return playlists, err
}

func (db *DB) PlaylistExists(providerID string) (bool, error) {
	query := `SELECT COUNT(*) FROM playlists WHERE provider_id = ?`
	var count int
	err := db.Get(&count, query, providerID)
	return count > 0, err
}

func (db *DB) ClearPlaylistTracks(playlistID int64) error {
	_, err := db.Exec(`DELETE FROM playlist_tracks WHERE playlist_id = ?`, playlistID)
	return err
}
