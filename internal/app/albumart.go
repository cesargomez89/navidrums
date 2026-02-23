package app

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/constants"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/storage"
)

type AlbumArtService interface {
	DownloadAndSaveAlbumArt(album *domain.Album, imageURL string) error
	DownloadAndSavePlaylistImage(pl *domain.Playlist, imageURL string) error
	DownloadImage(url string) ([]byte, error)
}

type albumArtService struct {
	config *config.Config
}

func NewAlbumArtService(cfg *config.Config) AlbumArtService {
	return &albumArtService{
		config: cfg,
	}
}

func (s *albumArtService) DownloadAndSaveAlbumArt(album *domain.Album, imageURL string) error {
	if imageURL == "" {
		return nil
	}

	imageData, err := s.DownloadImage(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download album art: %w", err)
	}

	// Generate album directory using the same template as tracks
	// Use first track's metadata if available, otherwise use album metadata with defaults
	artist := album.Artist

	year := album.Year
	if year == 0 && album.ReleaseDate != "" {
		// Try to parse year from release date (YYYY-MM-DD format)
		if len(album.ReleaseDate) >= 4 {
			var parsedYear int
			_, _ = fmt.Sscanf(album.ReleaseDate[:4], "%d", &parsedYear)
			year = parsedYear
		}
	}

	// Build template data with sensible defaults for disc/track
	templateData := storage.BuildPathTemplateData(
		artist,
		year,
		album.Title,
		1,       // Default to disc 1
		1,       // Default to track 1 (for folder creation purposes)
		"cover", // Placeholder title (won't be used since we just want the folder)
	)

	// Get the full path and extract just the directory portion
	fullPathNoExt, err := storage.BuildPath(s.config.SubdirTemplate, templateData)
	if err != nil {
		return fmt.Errorf("failed to build album path from template: %w", err)
	}

	fullPathNoExt = filepath.Join(s.config.DownloadsDir, fullPathNoExt)
	albumDir := filepath.Dir(fullPathNoExt)

	if err := storage.EnsureDir(albumDir); err != nil {
		return fmt.Errorf("failed to create album directory: %w", err)
	}

	imagePath := filepath.Join(albumDir, "cover.jpg")
	if len(imageData) > 0 {
		if err := storage.EnsureDir(filepath.Dir(imagePath)); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		if err := storage.WriteFile(imagePath, imageData); err != nil {
			return fmt.Errorf("failed to save album art: %w", err)
		}
	}

	return nil
}

func (s *albumArtService) DownloadAndSavePlaylistImage(pl *domain.Playlist, imageURL string) error {
	if imageURL == "" {
		return nil
	}

	imageData, err := s.DownloadImage(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download playlist image: %w", err)
	}

	playlistsDir := filepath.Join(s.config.DownloadsDir, "playlists")

	if err := storage.EnsureDir(playlistsDir); err != nil {
		return fmt.Errorf("failed to create playlists directory: %w", err)
	}

	imagePath := filepath.Join(playlistsDir, storage.Sanitize(pl.Title)+".jpg")
	if len(imageData) > 0 {
		if err := storage.WriteFile(imagePath, imageData); err != nil {
			return fmt.Errorf("failed to save playlist image: %w", err)
		}
	}

	return nil
}

func (s *albumArtService) DownloadImage(urlStr string) ([]byte, error) {
	if urlStr == "" {
		return nil, nil
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid image URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL scheme: %s (only http/https allowed)", parsedURL.Scheme)
	}

	client := &http.Client{Timeout: constants.ImageHTTPTimeout}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: status %d (URL: %s)", resp.StatusCode, urlStr)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}
	return buf.Bytes(), nil
}
