package tagging

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bogem/id3v2/v2"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"

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
	rawFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open FLAC file: %w", err)
	}
	defer func() { _ = rawFile.Close() }()

	// Validate magic
	magic := make([]byte, 4)
	if _, rErr := io.ReadFull(rawFile, magic); rErr != nil {
		return fmt.Errorf("failed to read fLaC magic: %w", rErr)
	}
	if string(magic) != "fLaC" {
		return fmt.Errorf("not a valid FLAC file: %s", filePath)
	}

	// Read STREAMINFO verbatim (38 bytes)
	rawStreamInfo := make([]byte, 38)
	if _, rErr := io.ReadFull(rawFile, rawStreamInfo); rErr != nil {
		return fmt.Errorf("failed to read STREAMINFO: %w", rErr)
	}

	stream, err := flac.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse FLAC metadata: %w", err)
	}
	audioOffset := calcAudioOffset(stream)

	var seekTableBlock *meta.Block
	for _, b := range stream.Blocks {
		if b.Type == meta.TypeSeekTable {
			seekTableBlock = b
			break
		}
	}
	if cErr := stream.Close(); cErr != nil {
		return fmt.Errorf("failed to close FLAC stream: %w", cErr)
	}

	// ---- Build new metadata ----

	vc := newVorbisComment(track)
	vcBody, err := encodeVorbisComment(vc)
	if err != nil {
		return err
	}

	var picBody []byte
	if len(albumArtData) > 0 {
		picBody, err = encodePictureData(albumArtData)
		if err != nil {
			return err
		}
	}

	// 🔥 FAST PATH: skip rewrite if metadata identical
	changed, err := metadataChanged(filePath, vcBody, picBody)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	var metaBuf bytes.Buffer

	// STREAMINFO (clear last flag)
	siHeader := rawStreamInfo[0] & 0x7F
	metaBuf.WriteByte(siHeader)
	metaBuf.Write(rawStreamInfo[1:])

	type rawBlock struct {
		body      []byte
		blockType byte
	}
	var blocks []rawBlock

	if seekTableBlock != nil {
		body, encErr := encodeSeekTable(seekTableBlock.Body.(*meta.SeekTable))
		if encErr != nil {
			return encErr
		}
		blocks = append(blocks, rawBlock{body: body, blockType: byte(meta.TypeSeekTable)})
	}

	blocks = append(blocks, rawBlock{body: vcBody, blockType: byte(meta.TypeVorbisComment)})

	if len(picBody) > 0 {
		blocks = append(blocks, rawBlock{body: picBody, blockType: byte(meta.TypePicture)})
	}

	for i, blk := range blocks {
		isLast := i == len(blocks)-1
		if wErr := writeRawBlock(&metaBuf, blk.blockType, blk.body, isLast); wErr != nil {
			return wErr
		}
	}

	if len(blocks) == 0 {
		b := metaBuf.Bytes()
		b[0] |= 0x80
	}

	// ---- Copy audio ----

	if _, seekErr := rawFile.Seek(audioOffset, io.SeekStart); seekErr != nil {
		return seekErr
	}

	dir := filepath.Dir(filePath)
	tmpFile, tmpErr := os.CreateTemp(dir, "*.flac.tmp")
	if tmpErr != nil {
		return tmpErr
	}
	tmpPath := tmpFile.Name()

	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write([]byte("fLaC")); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if _, err := tmpFile.Write(metaBuf.Bytes()); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if _, err := io.Copy(tmpFile, rawFile); err != nil {
		_ = tmpFile.Close()
		return err
	}

	// Ensure file is fully flushed
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	// Atomic replace
	if err := os.Rename(tmpPath, filePath); err != nil {
		return err
	}

	// 🔥 Guarantee Navidrome detection
	now := time.Now()
	if err := os.Chtimes(filePath, now, now); err != nil {
		return err
	}

	// Sync directory (extra safety)
	if dirHandle, err := os.Open(dir); err == nil {
		_ = dirHandle.Sync()
		_ = dirHandle.Close()
	}

	success = true
	return nil
}

