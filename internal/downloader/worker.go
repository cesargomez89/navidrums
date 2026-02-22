package downloader

import (
	"context"
	"encoding/json"
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
	"github.com/cesargomez89/navidrums/internal/logger"
	"github.com/cesargomez89/navidrums/internal/musicbrainz"
	"github.com/cesargomez89/navidrums/internal/storage"
	"github.com/cesargomez89/navidrums/internal/store"
	"github.com/cesargomez89/navidrums/internal/tagging"
)

var (
	ErrJobCancelled   = errors.New("job was cancelled")
	ErrDownloadFailed = errors.New("download failed after retries")
	ErrNoTracksFound  = errors.New("no tracks found")
)

type Worker struct {
	downloader        app.Downloader
	playlistGenerator app.PlaylistGenerator
	albumArtService   app.AlbumArtService
	ctx               context.Context
	Repo              *store.DB
	SettingsRepo      *store.SettingsRepo
	ProviderManager   *catalog.ProviderManager
	Config            *config.Config
	Logger            *logger.Logger
	musicBrainzClient musicbrainz.ClientInterface
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	MaxConcurrent     int
}

func NewWorker(repo *store.DB, settingsRepo *store.SettingsRepo, pm *catalog.ProviderManager, cfg *config.Config, log *logger.Logger) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	if log == nil {
		log = logger.Default()
	}

	worker := &Worker{
		Repo:            repo,
		SettingsRepo:    settingsRepo,
		ProviderManager: pm,
		Config:          cfg,
		MaxConcurrent:   constants.DefaultConcurrency,
		Logger:          log.WithComponent("worker"),
		ctx:             ctx,
		cancel:          cancel,
	}

	worker.downloader = app.NewDownloader(pm, cfg)
	worker.playlistGenerator = app.NewPlaylistGenerator(cfg)
	worker.albumArtService = app.NewAlbumArtService(cfg)

	baseMBClient := musicbrainz.NewClient(cfg.MusicBrainzURL)
	worker.musicBrainzClient = musicbrainz.NewCachedClient(baseMBClient, repo, 7*24*time.Hour)

	worker.loadGenreMap()

	return worker
}

func (w *Worker) Start() {
	w.Logger.Info("Starting worker")

	if err := w.Repo.ResetStuckJobs(); err != nil {
		w.Logger.Error("Failed to reset stuck jobs", "error", err)
	}

	w.recoverInterruptedTracks()

	w.wg.Add(1)
	go w.processJobs()
}

func (w *Worker) loadGenreMap() {
	if w.SettingsRepo == nil {
		return
	}

	genreMapJSON, err := w.SettingsRepo.Get(store.SettingGenreMap)
	if err != nil || genreMapJSON == "" {
		return
	}

	var customMap map[string]string
	if err := json.Unmarshal([]byte(genreMapJSON), &customMap); err != nil {
		w.Logger.Warn("Failed to parse custom genre map, using default", "error", err)
		return
	}

	w.musicBrainzClient.SetGenreMap(customMap)
}

func (w *Worker) recoverInterruptedTracks() {
	tracks, err := w.Repo.FindInterruptedTracks()
	if err != nil {
		w.Logger.Error("Failed to find interrupted tracks", "error", err)
		return
	}

	for _, t := range tracks {
		w.Logger.Info("Recovering interrupted track", "track_id", t.ID)

		// Attempt to clean up potential partial files
		// We need to reconstruct the path since it might not be saved in DB yet
		artistForFolder := t.AlbumArtist
		if artistForFolder == "" {
			artistForFolder = t.Artist
		}

		templateData := storage.BuildPathTemplateData(
			artistForFolder,
			t.Year,
			t.Album,
			t.DiscNumber,
			t.TrackNumber,
			t.Title,
		)

		fullPathNoExt, err := storage.BuildPath(w.Config.SubdirTemplate, templateData)
		if err == nil {
			fullPathNoExt = filepath.Join(w.Config.DownloadsDir, fullPathNoExt)
			// Remove known extensions if they exist
			// This is best-effort
			for _, ext := range []string{".flac", ".mp3", ".m4a"} {
				_ = storage.RemoveFile(fullPathNoExt + ext)
			}
		}

		if err := w.Repo.UpdateTrackStatus(t.ID, domain.TrackStatusQueued, ""); err != nil {
			w.Logger.Error("Failed to reset track status", "track_id", t.ID, "error", err)
		}
	}
}

