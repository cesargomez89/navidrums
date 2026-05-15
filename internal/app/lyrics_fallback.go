package app

import (
	"context"
	"log/slog"

	"github.com/cesargomez89/navidrums/internal/catalog"
	"github.com/cesargomez89/navidrums/internal/domain"
)

type LyricsFallback struct {
	enabled   bool
	providers []catalog.LyricsProvider
}

func NewLyricsFallback(enabled bool, providers []catalog.LyricsProvider) *LyricsFallback {
	return &LyricsFallback{
		enabled:   enabled,
		providers: providers,
	}
}

func (f *LyricsFallback) Fetch(ctx context.Context, track *domain.Track, logger *slog.Logger) {
	if !f.enabled {
		return
	}
	if len(f.providers) == 0 {
		return
	}
	if track.Lyrics != "" || track.Subtitles != "" {
		return
	}

	for _, provider := range f.providers {
		logger.Debug("Trying external lyrics provider", "provider", provider.Name(), "track", track.Title)

		lyrics, subtitles, err := provider.GetLyrics(ctx, track.Title, track.Artist, track.Album, track.Duration)
		if err != nil {
			logger.Debug("Failed to fetch lyrics from provider", "provider", provider.Name(), "error", err)
			continue
		}

		if lyrics != "" {
			track.Lyrics = lyrics
			logger.Debug("Fetched lyrics from external provider", "provider", provider.Name())
		}
		if subtitles != "" {
			track.Subtitles = subtitles
			logger.Debug("Fetched subtitles from external provider", "provider", provider.Name())
		}
		return
	}
}
