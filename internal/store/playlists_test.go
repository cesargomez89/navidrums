package store

import (
	"testing"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
)

func TestDB_Playlists(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	playlist := &domain.Playlist{
		ProviderID:  "spotify_123",
		Title:       "My Playlist",
		Description: "Test description",
		ImageURL:    "https://example.com/image.jpg",
	}
	err := db.CreatePlaylist(playlist)
	if err != nil {
		t.Fatalf("CreatePlaylist failed: %v", err)
	}
	if playlist.ID == 0 {
		t.Error("Expected playlist ID to be set")
	}

	fetched, err := db.GetPlaylistByID(playlist.ID)
	if err != nil {
		t.Fatalf("GetPlaylistByID failed: %v", err)
	}
	if fetched.Title != playlist.Title {
		t.Errorf("Title mismatch: got %s, want %s", fetched.Title, playlist.Title)
	}

	fetched, err = db.GetPlaylistByProviderID("spotify_123")
	if err != nil {
		t.Fatalf("GetPlaylistByProviderID failed: %v", err)
	}
	if fetched.ProviderID != playlist.ProviderID {
		t.Errorf("ProviderID mismatch: got %s, want %s", fetched.ProviderID, playlist.ProviderID)
	}

	playlist.Title = "Updated Title"
	err = db.UpdatePlaylist(playlist)
	if err != nil {
		t.Fatalf("UpdatePlaylist failed: %v", err)
	}
	fetched, _ = db.GetPlaylistByID(playlist.ID)
	if fetched.Title != "Updated Title" {
		t.Errorf("Title not updated: got %s, want Updated Title", fetched.Title)
	}

	list, err := db.ListPlaylists(10, 0)
	if err != nil {
		t.Fatalf("ListPlaylists failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 playlist, got %d", len(list))
	}

	exists, err := db.PlaylistExists("spotify_123")
	if err != nil {
		t.Fatalf("PlaylistExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected playlist to exist")
	}

	exists, err = db.PlaylistExists("nonexistent")
	if err != nil {
		t.Fatalf("PlaylistExists failed: %v", err)
	}
	if exists {
		t.Error("Expected playlist to not exist")
	}

	err = db.DeletePlaylist(playlist.ID)
	if err != nil {
		t.Fatalf("DeletePlaylist failed: %v", err)
	}
	_, err = db.GetPlaylistByID(playlist.ID)
	if err == nil {
		t.Error("Expected error after deleting playlist")
	}
}

func TestDB_PlaylistTracks(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	playlist := &domain.Playlist{
		ProviderID: "pl_1",
		Title:      "Test Playlist",
	}
	err := db.CreatePlaylist(playlist)
	if err != nil {
		t.Fatalf("CreatePlaylist failed: %v", err)
	}

	tracks := []*domain.Track{
		{ProviderID: "track_1", Title: "Track 1", Artist: "Artist", Album: "Album", TrackNumber: 1, Status: domain.TrackStatusCompleted, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ProviderID: "track_2", Title: "Track 2", Artist: "Artist", Album: "Album", TrackNumber: 2, Status: domain.TrackStatusCompleted, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ProviderID: "track_3", Title: "Track 3", Artist: "Artist", Album: "Album", TrackNumber: 3, Status: domain.TrackStatusCompleted, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, tr := range tracks {
		if createErr := db.CreateTrack(tr); createErr != nil {
			t.Fatalf("CreateTrack failed: %v", createErr)
		}
	}

	err = db.AddTrackToPlaylist(playlist.ID, tracks[0].ID, 0)
	if err != nil {
		t.Fatalf("AddTrackToPlaylist failed: %v", err)
	}
	err = db.AddTrackToPlaylist(playlist.ID, tracks[1].ID, 1)
	if err != nil {
		t.Fatalf("AddTrackToPlaylist failed: %v", err)
	}

	playlistTracks, err := db.GetTracksByPlaylistID(playlist.ID)
	if err != nil {
		t.Fatalf("GetTracksByPlaylistID failed: %v", err)
	}
	if len(playlistTracks) != 2 {
		t.Errorf("Expected 2 tracks in playlist, got %d", len(playlistTracks))
	}

	playlists, err := db.GetPlaylistsByTrackID(tracks[0].ID)
	if err != nil {
		t.Fatalf("GetPlaylistsByTrackID failed: %v", err)
	}
	if len(playlists) != 1 {
		t.Errorf("Expected 1 playlist containing track, got %d", len(playlists))
	}

	err = db.RemoveTrackFromPlaylist(playlist.ID, tracks[0].ID)
	if err != nil {
		t.Fatalf("RemoveTrackFromPlaylist failed: %v", err)
	}
	playlistTracks, _ = db.GetTracksByPlaylistID(playlist.ID)
	if len(playlistTracks) != 1 {
		t.Errorf("Expected 1 track after removal, got %d", len(playlistTracks))
	}

	err = db.ClearPlaylistTracks(playlist.ID)
	if err != nil {
		t.Fatalf("ClearPlaylistTracks failed: %v", err)
	}
	playlistTracks, _ = db.GetTracksByPlaylistID(playlist.ID)
	if len(playlistTracks) != 0 {
		t.Errorf("Expected 0 tracks after clear, got %d", len(playlistTracks))
	}

	_ = db.AddTrackToPlaylist(playlist.ID, tracks[0].ID, 0)
	_ = db.AddTrackToPlaylist(playlist.ID, tracks[1].ID, 1)
	_ = db.AddTrackToPlaylist(playlist.ID, tracks[2].ID, 2)

	playlistTracks, _ = db.GetTracksByPlaylistID(playlist.ID)
	if len(playlistTracks) != 3 {
		t.Errorf("Expected 3 tracks in playlist, got %d", len(playlistTracks))
	}

	for i, tr := range playlistTracks {
		if tr.TrackNumber != i+1 {
			t.Errorf("Expected track at position %d to have TrackNumber %d, got %d", i, i+1, tr.TrackNumber)
		}
	}
}

func TestDB_PlaylistDuplicateTrack(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	playlist := &domain.Playlist{
		ProviderID: "pl_dup",
		Title:      "Test Playlist",
	}
	if err := db.CreatePlaylist(playlist); err != nil {
		t.Fatalf("CreatePlaylist failed: %v", err)
	}

	track := &domain.Track{
		ProviderID: "track_dup",
		Title:      "Track",
		Artist:     "Artist",
		Album:      "Album",
		Status:     domain.TrackStatusCompleted,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := db.CreateTrack(track); err != nil {
		t.Fatalf("CreateTrack failed: %v", err)
	}

	err := db.AddTrackToPlaylist(playlist.ID, track.ID, 0)
	if err != nil {
		t.Fatalf("AddTrackToPlaylist failed: %v", err)
	}

	err = db.AddTrackToPlaylist(playlist.ID, track.ID, 0)
	if err != nil {
		t.Fatalf("Second AddTrackToPlaylist should not fail: %v", err)
	}

	playlistTracks, _ := db.GetTracksByPlaylistID(playlist.ID)
	if len(playlistTracks) != 1 {
		t.Errorf("Expected 1 track (no duplicates), got %d", len(playlistTracks))
	}
}
