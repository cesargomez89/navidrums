package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/filesystem"
	"github.com/cesargomez89/navidrums/internal/models"
)

// PlaylistGenerator handles M3U playlist file generation
type PlaylistGenerator interface {
	Generate(pl *models.Playlist) error
}

type playlistGenerator struct {
	config *config.Config
}

// NewPlaylistGenerator creates a new PlaylistGenerator service
func NewPlaylistGenerator(cfg *config.Config) PlaylistGenerator {
	return &playlistGenerator{
		config: cfg,
	}
}

// Generate creates an M3U playlist file
func (pg *playlistGenerator) Generate(pl *models.Playlist) error {
	if len(pl.Tracks) == 0 {
		return nil
	}

	playlistsDir := filepath.Join(pg.config.DownloadsDir, "playlists")
	if err := os.MkdirAll(playlistsDir, 0755); err != nil {
		return fmt.Errorf("failed to create playlists directory: %w", err)
	}

	filename := filesystem.Sanitize(pl.Title) + ".m3u"
	playlistPath := filepath.Join(playlistsDir, filename)

	f, err := os.Create(playlistPath)
	if err != nil {
		return fmt.Errorf("failed to create playlist file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString("#EXTM3U\n"); err != nil {
		return fmt.Errorf("failed to write playlist header: %w", err)
	}

	for _, t := range pl.Tracks {
		folderName := fmt.Sprintf("%s - %s", filesystem.Sanitize(t.Artist), filesystem.Sanitize(t.Album))
		trackFile := fmt.Sprintf("%02d - %s.flac", t.TrackNumber, filesystem.Sanitize(t.Title))
		// Path relative to 'playlists' folder: ../Artist - Album/01 - Title.flac
		relPath := filepath.Join("..", folderName, trackFile)

		line := fmt.Sprintf("#EXTINF:%d,%s - %s\n%s\n", t.Duration, t.Artist, t.Title, relPath)
		if _, err := f.WriteString(line); err != nil {
			return fmt.Errorf("failed to write track to playlist: %w", err)
		}
	}

	return nil
}
