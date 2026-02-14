package tagging

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/bogem/id3v2/v2"
	"github.com/cesargomez89/navidrums/internal/constants"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/storage"
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
)

func TagFile(filePath string, track *domain.Track, albumArtData []byte) error {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".flac":
		return tagFLAC(filePath, track, albumArtData)
	case ".mp3":
		return tagMP3(filePath, track, albumArtData)
	case ".mp4", ".m4a":
		return tagMP4(filePath, track, albumArtData)
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}
}

// tagFLAC writes metadata to a FLAC file
func tagFLAC(filePath string, track *domain.Track, albumArtData []byte) error {
	f, err := flac.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	// Create vorbis comment block
	var cmtmeta *flacvorbis.MetaDataBlockVorbisComment
	for _, meta := range f.Meta {
		if meta.Type == flac.VorbisComment {
			cmtmeta, err = flacvorbis.ParseFromMetaDataBlock(*meta)
			if err != nil {
				return fmt.Errorf("failed to parse vorbis comment: %w", err)
			}
			break
		}
	}

	if cmtmeta == nil {
		cmtmeta = flacvorbis.New()
	}

	// Set basic tags using standard vorbis comment field names
	if track.Title != "" {
		_ = cmtmeta.Add(flacvorbis.FIELD_TITLE, track.Title)
	}
	if track.Artist != "" {
		_ = cmtmeta.Add(flacvorbis.FIELD_ARTIST, track.Artist)
	}
	if track.AlbumArtist != "" {
		_ = cmtmeta.Add("ALBUMARTIST", track.AlbumArtist)
	}
	if track.Album != "" {
		_ = cmtmeta.Add(flacvorbis.FIELD_ALBUM, track.Album)
	}
	if track.TrackNumber > 0 {
		_ = cmtmeta.Add(flacvorbis.FIELD_TRACKNUMBER, fmt.Sprintf("%d", track.TrackNumber))
	}
	if track.TotalTracks > 0 {
		_ = cmtmeta.Add("TRACKTOTAL", fmt.Sprintf("%d", track.TotalTracks))
	}
	if track.DiscNumber > 0 {
		_ = cmtmeta.Add("DISCNUMBER", fmt.Sprintf("%d", track.DiscNumber))
	}
	if track.TotalDiscs > 0 {
		_ = cmtmeta.Add("DISCTOTAL", fmt.Sprintf("%d", track.TotalDiscs))
	}
	if track.Year > 0 {
		_ = cmtmeta.Add(flacvorbis.FIELD_DATE, fmt.Sprintf("%d", track.Year))
	}
	if track.Genre != "" {
		_ = cmtmeta.Add(flacvorbis.FIELD_GENRE, track.Genre)
	}
	if track.Label != "" {
		_ = cmtmeta.Add("LABEL", track.Label)
	}
	if track.ISRC != "" {
		_ = cmtmeta.Add(flacvorbis.FIELD_ISRC, track.ISRC)
	}
	if track.Copyright != "" {
		_ = cmtmeta.Add(flacvorbis.FIELD_COPYRIGHT, track.Copyright)
	}
	if track.Composer != "" {
		_ = cmtmeta.Add("COMPOSER", track.Composer)
	}
	// Additional metadata fields
	if track.BPM > 0 {
		_ = cmtmeta.Add("BPM", fmt.Sprintf("%d", track.BPM))
	}
	if track.Key != "" {
		_ = cmtmeta.Add("KEY", track.Key)
	}
	if track.KeyScale != "" {
		_ = cmtmeta.Add("KEY_SCALE", track.KeyScale)
	}
	if track.ReplayGain != 0 {
		_ = cmtmeta.Add("REPLAYGAIN_TRACK_GAIN", fmt.Sprintf("%.2f dB", track.ReplayGain))
	}
	if track.Peak != 0 {
		_ = cmtmeta.Add("REPLAYGAIN_TRACK_PEAK", fmt.Sprintf("%.6f", track.Peak))
	}
	if track.Version != "" {
		_ = cmtmeta.Add("VERSION", track.Version)
	}
	if track.Description != "" {
		_ = cmtmeta.Add("DESCRIPTION", track.Description)
	}
	if track.URL != "" {
		_ = cmtmeta.Add("URL", track.URL)
	}
	if track.AudioQuality != "" {
		_ = cmtmeta.Add("AUDIO_QUALITY", track.AudioQuality)
	}
	if track.AudioModes != "" {
		_ = cmtmeta.Add("AUDIO_MODE", track.AudioModes)
	}
	if track.Lyrics != "" {
		_ = cmtmeta.Add("UNSYNCEDLYRICS", track.Lyrics)
	}
	if track.Subtitles != "" {
		_ = cmtmeta.Add("LYRICS", formatToLRC(track.Subtitles))
	}
	if track.ReleaseDate != "" {
		_ = cmtmeta.Add("RELEASEDATE", track.ReleaseDate)
	}

	// Replace or add vorbis comment
	res := cmtmeta.Marshal()
	found := false
	for idx, meta := range f.Meta {
		if meta.Type == flac.VorbisComment {
			f.Meta[idx] = &res
			found = true
			break
		}
	}
	if !found {
		f.Meta = append(f.Meta, &res)
	}

	// Add album art if provided
	if len(albumArtData) > 0 {
		picture, err := flacpicture.NewFromImageData(
			flacpicture.PictureTypeFrontCover,
			"Front Cover",
			albumArtData,
			"image/jpeg",
		)
		if err == nil {
			picturemeta := picture.Marshal()
			f.Meta = append(f.Meta, &picturemeta)
		}
	}

	// Save the file
	return f.Save(filePath)
}

