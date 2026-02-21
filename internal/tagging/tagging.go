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
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"

	"github.com/cesargomez89/navidrums/internal/constants"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/storage"
)

// TagFile writes metadata tags to the audio file at filePath.
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

// tagFLAC rewrites a FLAC file with new metadata while preserving audio frames
// verbatim. Strategy:
//  1. Open the file raw to copy the original STREAMINFO bytes exactly (never
//     re-encode STREAMINFO — any bit-packing mistake will make Navidrome reject
//     the file silently).
//  2. Parse metadata with flac.ParseFile to enumerate all existing blocks and
//     find where audio starts.
//  3. Build new metadata: verbatim STREAMINFO + optional SeekTable + fresh
//     VorbisComment + optional Picture.
//  4. Atomic write: temp file → rename.

func tagFLAC(filePath string, track *domain.Track, albumArtData []byte) error {
	changed, err := metadataChanged(filePath, track, albumArtData)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	f, err := flac.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	vc := newVorbisComment(track)

	var pictureIdx = -1
	for i, meta := range f.Meta {
		if meta.Type == flac.Picture {
			pictureIdx = i
			break
		}
	}

	if len(albumArtData) > 0 {
		pic, err := flacpicture.NewFromImageData(
			flacpicture.PictureTypeFrontCover,
			"Front Cover",
			albumArtData,
			http.DetectContentType(albumArtData),
		)
		if err != nil {
			return fmt.Errorf("failed to create picture: %w", err)
		}
		picMeta := pic.Marshal()
		if pictureIdx >= 0 {
			f.Meta[pictureIdx] = &picMeta
		} else {
			f.Meta = append(f.Meta, &picMeta)
		}
	} else if pictureIdx >= 0 {
		f.Meta = append(f.Meta[:pictureIdx], f.Meta[pictureIdx+1:]...)
	}

	var vorbisIdx = -1
	for i, meta := range f.Meta {
		if meta.Type == flac.VorbisComment {
			vorbisIdx = i
			break
		}
	}

	vcMeta := vc.Marshal()
	if vorbisIdx >= 0 {
		f.Meta[vorbisIdx] = &vcMeta
	} else {
		f.Meta = append(f.Meta, &vcMeta)
	}

	if err := f.Save(filePath); err != nil {
		return fmt.Errorf("failed to save FLAC file: %w", err)
	}

	now := time.Now()
	if err := os.Chtimes(filePath, now, now); err != nil {
		return err
	}

	dir := filepath.Dir(filePath)
	if dirHandle, err := os.Open(dir); err == nil {
		_ = dirHandle.Sync()
		_ = dirHandle.Close()
	}

	return nil
}

func metadataChanged(filePath string, track *domain.Track, albumArtData []byte) (bool, error) {
	f, err := flac.ParseFile(filePath)
	if err != nil {
		return false, err
	}

	vc := newVorbisComment(track)
	newVCMeta := vc.Marshal()
	newVC := newVCMeta.Data

	var newPic []byte
	if len(albumArtData) > 0 {
		pic, err := flacpicture.NewFromImageData(
			flacpicture.PictureTypeFrontCover,
			"Front Cover",
			albumArtData,
			http.DetectContentType(albumArtData),
		)
		if err != nil {
			return false, err
		}
		newPic = pic.Marshal().Data
	}

	var currentVC []byte
	var currentPic []byte

	for _, b := range f.Meta {
		switch b.Type {
		case flac.VorbisComment:
			vcBlock, err := flacvorbis.ParseFromMetaDataBlock(*b)
			if err != nil {
				return false, err
			}
			currentVC = vcBlock.Marshal().Data
		case flac.Picture:
			picBlock, err := flacpicture.ParseFromMetaDataBlock(*b)
			if err != nil {
				return false, err
			}
			currentPic = picBlock.ImageData
		}
	}

	if !bytes.Equal(currentVC, newVC) {
		return true, nil
	}

	if len(newPic) > 0 && !bytes.Equal(currentPic, newPic) {
		return true, nil
	}

	return false, nil
}

