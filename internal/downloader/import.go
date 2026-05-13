package downloader

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/cesargomez89/navidrums/internal/app"
	"github.com/cesargomez89/navidrums/internal/catalog"
	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/constants"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/storage"
	"github.com/cesargomez89/navidrums/internal/store"
	"github.com/cesargomez89/navidrums/internal/tagging"
)

type ImportJobHandler struct {
	Repo            *store.DB
	Config          *config.Config
	ProviderManager *catalog.ProviderManager
	AlbumArtService app.AlbumArtService
	Enricher        *app.MetadataEnricher
}

func (h *ImportJobHandler) Handle(ctx context.Context, job *domain.Job, logger *slog.Logger) error {
	filePath := job.GetSourceID()
	if filePath == "" {
		return h.failJob(job, "source_id is empty", logger)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return h.failJob(job, fmt.Sprintf("file not found: %s", filePath), logger)
	}

	ffprobePath := h.Config.FFprobePath
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}
	track, err := app.ExtractTrackFromFile(filePath, ffprobePath)
	if err != nil {
		logger.Warn("Tag extraction had issues, proceeding with partial data", "error", err)
	}

	track.ParentJobID = job.ID

	provider := h.ProviderManager.GetProvider()
	if track.Title != "" && track.Artist != "" {
		searchQuery := fmt.Sprintf("\"%s\" \"%s\"", track.Artist, track.Title)
		result, searchErr := provider.Search(ctx, searchQuery, "tracks")
		if searchErr == nil && len(result.Tracks) > 0 {
			match := result.Tracks[0]
			track.ProviderID = match.ID
			h.Enricher.UpdateTrackFromCatalog(track, &match, logger)

			existing, lookupErr := h.Repo.GetTrackByProviderID(track.ProviderID)
			if lookupErr == nil && existing != nil {
				if existing.Status == domain.TrackStatusCompleted {
					logger.Info("Track already imported, moving file to final destination", "provider_id", track.ProviderID)
					if moveErr := h.moveToFinalDestination(track, filePath, logger); moveErr != nil {
						return h.failJob(job, fmt.Sprintf("failed to move file: %v", moveErr), logger)
					}
					if err := storage.RemoveFile(filePath); err != nil {
						logger.Warn("Failed to remove source file", "path", filePath, "error", err)
					}
					_ = h.cleanupSourceDir(filePath)
					_ = h.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100)
					return nil
				}
				track.ID = existing.ID
				track.CreatedAt = existing.CreatedAt
			}

			h.Enricher.EnrichComplete(ctx, track, logger)
		}
	}

	if track.ID == 0 {
		if err := h.Repo.CreateTrack(track); err != nil {
			return h.failJob(job, fmt.Sprintf("failed to create track: %v", err), logger)
		}
	} else {
		if err := h.Repo.UpdateTrack(track); err != nil {
			return h.failJob(job, fmt.Sprintf("failed to update track: %v", err), logger)
		}
	}

	if err := h.moveToFinalDestination(track, filePath, logger); err != nil {
		return h.failJob(job, fmt.Sprintf("failed to move file: %v", err), logger)
	}

	if err := h.Repo.UpdateTrack(track); err != nil {
		logger.Error("Failed to update track after move", "error", err)
	}

	if err := h.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100); err != nil {
		logger.Error("Failed to update job status", "error", err)
	}
	logger.Info("Import job completed", "track_id", track.ID, "path", track.FilePath)
	return nil
}

func (h *ImportJobHandler) moveToFinalDestination(track *domain.Track, sourcePath string, logger *slog.Logger) error {
	track.FileExtension = filepath.Ext(sourcePath)

	artistForFolder := track.PathArtist
	if artistForFolder == "" {
		artistForFolder = track.AlbumArtist
	}
	if artistForFolder == "" {
		artistForFolder = track.Artist
	}

	templateData := storage.BuildPathTemplateData(
		artistForFolder,
		track.Year,
		track.Album,
		track.DiscNumber,
		track.TrackNumber,
		track.Title,
	)

	finalPath, err := storage.BuildFullPath(h.Config.DownloadsDir, h.Config.SubdirTemplate, templateData, track.FileExtension)
	if err != nil {
		return fmt.Errorf("failed to build path: %w", err)
	}

	finalDir := filepath.Dir(finalPath)
	if err := storage.EnsureDir(finalDir); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	var albumArtData []byte
	if track.AlbumArtURL != "" {
		var dlErr error
		albumArtData, dlErr = h.AlbumArtService.DownloadImage(track.AlbumArtURL)
		if dlErr != nil {
			logger.Warn("Failed to download album art", "url", track.AlbumArtURL, "error", dlErr)
		}
	}

	if tagErr := tagging.TagFile(sourcePath, track, albumArtData); tagErr != nil {
		logger.Warn("Tagging had issues", "error", tagErr)
	}

	if err := storage.MoveFile(sourcePath, finalPath); err != nil {
		return fmt.Errorf("failed to move file from %s to %s: %w", sourcePath, finalPath, err)
	}

	h.maybeImportCoverArt(sourcePath, finalDir, logger)

	if err := h.cleanupSourceDir(sourcePath); err != nil {
		logger.Warn("Failed to clean up source directory", "error", err)
	}

	fileHash, hashErr := storage.HashFile(finalPath)
	if hashErr != nil {
		logger.Warn("Failed to hash file", "error", hashErr)
	}

	now := time.Now()
	track.FilePath = finalPath
	track.FileHash = fileHash
	track.Status = domain.TrackStatusCompleted
	track.CompletedAt = &now
	track.LastVerifiedAt = &now
	track.UpdatedAt = now

	return nil
}

func (h *ImportJobHandler) maybeImportCoverArt(sourcePath string, finalDir string, logger *slog.Logger) {
	sourceDir := filepath.Dir(sourcePath)
	sourceCover := filepath.Join(sourceDir, constants.CoverFileName)
	if _, err := os.Stat(sourceCover); os.IsNotExist(err) {
		return
	}
	destCover := filepath.Join(finalDir, constants.CoverFileName)
	if err := storage.CopyFile(sourceCover, destCover); err != nil {
		logger.Warn("Failed to copy cover art", "src", sourceCover, "dst", destCover, "error", err)
	} else {
		logger.Info("Imported cover art", "path", destCover)
	}
}

func (h *ImportJobHandler) cleanupSourceDir(sourcePath string) error {
	sourceDir := filepath.Dir(sourcePath)
	return storage.DeleteFolderWithCover(sourceDir)
}

func (h *ImportJobHandler) failJob(job *domain.Job, msg string, logger *slog.Logger) error {
	logger.Error(msg)
	_ = h.Repo.UpdateJobError(job.ID, msg)
	return fmt.Errorf(msg)
}
