package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/filesystem"
	"github.com/cesargomez89/navidrums/internal/models"
	"github.com/cesargomez89/navidrums/internal/tagging"
)

// AlbumArtService handles downloading and saving album artwork
type AlbumArtService interface {
	DownloadAndSaveAlbumArt(album *models.Album, imageURL string) error
	DownloadAndSavePlaylistImage(pl *models.Playlist, imageURL string) error
	DownloadImage(url string) ([]byte, error)
}

type albumArtService struct {
	config *config.Config
}

// NewAlbumArtService creates a new AlbumArtService
func NewAlbumArtService(cfg *config.Config) AlbumArtService {
	return &albumArtService{
		config: cfg,
	}
}

// DownloadAndSaveAlbumArt downloads and saves album artwork to the album folder
func (s *albumArtService) DownloadAndSaveAlbumArt(album *models.Album, imageURL string) error {
	if imageURL == "" {
		return nil
	}

	// Download image
	imageData, err := s.DownloadImage(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download album art: %w", err)
	}

	// Determine folder path
	folderName := fmt.Sprintf("%s - %s", filesystem.Sanitize(album.Artist), filesystem.Sanitize(album.Title))
	albumDir := filepath.Join(s.config.DownloadsDir, folderName)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(albumDir, 0755); err != nil {
		return fmt.Errorf("failed to create album directory: %w", err)
	}

	// Save image
	imagePath := filepath.Join(albumDir, "cover.jpg")
	if err := tagging.SaveImageToFile(imageData, imagePath); err != nil {
		return fmt.Errorf("failed to save album art: %w", err)
	}

	return nil
}

// DownloadAndSavePlaylistImage downloads and saves playlist cover image
func (s *albumArtService) DownloadAndSavePlaylistImage(pl *models.Playlist, imageURL string) error {
	if imageURL == "" {
		return nil
	}

	// Download image
	imageData, err := s.DownloadImage(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download playlist image: %w", err)
	}

	// Determine folder path
	playlistsDir := filepath.Join(s.config.DownloadsDir, "playlists")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(playlistsDir, 0755); err != nil {
		return fmt.Errorf("failed to create playlists directory: %w", err)
	}

	// Save image
	imagePath := filepath.Join(playlistsDir, filesystem.Sanitize(pl.Title)+".jpg")
	if err := tagging.SaveImageToFile(imageData, imagePath); err != nil {
		return fmt.Errorf("failed to save playlist image: %w", err)
	}

	return nil
}

// DownloadImage downloads an image from a URL and returns the image data
func (s *albumArtService) DownloadImage(url string) ([]byte, error) {
	return tagging.DownloadImage(url)
}
