package catalog

import "context"

type LyricsProvider interface {
	GetLyrics(ctx context.Context, track, artist, album string, duration int) (lyrics, subtitles string, err error)
	Name() string
}