// Detect if metadata actually changed
func metadataChanged(filePath string, newVC []byte, newPic []byte) (bool, error) {
	stream, err := flac.ParseFile(filePath)
	if err != nil {
		return false, err
	}
	defer func() { _ = stream.Close() }()

	var currentVC []byte
	var currentPic []byte

	for _, b := range stream.Blocks {
		switch b.Type {
		case meta.TypeVorbisComment:
			body, err := encodeVorbisComment(b.Body.(*meta.VorbisComment))
			if err != nil {
				return false, err
			}
			currentVC = body
		case meta.TypePicture:
			p := b.Body.(*meta.Picture)
			body, err := encodePictureData(p.Data)
			if err != nil {
				return false, err
			}
			currentPic = body
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

// writeRawBlock writes a single metadata block to w.
// [1-byte flags (last<<7 | type)] [3-byte big-endian body length] [body]
func writeRawBlock(w *bytes.Buffer, blockType byte, body []byte, isLast bool) error {
	length := len(body)
	if length > 0xFFFFFF {
		return fmt.Errorf("metadata block too large")
	}
	flags := blockType & 0x7F
	if isLast {
		flags |= 0x80
	}
	w.WriteByte(flags)
	w.WriteByte(byte(length >> 16))
	w.WriteByte(byte(length >> 8))
	w.WriteByte(byte(length))
	w.Write(body)
	return nil
}

// calcAudioOffset returns the byte offset where audio frames begin.
//
// Layout:
//
//	[4]  "fLaC" magic
//	[4]  STREAMINFO header
//	[34] STREAMINFO body  (always 34 bytes)
//	For each additional block:
//	  [4]  block header (1 flag byte + 3 length bytes)
//	  [N]  block body
//
// mewkiz/flac exposes STREAMINFO in stream.Info only — it is NOT in
// stream.Blocks — so we account for it explicitly.
func calcAudioOffset(stream *flac.Stream) int64 {
	offset := int64(4)
	offset += 4 + 34
	for _, b := range stream.Blocks {
		offset += 4 + int64(b.Length)
	}
	return offset
}

// encodeSeekTable encodes the seek table block body (18 bytes per point).
func encodeSeekTable(st *meta.SeekTable) ([]byte, error) {
	buf := make([]byte, len(st.Points)*18)
	for i, p := range st.Points {
		off := i * 18
		binary.BigEndian.PutUint64(buf[off:off+8], p.SampleNum)
		binary.BigEndian.PutUint64(buf[off+8:off+16], p.Offset)
		binary.BigEndian.PutUint16(buf[off+16:off+18], p.NSamples)
	}
	return buf, nil
}

// encodeVorbisComment encodes a VorbisComment block body.
// Framing: all lengths are 32-bit little-endian; strings are UTF-8.
func encodeVorbisComment(vc *meta.VorbisComment) ([]byte, error) {
	var buf bytes.Buffer
	writeLE32 := func(n uint32) {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], n)
		buf.Write(b[:])
	}

	vendor := []byte(vc.Vendor)
	writeLE32(uint32(len(vendor)))
	buf.Write(vendor)

	writeLE32(uint32(len(vc.Tags)))
	for _, tag := range vc.Tags {
		entry := []byte(tag[0] + "=" + tag[1])
		writeLE32(uint32(len(entry)))
		buf.Write(entry)
	}
	return buf.Bytes(), nil
}

// encodePictureData encodes a cover-art Picture block body from raw image bytes.
func encodePictureData(data []byte) ([]byte, error) {
	mime := http.DetectContentType(data)
	if idx := strings.Index(mime, ";"); idx != -1 {
		mime = strings.TrimSpace(mime[:idx])
	}
	mimeBytes := []byte(mime)
	desc := []byte("Front Cover")

	var buf bytes.Buffer
	write32 := func(v uint32) {
		b := [4]byte{}
		binary.BigEndian.PutUint32(b[:], v)
		buf.Write(b[:])
	}

	write32(3)
	write32(uint32(len(mimeBytes)))
	buf.Write(mimeBytes)
	write32(uint32(len(desc)))
	buf.Write(desc)
	write32(0)
	write32(0)
	write32(0)
	write32(0)
	write32(uint32(len(data)))
	buf.Write(data)

	return buf.Bytes(), nil
}

// newVorbisComment builds a populated VorbisComment from a Track.
// Only minimal metadata required for Navidrome is included.
func newVorbisComment(track *domain.Track) *meta.VorbisComment {
	vc := &meta.VorbisComment{
		Vendor: "navidrums",
	}

	add := func(name, value string) {
		if value != "" {
			vc.Tags = append(vc.Tags, [2]string{name, value})
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

	add("GENRE", track.Genre)

	if track.Compilation {
		add("COMPILATION", "1")
	}

	return vc
}

// ── MP3 ──────────────────────────────────────────────────────────────────────

// tagMP3 writes ID3v2.4 tags to an MP3 file.
// Only minimal metadata required for Navidrome is included.
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

	if track.Compilation {
		tag.AddTextFrame("TCMP", tag.DefaultEncoding(), "1")
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
