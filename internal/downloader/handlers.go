package downloader

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cesargomez89/navidrums/internal/app"
	"github.com/cesargomez89/navidrums/internal/catalog"
	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/constants"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/storage"
	"github.com/cesargomez89/navidrums/internal/store"
	"github.com/cesargomez89/navidrums/internal/tagging"
)

// TrackJobHandler handles downloading individual tracks.
type TrackJobHandler struct {
	Repo              *store.DB
	SettingsRepo      *store.SettingsRepo
	Config            *config.Config
	ProviderManager   *catalog.ProviderManager
	Downloader        app.Downloader
	AlbumArtService   app.AlbumArtService
	PlaylistGenerator app.PlaylistGenerator
	Enricher          *app.MetadataEnricher
	m3uLocks          sync.Map
}

func (h *TrackJobHandler) Handle(ctx context.Context, job *domain.Job, logger *slog.Logger) error {
	track, destPath, skipDownload, err := h.prepareTrackDownload(ctx, job, logger)
	if err != nil {
		return err
	}

	if skipDownload {
		return nil
	}

	if h.isCancelled(job.ID) {
		logger.Info("Job cancelled before download")
		return nil
	}

	finalPath, err := h.executeDownload(ctx, job, track, destPath, logger)
	if err != nil {
		return err
	}

	if err := h.postProcessTrack(ctx, track, finalPath, logger); err != nil {
		logger.Warn("Post-processing had issues", "error", err)
	}

	h.finalizeTrackDownload(job, track, finalPath, logger)
	return nil
}

func (h *TrackJobHandler) prepareTrackDownload(ctx context.Context, job *domain.Job, logger *slog.Logger) (*domain.Track, string, bool, error) {
	forceDownload := h.isForceDownload()

	existingTrack, _ := h.Repo.GetTrackByProviderID(job.GetSourceID())
	if existingTrack != nil && existingTrack.Status == domain.TrackStatusCompleted && !forceDownload {
		logger.Info("Track already downloaded", "file_path", existingTrack.FilePath)
		_ = h.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100)
		return nil, "", true, nil
	}

	var track *domain.Track
	if existingTrack != nil {
		track = existingTrack
	} else {
		track = &domain.Track{
			ProviderID:  job.GetSourceID(),
			Status:      domain.TrackStatusMissing,
			ParentJobID: job.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	h.Enricher.EnrichComplete(ctx, track, logger)

	if track.Title == "" && track.Artist == "" {
		err := fmt.Errorf("failed to fetch primary track metadata")
		logger.Error(err.Error())
		_ = h.Repo.UpdateJobError(job.ID, err.Error())
		return nil, "", false, err
	}

	if existingTrack != nil {
		if err := h.Repo.UpdateTrack(track); err != nil {
			logger.Warn("Failed to update track after enrichment", "error", err)
		}
	} else {
		if err := h.Repo.CreateTrack(track); err != nil {
			logger.Error("Failed to create track record", "error", err)
			_ = h.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to create track record: %v", err))
			return nil, "", false, err
		}
	}

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

	fullPathNoExt, err := storage.BuildPath(h.Config.SubdirTemplate, templateData)
	if err != nil {
		logger.Error("Failed to build path from template", "error", err)
		_ = h.Repo.MarkTrackFailed(track.ID, fmt.Sprintf("Failed to build path: %v", err))
		_ = h.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to build path: %v", err))
		return nil, "", false, err
	}

	fullPathNoExt = filepath.Join(h.Config.DownloadsDir, fullPathNoExt)

	ext := track.FileExtension
	if ext == "" {
		ext = ".flac"
	}
	predictedPath := fullPathNoExt + ext

	if track.Status == domain.TrackStatusCompleted && !forceDownload {
		exists := false
		if _, statErr := os.Stat(predictedPath); statErr == nil {
			exists = true
		} else if track.FilePath != "" {
			if _, statErr := os.Stat(track.FilePath); statErr == nil {
				exists = true
				predictedPath = track.FilePath
			}
		}

		if exists {
			match := false
			if track.FileHash != "" {
				verified, _ := storage.VerifyFile(predictedPath, track.FileHash)
				if verified {
					match = true
				}
			} else {
				newHash, hashErr := storage.HashFile(predictedPath)
				if hashErr == nil {
					track.FileHash = newHash
					_ = h.Repo.UpdateTrack(track)
					match = true
				}
			}

			if match {
				logger.Info("Track already exists and verified, skipping download", "path", predictedPath)
				_ = h.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100)
				return nil, "", true, nil
			} else {
				logger.Info("Track exists but hash mismatch, redownloading", "path", predictedPath)
				_ = storage.RemoveFile(predictedPath)
			}
		}
	} else if track.Status == domain.TrackStatusCompleted && forceDownload {
		logger.Info("Force download enabled, deleting existing file", "path", predictedPath)
		_ = storage.RemoveFile(predictedPath)
		if track.FilePath != "" && track.FilePath != predictedPath {
			_ = storage.RemoveFile(track.FilePath)
		}
	}

	return track, fullPathNoExt, false, nil
}

