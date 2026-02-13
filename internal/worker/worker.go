package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/constants"
	"github.com/cesargomez89/navidrums/internal/filesystem"
	"github.com/cesargomez89/navidrums/internal/logger"
	"github.com/cesargomez89/navidrums/internal/models"
	"github.com/cesargomez89/navidrums/internal/providers"
	"github.com/cesargomez89/navidrums/internal/repository"
	"github.com/cesargomez89/navidrums/internal/services"
	"github.com/cesargomez89/navidrums/internal/tagging"
	"github.com/google/uuid"
)

// Errors
var (
	ErrJobCancelled   = errors.New("job was cancelled")
	ErrDownloadFailed = errors.New("download failed after retries")
	ErrNoTracksFound  = errors.New("no tracks found")
)

// Worker handles background job processing
type Worker struct {
	Repo              *repository.DB
	Provider          providers.Provider
	ProviderManager   *providers.ProviderManager
	Config            *config.Config
	MaxConcurrent     int
	Logger            *logger.Logger
	downloader        services.Downloader
	playlistGenerator services.PlaylistGenerator
	albumArtService   services.AlbumArtService
	wg                sync.WaitGroup
	ctx               context.Context
	cancel            context.CancelFunc
}

// NewWorker creates a new Worker with all dependencies
func NewWorker(repo *repository.DB, pm *providers.ProviderManager, cfg *config.Config, log *logger.Logger) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	if log == nil {
		log = logger.Default()
	}

	worker := &Worker{
		Repo:            repo,
		ProviderManager: pm,
		Provider:        pm,
		Config:          cfg,
		MaxConcurrent:   constants.DefaultConcurrency,
		Logger:          log.WithComponent("worker"),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Initialize services
	worker.downloader = services.NewDownloader(pm, cfg, repo)
	worker.playlistGenerator = services.NewPlaylistGenerator(cfg)
	worker.albumArtService = services.NewAlbumArtService(cfg)

	return worker
}

func (w *Worker) Start() {
	w.Logger.Info("Starting worker")

	if err := w.Repo.ResetStuckJobs(); err != nil {
		w.Logger.Error("Failed to reset stuck jobs", "error", err)
	}

	// Start polling loop
	w.wg.Add(1)
	go w.processJobs()
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
			// List available jobs
			jobs, err := w.Repo.ListActiveJobs()
			if err != nil {
				w.Logger.Error("Failed to list jobs", "error", err)
				continue
			}

			if len(jobs) == 0 {
				continue
			}

			activeCount := 0
			queuedJobs := []*models.Job{}

			for _, j := range jobs {
				if j.Status == models.JobStatusDownloading || j.Status == models.JobStatusResolve {
					activeCount++
				} else if j.Status == models.JobStatusQueued {
					queuedJobs = append(queuedJobs, j)
				}
			}

			toStart := w.MaxConcurrent - activeCount
			if toStart <= 0 || len(queuedJobs) == 0 {
				continue
			}

			// Launch workers for queued jobs
			for i := 0; i < toStart && i < len(queuedJobs); i++ {
				job := queuedJobs[i]

				// Double check if it was cancelled while in memory
				current, _ := w.Repo.GetJob(job.ID)
				if current != nil && current.Status == models.JobStatusCancelled {
					continue
				}

				sem <- struct{}{}
				w.wg.Add(1)
				go func(j *models.Job) {
					defer w.wg.Done()
					defer func() { <-sem }()
					w.runJob(w.ctx, j)
				}(job)
			}
		}
	}
}

