package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/domain"
)

func TestPlaylistGenerator_Generate(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		DownloadsDir:   tmpDir,
		SubdirTemplate: "{{.AlbumArtist}}/{{.Album}}/{{.Track}} {{.Title}}",
	}

	pg := NewPlaylistGenerator(cfg)

	pl := &domain.Playlist{
		Title: "Test Playlist",
		Tracks: []domain.CatalogTrack{
			{
				ID:          "t1",
				Title:       "Track 1",
				Artist:      "Artist A",
				Album:       "Album 1",
				Year:        2023,
				TrackNumber: 1,
				Duration:    180,
			},
		},
	}

	lookup := func(id string) string {
		return ".flac"
	}

	err := pg.Generate(pl, lookup)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	playlistPath := filepath.Join(tmpDir, "playlists", "Test Playlist.m3u")
	content, err := os.ReadFile(playlistPath) //nolint:gosec
	if err != nil {
		t.Fatalf("Failed to read playlist file: %v", err)
	}

	sContent := string(content)
	if !strings.HasPrefix(sContent, "#EXTM3U") {
		t.Errorf("Missing M3U header")
	}
	if !strings.Contains(sContent, "Artist A/Album 1/01 Track 1.flac") {
		t.Errorf("Expected relative path not found in playlist: %s", sContent)
	}
}

func TestPlaylistGenerator_GenerateFromTracks(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		DownloadsDir:   tmpDir,
		SubdirTemplate: "{{.AlbumArtist}}/{{.Album}}/{{.Track}} {{.Title}}",
	}

	pg := NewPlaylistGenerator(cfg)

	tracks := []domain.CatalogTrack{
		{
			ID:          "t1",
			Title:       "Track 1",
			Artist:      "Famous Artist",
			Album:       "Greatest Hits",
			Year:        2020,
			TrackNumber: 5,
			Duration:    200,
		},
	}

	lookup := func(id string) string { return ".mp3" }

	err := pg.GenerateFromTracks("Famous Artist", tracks, lookup)
	if err != nil {
		t.Fatalf("GenerateFromTracks failed: %v", err)
	}

	playlistPath := filepath.Join(tmpDir, "playlists", "Famous Artist - Top Tracks.m3u")
	if _, err := os.Stat(playlistPath); os.IsNotExist(err) {
		t.Fatalf("Playlist file not created")
	}
}
