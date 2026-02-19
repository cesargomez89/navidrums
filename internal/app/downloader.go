package app

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/cesargomez89/navidrums/internal/catalog"
	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/constants"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/storage"
)

type Downloader interface {
	Download(ctx context.Context, track *domain.Track, destPathNoExt string) (string, error)
}

type downloader struct {
	providerManager *catalog.ProviderManager
	config          *config.Config
}

func NewDownloader(pm *catalog.ProviderManager, cfg *config.Config) Downloader {
	return &downloader{
		providerManager: pm,
		config:          cfg,
	}
}

func (d *downloader) Download(ctx context.Context, track *domain.Track, destPathNoExt string) (string, error) {
	var finalPath string
	var finalExt string

	provider := d.providerManager.GetProvider()

	for attempt := 0; attempt < constants.DefaultRetryCount; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		stream, mimeType, err := provider.GetStream(ctx, track.ProviderID, d.config.Quality)
		if err != nil {
			time.Sleep(time.Duration(attempt+1) * constants.DefaultRetryBase)
			continue
		}

		ext := constants.ExtFLAC
		switch mimeType {
		case constants.MimeTypeMP4:
			ext = constants.ExtMP4
		case constants.MimeTypeMP3:
			ext = constants.ExtMP3
		}
		finalExt = ext

		finalPath = destPathNoExt + finalExt

		f, err := storage.CreateFile(finalPath)
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

		storage.RemoveFile(finalPath)
		finalPath = ""

		time.Sleep(time.Duration(attempt+1) * constants.DefaultRetryBase)
	}

	return "", fmt.Errorf("download failed after %d attempts", constants.DefaultRetryCount)
}
