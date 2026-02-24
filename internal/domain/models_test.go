package domain

import (
	"testing"
)

func TestJobType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		jobType  JobType
		expected string
	}{
		{"track", JobTypeTrack, "track"},
		{"album", JobTypeAlbum, "album"},
		{"playlist", JobTypePlaylist, "playlist"},
		{"artist", JobTypeArtist, "artist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.jobType) != tt.expected {
				t.Errorf("JobType %s = %q, want %q", tt.name, tt.jobType, tt.expected)
			}
		})
	}
}

func TestJobStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		status   JobStatus
		expected string
	}{
		{"queued", JobStatusQueued, "queued"},
		{"running", JobStatusRunning, "running"},
		{"completed", JobStatusCompleted, "completed"},
		{"failed", JobStatusFailed, "failed"},
		{"cancelled", JobStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("JobStatus %s = %q, want %q", tt.name, tt.status, tt.expected)
			}
		})
	}
}

func TestTrackStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		status   TrackStatus
		expected string
	}{
		{"missing", TrackStatusMissing, "missing"},
		{"queued", TrackStatusQueued, "queued"},
		{"downloading", TrackStatusDownloading, "downloading"},
		{"processing", TrackStatusProcessing, "processing"},
		{"completed", TrackStatusCompleted, "completed"},
		{"failed", TrackStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("TrackStatus %s = %q, want %q", tt.name, tt.status, tt.expected)
			}
		})
	}
}

func TestJob_StatusAssignment(t *testing.T) {
	var job Job

	validStatuses := []JobStatus{
		JobStatusQueued,
		JobStatusRunning,
		JobStatusCompleted,
		JobStatusFailed,
		JobStatusCancelled,
	}

	for _, status := range validStatuses {
		job.Status = status
		if job.Status != status {
			t.Errorf("Status assignment failed: got %s, want %s", job.Status, status)
		}
	}
}

func TestTrack_StatusAssignment(t *testing.T) {
	var track Track

	validStatuses := []TrackStatus{
		TrackStatusMissing,
		TrackStatusQueued,
		TrackStatusDownloading,
		TrackStatusProcessing,
		TrackStatusCompleted,
		TrackStatusFailed,
	}

	for _, status := range validStatuses {
		track.Status = status
		if track.Status != status {
			t.Errorf("Status assignment failed: got %s, want %s", track.Status, status)
		}
	}
}

func TestCatalogTrack_Fields(t *testing.T) {
	track := CatalogTrack{
		ID:          "catalog_123",
		Title:       "Test Song",
		Duration:    180,
		TrackNumber: 1,
	}

	if track.ID != "catalog_123" {
		t.Errorf("ID = %s, want catalog_123", track.ID)
	}
	if track.Title != "Test Song" {
		t.Errorf("Title = %s, want Test Song", track.Title)
	}
	if track.Duration != 180 {
		t.Errorf("Duration = %d, want 180", track.Duration)
	}
	if track.TrackNumber != 1 {
		t.Errorf("TrackNumber = %d, want 1", track.TrackNumber)
	}
}

func TestAlbum_Fields(t *testing.T) {
	album := Album{
		ID:          "album_123",
		Title:       "Test Album",
		TotalTracks: 10,
	}

	if album.ID != "album_123" {
		t.Errorf("ID = %s, want album_123", album.ID)
	}
	if album.Title != "Test Album" {
		t.Errorf("Title = %s, want Test Album", album.Title)
	}
	if album.TotalTracks != 10 {
		t.Errorf("TotalTracks = %d, want 10", album.TotalTracks)
	}
}

func TestArtist_Fields(t *testing.T) {
	artist := Artist{
		ID:   "artist_123",
		Name: "Test Artist",
	}

	if artist.ID != "artist_123" {
		t.Errorf("ID = %s, want artist_123", artist.ID)
	}
	if artist.Name != "Test Artist" {
		t.Errorf("Name = %s, want Test Artist", artist.Name)
	}
}

func TestPlaylist_Fields(t *testing.T) {
	playlist := Playlist{
		ID:    "playlist_123",
		Title: "My Playlist",
	}

	if playlist.ID != "playlist_123" {
		t.Errorf("ID = %s, want playlist_123", playlist.ID)
	}
	if playlist.Title != "My Playlist" {
		t.Errorf("Title = %s, want My Playlist", playlist.Title)
	}
}

func TestSearchResult_Fields(t *testing.T) {
	result := SearchResult{
		Artists: []Artist{
			{ID: "artist_1", Name: "Artist One"},
		},
		Albums: []Album{
			{ID: "album_1", Title: "Album One"},
		},
		Tracks: []CatalogTrack{
			{ID: "track_1", Title: "Track One"},
		},
		Playlists: []Playlist{
			{ID: "playlist_1", Title: "Playlist One"},
		},
	}

	if len(result.Artists) != 1 {
		t.Errorf("Artists length = %d, want 1", len(result.Artists))
	}
	if len(result.Albums) != 1 {
		t.Errorf("Albums length = %d, want 1", len(result.Albums))
	}
	if len(result.Tracks) != 1 {
		t.Errorf("Tracks length = %d, want 1", len(result.Tracks))
	}
	if len(result.Playlists) != 1 {
		t.Errorf("Playlists length = %d, want 1", len(result.Playlists))
	}

	if result.Artists[0].ID != "artist_1" {
		t.Errorf("Artists[0].ID = %s, want artist_1", result.Artists[0].ID)
	}
	if result.Albums[0].ID != "album_1" {
		t.Errorf("Albums[0].ID = %s, want album_1", result.Albums[0].ID)
	}
	if result.Tracks[0].ID != "track_1" {
		t.Errorf("Tracks[0].ID = %s, want track_1", result.Tracks[0].ID)
	}
	if result.Playlists[0].ID != "playlist_1" {
		t.Errorf("Playlists[0].ID = %s, want playlist_1", result.Playlists[0].ID)
	}
}

func TestTrack_Normalize(t *testing.T) {
	tr := &Track{
		Genre: "Metal",
	}
	tr.Normalize() // should be a no-op? No, it lowercases now.
	if tr.Genre != "metal" {
		t.Errorf("Normalize() changed Genre to %q, want %q", tr.Genre, "metal")
	}
}