func (w *Worker) Stop() {
	w.Logger.Info("Stopping worker")
	w.cancel()
	w.wg.Wait()
}

func (w *Worker) processJobs() {
	defer w.wg.Done()
	ticker := time.NewTicker(constants.DefaultPollInterval)
	defer ticker.Stop()

	sem := make(chan struct{}, w.MaxConcurrent)

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			jobs, err := w.Repo.ListActiveJobs()
			if err != nil {
				w.Logger.Error("Failed to list jobs", "error", err)
				continue
			}

			if len(jobs) == 0 {
				continue
			}

			activeCount := 0
			queuedJobs := []*domain.Job{}

			for _, j := range jobs {
				switch j.Status {
				case domain.JobStatusRunning:
					activeCount++
				case domain.JobStatusQueued:
					queuedJobs = append(queuedJobs, j)
				}
			}

			toStart := w.MaxConcurrent - activeCount
			if toStart <= 0 || len(queuedJobs) == 0 {
				continue
			}

			for i := 0; i < toStart && i < len(queuedJobs); i++ {
				job := queuedJobs[i]

				current, err := w.Repo.GetJob(job.ID)
				if err != nil {
					w.Logger.Error("Failed to get job before starting", "job_id", job.ID, "error", err)
					continue
				}
				if current != nil && current.Status == domain.JobStatusCancelled {
					continue
				}

				sem <- struct{}{}
				w.wg.Add(1)
				go func(j *domain.Job) {
					defer w.wg.Done()
					defer func() { <-sem }()
					w.runJob(w.ctx, j)
				}(job)
			}
		}
	}
}

func (w *Worker) runJob(ctx context.Context, job *domain.Job) {
	defer func() {
		if r := recover(); r != nil {
			w.Logger.Error("Panic in job",
				"job_id", job.ID,
				"panic", r,
			)
			_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Panic: %v", r))
		}
	}()

	logger := w.Logger.With(
		"job_id", job.ID,
		"job_type", job.Type,
		"source_id", job.SourceID,
	)
	logger.Info("Running job")

	// Mark job as running
	if err := w.Repo.UpdateJobStatus(job.ID, domain.JobStatusRunning, 0); err != nil {
		logger.Error("Failed to update status", "error", err)
		return
	}

	if w.isCancelled(job.ID) {
		logger.Info("Job cancelled before processing")
		return
	}

	// Dispatch based on job type
	switch job.Type {
	case domain.JobTypeTrack:
		w.processTrackJob(ctx, job)
	case domain.JobTypeAlbum:
		w.processAlbumJob(ctx, job)
	case domain.JobTypePlaylist:
		w.processPlaylistJob(ctx, job)
	case domain.JobTypeArtist:
		w.processArtistJob(ctx, job)
	case domain.JobTypeSyncFile:
		w.processSyncFileJob(ctx, job)
	case domain.JobTypeSyncMusicBrainz:
		w.processSyncMusicBrainzJob(ctx, job)
	case domain.JobTypeSyncHiFi:
		w.processSyncHiFiJob(ctx, job)
	default:
		logger.Error("Unknown job type")
		_ = w.Repo.UpdateJobError(job.ID, "Unknown job type")
	}
}

func (w *Worker) processTrackJob(ctx context.Context, job *domain.Job) {
	logger := w.Logger.With("job_id", job.ID, "source_id", job.SourceID)

	track, destPath, skipDownload, err := w.prepareTrackDownload(ctx, job, logger)
	if err != nil {
		return
	}

	if skipDownload {
		return
	}

	if w.isCancelled(job.ID) {
		logger.Info("Job cancelled before download")
		return
	}

	finalPath, err := w.executeDownload(ctx, job, track, destPath, logger)
	if err != nil {
		return
	}

	if err := w.postProcessTrack(ctx, track, finalPath, logger); err != nil {
		logger.Warn("Post-processing had issues", "error", err)
	}

	w.finalizeTrackDownload(job, track, finalPath, logger)
}

