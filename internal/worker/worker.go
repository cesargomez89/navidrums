package worker

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/models"
	"github.com/cesargomez89/navidrums/internal/providers"
	"github.com/cesargomez89/navidrums/internal/repository"
	"github.com/google/uuid"
)

type Worker struct {
	Repo          *repository.DB
	Provider      providers.Provider
	Config        *config.Config
	MaxConcurrent int
	Logger        *log.Logger
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewWorker(repo *repository.DB, provider providers.Provider, cfg *config.Config) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		Repo:          repo,
		Provider:      provider,
		Config:        cfg,
		MaxConcurrent: 2,
		Logger:        log.New(os.Stdout, "[worker] ", log.LstdFlags),
		ctx:           ctx,
		cancel:        cancel,
	}
}

func (w *Worker) Start() {
	w.Logger.Println("Starting worker...")

	if err := w.Repo.ResetStuckJobs(); err != nil {
		w.Logger.Printf("Failed to reset stuck jobs: %v", err)
	}

	// Start polling loop
	w.wg.Add(1)
	go w.processJobs()
}

func (w *Worker) Stop() {
	w.Logger.Println("Stopping worker...")
	w.cancel()
	w.wg.Wait()
}

func (w *Worker) processJobs() {
	defer w.wg.Done()
	ticker := time.NewTicker(2 * time.Second)
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
				w.Logger.Printf("Failed to list jobs: %v", err)
				continue
			}

			if len(jobs) == 0 {
				continue
			}

			// Process jobs that fit within concurrency limit
			// Actually, "ListActiveJobs" might return running jobs too.
			// We only want 'queued' or 'resolving' that haven't started running properly?
			// No, the spec: "Handler creates job -> Worker picks job".
			// But concurrency is per *job*? Or per *file*?
			// Spec: "Worker picks job -> Resolve -> Download -> ...".
			// BUT: "Concurrent downloads: Max 2".
			// Does this mean max 2 *active* jobs? Or max 2 concurrent *file downloads*?
			// Spec: "Max 2 concurrent downloads" and "Concurrent jobs: Max 2 jobs simultaneously".
			// So 2 active jobs.

			activeCount := 0
			queuedJobs := []*models.Job{}

			for _, j := range jobs {
				if j.Status == models.JobStatusDownloading || j.Status == models.JobStatusTagging || j.Status == models.JobStatusResolve {
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
			w.Logger.Printf("Panic in job %s: %v", job.ID, r)
			w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Panic: %v", r))
		}
	}()

	w.Logger.Printf("Running job %s type %s id %s", job.ID, job.Type, job.SourceID)

	// 1. Resolution phase
	if err := w.Repo.UpdateJobStatus(job.ID, models.JobStatusResolve, 0); err != nil {
		w.Logger.Printf("Failed to update status: %v", err)
		return
	}

	var tracks []models.Track
	var err error

	// Fetch details based on type
	switch job.Type {
	case models.JobTypeTrack:
		var track *models.Track
		track, err = w.Provider.GetTrack(ctx, job.SourceID)
		if err == nil {
			tracks = []models.Track{*track}
		}
	case models.JobTypeAlbum:
		var album *models.Album
		album, err = w.Provider.GetAlbum(ctx, job.SourceID)
		if err == nil {
			tracks = album.Tracks
		}
	case models.JobTypePlaylist:
		var pl *models.Playlist
		pl, err = w.Provider.GetPlaylist(ctx, job.SourceID)
		if err == nil {
			tracks = pl.Tracks
			w.generatePlaylistFile(pl)
		}
	case models.JobTypeArtist:
		artist, err := w.Provider.GetArtist(ctx, job.SourceID)
		if err == nil {
			tracks = artist.TopTracks
		}
	}

	if err != nil {
		w.Logger.Printf("Job %s request error: %v", job.ID, err)
		w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Resolution failed: %v", err))
		return
	}

	if len(tracks) == 0 {
		w.Logger.Printf("Job %s: No tracks found", job.ID)
		w.Repo.UpdateJobError(job.ID, "No tracks found")
		return
	}

	// 2. Handle Container Types (Album, Playlist, Artist)
	if job.Type != models.JobTypeTrack {
		w.Logger.Printf("Job %s: decomposing container job into %d tracks", job.ID, len(tracks))

		for _, t := range tracks {
			// Check if already downloaded
			dl, _ := w.Repo.GetDownload(t.ID)
			if dl != nil {
				w.Logger.Printf("Track %s already downloaded, skipping", t.ID)
				continue
			}

			// Check if already active
			active, _ := w.Repo.IsTrackActive(t.ID)
			if active {
				w.Logger.Printf("Track %s already being processed, skipping", t.ID)
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
				w.Logger.Printf("Failed to create track job for %s: %v", t.ID, err)
			}
		}

		w.Repo.UpdateJobStatus(job.ID, models.JobStatusCompleted, 100)
		return
	}

	// 3. Handle Single Track Type
	track := tracks[0]

	// Check if already downloaded
	dl, _ := w.Repo.GetDownload(track.ID)
	if dl != nil {
		w.Logger.Printf("Track %s already downloaded at %s, completing job", track.ID, dl.FilePath)
		w.Repo.UpdateJobStatus(job.ID, models.JobStatusCompleted, 100)
		return
	}

	// Update job info if it was generic
	// (Note: Currently we don't have UpdateJobMetadata, but we can set title/artist in status update if we extend it)

	w.Repo.UpdateJobStatus(job.ID, models.JobStatusDownloading, 0)

	// Attempt download
	var filePath string
	var finalExt string
	for attempt := 0; attempt < 3; attempt++ {
		stream, mimeType, err := w.Provider.GetStream(ctx, track.ID, w.Config.Quality)
		if err == nil {
			// Determine extension
			ext := ".flac"
			if mimeType == "audio/mp4" {
				ext = ".mp4"
			} else if mimeType == "audio/mpeg" {
				ext = ".mp3"
			}
			finalExt = ext

			tempDir := os.TempDir()
			tempFile := filepath.Join(tempDir, fmt.Sprintf("%s-%s%s", job.ID, track.ID, ext))

			f, err := os.Create(tempFile)
			if err == nil {
				_, err = io.Copy(f, stream)
				stream.Close()
				f.Close()
				if err == nil {
					filePath = tempFile
					break
				}
			}
		}
		w.Logger.Printf("Download attempt %d failed for track %s: %v", attempt+1, track.ID, err)
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}

	if filePath == "" {
		w.Repo.UpdateJobError(job.ID, "Download failed after 3 attempts")
		return
	}

	// Success, move to final destination
	folderName := fmt.Sprintf("%s - %s", sanitize(track.Artist), sanitize(track.Album))
	trackFile := fmt.Sprintf("%02d - %s%s", track.TrackNumber, sanitize(track.Title), finalExt)
	finalPath := filepath.Join(w.Config.DownloadsDir, folderName, trackFile)

	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		w.Logger.Printf("Failed to create dir: %v", err)
		w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to create directory: %v", err))
		return
	}

	if err := os.Rename(filePath, finalPath); err != nil {
		w.Logger.Printf("Failed to move file: %v", err)
		w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to move file: %v", err))
		return
	}

	// Record download in DB
	w.Repo.CreateDownload(&models.Download{
		ProviderID:  track.ID,
		FilePath:    finalPath,
		CompletedAt: time.Now(),
	})

	w.Repo.UpdateJobStatus(job.ID, models.JobStatusCompleted, 100)
}

