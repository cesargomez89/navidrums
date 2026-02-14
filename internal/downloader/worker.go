package downloader

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cesargomez89/navidrums/internal/app"
	"github.com/cesargomez89/navidrums/internal/catalog"
	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/constants"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/logger"
	"github.com/cesargomez89/navidrums/internal/storage"
	"github.com/cesargomez89/navidrums/internal/store"
	"github.com/cesargomez89/navidrums/internal/tagging"
	"github.com/google/uuid"
)

var (
	ErrJobCancelled   = errors.New("job was cancelled")
	ErrDownloadFailed = errors.New("download failed after retries")
	ErrNoTracksFound  = errors.New("no tracks found")
)

type Worker struct {
	Repo              *store.DB
	Provider          catalog.Provider
	ProviderManager   *catalog.ProviderManager
	Config            *config.Config
	MaxConcurrent     int
	Logger            *logger.Logger
	downloader        app.Downloader
	playlistGenerator app.PlaylistGenerator
	albumArtService   app.AlbumArtService
	wg                sync.WaitGroup
	ctx               context.Context
	cancel            context.CancelFunc
}

func NewWorker(repo *store.DB, pm *catalog.ProviderManager, cfg *config.Config, log *logger.Logger) *Worker {
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

	// Initialize app
	worker.downloader = app.NewDownloader(pm, cfg)
	worker.playlistGenerator = app.NewPlaylistGenerator(cfg)
	worker.albumArtService = app.NewAlbumArtService(cfg)

	return worker
}

func (w *Worker) Start() {
	w.Logger.Info("Starting worker")

	if err := w.Repo.ResetStuckJobs(); err != nil {
		w.Logger.Error("Failed to reset stuck jobs", "error", err)
	}

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
			queuedJobs := []*domain.Job{}

			for _, j := range jobs {
				if j.Status == domain.JobStatusDownloading || j.Status == domain.JobStatusResolve {
					activeCount++
				} else if j.Status == domain.JobStatusQueued {
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
	if err := w.Repo.UpdateJobStatus(job.ID, domain.JobStatusResolve, 0); err != nil {
		logger.Error("Failed to update status", "error", err)
		return
	}

	// Check if cancelled
	if w.isCancelled(job.ID) {
		logger.Info("Job cancelled before resolution finished")
		return
	}

	var tracks []domain.Track
	var albumArtURL string
	var playlistImageURL string
	var err error

	// Fetch details based on type
	switch job.Type {
	case domain.JobTypeTrack:
		var track *domain.Track
		track, err = w.Provider.GetTrack(ctx, job.SourceID)
		if err == nil {
			tracks = []domain.Track{*track}
			albumArtURL = track.AlbumArtURL
		}
	case domain.JobTypeAlbum:
		var album *domain.Album
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
	case domain.JobTypePlaylist:
		var pl *domain.Playlist
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
			extLookup := func(trackID string) string {
				dl, _ := w.Repo.GetDownload(trackID)
				if dl != nil && dl.FileExtension != "" {
					return dl.FileExtension
				}
				return ".flac"
			}
			if err := w.playlistGenerator.Generate(pl, extLookup); err != nil {
				logger.Error("Failed to generate playlist file", "error", err)
			}
		}
	case domain.JobTypeArtist:
		var artist *domain.Artist
		artist, err = w.Provider.GetArtist(ctx, job.SourceID)
		if err == nil {
			tracks = artist.TopTracks
			// Generate playlist file for top tracks
			if len(tracks) > 0 {
				extLookup := func(trackID string) string {
					dl, _ := w.Repo.GetDownload(trackID)
					if dl != nil && dl.FileExtension != "" {
						return dl.FileExtension
					}
					return ".flac"
				}
				if err := w.playlistGenerator.GenerateFromTracks(artist.Name, tracks, extLookup); err != nil {
					logger.Error("Failed to generate playlist file", "error", err)
				}
			}
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
	if job.Type != domain.JobTypeTrack {
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
			newJob := &domain.Job{
				ID:        uuid.New().String(),
				Type:      domain.JobTypeTrack,
				Status:    domain.JobStatusQueued,
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

		if err := w.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100); err != nil {
			logger.Error("Failed to update job status to completed", "error", err)
		}

		// Update job metadata with container info before completing
		switch job.Type {
		case domain.JobTypeAlbum:
			if len(tracks) > 0 {
				if err := w.Repo.UpdateJobMetadata(job.ID, tracks[0].Album, tracks[0].AlbumArtist); err != nil {
					logger.Error("Failed to update job metadata", "error", err)
				}
			}
		case domain.JobTypePlaylist:
			if pl, err := w.Provider.GetPlaylist(ctx, job.SourceID); err == nil && pl != nil {
				if err := w.Repo.UpdateJobMetadata(job.ID, pl.Title, ""); err != nil {
					logger.Error("Failed to update job metadata", "error", err)
				}
			}
		case domain.JobTypeArtist:
			if artist, err := w.Provider.GetArtist(ctx, job.SourceID); err == nil && artist != nil {
				if err := w.Repo.UpdateJobMetadata(job.ID, artist.Name, ""); err != nil {
					logger.Error("Failed to update job metadata", "error", err)
				}
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
		if err := w.Repo.UpdateJobMetadata(job.ID, track.Title, track.Artist); err != nil {
			logger.Error("Failed to update job metadata", "error", err)
		}
		if err := w.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100); err != nil {
			logger.Error("Failed to update job status", "error", err)
		}
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
	folderName := fmt.Sprintf("%s - %s", storage.Sanitize(artistForFolder), storage.Sanitize(track.Album))
	finalDir := filepath.Join(w.Config.DownloadsDir, folderName)

	if err := storage.EnsureDir(finalDir); err != nil {
		logger.Error("Failed to create directory", "error", err)
		w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to create directory: %v", err))
		return
	}

	if w.isCancelled(job.ID) {
		logger.Info("Job cancelled before download")
		return
	}

	w.Repo.UpdateJobStatus(job.ID, domain.JobStatusDownloading, 0)

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
	if track.Lyrics == "" || track.Subtitles == "" {
		lyrics, subtitles, err := w.Provider.GetLyrics(ctx, track.ID)
		if err != nil {
			logger.Debug("Failed to fetch lyrics", "error", err)
		} else {
			if track.Lyrics == "" && lyrics != "" {
				track.Lyrics = lyrics
			}
			if track.Subtitles == "" && subtitles != "" {
				track.Subtitles = subtitles
			}
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
			if err := storage.WriteFile(artPath, albumArtData); err != nil {
				logger.Error("Failed to save album art", "path", artPath, "error", err)
			} else {
				logger.Info("Saved album art", "path", artPath)
			}
		}
	}

	// Record download in DB
	ext := filepath.Ext(finalPath)
	if ext == "" {
		ext = ".flac" // Default extension
	}
	err = w.Repo.CreateDownload(&domain.Download{
		ProviderID:    track.ID,
		FilePath:      finalPath,
		FileExtension: ext,
		CompletedAt:   time.Now(),
	})
	if err != nil {
		logger.Error("Failed to record download in DB", "error", err)
	}

	if err := w.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100); err != nil {
		logger.Error("Failed to update final status", "error", err)
	}

	logger.Info("Job completed successfully")
}

func (w *Worker) isCancelled(id string) bool {
	job, err := w.Repo.GetJob(id)
	if err != nil {
		return false
	}
	return job.Status == domain.JobStatusCancelled
}