func (w *Worker) prepareTrackDownload(ctx context.Context, job *domain.Job, logger *slog.Logger) (*domain.Track, string, bool, error) {
	existingTrack, _ := w.Repo.GetTrackByProviderID(job.SourceID)
	if existingTrack != nil && existingTrack.Status == domain.TrackStatusCompleted {
		logger.Info("Track already downloaded", "file_path", existingTrack.FilePath)
		_ = w.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100)
		return nil, "", true, nil
	}

	var track *domain.Track
	if existingTrack != nil {
		track = existingTrack
	} else {
		catalogTrack, err := w.ProviderManager.GetProvider().GetTrack(ctx, job.SourceID)
		if err != nil {
			logger.Error("Failed to fetch track metadata", "error", err)
			_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to fetch track: %v", err))
			return nil, "", false, err
		}

		track = w.catalogTrackToDomainTrack(catalogTrack)
		track.Status = domain.TrackStatusMissing
		track.ParentJobID = job.ID
		track.CreatedAt = time.Now()
		track.UpdatedAt = time.Now()

		if err := w.Repo.CreateTrack(track); err != nil {
			logger.Error("Failed to create track record", "error", err)
			_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to create track record: %v", err))
			return nil, "", false, err
		}
	}

	artistForFolder := track.AlbumArtist
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

	fullPathNoExt, err := storage.BuildPath(w.Config.SubdirTemplate, templateData)
	if err != nil {
		logger.Error("Failed to build path from template", "error", err)
		_ = w.Repo.MarkTrackFailed(track.ID, fmt.Sprintf("Failed to build path: %v", err))
		_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to build path: %v", err))
		return nil, "", false, err
	}

	fullPathNoExt = filepath.Join(w.Config.DownloadsDir, fullPathNoExt)

	ext := track.FileExtension
	if ext == "" {
		ext = ".flac"
	}
	predictedPath := fullPathNoExt + ext

	if track.Status == domain.TrackStatusCompleted {
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
					_ = w.Repo.UpdateTrack(track)
					match = true
				}
			}

			if match {
				logger.Info("Track already exists and verified, skipping download", "path", predictedPath)
				_ = w.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100)
				return nil, "", true, nil
			} else {
				logger.Info("Track exists but hash mismatch, redownloading", "path", predictedPath)
				_ = storage.RemoveFile(predictedPath)
			}
		}
	}

	return track, fullPathNoExt, false, nil
}

func (w *Worker) executeDownload(ctx context.Context, job *domain.Job, track *domain.Track, destPath string, logger *slog.Logger) (string, error) {
	if updateErr := w.Repo.UpdateTrackStatus(track.ID, domain.TrackStatusDownloading, ""); updateErr != nil {
		logger.Error("Failed to update track status to downloading", "error", updateErr)
		return "", updateErr
	}

	finalDir := filepath.Dir(destPath)
	if dirErr := storage.EnsureDir(finalDir); dirErr != nil {
		logger.Error("Failed to create directory", "error", dirErr)
		_ = w.Repo.MarkTrackFailed(track.ID, fmt.Sprintf("Failed to create directory: %v", dirErr))
		_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to create directory: %v", dirErr))
		return "", dirErr
	}

	finalPath, err := w.downloader.Download(ctx, track, destPath)
	if err != nil {
		logger.Error("Download failed", "error", err)
		_ = w.Repo.MarkTrackFailed(track.ID, err.Error())
		_ = w.Repo.UpdateJobError(job.ID, err.Error())
		return "", err
	}

	if statusErr := w.Repo.UpdateTrackStatus(track.ID, domain.TrackStatusProcessing, finalPath); statusErr != nil {
		logger.Error("Failed to update track status to processing", "error", statusErr)
	}

	logger.Info("Download finished, starting tagging", "file_path", finalPath)
	return finalPath, nil
}

