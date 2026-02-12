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
	"github.com/cesargomez89/navidrums/internal/tagging"
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

	// Check if cancelled
	if w.isCancelled(job.ID) {
		w.Logger.Printf("Job %s cancelled before resolution finished", job.ID)
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
			// Save album art to album folder
			if albumArtURL != "" {
				w.saveAlbumArt(album, albumArtURL)
			}
		}
	case models.JobTypePlaylist:
		var pl *models.Playlist
		pl, err = w.Provider.GetPlaylist(ctx, job.SourceID)
		if err == nil {
			tracks = pl.Tracks
			playlistImageURL = pl.ImageURL
			// Save playlist image
			if playlistImageURL != "" {
				w.savePlaylistImage(pl, playlistImageURL)
			}
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
		if w.isCancelled(job.ID) {
			w.Logger.Printf("Job %s cancelled before decomposition", job.ID)
			return
		}
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
		// Update job metadata with container info before completing
		switch job.Type {
		case models.JobTypeAlbum:
			if len(tracks) > 0 {
				w.Repo.UpdateJobMetadata(job.ID, tracks[0].Album, tracks[0].AlbumArtist)
			}
		case models.JobTypePlaylist:
			// Try to get playlist title from provider if available
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
		w.Logger.Printf("Track %s already downloaded at %s, completing job", track.ID, dl.FilePath)
		// Update metadata even for already-downloaded tracks
		w.Repo.UpdateJobMetadata(job.ID, track.Title, track.Artist)
		w.Repo.UpdateJobStatus(job.ID, models.JobStatusCompleted, 100)
		return
	}

	// Update job metadata with actual track info
	if err := w.Repo.UpdateJobMetadata(job.ID, track.Title, track.Artist); err != nil {
		w.Logger.Printf("Failed to update job metadata: %v", err)
	}

	// Prepare final destination path
	artistForFolder := track.AlbumArtist
	if artistForFolder == "" {
		artistForFolder = track.Artist
	}
	folderName := fmt.Sprintf("%s - %s", sanitize(artistForFolder), sanitize(track.Album))
	finalDir := filepath.Join(w.Config.DownloadsDir, folderName)

	if err := os.MkdirAll(finalDir, 0755); err != nil {
		w.Logger.Printf("Failed to create dir: %v", err)
		w.Repo.UpdateJobError(job.ID, fmt.Sprintf("Failed to create directory: %v", err))
		return
	}

	if w.isCancelled(job.ID) {
		w.Logger.Printf("Job %s cancelled before download", job.ID)
		return
	}

	w.Repo.UpdateJobStatus(job.ID, models.JobStatusDownloading, 0)

	// Attempt download directly to final destination
	var finalPath string
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

			trackFile := fmt.Sprintf("%02d - %s%s", track.TrackNumber, sanitize(track.Title), finalExt)
			finalPath = filepath.Join(finalDir, trackFile)

			f, err := os.Create(finalPath)
			if err == nil {
				// Use a limited reader or just monitor progress
				// For now, let's just do io.Copy but check for cancellation occasionally?
				// Actually, we can use a custom writer that checks context
				pw := &progressWriter{
					jobID:  job.ID,
					repo:   w.Repo,
					logger: w.Logger,
					ctx:    ctx,
				}
				_, err = io.Copy(io.MultiWriter(f, pw), stream)
				stream.Close()
				f.Close()

				if err == nil {
					// Download successful
					break
				} else {
					// Clean up partial file on error/cancel
					os.Remove(finalPath)
					finalPath = ""
					if ctx.Err() != nil || w.isCancelled(job.ID) {
						w.Logger.Printf("Download cancelled for track %s", track.ID)
						return
					}
				}
			}
		}
		w.Logger.Printf("Download attempt %d failed for track %s: %v", attempt+1, track.ID, err)
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}

	if finalPath == "" {
		w.Repo.UpdateJobError(job.ID, "Download failed after 3 attempts")
		return
	}

	w.Logger.Printf("Job %s: Download finished, starting tagging", job.ID)

	// Download album art for tagging
	var albumArtData []byte
	if track.AlbumArtURL != "" {
		albumArtData, err = w.downloadImage(track.AlbumArtURL)
		if err != nil {
			w.Logger.Printf("Failed to download album art for tagging: %v", err)
		}
	}

	// Tag the file with metadata
	if err := w.tagFile(finalPath, &track, albumArtData); err != nil {
		w.Logger.Printf("Failed to tag file %s: %v", finalPath, err)
		// Don't fail the job if tagging fails, just log it
	}

	// Save album art to folder if not already saved
	if len(albumArtData) > 0 {
		artPath := filepath.Join(finalDir, "cover.jpg")
		if _, err := os.Stat(artPath); os.IsNotExist(err) {
			if err := os.WriteFile(artPath, albumArtData, 0644); err != nil {
				w.Logger.Printf("Failed to save album art to %s: %v", artPath, err)
			} else {
				w.Logger.Printf("Saved cover.jpg to %s", artPath)
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
		w.Logger.Printf("Failed to record download in DB for job %s: %v", job.ID, err)
		// Still try to complete the job
	}

	if err := w.Repo.UpdateJobStatus(job.ID, models.JobStatusCompleted, 100); err != nil {
		w.Logger.Printf("Failed to update final status for job %s: %v", job.ID, err)
	}
	w.Logger.Printf("Job %s completed successfully", job.ID)
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

// downloadImage downloads an image from a URL and returns the image data
func (w *Worker) downloadImage(url string) ([]byte, error) {
	return tagging.DownloadImage(url)
}

// tagFile tags an audio file with metadata
func (w *Worker) tagFile(filePath string, track *models.Track, albumArtData []byte) error {
	return tagging.TagFile(filePath, track, albumArtData)
}

// saveAlbumArt saves album artwork to the album folder
func (w *Worker) saveAlbumArt(album *models.Album, imageURL string) {
	if imageURL == "" {
		return
	}

	// Download image
	imageData, err := w.downloadImage(imageURL)
	if err != nil {
		w.Logger.Printf("Failed to download album art for %s: %v", album.Title, err)
		return
	}

	// Determine folder path
	folderName := fmt.Sprintf("%s - %s", sanitize(album.Artist), sanitize(album.Title))
	albumDir := filepath.Join(w.Config.DownloadsDir, folderName)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(albumDir, 0755); err != nil {
		w.Logger.Printf("Failed to create album directory: %v", err)
		return
	}

	// Save image
	imagePath := filepath.Join(albumDir, "cover.jpg")
	if err := tagging.SaveImageToFile(imageData, imagePath); err != nil {
		w.Logger.Printf("Failed to save album art to %s: %v", imagePath, err)
		return
	}

	w.Logger.Printf("Saved album art to %s (URL: %s)", imagePath, imageURL)
}

// savePlaylistImage saves playlist cover image to the playlists folder
func (w *Worker) savePlaylistImage(playlist *models.Playlist, imageURL string) {
	if imageURL == "" {
		return
	}

	// Download image
	imageData, err := w.downloadImage(imageURL)
	if err != nil {
		w.Logger.Printf("Failed to download playlist image for %s: %v", playlist.Title, err)
		return
	}

	// Determine folder path
	playlistsDir := filepath.Join(w.Config.DownloadsDir, "playlists")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(playlistsDir, 0755); err != nil {
		w.Logger.Printf("Failed to create playlists directory: %v", err)
		return
	}

	// Save image
	imagePath := filepath.Join(playlistsDir, sanitize(playlist.Title)+".jpg")
	if err := tagging.SaveImageToFile(imageData, imagePath); err != nil {
		w.Logger.Printf("Failed to save playlist image to %s: %v", imagePath, err)
		return
	}

	w.Logger.Printf("Saved playlist image to %s", imagePath)
}

func (w *Worker) isCancelled(id string) bool {
	job, err := w.Repo.GetJob(id)
	if err != nil {
		return false
	}
	return job.Status == models.JobStatusCancelled
}

type progressWriter struct {
	jobID      string
	repo       *repository.DB
	logger     *log.Logger
	ctx        context.Context
	total      int64
	written    int64
	lastUpdate time.Time
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	if pw.ctx.Err() != nil {
		return 0, pw.ctx.Err()
	}

	n = len(p)
	pw.written += int64(n)

	// Since we don't always know the total size from the stream,
	// we can't easily calculate percentage here unless we get it from elsewhere.
	// But we can update 'updated_at' to show it's still alive.

	if time.Since(pw.lastUpdate) > 2*time.Second {
		// Just update status to keep it "fresh" in DB
		_ = pw.repo.UpdateJobStatus(pw.jobID, models.JobStatusDownloading, 0) // Progress 0 if unknown
		pw.lastUpdate = time.Now()

		// Also check for cancellation in DB
		job, _ := pw.repo.GetJob(pw.jobID)
		if job != nil && job.Status == models.JobStatusCancelled {
			return 0, fmt.Errorf("job cancelled")
		}
	}

	return n, nil
}