// tagMP3 writes ID3v2 tags to an MP3 file
func tagMP3(filePath string, track *domain.Track, albumArtData []byte) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer tag.Close()

	// Set version to ID3v2.4
	tag.SetVersion(4)

	// Set basic tags
	if track.Title != "" {
		tag.SetTitle(track.Title)
	}
	if track.Artist != "" {
		tag.SetArtist(track.Artist)
	}
	if track.Album != "" {
		tag.SetAlbum(track.Album)
	}
	if track.Year > 0 {
		tag.SetYear(fmt.Sprintf("%d", track.Year))
	}
	if track.Genre != "" {
		tag.SetGenre(track.Genre)
	}

	// Add frames for additional metadata
	if track.AlbumArtist != "" {
		tag.AddTextFrame(tag.CommonID("Band/Orchestra/Accompaniment"), tag.DefaultEncoding(), track.AlbumArtist)
	}
	if track.TrackNumber > 0 {
		trackStr := fmt.Sprintf("%d", track.TrackNumber)
		if track.TotalTracks > 0 {
			trackStr = fmt.Sprintf("%d/%d", track.TrackNumber, track.TotalTracks)
		}
		tag.AddTextFrame(tag.CommonID("Track number/Position in set"), tag.DefaultEncoding(), trackStr)
	}
	if track.DiscNumber > 0 {
		discStr := fmt.Sprintf("%d", track.DiscNumber)
		if track.TotalDiscs > 0 {
			discStr = fmt.Sprintf("%d/%d", track.DiscNumber, track.TotalDiscs)
		}
		tag.AddTextFrame(tag.CommonID("Part of a set"), tag.DefaultEncoding(), discStr)
	}
	if track.Composer != "" {
		tag.AddTextFrame(tag.CommonID("Composer"), tag.DefaultEncoding(), track.Composer)
	}
	if track.Label != "" {
		tag.AddTextFrame(tag.CommonID("Publisher"), tag.DefaultEncoding(), track.Label)
	}
	if track.ISRC != "" {
		tag.AddTextFrame(tag.CommonID("ISRC"), tag.DefaultEncoding(), track.ISRC)
	}
	if track.Copyright != "" {
		tag.AddTextFrame(tag.CommonID("Copyright message"), tag.DefaultEncoding(), track.Copyright)
	}

	// Additional metadata fields
	if track.BPM > 0 {
		tag.AddTextFrame(tag.CommonID("BPM"), tag.DefaultEncoding(), fmt.Sprintf("%d", track.BPM))
	}
	if track.Key != "" {
		tag.AddTextFrame(tag.CommonID("Key"), tag.DefaultEncoding(), track.Key)
	}
	if track.KeyScale != "" {
		tag.AddTextFrame("TKEYSCALE", tag.DefaultEncoding(), track.KeyScale)
	}
	if track.ReplayGain != 0 {
		tag.AddTextFrame("TXXX", tag.DefaultEncoding(), fmt.Sprintf("REPLAYGAIN_TRACK_GAIN=%.2f dB", track.ReplayGain))
		tag.AddTextFrame("TXXX", tag.DefaultEncoding(), fmt.Sprintf("REPLAYGAIN_TRACK_PEAK=%.6f", track.Peak))
	}
	if track.Version != "" {
		tag.AddTextFrame(tag.CommonID("Version"), tag.DefaultEncoding(), track.Version)
	}
	if track.Description != "" {
		tag.AddTextFrame(tag.CommonID("Comments"), tag.DefaultEncoding(), track.Description)
	}
	if track.URL != "" {
		tag.AddTextFrame(tag.CommonID("WWWAudioSource"), tag.DefaultEncoding(), track.URL)
	}
	if track.AudioQuality != "" {
		tag.AddTextFrame("TXXX", tag.DefaultEncoding(), fmt.Sprintf("AUDIO_QUALITY=%s", track.AudioQuality))
	}
	if track.AudioModes != "" {
		tag.AddTextFrame("TXXX", tag.DefaultEncoding(), fmt.Sprintf("AUDIO_MODE=%s", track.AudioModes))
	}
	if track.Lyrics != "" {
		tag.AddTextFrame(tag.CommonID("Lyrics"), tag.DefaultEncoding(), track.Lyrics)
	}

	// Add synchronized lyrics via USLT frame with LRC-formatted text as fallback
	if track.Subtitles != "" {
		tag.AddUnsynchronisedLyricsFrame(id3v2.UnsynchronisedLyricsFrame{
			Encoding:          id3v2.EncodingUTF8,
			Language:          "eng",
			ContentDescriptor: "LRC",
			Lyrics:            formatToLRC(track.Subtitles),
		})
	}

	if track.ReleaseDate != "" {
		tag.AddTextFrame(tag.CommonID("Release time"), tag.DefaultEncoding(), track.ReleaseDate)
	}

	// Add album art
	if len(albumArtData) > 0 {
		pic := id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    "image/jpeg",
			PictureType: id3v2.PTFrontCover,
			Description: "Front Cover",
			Picture:     albumArtData,
		}
		tag.AddAttachedPicture(pic)
	}

	return tag.Save()
}