func (h *TrackJobHandler) executeDownload(ctx context.Context, job *domain.Job, track *domain.Track, destPath string, logger *slog.Logger) (string, error) {
	if updateErr := h.Repo.UpdateTrackStatus(track.ID, domain.TrackStatusDownloading, ""); updateErr != nil {
		logger.Error("Failed to update track status to downloading", "error", updateErr)
		return "", updateErr
	}

	finalDir := filepath.Dir(destPath)
	if dirErr := storage.EnsureDir(finalDir); dirErr != nil {
		logger.Error("Failed to create directory", "error", dirErr)
		_ = h.Repo.MarkTrackFailed(track.ID, fmt.Sprintf("Failed to create directory: %v", dirErr))
		_ = h.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to create directory: %v", dirErr))
		return "", dirErr
	}

	quality := h.getQuality()
	finalPath, err := h.Downloader.Download(ctx, track, destPath, quality, logger)
	if err != nil {
		logger.Error("Download failed", "error", err)
		_ = h.Repo.MarkTrackFailed(track.ID, err.Error())
		_ = h.Repo.UpdateJobError(job.ID, err.Error())
		return "", err
	}

	if statusErr := h.Repo.UpdateTrackStatus(track.ID, domain.TrackStatusDownloaded, finalPath); statusErr != nil {
		logger.Error("Failed to update track status to downloaded", "error", statusErr)
	}

	logger.Info("Download finished, preparing for tagging", "file_path", finalPath)
	return finalPath, nil
}

func (h *TrackJobHandler) postProcessTrack(ctx context.Context, track *domain.Track, finalPath string, logger *slog.Logger) error {
	if statusErr := h.Repo.UpdateTrackStatus(track.ID, domain.TrackStatusProcessing, finalPath); statusErr != nil {
		logger.Error("Failed to update track status to processing", "error", statusErr)
	}

	var albumArtData []byte
	finalDir := filepath.Dir(finalPath)
	artPath := filepath.Join(finalDir, "cover.jpg")

	if data, err := os.ReadFile(artPath); err == nil && len(data) > 0 { //nolint:gosec
		albumArtData = data
	} else if track.AlbumArtURL != "" {
		var err error
		albumArtData, err = h.AlbumArtService.DownloadImage(track.AlbumArtURL)
		if err != nil {
			logger.Error("Failed to download album art for tagging", "error", err)
		}
	}

	if tagErr := tagging.TagFile(finalPath, track, albumArtData); tagErr != nil {
		if errors.Is(tagErr, tagging.ErrUnsupportedFormat) {
			logger.Warn("Tagging skipped: unsupported format", "file_path", finalPath, "error", tagErr)
		} else {
			logger.Error("Failed to tag file", "file_path", finalPath, "error", tagErr)
		}
	}

	if len(albumArtData) > 0 {
		if _, artStatErr := os.Stat(artPath); os.IsNotExist(artStatErr) {
			if writeErr := storage.WriteFile(artPath, albumArtData); writeErr != nil {
				logger.Error("Failed to save album art", "path", artPath, "error", writeErr)
			} else {
				logger.Info("Saved album art", "path", artPath)
			}
		}
	}

	logger.Info("File finalized", "original_path", finalPath)
	return nil
}