func (w *Worker) runJob(ctx context.Context, job *models.Job) {
	defer func() {
		if r := recover(); r != nil {
			w.Logger.Error("Panic in job",
				"job_id", job.ID,
				"panic", r,
			)
			w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Panic: %v", r))
		}
	}()

	logger := w.Logger.With(
		"job_id", job.ID,
		"job_type", job.Type,
		"source_id", job.SourceID,
	)
	logger.Info("Running job")

	// 1. Resolution phase
	if err := w.Repo.UpdateJobStatus(job.ID, models.JobStatusResolve, 0); err != nil {
		logger.Error("Failed to update status", "error", err)
		return
	}

	// Check if cancelled
	if w.isCancelled(job.ID) {
		logger.Info("Job cancelled before resolution finished")
		return
	}

	var tracks []models.Track
	var albumArtURL string
	var playlistImageURL string
	var err error

	// Fetch details based on type
	switch job.Type {
	case models.JobTypeTrack:
		var track *models.Track
		track, err = w.Provider.GetTrack(ctx, job.SourceID)
		if err == nil {
			tracks = []models.Track{*track}
			albumArtURL = track.AlbumArtURL
		}
	case models.JobTypeAlbum:
		var album *models.Album
		album, err = w.Provider.GetAlbum(ctx, job.SourceID)
		if err == nil {
			tracks = album.Tracks
			albumArtURL = album.AlbumArtURL
			// Save album art using service
			if albumArtURL != "" {
				if err := w.albumArtService.DownloadAndSaveAlbumArt(album, albumArtURL); err != nil {
					logger.Error("Failed to save album art", "error", err)
				}
			}
		}
	case models.JobTypePlaylist:
		var pl *models.Playlist
		pl, err = w.Provider.GetPlaylist(ctx, job.SourceID)
		if err == nil {
			tracks = pl.Tracks
			playlistImageURL = pl.ImageURL
			// Save playlist image using service
			if playlistImageURL != "" {
				if err := w.albumArtService.DownloadAndSavePlaylistImage(pl, playlistImageURL); err != nil {
					logger.Error("Failed to save playlist image", "error", err)
				}
			}
			// Generate playlist file using service
			if err := w.playlistGenerator.Generate(pl); err != nil {
				logger.Error("Failed to generate playlist file", "error", err)
			}
		}
	case models.JobTypeArtist:
		artist, err := w.Provider.GetArtist(ctx, job.SourceID)
		if err == nil {
			tracks = artist.TopTracks
		}
	}

	if err != nil {
		logger.Error("Job request error", "error", err)
		w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Resolution failed: %v", err))
		return
	}

	if len(tracks) == 0 {
		logger.Error("No tracks found")
		w.Repo.UpdateJobError(job.ID, "No tracks found")
		return
	}

	// 2. Handle Container Types (Album, Playlist, Artist)
	if job.Type != models.JobTypeTrack {
		if w.isCancelled(job.ID) {
			logger.Info("Job cancelled before decomposition")
			return
		}
		logger.Info("Decomposing container job", "track_count", len(tracks))

		for _, t := range tracks {
			// Check if already downloaded
			dl, _ := w.Repo.GetDownload(t.ID)
			if dl != nil {
				logger.Debug("Track already downloaded, skipping", "track_id", t.ID)
				continue
			}

			// Check if already active
			active, _ := w.Repo.IsTrackActive(t.ID)
			if active {
				logger.Debug("Track already being processed, skipping", "track_id", t.ID)
				continue
			}

			// Enqueue new track job
			newJob := &models.Job{
				ID:        uuid.New().String(),
				Type:      models.JobTypeTrack,
				Status:    models.JobStatusQueued,
				SourceID:  t.ID,
				Title:     t.Title,
				Artist:    t.Artist,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := w.Repo.CreateJob(newJob); err != nil {
				logger.Error("Failed to create track job", "track_id", t.ID, "error", err)
			}
		}

		w.Repo.UpdateJobStatus(job.ID, models.JobStatusCompleted, 100)

		// Update job metadata with container info before completing
		switch job.Type {
		case models.JobTypeAlbum:
			if len(tracks) > 0 {
				w.Repo.UpdateJobMetadata(job.ID, tracks[0].Album, tracks[0].AlbumArtist)
			}
		case models.JobTypePlaylist:
			if pl, err := w.Provider.GetPlaylist(ctx, job.SourceID); err == nil && pl != nil {
				w.Repo.UpdateJobMetadata(job.ID, pl.Title, "")
			}
		case models.JobTypeArtist:
			if artist, err := w.Provider.GetArtist(ctx, job.SourceID); err == nil && artist != nil {
				w.Repo.UpdateJobMetadata(job.ID, artist.Name, "")
			}
		}
		return
	}

	// 3. Handle Single Track Type
	track := tracks[0]

	// Check if already downloaded
	dl, _ := w.Repo.GetDownload(track.ID)
	if dl != nil {
		logger.Info("Track already downloaded", "file_path", dl.FilePath)
		w.Repo.UpdateJobMetadata(job.ID, track.Title, track.Artist)
		w.Repo.UpdateJobStatus(job.ID, models.JobStatusCompleted, 100)
		return
	}

	// Update job metadata with actual track info
	if err := w.Repo.UpdateJobMetadata(job.ID, track.Title, track.Artist); err != nil {
		logger.Error("Failed to update job metadata", "error", err)
	}

	// Prepare final destination path
	artistForFolder := track.AlbumArtist
	if artistForFolder == "" {
		artistForFolder = track.Artist
	}
	folderName := fmt.Sprintf("%s - %s", filesystem.Sanitize(artistForFolder), filesystem.Sanitize(track.Album))
	finalDir := filepath.Join(w.Config.DownloadsDir, folderName)

	if err := os.MkdirAll(finalDir, 0755); err != nil {
		logger.Error("Failed to create directory", "error", err)
		w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to create directory: %v", err))
		return
	}

	if w.isCancelled(job.ID) {
		logger.Info("Job cancelled before download")
		return
	}

	w.Repo.UpdateJobStatus(job.ID, models.JobStatusDownloading, 0)

	// Download using service
	finalPath, err := w.downloader.Download(ctx, track, finalDir)

	if err != nil {
		logger.Error("Download failed", "track_id", track.ID, "error", err)
		w.Repo.UpdateJobError(job.ID, err.Error())
		return
	}

	logger.Info("Download finished, starting tagging",
		"file_path", finalPath,
	)

	// Download album art for tagging
	var albumArtData []byte
	if track.AlbumArtURL != "" {
		albumArtData, err = w.albumArtService.DownloadImage(track.AlbumArtURL)
		if err != nil {
			logger.Error("Failed to download album art for tagging", "error", err)
		}
	}

	// Fetch lyrics if not already present
	if track.Lyrics == "" {
		lyrics, _, err := w.Provider.GetLyrics(ctx, track.ID)
		if err != nil {
			logger.Debug("Failed to fetch lyrics", "error", err)
		} else {
			track.Lyrics = lyrics
			logger.Debug("Fetched lyrics successfully")
		}
	}

	// Tag the file with metadata
	if err := tagging.TagFile(finalPath, &track, albumArtData); err != nil {
		logger.Error("Failed to tag file", "file_path", finalPath, "error", err)
	}

	// Save album art to folder if not already saved
	if len(albumArtData) > 0 {
		artPath := filepath.Join(finalDir, "cover.jpg")
		if _, err := os.Stat(artPath); os.IsNotExist(err) {
			if err := os.WriteFile(artPath, albumArtData, 0644); err != nil {
				logger.Error("Failed to save album art", "path", artPath, "error", err)
			} else {
				logger.Info("Saved album art", "path", artPath)
			}
		}
	}

	// Record download in DB
	err = w.Repo.CreateDownload(&models.Download{
		ProviderID:  track.ID,
		FilePath:    finalPath,
		CompletedAt: time.Now(),
	})
	if err != nil {
		logger.Error("Failed to record download in DB", "error", err)
	}

	if err := w.Repo.UpdateJobStatus(job.ID, models.JobStatusCompleted, 100); err != nil {
		logger.Error("Failed to update final status", "error", err)
	}

	logger.Info("Job completed successfully")
}

func (w *Worker) isCancelled(id string) bool {
	job, err := w.Repo.GetJob(id)
	if err != nil {
		return false
	}
	return job.Status == models.JobStatusCancelled
}