// formatToLRC converts subtitles format to LRC format
// Input: "[00:39.98] Lyrics text"
// Output: "[00:39.98]Lyrics text"
func formatToLRC(subtitles string) string {
	var result strings.Builder
	lines := strings.Split(subtitles, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Check for timestamp pattern [MM:SS.mm] or [MM:SS.mmm]
		// The timestamp is 10 chars: [00:39.98] or [00:39.983]
		if len(line) >= 10 && line[0] == '[' && line[9] == ']' {
			result.WriteString(line[:10]) // Timestamp part
			if len(line) > 10 {
				result.WriteString(line[10:]) // Everything after timestamp
			}
		} else {
			result.WriteString(line)
		}
		result.WriteString("\n")
	}
	return result.String()
}

// tagMP4 writes metadata to an MP4/M4A file
// Note: This is a basic implementation. For full MP4 support, consider using a dedicated library
func tagMP4(filePath string, track *domain.Track, albumArtData []byte) error {
	// MP4 tagging is more complex and requires atom manipulation
	// For now, we'll skip MP4 tagging or use a simpler approach
	// You could use github.com/abema/go-mp4 or similar libraries
	return fmt.Errorf("MP4 tagging not yet implemented")
}

// DownloadImage downloads an image from a URL and returns the image data
func DownloadImage(url string) ([]byte, error) {
	if url == "" {
		return nil, nil
	}

	client := &http.Client{
		Timeout: constants.DefaultHTTPTimeout,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://tidal.com/")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: status %d (URL: %s)", resp.StatusCode, url)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	return buf.Bytes(), nil
}

// SaveImageToFile saves image data to a file
func SaveImageToFile(imageData []byte, filePath string) error {
	if len(imageData) == 0 {
		return nil
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := storage.EnsureDir(dir); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write image to file
	if err := storage.WriteFile(filePath, imageData); err != nil {
		return fmt.Errorf("failed to write image file: %w", err)
	}

	return nil
}
