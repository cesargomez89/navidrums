package tagging

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
)

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
	if dirHandle, err := os.Open(dir); err == nil { //nolint:gosec
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

	if tags.Genre != "" {
		genres := strings.Split(tags.Genre, GenreSeparator)
		for _, g := range genres {
			g = strings.TrimSpace(g)
			if g != "" {
				add("GENRE", g)
			}
		}
	}
	add("COPYRIGHT", tags.Copyright)
	add("COMPOSER", tags.Composer)

	if tags.BPM > 0 {
		add("BPM", fmt.Sprintf("%d", tags.BPM))
	}

	add("UNSYNCEDLYRICS", tags.Lyrics)
	add("LANGUAGE", tags.Language)

	// Dump all custom normalized tags
	for k, v := range tags.Custom {
		add(k, v)
	}

	return vc
}
