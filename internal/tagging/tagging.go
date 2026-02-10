package tagging

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bogem/id3v2/v2"
	"github.com/cesargomez89/navidrums/internal/models"
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
)

// TagFile writes metadata tags to an audio file based on its extension
func TagFile(filePath string, track *models.Track, albumArtData []byte) error {
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
func tagFLAC(filePath string, track *models.Track, albumArtData []byte) error {
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
func tagMP3(filePath string, track *models.Track, albumArtData []byte) error {
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

// tagMP4 writes metadata to an MP4/M4A file
// Note: This is a basic implementation. For full MP4 support, consider using a dedicated library
func tagMP4(filePath string, track *models.Track, albumArtData []byte) error {
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
		Timeout: 30 * time.Second,
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write image to file
	if err := os.WriteFile(filePath, imageData, 0644); err != nil {
		return fmt.Errorf("failed to write image file: %w", err)
	}

	return nil
}