func (h *TrackJobHandler) finalizeTrackDownload(job *domain.Job, track *domain.Track, finalPath string, logger *slog.Logger) {
	fileHash, err := storage.HashFile(finalPath)
	if err != nil {
		logger.Error("Failed to hash file", "error", err)
	}

	ext := filepath.Ext(finalPath)
	if ext == "" {
		ext = ".flac"
	}
	track.FileExtension = ext
	track.Status = domain.TrackStatusCompleted
	track.FilePath = finalPath
	track.FileHash = fileHash
	now := time.Now()
	track.CompletedAt = &now
	track.LastVerifiedAt = &now
	track.UpdatedAt = time.Now()

	if err := h.Repo.UpdateTrack(track); err != nil {
		logger.Error("Failed to update track", "error", err)
	}

	if track.AlbumID != "" {
		_, _ = h.Repo.RecomputeAlbumState(track.AlbumID)
	}

	if err := h.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100); err != nil {
		logger.Error("Failed to update final job status", "error", err)
	}

	if track.ParentJobID != "" {
		parentJob, err := h.Repo.GetJob(track.ParentJobID)
		if err == nil && parentJob != nil && parentJob.Type == domain.JobTypePlaylist {
			playlist, err := h.Repo.GetPlaylistByProviderID(parentJob.GetSourceID())
			if err == nil && playlist != nil {
				if err := h.Repo.AddTrackToPlaylist(playlist.ID, track.ID, track.TrackNumber); err != nil {
					logger.Warn("Failed to add track to playlist", "error", err)
				}
			}
		}

		h.updateParentJobProgress(track.ParentJobID, logger)
		h.triggerPlaylistGenerationIfComplete(track.ParentJobID, logger)
	}

	logger.Info("Job completed successfully")
}

func (h *TrackJobHandler) triggerPlaylistGenerationIfComplete(parentJobID string, logger *slog.Logger) {
	parentJob, err := h.Repo.GetJob(parentJobID)
	if err != nil || parentJob == nil {
		return
	}

	if parentJob.Status != domain.JobStatusCompleted && parentJob.Status != domain.JobStatusDecomposed {
		return
	}

	total, pending, err := h.Repo.CountJobsForParent(parentJobID)
	if err != nil || total == 0 || pending > 0 {
		return
	}

	if parentJob.Type != domain.JobTypePlaylist && parentJob.Type != domain.JobTypeArtist {
		return
	}

	_, loaded := h.m3uLocks.LoadOrStore(parentJobID, struct{}{})
	if loaded {
		return
	}

	defer h.m3uLocks.Delete(parentJobID)

	switch parentJob.Type {
	case domain.JobTypePlaylist:
		playlist, err := h.Repo.GetPlaylistByProviderID(parentJob.GetSourceID())
		if err == nil && playlist != nil {
			if genErr := h.PlaylistGenerator.GenerateFromDB(playlist.ID, lookupTrack(h.Repo)); genErr != nil {
				logger.Error("Failed to generate complete playlist from DB", "error", genErr)
			} else {
				logger.Info("Successfully generated complete playlist from DB", "playlist_id", playlist.ID)
			}
		}
	case domain.JobTypeArtist:
		tracks, err := h.Repo.ListTracksByParentJobID(parentJobID)
		if err == nil && len(tracks) > 0 {
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
			artistName := tracks[0].AlbumArtist
			if artistName == "" {
				artistName = tracks[0].Artist
			}
			if genErr := h.PlaylistGenerator.GenerateFromTracks(artistName, catalogTracks, lookupTrack(h.Repo)); genErr != nil {
				logger.Error("Failed to generate complete artist playlist", "error", genErr)
			} else {
				logger.Info("Successfully generated complete artist playlist", "artist", artistName)
			}
		}
	}
}

func lookupTrack(repo *store.DB) func(string) *domain.Track {
	return func(trackID string) *domain.Track {
		t, _ := repo.GetTrackByProviderID(trackID)
		return t
	}
}

func (h *TrackJobHandler) updateParentJobProgress(parentJobID string, logger *slog.Logger) {
	total, pending, err := h.Repo.CountJobsForParent(parentJobID)
	if err != nil {
		logger.Error("Failed to count jobs for parent", "parent_job", parentJobID, "error", err)
		return
	}

	if total == 0 {
		return
	}

	progress := float64(total-pending) / float64(total) * 100
	if err := h.Repo.UpdateJobProgress(parentJobID, progress); err != nil {
		logger.Error("Failed to update parent job progress", "parent_job", parentJobID, "error", err)
	}

	if pending == 0 {
		if err := h.Repo.UpdateJobStatus(parentJobID, domain.JobStatusCompleted, 100); err != nil {
			logger.Error("Failed to mark parent job as completed", "parent_job", parentJobID, "error", err)
		}
	}
}