func (w *Worker) generatePlaylistFile(pl *models.Playlist) {
	if len(pl.Tracks) == 0 {
		return
	}

	playlistsDir := filepath.Join(w.Config.DownloadsDir, "playlists")
	if err := os.MkdirAll(playlistsDir, 0755); err != nil {
		w.Logger.Printf("Failed to create playlists dir: %v", err)
		return
	}

	filename := sanitize(pl.Title) + ".m3u"
	playlistPath := filepath.Join(playlistsDir, filename)

	f, err := os.Create(playlistPath)
	if err != nil {
		w.Logger.Printf("Failed to create playlist file: %v", err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString("#EXTM3U\n"); err != nil {
		w.Logger.Printf("Failed to write to playlist file: %v", err)
		return
	}

	for _, t := range pl.Tracks {
		folderName := fmt.Sprintf("%s - %s", sanitize(t.Artist), sanitize(t.Album))
		trackFile := fmt.Sprintf("%02d - %s.flac", t.TrackNumber, sanitize(t.Title))
		// Path relative to 'playlists' folder: ../Artist - Album/01 - Title.flac
		relPath := filepath.Join("..", folderName, trackFile)

		line := fmt.Sprintf("#EXTINF:%d,%s - %s\n%s\n", t.Duration, t.Artist, t.Title, relPath)
		if _, err := f.WriteString(line); err != nil {
			w.Logger.Printf("Failed to write track to playlist file: %v", err)
			continue
		}
	}

	w.Logger.Printf("Generated playlist file: %s", playlistPath)
}

func sanitize(s string) string {
	// Simple sanitize
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune("<>:\"/\\|?*", r) {
			return -1
		}
		return r
	}, s)
}
