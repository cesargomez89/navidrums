package app

import (
	"fmt"
	"path/filepath"

	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/storage"
	"github.com/cesargomez89/navidrums/internal/tagging"
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

	folderName := fmt.Sprintf("%s - %s", storage.Sanitize(album.Artist), storage.Sanitize(album.Title))
	albumDir := filepath.Join(s.config.DownloadsDir, folderName)

	if err := storage.EnsureDir(albumDir); err != nil {
		return fmt.Errorf("failed to create album directory: %w", err)
	}

	imagePath := filepath.Join(albumDir, "cover.jpg")
	if err := tagging.SaveImageToFile(imageData, imagePath); err != nil {
		return fmt.Errorf("failed to save album art: %w", err)
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
	if err := tagging.SaveImageToFile(imageData, imagePath); err != nil {
		return fmt.Errorf("failed to save playlist image: %w", err)
	}

	return nil
}

func (s *albumArtService) DownloadImage(url string) ([]byte, error) {
	return tagging.DownloadImage(url)
}