func (h *TrackJobHandler) isCancelled(id string) bool {
	job, err := h.Repo.GetJob(id)
	if err != nil {
		return false
	}
	return job.Status == domain.JobStatusCancelled
}

// ContainerJobHandler handles albums, playlists, and artists by decomposing them into track jobs.
type ContainerJobHandler struct {
	Repo              *store.DB
	SettingsRepo      *store.SettingsRepo
	ProviderManager   *catalog.ProviderManager
	AlbumArtService   app.AlbumArtService
	PlaylistGenerator app.PlaylistGenerator
	Enricher          *app.MetadataEnricher
}

func (h *ContainerJobHandler) Handle(ctx context.Context, job *domain.Job, logger *slog.Logger) error {
	switch job.Type {
	case domain.JobTypeAlbum:
		return h.processAlbumJob(ctx, job, logger)
	case domain.JobTypePlaylist:
		return h.processPlaylistJob(ctx, job, logger)
	case domain.JobTypeArtist:
		return h.processArtistJob(ctx, job, logger)
	case domain.JobTypeDiscography:
		return h.processDiscographyJob(ctx, job, logger)
	default:
		return ErrUnknownJobType
	}
}

func (h *ContainerJobHandler) processAlbumJob(ctx context.Context, job *domain.Job, logger *slog.Logger) error {
	album, err := h.ProviderManager.GetProvider().GetAlbum(ctx, job.GetSourceID())
	if err != nil {
		logger.Error("Failed to fetch album", "error", err)
		_ = h.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to fetch album: %v", err))
		return err
	}

	if len(album.Tracks) == 0 {
		logger.Error("No tracks found in album")
		_ = h.Repo.UpdateJobError(job.ID, "No tracks found")
		return ErrNoTracksFound
	}

	if album.AlbumArtURL != "" {
		if err := h.AlbumArtService.DownloadAndSaveAlbumArt(album, album.AlbumArtURL); err != nil {
			logger.Error("Failed to save album art", "error", err)
		}
	}

	logger.Info("Creating track jobs", "track_count", len(album.Tracks))
	createdCount := h.createTracksAndJobs(job.ID, album.Tracks, logger)

	if err := h.Repo.UpdateJobStatus(job.ID, domain.JobStatusDecomposed, 0); err != nil {
		logger.Error("Failed to update job status to decomposed", "error", err)
	}

	logger.Info("Album job completed", "tracks_created", createdCount)
	return nil
}

func (h *ContainerJobHandler) processPlaylistJob(ctx context.Context, job *domain.Job, logger *slog.Logger) error {
	pl, err := h.ProviderManager.GetProvider().GetPlaylist(ctx, job.GetSourceID())
	if err != nil {
		logger.Error("Failed to fetch playlist", "error", err)
		_ = h.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to fetch playlist: %v", err))
		return err
	}

	if len(pl.Tracks) == 0 {
		logger.Error("No tracks found in playlist")
		_ = h.Repo.UpdateJobError(job.ID, "No tracks found")
		return ErrNoTracksFound
	}

	if pl.ImageURL != "" {
		if imgErr := h.AlbumArtService.DownloadAndSavePlaylistImage(pl, pl.ImageURL); imgErr != nil {
			logger.Error("Failed to save playlist image", "error", imgErr)
		}
	}

	playlist := &domain.Playlist{
		ProviderID:  pl.ProviderID,
		Title:       pl.Title,
		Description: pl.Description,
		ImageURL:    pl.ImageURL,
	}

	existing, err := h.Repo.GetPlaylistByProviderID(pl.ProviderID)
	if err == nil && existing != nil {
		playlist.ID = existing.ID
		playlist.CreatedAt = existing.CreatedAt
		if err := h.Repo.UpdatePlaylist(playlist); err != nil {
			logger.Error("Failed to update playlist", "error", err)
		}
		if err := h.Repo.ClearPlaylistTracks(playlist.ID); err != nil {
			logger.Error("Failed to clear playlist tracks", "error", err)
		}
	} else {
		if err := h.Repo.CreatePlaylist(playlist); err != nil {
			logger.Error("Failed to create playlist", "error", err)
		}
	}

	logger.Info("Creating track jobs", "track_count", len(pl.Tracks))
	createdCount := h.createTracksAndJobs(job.ID, pl.Tracks, logger)

	if err := h.Repo.UpdateJobStatus(job.ID, domain.JobStatusDecomposed, 0); err != nil {
		logger.Error("Failed to update job status to decomposed", "error", err)
	}

	if createdCount == 0 {
		if genErr := h.PlaylistGenerator.Generate(pl, lookupTrack(h.Repo)); genErr != nil {
			logger.Error("Failed to generate playlist file", "error", genErr)
		}
	}

	logger.Info("Playlist job completed", "tracks_created", createdCount)
	return nil
}