// newVorbisComment builds a populated VorbisComment from a Track.
func newVorbisComment(track *domain.Track) *flacvorbis.MetaDataBlockVorbisComment {
	vc := flacvorbis.New()

	add := func(name, value string) {
		if value != "" {
			_ = vc.Add(name, value)
		}
	}

	add("TITLE", track.Title)

	if len(track.Artists) > 0 {
		for _, a := range track.Artists {
			add("ARTIST", a)
		}
	} else {
		add("ARTIST", track.Artist)
	}

	if len(track.AlbumArtists) > 0 {
		for _, a := range track.AlbumArtists {
			add("ALBUMARTIST", a)
		}
	} else {
		add("ALBUMARTIST", track.AlbumArtist)
	}

	add("ALBUM", track.Album)

	if track.TrackNumber > 0 {
		add("TRACKNUMBER", fmt.Sprintf("%d", track.TrackNumber))
	}
	if track.TotalTracks > 0 {
		add("TRACKTOTAL", fmt.Sprintf("%d", track.TotalTracks))
	}
	if track.DiscNumber > 0 {
		add("DISCNUMBER", fmt.Sprintf("%d", track.DiscNumber))
	}
	if track.TotalDiscs > 0 {
		add("DISCTOTAL", fmt.Sprintf("%d", track.TotalDiscs))
	}
	if track.Year > 0 {
		add("DATE", fmt.Sprintf("%d", track.Year))
	}

	add("RELEASEDATE", track.ReleaseDate)
	add("GENRE", track.Genre)
	add("LABEL", track.Label)
	add("ISRC", track.ISRC)
	add("COPYRIGHT", track.Copyright)
	add("COMPOSER", track.Composer)

	if track.BPM > 0 {
		add("BPM", fmt.Sprintf("%d", track.BPM))
	}
	add("KEY", track.Key)
	add("KEY_SCALE", track.KeyScale)

	if track.ReplayGain != 0 {
		add("REPLAYGAIN_TRACK_GAIN", fmt.Sprintf("%.2f dB", track.ReplayGain))
	}
	if track.Peak != 0 {
		add("REPLAYGAIN_TRACK_PEAK", fmt.Sprintf("%.6f", track.Peak))
	}

	add("VERSION", track.Version)
	add("DESCRIPTION", track.Description)
	add("URL", track.URL)
	add("AUDIO_QUALITY", track.AudioQuality)
	add("AUDIO_MODE", track.AudioModes)
	add("UNSYNCEDLYRICS", track.Lyrics)

	if track.Subtitles != "" {
		add("LYRICS", formatToLRC(track.Subtitles))
	}

	if track.Compilation {
		add("COMPILATION", "1")
	}

	for _, id := range track.ArtistIDs {
		add("MUSICBRAINZ_ARTISTID", id)
	}
	for _, id := range track.AlbumArtistIDs {
		add("MUSICBRAINZ_ALBUMARTISTID", id)
	}

	add("BARCODE", track.Barcode)
	add("CATALOGNUMBER", track.CatalogNumber)
	add("RELEASETYPE", track.ReleaseType)
	add("MUSICBRAINZ_RELEASEGROUPID", track.ReleaseID)

	return vc
}

// formatToLRC converts subtitle lines to LRC format.
// Input lines are expected to look like "[MM:SS.mm] Lyrics text".
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

// ── MP3 ──────────────────────────────────────────────────────────────────────

