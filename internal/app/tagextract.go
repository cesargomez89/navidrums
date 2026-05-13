package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
)

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	CodecType string `json:"codec_type"`
}

type ffprobeFormat struct {
	Duration string           `json:"duration"`
	Tags     map[string]string `json:"tags"`
}

func ExtractTrackFromFile(filePath string, ffprobePath string) (*domain.Track, error) {
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}

	cmd := exec.Command(ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return extractFromFilename(filePath)
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return extractFromFilename(filePath)
	}

	track := &domain.Track{
		Status:        domain.TrackStatusQueued,
		FileExtension: filepath.Ext(filePath),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if probe.Format.Duration != "" {
		if dur, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
			track.Duration = int(math.Round(dur))
		}
	}

	if tags := probe.Format.Tags; tags != nil {
		if v, ok := tags["title"]; ok {
			track.Title = v
		}
		if v, ok := tags["artist"]; ok {
			track.Artist = v
		}
		if v, ok := tags["album"]; ok {
			track.Album = v
		}
		if v, ok := tags["album_artist"]; ok {
			track.AlbumArtist = v
		}
		if v, ok := tags["track"]; ok {
			if parts := strings.Split(v, "/"); len(parts) > 0 {
				if n, err := strconv.Atoi(parts[0]); err == nil {
					track.TrackNumber = n
				}
				if len(parts) > 1 {
					if n, err := strconv.Atoi(parts[1]); err == nil {
						track.TotalTracks = n
					}
				}
			}
		}
		if v, ok := tags["disc"]; ok {
			if parts := strings.Split(v, "/"); len(parts) > 0 {
				if n, err := strconv.Atoi(parts[0]); err == nil {
					track.DiscNumber = n
				}
				if len(parts) > 1 {
					if n, err := strconv.Atoi(parts[1]); err == nil {
						track.TotalDiscs = n
					}
				}
			}
		}
		if v, ok := tags["date"]; ok {
			if len(v) >= 4 {
				if n, err := strconv.Atoi(v[:4]); err == nil {
					track.Year = n
				}
			}
		}
		if v, ok := tags["genre"]; ok {
			track.Genre = v
		}
	}

	hash := sha256.Sum256([]byte(filePath))
	track.ProviderID = "local:" + hex.EncodeToString(hash[:])[:12]

	if track.Title == "" && track.Artist == "" {
		fallback, fbErr := extractFromFilename(filePath)
		if fbErr != nil {
			return track, nil
		}
		if track.Title == "" {
			track.Title = fallback.Title
		}
		if track.Artist == "" {
			track.Artist = fallback.Artist
		}
	}

	return track, nil
}

func extractFromFilename(filePath string) (*domain.Track, error) {
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	parts := strings.SplitN(name, " - ", 2)

	track := &domain.Track{
		Status:        domain.TrackStatusQueued,
		FileExtension: ext,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	hash := sha256.Sum256([]byte(filePath))
	track.ProviderID = "local:" + hex.EncodeToString(hash[:])[:12]

	if len(parts) == 2 {
		track.Artist = strings.TrimSpace(parts[0])
		track.Title = strings.TrimSpace(parts[1])
		return track, nil
	}

	track.Title = name
	return track, nil
}
