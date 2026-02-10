package tagging

import (
	"fmt"
	"os/exec"

	"github.com/cesargomez89/navidrums/internal/models"
)

// TagFile uses ffmpeg CLI to tag the file
func TagFile(filePath string, track *models.Track) error {
	// ffmpeg -i input.flac -metadata title="Title" -metadata artist="Artist" -metadata album="Album" -metadata track="1" -c copy output.flac
	// But in-place editing with ffmpeg is tricky (needs tmp file).
	// TagLib bindings might be better but require CGO and system libs.
	// User said "go-mp3 / taglib bindings / ffmpeg CLI for tagging".
	// "ffmpeg CLI" suggests exec.

	// Create temp output
	tempOut := filePath + ".tagged.flac"

	args := []string{
		"-y", "-i", filePath,
		"-metadata", fmt.Sprintf("title=%s", track.Title),
		"-metadata", fmt.Sprintf("artist=%s", track.Artist),
		"-metadata", fmt.Sprintf("album=%s", track.Album),
		"-metadata", fmt.Sprintf("track=%d", track.TrackNumber),
		"-c", "copy",
		tempOut,
	}

	cmd := exec.Command("ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg failed: %s (%v)", string(out), err)
	}

	// Move temp back to original
	// On Linux, Rename is atomic replace usually, but here we overwrite
	if err := exec.Command("mv", tempOut, filePath).Run(); err != nil {
		return fmt.Errorf("failed to move tagged file: %v", err)
	}

	return nil
}