// tagMP3 writes ID3v2.4 tags to an MP3 file.
func tagMP3(filePath string, track *domain.Track, albumArtData []byte) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer func() { _ = tag.Close() }()

	tag.SetVersion(4)

	if track.Title != "" {
		tag.SetTitle(track.Title)
	}
	if len(track.Artists) > 0 {
		tag.AddTextFrame("TPE1", tag.DefaultEncoding(), strings.Join(track.Artists, "\x00"))
	} else if track.Artist != "" {
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

	if len(track.AlbumArtists) > 0 {
		tag.AddTextFrame("TPE2", tag.DefaultEncoding(), strings.Join(track.AlbumArtists, "\x00"))
	} else if track.AlbumArtist != "" {
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
	if track.BPM > 0 {
		tag.AddTextFrame(tag.CommonID("BPM"), tag.DefaultEncoding(), fmt.Sprintf("%d", track.BPM))
	}
	if track.Key != "" {
		tag.AddTextFrame(tag.CommonID("Key"), tag.DefaultEncoding(), track.Key)
	}
	if track.ReplayGain != 0 {
		tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
			Encoding:    id3v2.EncodingUTF8,
			Description: "REPLAYGAIN_TRACK_GAIN",
			Value:       fmt.Sprintf("%.2f dB", track.ReplayGain),
		})
		tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
			Encoding:    id3v2.EncodingUTF8,
			Description: "REPLAYGAIN_TRACK_PEAK",
			Value:       fmt.Sprintf("%.6f", track.Peak),
		})
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
		tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
			Encoding:    id3v2.EncodingUTF8,
			Description: "AUDIO_QUALITY",
			Value:       track.AudioQuality,
		})
	}
	if track.AudioModes != "" {
		tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
			Encoding:    id3v2.EncodingUTF8,
			Description: "AUDIO_MODE",
			Value:       track.AudioModes,
		})
	}
	if track.Lyrics != "" {
		tag.AddTextFrame(tag.CommonID("Lyrics"), tag.DefaultEncoding(), track.Lyrics)
	}
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
	if track.Compilation {
		tag.AddTextFrame("TCMP", tag.DefaultEncoding(), "1")
	}
	for _, id := range track.ArtistIDs {
		tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
			Encoding:    id3v2.EncodingUTF8,
			Description: "MUSICBRAINZ_ARTISTID",
			Value:       id,
		})
	}
	for _, id := range track.AlbumArtistIDs {
		tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
			Encoding:    id3v2.EncodingUTF8,
			Description: "MUSICBRAINZ_ALBUMARTISTID",
			Value:       id,
		})
	}
	if track.Barcode != "" {
		tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
			Encoding:    id3v2.EncodingUTF8,
			Description: "BARCODE",
			Value:       track.Barcode,
		})
	}
	if track.CatalogNumber != "" {
		tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
			Encoding:    id3v2.EncodingUTF8,
			Description: "CATALOGNUMBER",
			Value:       track.CatalogNumber,
		})
	}
	if track.ReleaseType != "" {
		tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
			Encoding:    id3v2.EncodingUTF8,
			Description: "RELEASETYPE",
			Value:       track.ReleaseType,
		})
	}
	if track.ReleaseID != "" {
		tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
			Encoding:    id3v2.EncodingUTF8,
			Description: "MUSICBRAINZ_RELEASEGROUPID",
			Value:       track.ReleaseID,
		})
	}

	if len(albumArtData) > 0 {
		mime := http.DetectContentType(albumArtData)
		if idx := strings.Index(mime, ";"); idx != -1 {
			mime = strings.TrimSpace(mime[:idx])
		}
		tag.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    mime,
			PictureType: id3v2.PTFrontCover,
			Description: "Front Cover",
			Picture:     albumArtData,
		})
	}

	return tag.Save()
}

// ── MP4 ──────────────────────────────────────────────────────────────────────

// tagMP4 is not yet implemented.
func tagMP4(_ string, _ *domain.Track, _ []byte) error {
	return fmt.Errorf("MP4 tagging not yet implemented")
}

// ── Utilities ─────────────────────────────────────────────────────────────────

// DownloadImage fetches raw image bytes from a URL.
func DownloadImage(url string) ([]byte, error) {
	if url == "" {
		return nil, nil
	}

	client := &http.Client{Timeout: constants.DefaultHTTPTimeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: status %d (URL: %s)", resp.StatusCode, url)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}
	return buf.Bytes(), nil
}

// SaveImageToFile persists image bytes to filePath, creating directories as needed.
func SaveImageToFile(imageData []byte, filePath string) error {
	if len(imageData) == 0 {
		return nil
	}
	if err := storage.EnsureDir(filepath.Dir(filePath)); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := storage.WriteFile(filePath, imageData); err != nil {
		return fmt.Errorf("failed to write image file: %w", err)
	}
	return nil
}
