package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/cesargomez89/navidrums/internal/catalog"
	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/constants"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/ffmpeg"
	"github.com/cesargomez89/navidrums/internal/storage"
)

type Downloader interface {
	Download(ctx context.Context, track *domain.Track, destPathNoExt string, logger *slog.Logger) (string, error)
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

func (d *downloader) Download(ctx context.Context, track *domain.Track, destPathNoExt string, logger *slog.Logger) (string, error) {
	provider := d.providerManager.GetDownloadProvider()

	shouldConvertToFLAC := d.config.Quality == constants.QualityHiResLossless

	var lastErr error

	for attempt := 0; attempt < constants.DefaultRetryCount; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		stream, mimeType, err := provider.GetStream(ctx, track.ProviderID, d.config.Quality)
		if err != nil {
			lastErr = err
			logger.Error("Download attempt failed",
				"attempt", attempt+1,
				"total_attempts", constants.DefaultRetryCount,
				"track_id", track.ID,
				"track_title", track.Title,
				"provider_id", track.ProviderID,
				"error", err,
			)
			time.Sleep(time.Duration(attempt+1) * constants.DefaultRetryBase)
			continue
		}

		ext := constants.ExtFLAC
		switch mimeType {
		case constants.MimeTypeMP4:
			ext = constants.ExtM4A
		case constants.MimeTypeMP3:
			ext = constants.ExtMP3
		}

		downloadPath := destPathNoExt + ext

		f, err := storage.CreateFile(downloadPath)
		if err != nil {
			_ = stream.Close()
			continue
		}

		_, err = io.Copy(f, stream)
		_ = stream.Close()
		_ = f.Close()

		if err != nil {
			lastErr = err
			_ = storage.RemoveFile(downloadPath)
			time.Sleep(time.Duration(attempt+1) * constants.DefaultRetryBase)
			continue
		}

		if shouldConvertToFLAC && mimeType == constants.MimeTypeMP4 {
			flacPath, convErr := ffmpeg.ConvertToFLAC(ctx, downloadPath)
			if convErr != nil {
				lastErr = convErr
				_ = storage.RemoveFile(downloadPath)
				time.Sleep(time.Duration(attempt+1) * constants.DefaultRetryBase)
				continue
			}

			if err := storage.RemoveFile(downloadPath); err != nil {
				// We won't retry just because cleanup failed, success is the .flac
				return flacPath, nil
			}

			return flacPath, nil
		}

		return downloadPath, nil
	}

	return "", fmt.Errorf("download failed after %d attempts: %w", constants.DefaultRetryCount, lastErr)
}
