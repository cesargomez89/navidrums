package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/storage"
)

type PlaylistGenerator interface {
	Generate(pl *domain.Playlist) error
	GenerateFromTracks(artistName string, tracks []domain.Track) error
}

type playlistGenerator struct {
	config *config.Config
}

func NewPlaylistGenerator(cfg *config.Config) PlaylistGenerator {
	return &playlistGenerator{
		config: cfg,
	}
}

func (pg *playlistGenerator) Generate(pl *domain.Playlist) error {
	if len(pl.Tracks) == 0 {
		return nil
	}

	playlistsDir := filepath.Join(pg.config.DownloadsDir, "playlists")
	if err := os.MkdirAll(playlistsDir, 0755); err != nil {
		return fmt.Errorf("failed to create playlists directory: %w", err)
	}

	filename := storage.Sanitize(pl.Title) + ".m3u"
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
		folderName := fmt.Sprintf("%s - %s", storage.Sanitize(t.Artist), storage.Sanitize(t.Album))
		trackFile := fmt.Sprintf("%02d - %s.flac", t.TrackNumber, storage.Sanitize(t.Title))
		relPath := filepath.Join("..", folderName, trackFile)

		line := fmt.Sprintf("#EXTINF:%d,%s - %s\n%s\n", t.Duration, t.Artist, t.Title, relPath)
		if _, err := f.WriteString(line); err != nil {
			return fmt.Errorf("failed to write track to playlist: %w", err)
		}
	}

	return nil
}

func (pg *playlistGenerator) GenerateFromTracks(artistName string, tracks []domain.Track) error {
	if len(tracks) == 0 {
		return nil
	}

	playlistsDir := filepath.Join(pg.config.DownloadsDir, "playlists")
	if err := os.MkdirAll(playlistsDir, 0755); err != nil {
		return fmt.Errorf("failed to create playlists directory: %w", err)
	}

	filename := fmt.Sprintf("%s - Top Tracks.m3u", storage.Sanitize(artistName))
	playlistPath := filepath.Join(playlistsDir, filename)

	f, err := os.Create(playlistPath)
	if err != nil {
		return fmt.Errorf("failed to create playlist file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString("#EXTM3U\n"); err != nil {
		return fmt.Errorf("failed to write playlist header: %w", err)
	}

	for _, t := range tracks {
		folderName := fmt.Sprintf("%s - %s", storage.Sanitize(t.Artist), storage.Sanitize(t.Album))
		trackFile := fmt.Sprintf("%02d - %s.flac", t.TrackNumber, storage.Sanitize(t.Title))
		relPath := filepath.Join("..", folderName, trackFile)

		line := fmt.Sprintf("#EXTINF:%d,%s - %s\n%s\n", t.Duration, t.Artist, t.Title, relPath)
		if _, err := f.WriteString(line); err != nil {
			return fmt.Errorf("failed to write track to playlist: %w", err)
		}
	}

	return nil
}
