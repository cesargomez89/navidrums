package app

import (
	"fmt"
	"path/filepath"

	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/storage"
	"github.com/cesargomez89/navidrums/internal/store"
)

const defaultLimit = 30

type DownloadsService struct {
	Repo *store.DB
}

func NewDownloadsService(repo *store.DB) *DownloadsService {
	return &DownloadsService{Repo: repo}
}

func (s *DownloadsService) ListDownloads() ([]*domain.Track, error) {
	return s.Repo.ListCompletedTracks(defaultLimit)
}

func (s *DownloadsService) SearchDownloads(query string) ([]*domain.Track, error) {
	return s.Repo.SearchTracks(query, defaultLimit)
}

func (s *DownloadsService) DeleteDownload(providerID string) error {
	track, err := s.Repo.GetDownloadedTrack(providerID)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}
	if track == nil {
		return nil
	}

	if err := storage.RemoveFile(track.FilePath); err != nil {
		if !storage.IsNotExist(err) {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}

	folderPath := filepath.Dir(track.FilePath)
	if err := storage.DeleteFolderIfEmpty(folderPath); err != nil {
		return fmt.Errorf("failed to clean up folder: %w", err)
	}

	if err := s.Repo.DeleteTrack(track.ID); err != nil {
		return fmt.Errorf("failed to delete track record: %w", err)
	}

	return nil
}