func (h *ContainerJobHandler) processArtistJob(ctx context.Context, job *domain.Job, logger *slog.Logger) error {
	artist, err := h.ProviderManager.GetProvider().GetArtist(ctx, job.GetSourceID())
	if err != nil {
		logger.Error("Failed to fetch artist", "error", err)
		_ = h.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to fetch artist: %v", err))
		return err
	}

	if len(artist.TopTracks) == 0 {
		logger.Error("No tracks found for artist")
		_ = h.Repo.UpdateJobError(job.ID, "No tracks found")
		return ErrNoTracksFound
	}

	logger.Info("Creating track jobs", "track_count", len(artist.TopTracks))
	createdCount := h.createTracksAndJobs(job.ID, artist.TopTracks, logger)

	if err := h.Repo.UpdateJobStatus(job.ID, domain.JobStatusDecomposed, 0); err != nil {
		logger.Error("Failed to update job status to decomposed", "error", err)
	}

	if createdCount == 0 {
		catalogTracks := make([]domain.CatalogTrack, len(artist.TopTracks))
		copy(catalogTracks, artist.TopTracks)
		if genErr := h.PlaylistGenerator.GenerateFromTracks(artist.Name, catalogTracks, lookupTrack(h.Repo)); genErr != nil {
			logger.Error("Failed to generate playlist file", "error", genErr)
		}
	}

	logger.Info("Artist job completed", "tracks_created", createdCount)
	return nil
}

func (h *ContainerJobHandler) processDiscographyJob(ctx context.Context, job *domain.Job, logger *slog.Logger) error {
	artist, err := h.ProviderManager.GetProvider().GetArtist(ctx, job.GetSourceID())
	if err != nil {
		logger.Error("Failed to fetch artist", "error", err)
		_ = h.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to fetch artist: %v", err))
		return err
	}
	if len(artist.Albums) == 0 {
		logger.Error("No albums found for artist")
		_ = h.Repo.UpdateJobError(job.ID, "No albums found")
		return ErrNoTracksFound
	}

	logger.Info("Processing discography", "album_count", len(artist.Albums))
	for _, album := range artist.Albums {
		albumJob := &domain.Job{
			Type:     domain.JobTypeAlbum,
			SourceID: sql.NullString{String: album.ID, Valid: true},
		}
		if err := h.processAlbumJob(ctx, albumJob, logger); err != nil {
			logger.Error("Failed to process album", "album_id", album.ID, "error", err)
			continue
		}
	}

	if err := h.Repo.UpdateJobStatus(job.ID, domain.JobStatusDecomposed, 0); err != nil {
		logger.Error("Failed to update job status to decomposed", "error", err)
	}
	logger.Info("Discography job completed")
	return nil
}

