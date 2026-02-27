package tagging

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sorrow446/go-mp4tag"
	"github.com/bogem/id3v2/v2"
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"

	"github.com/cesargomez89/navidrums/internal/domain"
)

var ErrUnsupportedFormat = errors.New("unsupported file format")

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
		return fmt.Errorf("%w: %s", ErrUnsupportedFormat, ext)
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
	addCustom("GENRE", track.Genre)
	addCustom("MOOD", track.Mood)
	addCustom("STYLE", track.Style)
	addCustom("BARCODE", track.Barcode)
	addCustom("CATALOGNUMBER", track.CatalogNumber)
	addCustom("RELEASETYPE", track.ReleaseType)
	addCustom("MUSICBRAINZ_RELEASEGROUPID", track.ReleaseID)
	addCustom("AUDIO_QUALITY", track.AudioQuality)
	addCustom("AUDIO_MODE", track.AudioModes)
	addCustom("KEY", track.Key)
	addCustom("KEY_SCALE", track.KeyScale)
	addCustom("VERSION", track.Version)
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

// ── MP4 Strategy ─────────────────────────────────────────────────────────────

type MP4Tagger struct{}

func (t *MP4Tagger) WriteTags(filePath string, tags *TagMap) error {
	mp4, err := mp4tag.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open MP4 file: %w", err)
	}
	defer func() { _ = mp4.Close() }()

	mp4Tags := &mp4tag.MP4Tags{
		Title:       tags.Title,
		Album:       tags.Album,
		TrackNumber: int16(tags.TrackNum),
		TrackTotal:  int16(tags.TrackTotal),
		DiscNumber:  int16(tags.DiscNum),
		DiscTotal:   int16(tags.DiscTotal),
		BPM:         int16(tags.BPM),
		Composer:    tags.Composer,
		Copyright:   tags.Copyright,
		Lyrics:      tags.Lyrics,
	}

	if len(tags.Artists) > 0 {
		mp4Tags.Artist = strings.Join(tags.Artists, ", ")
	}
	if len(tags.AlbumArtists) > 0 {
		mp4Tags.AlbumArtist = strings.Join(tags.AlbumArtists, ", ")
	}

	if len(tags.CoverArt) > 0 {
		mp4Tags.Pictures = []*mp4tag.MP4Picture{
			{Data: tags.CoverArt},
		}
	}

	// Apply all advanced parity tags to iTunes freeform atoms
	mp4Tags.Custom = tags.Custom

	return mp4.Write(mp4Tags, []string{})
}

// ── MP3 Strategy ─────────────────────────────────────────────────────────────

type MP3Tagger struct{}

func (t *MP3Tagger) WriteTags(filePath string, tags *TagMap) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer func() { _ = tag.Close() }()

	tag.SetVersion(4)

	if tags.Title != "" {
		tag.SetTitle(tags.Title)
	}
	if len(tags.Artists) > 0 {
		tag.AddTextFrame("TPE1", tag.DefaultEncoding(), strings.Join(tags.Artists, "\x00"))
	}
	if tags.Album != "" {
		tag.SetAlbum(tags.Album)
	}
	if tags.Year > 0 {
		tag.SetYear(fmt.Sprintf("%d", tags.Year))
	}
	if tags.Genre != "" {
		tag.SetGenre(tags.Genre)
	}
	tag.DeleteFrames("TIT3")

	if len(tags.AlbumArtists) > 0 {
		tag.AddTextFrame("TPE2", tag.DefaultEncoding(), strings.Join(tags.AlbumArtists, "\x00"))
	}

	if tags.TrackNum > 0 {
		trackStr := fmt.Sprintf("%d", tags.TrackNum)
		if tags.TrackTotal > 0 {
			trackStr = fmt.Sprintf("%d/%d", tags.TrackNum, tags.TrackTotal)
		}
		tag.AddTextFrame(tag.CommonID("Track number/Position in set"), tag.DefaultEncoding(), trackStr)
	}
	if tags.DiscNum > 0 {
		discStr := fmt.Sprintf("%d", tags.DiscNum)
		if tags.DiscTotal > 0 {
			discStr = fmt.Sprintf("%d/%d", tags.DiscNum, tags.DiscTotal)
		}
		tag.AddTextFrame(tag.CommonID("Part of a set"), tag.DefaultEncoding(), discStr)
	}

	if tags.Composer != "" {
		tag.AddTextFrame(tag.CommonID("Composer"), tag.DefaultEncoding(), tags.Composer)
	}
	if tags.Copyright != "" {
		tag.AddTextFrame(tag.CommonID("Copyright message"), tag.DefaultEncoding(), tags.Copyright)
	}
	if tags.BPM > 0 {
		tag.AddTextFrame(tag.CommonID("BPM"), tag.DefaultEncoding(), fmt.Sprintf("%d", tags.BPM))
	}
	if tags.Lyrics != "" {
		tag.AddTextFrame(tag.CommonID("Lyrics"), tag.DefaultEncoding(), tags.Lyrics)
	}

	// Apply Custom Metadata Mapping
	for k, v := range tags.Custom {
		if k == "LYRICS" {
			tag.AddUnsynchronisedLyricsFrame(id3v2.UnsynchronisedLyricsFrame{
				Encoding:          id3v2.EncodingUTF8,
				Language:          "eng",
				ContentDescriptor: "LRC",
				Lyrics:            v,
			})
			continue
		}
		// Map known custom fields to common IDs if applicable, else UserDefined
		switch k {
		case "LABEL":
			tag.AddTextFrame(tag.CommonID("Publisher"), tag.DefaultEncoding(), v)
		case "ISRC":
			tag.AddTextFrame(tag.CommonID("ISRC"), tag.DefaultEncoding(), v)
		case "KEY":
			tag.AddTextFrame(tag.CommonID("Key"), tag.DefaultEncoding(), v)
		case "VERSION":
			tag.AddTextFrame(tag.CommonID("Version"), tag.DefaultEncoding(), v)
		case "URL":
			tag.AddTextFrame(tag.CommonID("WWWAudioSource"), tag.DefaultEncoding(), v)
		case "COMPILATION":
			tag.AddTextFrame("TCMP", tag.DefaultEncoding(), v)
		default:
			tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
				Encoding:    id3v2.EncodingUTF8,
				Description: k,
				Value:       v,
			})
		}
	}

	if len(tags.CoverArt) > 0 {
		tag.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    tags.CoverMime,
			PictureType: id3v2.PTFrontCover,
			Description: "Front Cover",
			Picture:     tags.CoverArt,
		})
	}

	return tag.Save()
}

