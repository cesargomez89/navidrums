package app

import (
	"fmt"
	"path/filepath"

	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/storage"
	"github.com/cesargomez89/navidrums/internal/store"
)

type DownloadsService struct {
	Repo *store.DB
}

func NewDownloadsService(repo *store.DB) *DownloadsService {
	return &DownloadsService{Repo: repo}
}

func (s *DownloadsService) ListDownloads() ([]*domain.Download, error) {
	return s.Repo.ListDownloads(30)
}

func (s *DownloadsService) SearchDownloads(query string) ([]*domain.Download, error) {
	return s.Repo.SearchDownloads(query, 30)
}

func (s *DownloadsService) DeleteDownload(providerID string) error {
	download, err := s.Repo.GetDownload(providerID)
	if err != nil {
		return fmt.Errorf("failed to get download: %w", err)
	}
	if download == nil {
		return nil
	}

	if err := storage.RemoveFile(download.FilePath); err != nil {
		if !storage.IsNotExist(err) {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}

	folderPath := filepath.Dir(download.FilePath)
	if err := storage.DeleteFolderIfEmpty(folderPath); err != nil {
		return fmt.Errorf("failed to clean up folder: %w", err)
	}

	if err := s.Repo.DeleteDownload(providerID); err != nil {
		return fmt.Errorf("failed to delete download record: %w", err)
	}

	return nil
}
