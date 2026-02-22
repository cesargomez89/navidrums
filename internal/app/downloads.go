package app

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/logger"
	"github.com/cesargomez89/navidrums/internal/storage"
	"github.com/cesargomez89/navidrums/internal/store"
)

const defaultLimit = 30

type DownloadsService struct {
	Repo   *store.DB
	Logger *logger.Logger
}

func NewDownloadsService(repo *store.DB, log *logger.Logger) *DownloadsService {
	return &DownloadsService{Repo: repo, Logger: log}
}

func (s *DownloadsService) ListDownloads() ([]*domain.Track, error) {
	return s.Repo.ListCompletedTracks(defaultLimit)
}

func (s *DownloadsService) SearchDownloads(query string) ([]*domain.Track, error) {
	return s.Repo.SearchTracks(query, defaultLimit)
}

func (s *DownloadsService) GetTrackByID(id int) (*domain.Track, error) {
	return s.Repo.GetTrackByID(id)
}

func (s *DownloadsService) UpdateTrackPartial(id int, updates map[string]interface{}) error {
	return s.Repo.UpdateTrackPartial(id, updates)
}

func (s *DownloadsService) EnqueueSyncFileJob(providerID string) error {
	job := &domain.Job{
		ID:        uuid.New().String(),
		Type:      domain.JobTypeSyncFile,
		Status:    domain.JobStatusQueued,
		SourceID:  providerID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return s.Repo.CreateJob(job)
}

func (s *DownloadsService) EnqueueSyncMetadataJob(providerID string) error {
	job := &domain.Job{
		ID:        uuid.New().String(),
		Type:      domain.JobTypeSyncMusicBrainz,
		Status:    domain.JobStatusQueued,
		SourceID:  providerID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return s.Repo.CreateJob(job)
}

func (s *DownloadsService) EnqueueSyncHiFiJob(providerID string) error {
	job := &domain.Job{
		ID:        uuid.New().String(),
		Type:      domain.JobTypeSyncHiFi,
		Status:    domain.JobStatusQueued,
		SourceID:  providerID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return s.Repo.CreateJob(job)
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
	if err := storage.DeleteFolderWithCover(folderPath); err != nil {
		return fmt.Errorf("failed to clean up folder: %w", err)
	}

	albumPath := filepath.Dir(folderPath)
	if err := storage.DeleteFolderIfEmpty(albumPath); err != nil {
		return fmt.Errorf("failed to clean up album folder: %w", err)
	}

	artistPath := filepath.Dir(albumPath)
	if err := storage.DeleteFolderIfEmpty(artistPath); err != nil {
		return fmt.Errorf("failed to clean up artist folder: %w", err)
	}

	if err := s.Repo.DeleteTrack(track.ID); err != nil {
		return fmt.Errorf("failed to delete track record: %w", err)
	}

	s.Logger.Info("Download deleted", "provider_id", providerID, "file_path", track.FilePath)
	return nil
}

func (s *DownloadsService) EnqueueSyncJobs() (int, error) {
	tracks, err := s.Repo.ListCompletedTracks(defaultLimit)
	if err != nil {
		return 0, fmt.Errorf("failed to list tracks: %w", err)
	}

	count := 0
	for _, track := range tracks {
		existing, _ := s.Repo.GetActiveJobBySourceID(track.ProviderID, domain.JobTypeSyncHiFi)
		if existing != nil {
			continue
		}

		job := &domain.Job{
			ID:        uuid.New().String(),
			Type:      domain.JobTypeSyncHiFi,
			Status:    domain.JobStatusQueued,
			SourceID:  track.ProviderID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.Repo.CreateJob(job); err != nil {
			s.Logger.Error("Failed to create sync job", "track_id", track.ID, "error", err)
			continue
		}
		count++
	}

	return count, nil
}
