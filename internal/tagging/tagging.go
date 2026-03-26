package tagging

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cesargomez89/navidrums/internal/domain"
)

var ErrUnsupportedFormat = errors.New("unsupported file format")

var GenreSeparator = ";"

func SetGenreSeparator(sep string) {
	if sep != "" {
		GenreSeparator = sep
	}
}

// ── Models & Interfaces ──────────────────────────────────────────────────────

// TagMap represents the normalized metadata payload for all audio formats.
type TagMap struct {
	Custom       map[string]string
	Lyrics       string
	Title        string
	Album        string
	Genre        string
	Mood         string
	Style        string
	Language     string
	Country      string
	Composer     string
	Copyright    string
	CoverMime    string
	AlbumArtists []string
	CoverArt     []byte
	Artists      []string
	Year         int
	TrackTotal   int
	DiscNum      int
	DiscTotal    int
	BPM          int
	TrackNum     int
}

// AudioTagger defines the Strategy interface for our format adapters.
type AudioTagger interface {
	WriteTags(filePath string, tags *TagMap) error
}

// ── Factory & Normalizer ─────────────────────────────────────────────────────

// TagFile writes metadata tags to the audio file using the Strategy pattern.
func TagFile(filePath string, track *domain.Track, albumArtData []byte) error {
	// 1. Normalize the data ONCE
	tags := buildTagMap(track, albumArtData)

	// 2. Select the strategy based on extension
	var tagger AudioTagger
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".flac":
		tagger = &FLACTagger{}
	case ".mp3":
		tagger = &MP3Tagger{}
	case ".mp4", ".m4a":
		tagger = &MP4Tagger{}
	default:
		// Fallback: try FFmpeg for unknown formats
		tagger = &FFmpegFallbackTagger{}
	}

	// 3. Execute
	return tagger.WriteTags(filePath, tags)
}

// buildTagMap normalizes the domain.Track into a standard map, resolving fallbacks.
func buildTagMap(track *domain.Track, art []byte) *TagMap {
	tm := &TagMap{
		Title:        track.Title,
		Artists:      track.Artists,
		Album:        track.Album,
		AlbumArtists: track.AlbumArtists,
		Genre:        track.Genre,
		Mood:         track.Mood,
		Style:        track.Style,
		Language:     track.Language,
		Country:      track.Country,
		Year:         track.Year,
		TrackNum:     track.TrackNumber,
		TrackTotal:   track.TotalTracks,
		DiscNum:      track.DiscNumber,
		DiscTotal:    track.TotalDiscs,
		BPM:          track.BPM,
		Composer:     track.Composer,
		Copyright:    track.Copyright,
		Lyrics:       track.Lyrics,
		CoverArt:     art,
		Custom:       make(map[string]string),
	}

	// Array Fallbacks
	if len(tm.Artists) == 0 && track.Artist != "" {
		tm.Artists = []string{track.Artist}
	}
	if len(tm.AlbumArtists) == 0 && track.AlbumArtist != "" {
		tm.AlbumArtists = []string{track.AlbumArtist}
	}

	// Description/Lyrics Fallback
	if track.Description != "" && tm.Lyrics == "" {
		tm.Lyrics = track.Description
	}

	// Subtitles -> LRC
	if track.Subtitles != "" {
		tm.Custom["LYRICS"] = formatToLRC(track.Subtitles)
	}

	// Advanced Parity Tags Helper
	addCustom := func(k, v string) {
		if v != "" {
			tm.Custom[k] = v
		}
	}

	addCustom("ISRC", track.ISRC)
	addCustom("LABEL", track.Label)
	addCustom("MOOD", track.Mood)
	addCustom("STYLE", track.Style)
	addCustom("LANGUAGE", track.Language)
	addCustom("COUNTRY", track.Country)
	addCustom("BARCODE", track.Barcode)
	addCustom("CATALOGNUMBER", track.CatalogNumber)
	addCustom("RELEASETYPE", track.ReleaseType)
	addCustom("MUSICBRAINZ_RELEASEGROUPID", track.ReleaseID)
	addCustom("AUDIO_QUALITY", track.AudioQuality)
	addCustom("AUDIO_MODE", track.AudioModes)
	addCustom("KEY", track.Key)
	addCustom("KEY_SCALE", track.KeyScale)
	addCustom("URL", track.URL)

	if track.ReplayGain != 0 {
		addCustom("REPLAYGAIN_TRACK_GAIN", fmt.Sprintf("%.2f dB", track.ReplayGain))
	}
	if track.Peak != 0 {
		addCustom("REPLAYGAIN_TRACK_PEAK", fmt.Sprintf("%.6f", track.Peak))
	}
	if track.Compilation {
		addCustom("COMPILATION", "1")
	}

	// Mime Type Detection
	if len(art) > 0 {
		mime := http.DetectContentType(art)
		if idx := strings.Index(mime, ";"); idx != -1 {
			mime = strings.TrimSpace(mime[:idx])
		}
		tm.CoverMime = mime
	}

	// Array Joins for Custom Maps
	if len(track.ArtistIDs) > 0 {
		addCustom("MUSICBRAINZ_ARTISTID", strings.Join(track.ArtistIDs, "; "))
	}
	if len(track.AlbumArtistIDs) > 0 {
		addCustom("MUSICBRAINZ_ALBUMARTISTID", strings.Join(track.AlbumArtistIDs, "; "))
	}
	for _, t := range track.Tags {
		addCustom("MUSICBRAINZ_TAG", t)
	}

	return tm
}

// ── Utilities ────────────────────────────────────────────────────────────────

// formatToLRC converts subtitle lines to LRC format.
func formatToLRC(subtitles string) string {
	var sb strings.Builder
	for _, line := range strings.Split(subtitles, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return sb.String()
}