func (h *ContainerJobHandler) createTracksAndJobs(parentJobID string, catalogTracks []domain.CatalogTrack, logger *slog.Logger) int {
	createdCount := 0
	forceDownload := h.isForceDownload()

	var tracksToCreate []*domain.Track
	var jobsToCreate []*domain.Job

	for _, catalogTrack := range catalogTracks {
		if downloaded, _ := h.Repo.IsTrackDownloaded(catalogTrack.ID); downloaded && !forceDownload {
			continue
		}

		if active, _ := h.Repo.IsTrackActive(catalogTrack.ID); active && !forceDownload {
			continue
		}

		track := &domain.Track{
			ProviderID: catalogTrack.ID,
		}
		h.Enricher.UpdateTrackFromCatalog(track, &catalogTrack, logger)
		track.Status = domain.TrackStatusQueued
		track.ParentJobID = parentJobID
		track.CreatedAt = time.Now()
		track.UpdatedAt = time.Now()
		tracksToCreate = append(tracksToCreate, track)

		job := &domain.Job{
			ID:          uuid.New().String(),
			Type:        domain.JobTypeTrack,
			Status:      domain.JobStatusQueued,
			SourceID:    sql.NullString{String: catalogTrack.ID, Valid: true},
			ParentJobID: sql.NullString{String: parentJobID, Valid: true},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		jobsToCreate = append(jobsToCreate, job)
	}

	if len(tracksToCreate) > 0 {
		n, err := h.Repo.CreateTrackBatch(tracksToCreate)
		if err != nil {
			logger.Error("Failed to create tracks batch", "error", err)
		} else {
			createdCount += n
		}
	}

	if len(jobsToCreate) > 0 {
		if err := h.Repo.CreateJobBatch(jobsToCreate); err != nil {
			logger.Error("Failed to create jobs batch", "error", err)
			_ = h.Repo.DeletePendingTracksByParentJobID(parentJobID)
			return 0
		}
	}

	return createdCount
}

// SyncJobHandler handles all metadata resyncs (Hi-Fi, MusicBrainz, File).
type SyncJobHandler struct {
	Repo            *store.DB
	Config          *config.Config
	ProviderManager *catalog.ProviderManager
	AlbumArtService app.AlbumArtService
	Enricher        *app.MetadataEnricher
}

func (h *SyncJobHandler) Handle(ctx context.Context, job *domain.Job, logger *slog.Logger) error {
	switch job.Type {
	case domain.JobTypeSyncMusicBrainz:
		return h.processSyncMusicBrainzJob(ctx, job, logger)
	case domain.JobTypeSyncHiFi:
		return h.processSyncHiFiJob(ctx, job, logger)
	case domain.JobTypeSyncFile:
		return h.processSyncFileJob(ctx, job, logger)
	default:
		return ErrUnknownJobType
	}
}

func (h *SyncJobHandler) processSyncHiFiJob(ctx context.Context, job *domain.Job, logger *slog.Logger) error {
	track, ok := h.getTrackForSync(job, logger)
	if !ok {
		return nil
	}

	h.Enricher.EnrichComplete(ctx, track, logger)

	if h.isCancelled(job.ID) {
		logger.Info("Job cancelled")
		return nil
	}

	h.completeSyncBasic(job, track, logger, "Sync Hi-Fi job completed")
	return nil
}

func (h *SyncJobHandler) processSyncMusicBrainzJob(ctx context.Context, job *domain.Job, logger *slog.Logger) error {
	track, ok := h.getTrackForSync(job, logger)
	if !ok {
		return nil
	}

	h.completeSyncWithEnrichment(ctx, job, track, logger, "Sync job completed")
	return nil
}

func (h *SyncJobHandler) processSyncFileJob(ctx context.Context, job *domain.Job, logger *slog.Logger) error {
	track, ok := h.getTrackForSync(job, logger)
	if !ok {
		return nil
	}
	h.completeSyncBasic(job, track, logger, "Sync file job completed")
	return nil
}

func (h *SyncJobHandler) getTrackForSync(job *domain.Job, logger *slog.Logger) (*domain.Track, bool) {
	track, err := h.Repo.GetTrackByProviderID(job.GetSourceID())
	if err != nil {
		logger.Error("Failed to get track", "error", err)
		_ = h.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to get track: %v", err))
		return nil, false
	}
	if track == nil {
		logger.Error("Track not found")
		_ = h.Repo.UpdateJobError(job.ID, "Track not found")
		return nil, false
	}
	if h.isCancelled(job.ID) {
		logger.Info("Job cancelled")
		return nil, false
	}
	return track, true
}

func (h *SyncJobHandler) isCancelled(id string) bool {
	job, err := h.Repo.GetJob(id)
	if err != nil {
		return false
	}
	return job.Status == domain.JobStatusCancelled
}

func (h *SyncJobHandler) completeSyncBasic(job *domain.Job, track *domain.Track, logger *slog.Logger, successMsg string) {
	oldFilePath := track.FilePath

	if err := h.reTagTrack(track, logger); err != nil {
		_ = h.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to tag file: %v", err))
		return
	}

	if err := h.maybeMoveTrackFile(track, oldFilePath, logger); err != nil {
		_ = h.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to move file: %v", err))
		return
	}

	track.UpdatedAt = time.Now()
	if err := h.Repo.UpdateTrack(track); err != nil {
		logger.Error("Failed to update track", "error", err)
		_ = h.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to update track: %v", err))
		return
	}

	_ = h.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100)
	logger.Info(successMsg)
}

