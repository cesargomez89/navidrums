// Package services provides business logic services for the application
package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/constants"
	"github.com/cesargomez89/navidrums/internal/filesystem"
	"github.com/cesargomez89/navidrums/internal/models"
	"github.com/cesargomez89/navidrums/internal/providers"
	"github.com/cesargomez89/navidrums/internal/repository"
)

// Downloader handles track downloading with retry logic
type Downloader interface {
	Download(ctx context.Context, track models.Track, destDir string) (string, error)
}

type downloader struct {
	provider providers.Provider
	config   *config.Config
	repo     *repository.DB
}

// NewDownloader creates a new Downloader service
func NewDownloader(provider providers.Provider, cfg *config.Config, repo *repository.DB) Downloader {
	return &downloader{
		provider: provider,
		config:   cfg,
		repo:     repo,
	}
}

// Download downloads a track with retry logic
func (d *downloader) Download(ctx context.Context, track models.Track, destDir string) (string, error) {
	var finalPath string
	var finalExt string

	for attempt := 0; attempt < constants.DefaultRetryCount; attempt++ {
		stream, mimeType, err := d.provider.GetStream(ctx, track.ID, d.config.Quality)
		if err != nil {
			time.Sleep(time.Duration(attempt+1) * constants.DefaultRetryBase)
			continue
		}

		// Determine file extension from MIME type
		ext := constants.ExtFLAC
		switch mimeType {
		case constants.MimeTypeMP4:
			ext = constants.ExtMP4
		case constants.MimeTypeMP3:
			ext = constants.ExtMP3
		}
		finalExt = ext

		trackFile := fmt.Sprintf("%02d - %s%s", track.TrackNumber, filesystem.Sanitize(track.Title), finalExt)
		finalPath = filepath.Join(destDir, trackFile)

		f, err := os.Create(finalPath)
		if err != nil {
			stream.Close()
			continue
		}

		_, err = io.Copy(f, stream)
		stream.Close()
		f.Close()

		if err == nil {
			return finalPath, nil
		}

		// Clean up partial file on error
		os.Remove(finalPath)
		finalPath = ""

		time.Sleep(time.Duration(attempt+1) * constants.DefaultRetryBase)
	}

	return "", fmt.Errorf("download failed after %d attempts", constants.DefaultRetryCount)
}
