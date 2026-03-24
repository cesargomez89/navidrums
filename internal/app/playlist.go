package app

import (
	"fmt"
	"path/filepath"

	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/storage"
	"github.com/cesargomez89/navidrums/internal/store"
)

type TrackLookupFunc func(trackID string) *domain.Track

type PlaylistGenerator interface {
	Generate(pl *domain.Playlist, lookup TrackLookupFunc) error
	GenerateFromTracks(artistName string, tracks []domain.CatalogTrack, lookup TrackLookupFunc) error
	GenerateFromDB(playlistID int64, lookup TrackLookupFunc) error
}

type playlistGenerator struct {
	config *config.Config
	Repo   *store.DB
}

func NewPlaylistGenerator(cfg *config.Config, repo *store.DB) PlaylistGenerator {
	return &playlistGenerator{
		config: cfg,
		Repo:   repo,
	}
}

func (pg *playlistGenerator) GenerateFromDB(playlistID int64, lookup TrackLookupFunc) error {
	playlist, err := pg.Repo.GetPlaylistByID(playlistID)
	if err != nil {
		return fmt.Errorf("failed to get playlist: %w", err)
	}

	tracks, err := pg.Repo.GetTracksByPlaylistID(playlistID)
	if err != nil {
		return fmt.Errorf("failed to get playlist tracks: %w", err)
	}

	catalogTracks := make([]domain.CatalogTrack, len(tracks))
	for i, t := range tracks {
		catalogTracks[i] = domain.CatalogTrack{
			ID:          t.ProviderID,
			Title:       t.Title,
			Artist:      t.Artist,
			Album:       t.Album,
			AlbumArtist: t.AlbumArtist,
			Duration:    t.Duration,
		}
	}

	filename := fmt.Sprintf("%s - %s.m3u", storage.Sanitize(playlist.Title), storage.Sanitize(playlist.ProviderID))
	return pg.writePlaylist(filename, playlist.Title, catalogTracks, lookup)
}

func (pg *playlistGenerator) Generate(pl *domain.Playlist, lookup TrackLookupFunc) error {
	var filename string
	if pl.ProviderID != "" {
		filename = fmt.Sprintf("%s - %s.m3u", storage.Sanitize(pl.Title), storage.Sanitize(pl.ProviderID))
	} else {
		filename = storage.Sanitize(pl.Title) + ".m3u"
	}
	return pg.writePlaylist(filename, pl.Title, pl.Tracks, lookup)
}

func (pg *playlistGenerator) GenerateFromTracks(artistName string, tracks []domain.CatalogTrack, lookup TrackLookupFunc) error {
	filename := fmt.Sprintf("%s - Top Tracks.m3u", storage.Sanitize(artistName))
	return pg.writePlaylist(filename, fmt.Sprintf("%s - Top Tracks", artistName), tracks, lookup)
}

func (pg *playlistGenerator) writePlaylist(filename string, title string, tracks []domain.CatalogTrack, lookup TrackLookupFunc) error {
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
	writeErr := error(nil)
	defer func() {
		_ = f.Close()
		if writeErr != nil {
			_ = storage.RemoveFile(playlistPath)
		}
	}()

	if _, err := f.WriteString("#EXTM3U\n"); err != nil {
		writeErr = fmt.Errorf("failed to write playlist header: %w", err)
		return writeErr
	}

	if title != "" {
		if _, err := fmt.Fprintf(f, "#PLAYLIST:%s\n", title); err != nil {
			writeErr = fmt.Errorf("failed to write playlist title: %w", err)
			return writeErr
		}
	}

	for _, t := range tracks {
		var relPath string
		var err error

		dbTrack := lookup(t.ID)

		if dbTrack != nil && dbTrack.Status == domain.TrackStatusCompleted && dbTrack.FilePath != "" {
			// If track is already downloaded, use its exact file path relative to playlists dir
			// FilePath is absolute. DownloadsDir is absolute.
			// The playlist is in <DownloadsDir>/playlists/
			// We can figure out the relative path easily:
			rel, relErr := filepath.Rel(playlistsDir, dbTrack.FilePath)
			if relErr == nil {
				relPath = rel
			}
		}

		if relPath == "" {
			var templateData *storage.PathTemplateData
			if dbTrack != nil {
				artistForFolder := dbTrack.PathArtist
				if artistForFolder == "" {
					artistForFolder = dbTrack.AlbumArtist
				}
				if artistForFolder == "" {
					artistForFolder = dbTrack.Artist
				}
				templateData = storage.BuildPathTemplateData(
					artistForFolder,
					dbTrack.Year,
					dbTrack.Album,
					dbTrack.DiscNumber,
					dbTrack.TrackNumber,
					dbTrack.Title,
				)
			} else {
				artistForFolder := t.AlbumArtist
				if artistForFolder == "" {
					artistForFolder = t.Artist
				}
				templateData = storage.BuildPathTemplateData(
					artistForFolder,
					t.Year,
					t.Album,
					t.DiscNumber,
					t.TrackNumber,
					t.Title,
				)
			}

			relPath, err = storage.BuildPath(pg.config.SubdirTemplate, templateData)
			if err != nil {
				// Fallback to old behavior if template fails
				folderName := fmt.Sprintf("%s - %s", storage.Sanitize(t.Artist), storage.Sanitize(t.Album))
				trackFile := fmt.Sprintf("%02d - %s%s", t.TrackNumber, storage.Sanitize(t.Title), ".flac")
				relPath = filepath.Join("..", folderName, trackFile)
			} else {
				ext := ".flac"
				if dbTrack != nil && dbTrack.FileExtension != "" {
					ext = dbTrack.FileExtension
				}
				// Format with path.Join to ensure forward slashes, but filepath.Join is what was used
				relPath = filepath.Join("..", relPath+ext)
			}
		}

		line := fmt.Sprintf("#EXTINF:%d,%s - %s\n%s\n", t.Duration, t.Artist, t.Title, filepath.ToSlash(relPath))
		if _, err := f.WriteString(line); err != nil {
			writeErr = fmt.Errorf("failed to write track to playlist: %w", err)
			return writeErr
		}
	}

	return nil
}
