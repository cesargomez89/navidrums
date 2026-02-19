package downloader

import (
	"context"
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
	ProviderManager   *catalog.ProviderManager
	Config            *config.Config
	Logger            *logger.Logger
	musicBrainzClient *musicbrainz.Client
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	MaxConcurrent     int
}

func NewWorker(repo *store.DB, pm *catalog.ProviderManager, cfg *config.Config, log *logger.Logger) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	if log == nil {
		log = logger.Default()
	}

	worker := &Worker{
		Repo:            repo,
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
	worker.musicBrainzClient = musicbrainz.NewClient(cfg.MusicBrainzURL)

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
	default:
		logger.Error("Unknown job type")
		_ = w.Repo.UpdateJobError(job.ID, "Unknown job type")
	}
}

func (w *Worker) processTrackJob(ctx context.Context, job *domain.Job) {
	logger := w.Logger.With("job_id", job.ID, "source_id", job.SourceID)

	// Check if track already exists and is completed
	existingTrack, _ := w.Repo.GetTrackByProviderID(job.SourceID)
	if existingTrack != nil && existingTrack.Status == domain.TrackStatusCompleted {
		logger.Info("Track already downloaded", "file_path", existingTrack.FilePath)
		_ = w.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100)
		return
	}

	// Fetch track metadata from provider if not already stored
	var track *domain.Track
	if existingTrack != nil {
		track = existingTrack
	} else {
		catalogTrack, err := w.ProviderManager.GetProvider().GetTrack(ctx, job.SourceID)
		if err != nil {
			logger.Error("Failed to fetch track metadata", "error", err)
			_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to fetch track: %v", err))
			return
		}

		// Convert catalog track to domain track
		track = w.catalogTrackToDomainTrack(catalogTrack)
		track.Status = domain.TrackStatusMissing
		track.ParentJobID = job.ID
		track.CreatedAt = time.Now()
		track.UpdatedAt = time.Now()

		if err := w.Repo.CreateTrack(track); err != nil {
			logger.Error("Failed to create track record", "error", err)
			_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to create track record: %v", err))
			return
		}
	}

	if w.isCancelled(job.ID) {
		logger.Info("Job cancelled before download")
		return
	}

	// Prepare download path using template (Early for idempotency check)
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
		return
	}

	fullPathNoExt = filepath.Join(w.Config.DownloadsDir, fullPathNoExt)

	// Check for existing file (Idempotency)
	// We guess extension - usually .flac for now, or check generic
	ext := track.FileExtension
	if ext == "" {
		ext = ".flac" // Default guess
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
			// Verify hash if available
			match := false
			if track.FileHash != "" {
				verified, _ := storage.VerifyFile(predictedPath, track.FileHash)
				if verified {
					match = true
				}
			} else {
				// Trust existing if completed? Or re-hash?
				// Be safe: re-hash and update
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
				return
			} else {
				logger.Info("Track exists but hash mismatch, redownloading", "path", predictedPath)
				_ = storage.RemoveFile(predictedPath)
			}
		}
	}

	// Update track status to downloading
	if updateErr := w.Repo.UpdateTrackStatus(track.ID, domain.TrackStatusDownloading, ""); updateErr != nil {
		logger.Error("Failed to update track status to downloading", "error", err)
		return
	}

	finalDir := filepath.Dir(fullPathNoExt)
	if dirErr := storage.EnsureDir(finalDir); dirErr != nil {
		logger.Error("Failed to create directory", "error", dirErr)
		_ = w.Repo.MarkTrackFailed(track.ID, fmt.Sprintf("Failed to create directory: %v", dirErr))
		_ = w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to create directory: %v", dirErr))
		return
	}

	// Download the track
	finalPath, err := w.downloader.Download(ctx, track, fullPathNoExt)
	if err != nil {
		logger.Error("Download failed", "error", err)
		_ = w.Repo.MarkTrackFailed(track.ID, err.Error())
		_ = w.Repo.UpdateJobError(job.ID, err.Error())
		return
	}

	// Set track.Status = processing
	if statusErr := w.Repo.UpdateTrackStatus(track.ID, domain.TrackStatusProcessing, finalPath); statusErr != nil {
		logger.Error("Failed to update track status to processing", "error", statusErr)
	}

	logger.Info("Download finished, starting tagging", "file_path", finalPath)

	// Download album art for tagging
	var albumArtData []byte
	if track.AlbumArtURL != "" {
		albumArtData, err = w.albumArtService.DownloadImage(track.AlbumArtURL)
		if err != nil {
			logger.Error("Failed to download album art for tagging", "error", err)
		}
	}

	// Fetch lyrics if not already present
	if track.Lyrics == "" || track.Subtitles == "" {
		lyrics, subtitles, lyricsErr := w.ProviderManager.GetProvider().GetLyrics(ctx, track.ProviderID)
		if lyricsErr != nil {
			logger.Debug("Failed to fetch lyrics", "error", err)
		} else {
			if track.Lyrics == "" && lyrics != "" {
				track.Lyrics = lyrics
			}
			if track.Subtitles == "" && subtitles != "" {
				track.Subtitles = subtitles
			}
		}
	}

	// Fetch metadata from MusicBrainz if ISRC available
	if track.ISRC != "" {
		// Fetch recording metadata (album, artist, duration, release date, etc.)
		meta, mbErr := w.musicBrainzClient.GetRecordingByISRC(ctx, track.ISRC)
		if mbErr != nil {
			logger.Warn("Failed to fetch recording from MusicBrainz", "isrc", track.ISRC, "error", mbErr)
		} else if meta != nil {
			// Replace with MusicBrainz data (authoritative source)
			if meta.Album != "" {
				track.Album = meta.Album
			}
			if meta.Artist != "" {
				track.Artist = meta.Artist
			}
			if len(meta.Artists) > 0 {
				track.Artists = meta.Artists
			}
			if meta.Title != "" {
				track.Title = meta.Title
			}
			if meta.Duration > 0 {
				track.Duration = meta.Duration
			}
			if meta.Year > 0 {
				track.Year = meta.Year
			}
			if meta.ReleaseDate != "" {
				track.ReleaseDate = meta.ReleaseDate
			}
			if meta.Barcode != "" {
				track.Barcode = meta.Barcode
			}
			if meta.CatalogNumber != "" {
				track.CatalogNumber = meta.CatalogNumber
			}
			if meta.ReleaseType != "" {
				track.ReleaseType = meta.ReleaseType
			}
			if meta.ReleaseID != "" {
				track.ReleaseID = meta.ReleaseID
			}
		}

		// Fetch genre from MusicBrainz if not available from provider
		if track.Genre == "" {
			genres, genreErr := w.musicBrainzClient.GetGenresByISRC(ctx, track.ISRC)
			if genreErr != nil {
				logger.Warn("Failed to fetch genre from MusicBrainz", "isrc", track.ISRC, "error", genreErr)
			} else if len(genres) > 0 {
				track.Genre = genres[0]
			}
		}
	}

	// Tag the file
	if tagErr := tagging.TagFile(finalPath, track, albumArtData); tagErr != nil {
		logger.Error("Failed to tag file", "file_path", finalPath, "error", err)
	}

	// Save album art to folder
	if len(albumArtData) > 0 {
		artPath := filepath.Join(finalDir, "cover.jpg")
		if _, artStatErr := os.Stat(artPath); os.IsNotExist(artStatErr) {
			if writeErr := storage.WriteFile(artPath, albumArtData); writeErr != nil {
				logger.Error("Failed to save album art", "path", artPath, "error", err)
			} else {
				logger.Info("Saved album art", "path", artPath)
			}
		}
	}

	// Hash file
	fileHash, err := storage.HashFile(finalPath)
	if err != nil {
		logger.Error("Failed to hash file", "error", err)
		// Proceed but log? Or fail?
		// "store current hash"
	}

	// Mark track as completed
	ext = filepath.Ext(finalPath)
	if ext == "" {
		ext = ".flac"
	}
	track.FileExtension = ext

	// w.Repo.MarkTrackCompleted(track.ID, finalPath, fileHash)
	if err := w.Repo.MarkTrackCompleted(track.ID, finalPath, fileHash); err != nil {
		logger.Error("Failed to mark track completed", "error", err)
	}

	// Recompute album state
	if track.AlbumID != "" {
		// Just call it, we don't do anything with result yet other than maybe log or if we had a table
		// User requirement: "Create function: RecomputeAlbumState(albumID)"
		_, _ = w.Repo.RecomputeAlbumState(track.AlbumID)
	}

	// Mark job as completed
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
		// Check if already downloaded
		if downloaded, _ := w.Repo.IsTrackDownloaded(catalogTrack.ID); downloaded {
			continue
		}

		// Check if already active
		if active, _ := w.Repo.IsTrackActive(catalogTrack.ID); active {
			continue
		}

		// Create track record
		track := w.catalogTrackToDomainTrack(&catalogTrack)
		track.Status = domain.TrackStatusQueued
		track.ParentJobID = parentJobID
		track.CreatedAt = time.Now()
		track.UpdatedAt = time.Now()

		if err := w.Repo.CreateTrack(track); err != nil {
			logger.Error("Failed to create track record", "track_id", catalogTrack.ID, "error", err)
			continue
		}

		// Create child job
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