// ── FLAC Strategy ────────────────────────────────────────────────────────────

type FLACTagger struct{}

func (t *FLACTagger) WriteTags(filePath string, tags *TagMap) error {
	f, err := flac.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	vc := t.newVorbisComment(tags)
	newVCMeta := vc.Marshal()

	var newPicMeta *flac.MetaDataBlock
	if len(tags.CoverArt) > 0 {
		pic, err := flacpicture.NewFromImageData(
			flacpicture.PictureTypeFrontCover,
			"Front Cover",
			tags.CoverArt,
			tags.CoverMime,
		)
		if err != nil {
			return fmt.Errorf("failed to create picture: %w", err)
		}
		pm := pic.Marshal()
		newPicMeta = &pm
	}

	var currentVC []byte
	var currentPic []byte
	var vorbisIdx = -1
	var pictureIdx = -1

	for i, b := range f.Meta {
		switch b.Type {
		case flac.VorbisComment:
			vorbisIdx = i
			vcBlock, err := flacvorbis.ParseFromMetaDataBlock(*b)
			if err == nil {
				currentVC = vcBlock.Marshal().Data
			}
		case flac.Picture:
			pictureIdx = i
			picBlock, err := flacpicture.ParseFromMetaDataBlock(*b)
			if err == nil {
				currentPic = picBlock.ImageData
			}
		}
	}

	changed := !bytes.Equal(currentVC, newVCMeta.Data)
	if newPicMeta != nil && !bytes.Equal(currentPic, newPicMeta.Data) {
		changed = true
	} else if len(tags.CoverArt) == 0 && pictureIdx >= 0 {
		changed = true
	}

	if !changed {
		return nil
	}

	if pictureIdx >= 0 {
		f.Meta = append(f.Meta[:pictureIdx], f.Meta[pictureIdx+1:]...)
		if vorbisIdx > pictureIdx {
			vorbisIdx--
		}
	}

	if newPicMeta != nil {
		f.Meta = append(f.Meta, newPicMeta)
	}

	if vorbisIdx >= 0 {
		f.Meta[vorbisIdx] = &newVCMeta
	} else {
		f.Meta = append(f.Meta, &newVCMeta)
	}

	tempFile := filePath + ".tmp"
	if err := f.Save(tempFile); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to save temp FLAC file: %w", err)
	}

	if err := os.Rename(tempFile, filePath); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to rename temp file: %w", err)
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

func (t *FLACTagger) newVorbisComment(tags *TagMap) *flacvorbis.MetaDataBlockVorbisComment {
	vc := flacvorbis.New()

	add := func(name, value string) {
		if value != "" {
			_ = vc.Add(name, value)
		}
	}

	add("TITLE", tags.Title)
	for _, a := range tags.Artists {
		add("ARTIST", a)
	}
	for _, a := range tags.AlbumArtists {
		add("ALBUMARTIST", a)
	}
	add("ALBUM", tags.Album)

	if tags.TrackNum > 0 {
		add("TRACKNUMBER", fmt.Sprintf("%d", tags.TrackNum))
	}
	if tags.TrackTotal > 0 {
		add("TRACKTOTAL", fmt.Sprintf("%d", tags.TrackTotal))
	}
	if tags.DiscNum > 0 {
		add("DISCNUMBER", fmt.Sprintf("%d", tags.DiscNum))
	}
	if tags.DiscTotal > 0 {
		add("DISCTOTAL", fmt.Sprintf("%d", tags.DiscTotal))
	}
	if tags.Year > 0 {
		add("DATE", fmt.Sprintf("%d", tags.Year))
	}

	add("GENRE", tags.Genre)
	add("COPYRIGHT", tags.Copyright)
	add("COMPOSER", tags.Composer)

	if tags.BPM > 0 {
		add("BPM", fmt.Sprintf("%d", tags.BPM))
	}

	add("UNSYNCEDLYRICS", tags.Lyrics)

	// Dump all custom normalized tags
	for k, v := range tags.Custom {
		add(k, v)
	}

	return vc
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
