package app

import (
	"fmt"
	"path/filepath"

	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/storage"
)

type ExtensionLookupFunc func(trackID string) string

type PlaylistGenerator interface {
	Generate(pl *domain.Playlist, lookup ExtensionLookupFunc) error
	GenerateFromTracks(artistName string, tracks []domain.Track, lookup ExtensionLookupFunc) error
}

type playlistGenerator struct {
	config *config.Config
}

func NewPlaylistGenerator(cfg *config.Config) PlaylistGenerator {
	return &playlistGenerator{
		config: cfg,
	}
}

func (pg *playlistGenerator) Generate(pl *domain.Playlist, lookup ExtensionLookupFunc) error {
	filename := storage.Sanitize(pl.Title) + ".m3u"
	return pg.writePlaylist(filename, pl.Tracks, lookup)
}

func (pg *playlistGenerator) GenerateFromTracks(artistName string, tracks []domain.Track, lookup ExtensionLookupFunc) error {
	filename := fmt.Sprintf("%s - Top Tracks.m3u", storage.Sanitize(artistName))
	return pg.writePlaylist(filename, tracks, lookup)
}

func (pg *playlistGenerator) writePlaylist(filename string, tracks []domain.Track, lookup ExtensionLookupFunc) error {
	if len(tracks) == 0 {
		return nil
	}

	playlistsDir := filepath.Join(pg.config.DownloadsDir, "playlists")
	if err := storage.EnsureDir(playlistsDir); err != nil {
		return fmt.Errorf("failed to create playlists directory: %w", err)
	}

	playlistPath := filepath.Join(playlistsDir, filename)

	f, err := storage.CreateFile(playlistPath)
	if err != nil {
		return fmt.Errorf("failed to create playlist file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString("#EXTM3U\n"); err != nil {
		return fmt.Errorf("failed to write playlist header: %w", err)
	}

	for _, t := range tracks {
		folderName := fmt.Sprintf("%s - %s", storage.Sanitize(t.Artist), storage.Sanitize(t.Album))
		ext := lookup(t.ID)
		if ext == "" {
			ext = ".flac" // Default fallback
		}
		trackFile := fmt.Sprintf("%02d - %s%s", t.TrackNumber, storage.Sanitize(t.Title), ext)
		relPath := filepath.Join("..", folderName, trackFile)

		line := fmt.Sprintf("#EXTINF:%d,%s - %s\n%s\n", t.Duration, t.Artist, t.Title, relPath)
		if _, err := f.WriteString(line); err != nil {
			return fmt.Errorf("failed to write track to playlist: %w", err)
		}
	}

	return nil
}