func (w *Worker) postProcessTrack(ctx context.Context, track *domain.Track, finalPath string, logger *slog.Logger) error {
	var albumArtData []byte
	if track.AlbumArtURL != "" {
		var err error
		albumArtData, err = w.albumArtService.DownloadImage(track.AlbumArtURL)
		if err != nil {
			logger.Error("Failed to download album art for tagging", "error", err)
		}
	}

	if track.Lyrics == "" || track.Subtitles == "" {
		lyrics, subtitles, lyricsErr := w.ProviderManager.GetProvider().GetLyrics(ctx, track.ProviderID)
		if lyricsErr != nil {
			logger.Debug("Failed to fetch lyrics", "error", lyricsErr)
		} else {
			if track.Lyrics == "" && lyrics != "" {
				track.Lyrics = lyrics
			}
			if track.Subtitles == "" && subtitles != "" {
				track.Subtitles = subtitles
			}
		}
	}

	if err := w.enrichFromMusicBrainz(ctx, track, logger); err != nil {
		logger.Warn("MusicBrainz enrichment failed", "isrc", track.ISRC, "error", err)
	}

	if tagErr := tagging.TagFile(finalPath, track, albumArtData); tagErr != nil {
		logger.Error("Failed to tag file", "file_path", finalPath, "error", tagErr)
	}

	if len(albumArtData) > 0 {
		finalDir := filepath.Dir(finalPath)
		artPath := filepath.Join(finalDir, "cover.jpg")
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

func (w *Worker) finalizeTrackDownload(job *domain.Job, track *domain.Track, finalPath string, logger *slog.Logger) {
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

	if err := w.Repo.UpdateTrack(track); err != nil {
		logger.Error("Failed to update track", "error", err)
	}

	if track.AlbumID != "" {
		_, _ = w.Repo.RecomputeAlbumState(track.AlbumID)
	}

	if err := w.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100); err != nil {
		logger.Error("Failed to update final job status", "error", err)
	}

	logger.Info("Job completed successfully")
}

func (w *Worker) processAlbumJob(ctx context.Context, job *domain.Job) {
	logger := w.Logger.With("job_id", job.ID, "source_id", job.SourceID)

	album, err := w.ProviderManager.GetProvider().GetAlbum(ctx, job.SourceID)
	if err != nil {
		logger.Error("Failed to fetch album", "error", err)
		_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to fetch album: %v", err))
		return
	}

	if len(album.Tracks) == 0 {
		logger.Error("No tracks found in album")
		_ = w.Repo.UpdateJobError(job.ID, "No tracks found")
		return
	}

	// Save album art
	if album.AlbumArtURL != "" {
		if err := w.albumArtService.DownloadAndSaveAlbumArt(album, album.AlbumArtURL); err != nil {
			logger.Error("Failed to save album art", "error", err)
		}
	}

	// Create tracks and child jobs for each track
	logger.Info("Creating track jobs", "track_count", len(album.Tracks))
	createdCount := w.createTracksAndJobs(job.ID, album.Tracks, logger)

	// Mark album job as completed
	if err := w.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100); err != nil {
		logger.Error("Failed to update job status to completed", "error", err)
	}

	logger.Info("Album job completed", "tracks_created", createdCount)
}

func (w *Worker) processPlaylistJob(ctx context.Context, job *domain.Job) {
	logger := w.Logger.With("job_id", job.ID, "source_id", job.SourceID)

	pl, err := w.ProviderManager.GetProvider().GetPlaylist(ctx, job.SourceID)
	if err != nil {
		logger.Error("Failed to fetch playlist", "error", err)
		_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to fetch playlist: %v", err))
		return
	}

	if len(pl.Tracks) == 0 {
		logger.Error("No tracks found in playlist")
		_ = w.Repo.UpdateJobError(job.ID, "No tracks found")
		return
	}

	// Save playlist image
	if pl.ImageURL != "" {
		if err := w.albumArtService.DownloadAndSavePlaylistImage(pl, pl.ImageURL); err != nil {
			logger.Error("Failed to save playlist image", "error", err)
		}
	}

	// Create tracks and child jobs
	logger.Info("Creating track jobs", "track_count", len(pl.Tracks))
	createdCount := w.createTracksAndJobs(job.ID, pl.Tracks, logger)

	if err := w.playlistGenerator.Generate(pl, w.lookupTrackExtension); err != nil {
		logger.Error("Failed to generate playlist file", "error", err)
	}

	// Mark job as completed
	if err := w.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100); err != nil {
		logger.Error("Failed to update job status to completed", "error", err)
	}

	logger.Info("Playlist job completed", "tracks_created", createdCount)
}

func (w *Worker) processArtistJob(ctx context.Context, job *domain.Job) {
	logger := w.Logger.With("job_id", job.ID, "source_id", job.SourceID)

	artist, err := w.ProviderManager.GetProvider().GetArtist(ctx, job.SourceID)
	if err != nil {
		logger.Error("Failed to fetch artist", "error", err)
		_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to fetch artist: %v", err))
		return
	}

	if len(artist.TopTracks) == 0 {
		logger.Error("No tracks found for artist")
		_ = w.Repo.UpdateJobError(job.ID, "No tracks found")
		return
	}

	// Create tracks and child jobs
	logger.Info("Creating track jobs", "track_count", len(artist.TopTracks))
	createdCount := w.createTracksAndJobs(job.ID, artist.TopTracks, logger)

	catalogTracks := make([]domain.CatalogTrack, len(artist.TopTracks))
	copy(catalogTracks, artist.TopTracks)
	if err := w.playlistGenerator.GenerateFromTracks(artist.Name, catalogTracks, w.lookupTrackExtension); err != nil {
		logger.Error("Failed to generate playlist file", "error", err)
	}

	// Mark job as completed
	if err := w.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100); err != nil {
		logger.Error("Failed to update job status to completed", "error", err)
	}

	logger.Info("Artist job completed", "tracks_created", createdCount)
}

