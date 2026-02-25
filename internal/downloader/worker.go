package downloader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/cesargomez89/navidrums/internal/app"
	"github.com/cesargomez89/navidrums/internal/catalog"
	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/constants"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/logger"
	"github.com/cesargomez89/navidrums/internal/musicbrainz"
	"github.com/cesargomez89/navidrums/internal/storage"
	"github.com/cesargomez89/navidrums/internal/store"
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
	enricher          *app.MetadataEnricher
	dispatcher        *Dispatcher
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
	worker.enricher = app.NewMetadataEnricher(worker.musicBrainzClient, pm)

	worker.dispatcher = NewDispatcher()

	trackHandler := &TrackJobHandler{
		Repo:            repo,
		Config:          cfg,
		ProviderManager: pm,
		Downloader:      worker.downloader,
		AlbumArtService: worker.albumArtService,
		Enricher:        worker.enricher,
	}

	containerHandler := &ContainerJobHandler{
		Repo:              repo,
		ProviderManager:   pm,
		AlbumArtService:   worker.albumArtService,
		PlaylistGenerator: worker.playlistGenerator,
	}

	syncHandler := &SyncJobHandler{
		Repo:            repo,
		ProviderManager: pm,
		AlbumArtService: worker.albumArtService,
		Enricher:        worker.enricher,
	}

	worker.dispatcher.Register(domain.JobTypeTrack, trackHandler)
	worker.dispatcher.Register(domain.JobTypeAlbum, containerHandler)
	worker.dispatcher.Register(domain.JobTypePlaylist, containerHandler)
	worker.dispatcher.Register(domain.JobTypeArtist, containerHandler)
	worker.dispatcher.Register(domain.JobTypeSyncFile, syncHandler)
	worker.dispatcher.Register(domain.JobTypeSyncMusicBrainz, syncHandler)
	worker.dispatcher.Register(domain.JobTypeSyncHiFi, syncHandler)

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
			jobs, err := w.Repo.ListActiveJobs(0, 50)
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
	if err := w.dispatcher.Dispatch(ctx, job, logger); err != nil {
		logger.Error("Job processing failed", "error", err)
		if err == ErrUnknownJobType {
			_ = w.Repo.UpdateJobError(job.ID, "Unknown job type")
		}
	}
}

func (w *Worker) isCancelled(id string) bool {
	job, err := w.Repo.GetJob(id)
	if err != nil {
		return false
	}
	return job.Status == domain.JobStatusCancelled
}