func (h *SyncJobHandler) completeSyncWithEnrichment(ctx context.Context, job *domain.Job, track *domain.Track, logger *slog.Logger, successMsg string) {
	oldFilePath := track.FilePath

	if err := h.Enricher.EnrichTrack(ctx, track, logger); err != nil {
		logger.Warn("MusicBrainz enrichment failed, continuing with existing data", "error", err)
	}

	if h.isCancelled(job.ID) {
		logger.Info("Job cancelled")
		return
	}

	if err := h.reTagTrack(track, logger); err != nil {
		_ = h.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to tag file: %v", err))
		return
	}

	if err := h.maybeMoveTrackFile(track, oldFilePath, logger); err != nil {
		_ = h.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to move file: %v", err))
		return
	}

	track.UpdatedAt = time.Now()
	if err := h.Repo.UpdateTrack(track); err != nil {
		logger.Error("Failed to update track", "error", err)
		_ = h.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to update track: %v", err))
		return
	}

	_ = h.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100)
	logger.Info(successMsg)
}

func (h *SyncJobHandler) reTagTrack(track *domain.Track, logger *slog.Logger) error {
	var albumArtData []byte

	if track.FilePath != "" {
		albumDir := filepath.Dir(track.FilePath)
		coverPath := filepath.Join(albumDir, "cover.jpg")
		if data, err := os.ReadFile(coverPath); err == nil && len(data) > 0 { //nolint:gosec
			albumArtData = data
		}
	}

	if len(albumArtData) == 0 && track.AlbumArtURL != "" {
		var err error
		albumArtData, err = h.AlbumArtService.DownloadImage(track.AlbumArtURL)
		if err != nil {
			logger.Error("Failed to download album art for tagging", "error", err)
		}
	}

	if tagErr := tagging.TagFile(track.FilePath, track, albumArtData); tagErr != nil {
		if errors.Is(tagErr, tagging.ErrUnsupportedFormat) {
			logger.Warn("Tagging skipped: unsupported format", "file_path", track.FilePath, "error", tagErr)
			return nil
		}
		logger.Error("Failed to tag file", "error", tagErr)
		return tagErr
	}
	return nil
}

func (h *SyncJobHandler) maybeMoveTrackFile(track *domain.Track, oldFilePath string, logger *slog.Logger) error {
	if oldFilePath == "" {
		return nil
	}

	oldDir := filepath.Dir(oldFilePath)

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

	expectedPath, err := storage.BuildFullPath(h.Config.DownloadsDir, h.Config.SubdirTemplate, templateData, track.FileExtension)
	if err != nil {
		logger.Error("Failed to build expected path", "error", err)
		return err
	}

	if oldFilePath == expectedPath {
		return nil
	}

	track.FilePath = expectedPath
	newDir := filepath.Dir(track.FilePath)

	if err := os.MkdirAll(newDir, constants.DirPermissions); err != nil {
		logger.Error("Failed to create new directory", "dir", newDir, "error", err)
		return err
	}

	if err := storage.MoveFile(oldFilePath, track.FilePath); err != nil {
		logger.Error("Failed to move audio file", "old", oldFilePath, "new", track.FilePath, "error", err)
		return err
	}

	oldCoverPath := filepath.Join(oldDir, "cover.jpg")
	newCoverPath := filepath.Join(newDir, "cover.jpg")
	if _, err := os.Stat(oldCoverPath); err == nil {
		if err := storage.CopyFile(oldCoverPath, newCoverPath); err != nil {
			logger.Warn("Failed to copy cover file", "old", oldCoverPath, "new", newCoverPath, "error", err)
		}
	}

	if err := storage.DeleteFolderWithCover(oldDir); err != nil {
		logger.Warn("Failed to clean up old directory", "dir", oldDir, "error", err)
	}

	return nil
}

func (h *TrackJobHandler) isForceDownload() bool {
	if h.SettingsRepo == nil {
		return false
	}
	val, err := h.SettingsRepo.Get(store.SettingForceDownload)
	return err == nil && val == "true"
}

func (h *TrackJobHandler) getQuality() string {
	if h.SettingsRepo == nil {
		return h.Config.Quality
	}
	val, err := h.SettingsRepo.Get(store.SettingQuality)
	if err == nil && val != "" {
		return val
	}
	return h.Config.Quality
}

func (h *ContainerJobHandler) isForceDownload() bool {
	if h.SettingsRepo == nil {
		return false
	}
	val, err := h.SettingsRepo.Get(store.SettingForceDownload)
	return err == nil && val == "true"
}