func (w *Worker) isCancelled(id string) bool {
	job, err := w.Repo.GetJob(id)
	if err != nil {
		return false
	}
	return job.Status == domain.JobStatusCancelled
}

// catalogTrackToDomainTrack converts a CatalogTrack to a domain Track
func (w *Worker) catalogTrackToDomainTrack(ct *domain.CatalogTrack) *domain.Track {
	return &domain.Track{
		ProviderID:     ct.ID,
		Title:          ct.Title,
		Artist:         ct.Artist,
		Artists:        ct.Artists,
		ArtistIDs:      ct.ArtistIDs,
		Album:          ct.Album,
		AlbumArtist:    ct.AlbumArtist,
		AlbumArtists:   ct.AlbumArtists,
		AlbumArtistIDs: ct.AlbumArtistIDs,
		TrackNumber:    ct.TrackNumber,
		DiscNumber:     ct.DiscNumber,
		TotalTracks:    ct.TotalTracks,
		TotalDiscs:     ct.TotalDiscs,
		Year:           ct.Year,
		ReleaseDate:    ct.ReleaseDate,
		Genre:          ct.Genre,
		Label:          ct.Label,
		ISRC:           ct.ISRC,
		Copyright:      ct.Copyright,
		Composer:       ct.Composer,
		Duration:       ct.Duration,
		Explicit:       ct.ExplicitLyrics,
		Compilation:    ct.Compilation,
		AlbumArtURL:    ct.AlbumArtURL,
		Lyrics:         ct.Lyrics,
		Subtitles:      ct.Subtitles,
		BPM:            ct.BPM,
		Key:            ct.Key,
		KeyScale:       ct.KeyScale,
		ReplayGain:     ct.ReplayGain,
		Peak:           ct.Peak,
		Version:        ct.Version,
		Description:    ct.Description,
		URL:            ct.URL,
		AudioQuality:   ct.AudioQuality,
		AudioModes:     ct.AudioModes,
	}
}

func (w *Worker) lookupTrackExtension(trackID string) string {
	t, _ := w.Repo.GetTrackByProviderID(trackID)
	if t != nil && t.FileExtension != "" {
		return t.FileExtension
	}
	return ".flac"
}

