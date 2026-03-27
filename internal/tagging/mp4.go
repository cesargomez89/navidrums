package tagging

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cesargomez89/navidrums/internal/ffmpeg"
)

// ── MP4 Strategy ─────────────────────────────────────────────────────────────

type MP4Tagger struct{}

func (t *MP4Tagger) WriteTags(filePath string, tags *TagMap) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	meta := &ffmpeg.Metadata{
		Title:        tags.Title,
		Artists:      tags.Artists,
		Album:        tags.Album,
		AlbumArtists: tags.AlbumArtists,
		Genre:        tags.Genre,
		Mood:         tags.Mood,
		Language:     tags.Language,
		Year:         tags.Year,
		TrackNum:     tags.TrackNum,
		TrackTotal:   tags.TrackTotal,
		DiscNum:      tags.DiscNum,
		DiscTotal:    tags.DiscTotal,
		BPM:          tags.BPM,
		Composer:     tags.Composer,
		Copyright:    tags.Copyright,
		Lyrics:       tags.Lyrics,
		CoverArt:     tags.CoverArt,
		CoverMime:    tags.CoverMime,
		Custom:       tags.Custom,
	}

	tempPath, err := ffmpeg.WriteTags(ctx, filePath, meta)
	if err != nil {
		return fmt.Errorf("failed to write MP4 tags via ffmpeg: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	now := time.Now()
	if err := os.Chtimes(filePath, now, now); err != nil {
		return err
	}

	dir := filepath.Dir(filePath)
	if dirHandle, err := os.Open(dir); err == nil { //nolint:gosec
		_ = dirHandle.Sync()
		_ = dirHandle.Close()
	}

	return nil
}