func (w *Worker) createTracksAndJobs(parentJobID string, catalogTracks []domain.CatalogTrack, logger *slog.Logger) int {
	createdCount := 0

	for _, catalogTrack := range catalogTracks {
		if downloaded, _ := w.Repo.IsTrackDownloaded(catalogTrack.ID); downloaded {
			continue
		}

		if active, _ := w.Repo.IsTrackActive(catalogTrack.ID); active {
			continue
		}

		track := w.catalogTrackToDomainTrack(&catalogTrack)
		track.Status = domain.TrackStatusQueued
		track.ParentJobID = parentJobID
		track.CreatedAt = time.Now()
		track.UpdatedAt = time.Now()

		if err := w.Repo.CreateTrack(track); err != nil {
			logger.Error("Failed to create track record", "track_id", catalogTrack.ID, "error", err)
			continue
		}

		childJob := &domain.Job{
			ID:        uuid.New().String(),
			Type:      domain.JobTypeTrack,
			Status:    domain.JobStatusQueued,
			SourceID:  catalogTrack.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := w.Repo.CreateJob(childJob); err != nil {
			logger.Error("Failed to create child track job", "track_id", catalogTrack.ID, "error", err)
		} else {
			createdCount++
		}
	}

	return createdCount
}

func (w *Worker) enrichFromMusicBrainz(ctx context.Context, track *domain.Track, logger *slog.Logger) error {
	if track.ISRC == "" && track.RecordingID == "" {
		return nil
	}

	meta, mbErr := w.musicBrainzClient.GetRecording(ctx, track.RecordingID, track.ISRC, track.Album)
	if mbErr != nil {
		return mbErr
	}
	if meta == nil {
		return nil
	}

	if meta.RecordingID != "" && track.RecordingID == "" {
		track.RecordingID = meta.RecordingID
	}
	if track.Artist == "" && meta.Artist != "" {
		track.Artist = meta.Artist
	}
	if len(track.Artists) == 0 && len(meta.Artists) > 0 {
		track.Artists = meta.Artists
	}
	if track.Title == "" && meta.Title != "" {
		track.Title = meta.Title
	}
	if track.Duration == 0 && meta.Duration > 0 {
		track.Duration = meta.Duration
	}
	if track.Year == 0 && meta.Year > 0 {
		track.Year = meta.Year
	}
	if track.Barcode == "" && meta.Barcode != "" {
		track.Barcode = meta.Barcode
	}
	if track.CatalogNumber == "" && meta.CatalogNumber != "" {
		track.CatalogNumber = meta.CatalogNumber
	}
	if track.ReleaseType == "" && meta.ReleaseType != "" {
		track.ReleaseType = meta.ReleaseType
	}
	if meta.ReleaseID != "" {
		track.ReleaseID = meta.ReleaseID
	}
	if len(track.ArtistIDs) == 0 && len(meta.ArtistIDs) > 0 {
		track.ArtistIDs = meta.ArtistIDs
	}
	if len(track.AlbumArtistIDs) == 0 && len(meta.AlbumArtistIDs) > 0 {
		track.AlbumArtistIDs = meta.AlbumArtistIDs
	}
	if len(track.AlbumArtists) == 0 && len(meta.AlbumArtists) > 0 {
		track.AlbumArtists = meta.AlbumArtists
	}
	if track.Composer == "" && meta.Composer != "" {
		track.Composer = meta.Composer
	}

	if track.Genre == "" {
		result, genreErr := w.musicBrainzClient.GetGenres(ctx, track.RecordingID, track.ISRC)
		if genreErr != nil {
			logger.Warn("Failed to fetch genre from MusicBrainz", "isrc", track.ISRC, "error", genreErr)
		} else {
			if result.MainGenre != "" {
				track.Genre = result.MainGenre
			}
			if result.SubGenre != "" {
				track.SubGenre = result.SubGenre
			}
		}
	}

	return nil
}

func (w *Worker) updateTrackFromCatalog(track *domain.Track, ct *domain.CatalogTrack) {
	track.Title = ct.Title
	track.Artist = ct.Artist
	track.Artists = ct.Artists
	track.ArtistIDs = ct.ArtistIDs
	track.Album = ct.Album
	track.AlbumArtist = ct.AlbumArtist
	track.AlbumArtists = ct.AlbumArtists
	track.AlbumArtistIDs = ct.AlbumArtistIDs
	track.AlbumID = ct.AlbumID
	track.TrackNumber = ct.TrackNumber
	track.DiscNumber = ct.DiscNumber
	track.TotalTracks = ct.TotalTracks
	track.TotalDiscs = ct.TotalDiscs
	track.Year = ct.Year
	track.ReleaseDate = ct.ReleaseDate
	track.Genre = ct.Genre
	track.Label = ct.Label
	track.ISRC = ct.ISRC
	track.Copyright = ct.Copyright
	track.Composer = ct.Composer
	track.Duration = ct.Duration
	track.Explicit = ct.ExplicitLyrics
	track.Compilation = ct.Compilation
	track.AlbumArtURL = ct.AlbumArtURL
	track.BPM = ct.BPM
	track.Key = ct.Key
	track.KeyScale = ct.KeyScale
	track.ReplayGain = ct.ReplayGain
	track.Peak = ct.Peak
	track.Version = ct.Version
	track.Description = ct.Description
	track.URL = ct.URL
	track.AudioQuality = ct.AudioQuality
	track.AudioModes = ct.AudioModes
}

func (w *Worker) reTagTrack(track *domain.Track, logger *slog.Logger) {
	if track.FilePath == "" {
		return
	}

	var albumArtData []byte
	if track.AlbumArtURL != "" {
		albumArtData, _ = w.albumArtService.DownloadImage(track.AlbumArtURL)
	}
	if tagErr := tagging.TagFile(track.FilePath, track, albumArtData); tagErr != nil {
		logger.Error("Failed to tag file", "error", tagErr)
	}
}

func (w *Worker) processSyncHiFiJob(ctx context.Context, job *domain.Job) {
	logger := w.Logger.With("job_id", job.ID, "source_id", job.SourceID)

	track, err := w.Repo.GetTrackByProviderID(job.SourceID)
	if err != nil {
		logger.Error("Failed to get track", "error", err)
		_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to get track: %v", err))
		return
	}
	if track == nil {
		logger.Error("Track not found")
		_ = w.Repo.UpdateJobError(job.ID, "Track not found")
		return
	}

	if w.isCancelled(job.ID) {
		logger.Info("Job cancelled")
		return
	}

	catalogTrack, err := w.ProviderManager.GetProvider().GetTrack(ctx, job.SourceID)
	if err != nil {
		logger.Warn("Failed to fetch Hi-Fi metadata, using existing data", "error", err)
	} else {
		w.updateTrackFromCatalog(track, catalogTrack)

		if catalogTrack.AlbumID != "" {
			album, albumErr := w.ProviderManager.GetProvider().GetAlbum(ctx, catalogTrack.AlbumID)
			if albumErr != nil {
				logger.Debug("Failed to fetch album metadata", "album_id", catalogTrack.AlbumID, "error", albumErr)
			} else {
				track.ReleaseDate = album.ReleaseDate
				track.Label = album.Label
				track.Genre = album.Genre
				track.TotalTracks = album.TotalTracks
				track.TotalDiscs = album.TotalDiscs
				track.Barcode = album.UPC
				if album.AlbumArtURL != "" {
					track.AlbumArtURL = album.AlbumArtURL
				}
			}
		}
	}

	lyrics, subtitles, lyricsErr := w.ProviderManager.GetProvider().GetLyrics(ctx, track.ProviderID)
	if lyricsErr != nil {
		logger.Debug("Failed to fetch lyrics", "error", lyricsErr)
	} else {
		if lyrics != "" {
			track.Lyrics = lyrics
		}
		if subtitles != "" {
			track.Subtitles = subtitles
		}
	}

	w.finalizeSyncJob(ctx, job, track, logger, "Sync Hi-Fi job completed")
}

func (w *Worker) processSyncMusicBrainzJob(ctx context.Context, job *domain.Job) {
	logger := w.Logger.With("job_id", job.ID, "source_id", job.SourceID)

	track, err := w.Repo.GetTrackByProviderID(job.SourceID)
	if err != nil {
		logger.Error("Failed to get track", "error", err)
		_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to get track: %v", err))
		return
	}
	if track == nil {
		logger.Error("Track not found")
		_ = w.Repo.UpdateJobError(job.ID, "Track not found")
		return
	}

	if w.isCancelled(job.ID) {
		logger.Info("Job cancelled")
		return
	}

	w.finalizeSyncJob(ctx, job, track, logger, "Sync job completed")
}

func (w *Worker) finalizeSyncJob(ctx context.Context, job *domain.Job, track *domain.Track, logger *slog.Logger, successMsg string) {
	if err := w.enrichFromMusicBrainz(ctx, track, logger); err != nil {
		_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("MusicBrainz enrichment failed: %v", err))
		return
	}

	track.UpdatedAt = time.Now()
	if err := w.Repo.UpdateTrack(track); err != nil {
		logger.Error("Failed to update track", "error", err)
		_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to update track: %v", err))
		return
	}

	w.reTagTrack(track, logger)

	_ = w.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100)
	logger.Info(successMsg)
}

func (w *Worker) processSyncFileJob(ctx context.Context, job *domain.Job) {
	logger := w.Logger.With("job_id", job.ID, "source_id", job.SourceID)

	track, err := w.Repo.GetTrackByProviderID(job.SourceID)
	if err != nil {
		logger.Error("Failed to get track", "error", err)
		_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to get track: %v", err))
		return
	}
	if track == nil {
		logger.Error("Track not found")
		_ = w.Repo.UpdateJobError(job.ID, "Track not found")
		return
	}

	if w.isCancelled(job.ID) {
		logger.Info("Job cancelled")
		return
	}

	track.UpdatedAt = time.Now()
	if err := w.Repo.UpdateTrack(track); err != nil {
		logger.Error("Failed to update track", "error", err)
		_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to update track: %v", err))
		return
	}

	w.reTagTrack(track, logger)

	_ = w.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100)
	logger.Info("Sync file job completed")
}
