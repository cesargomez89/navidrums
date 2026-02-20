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
	// ── Step 1: grab raw STREAMINFO bytes (4-byte header + 34-byte body = 38) ──
	rawFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open FLAC file: %w", err)
	}

	// Skip the 4-byte "fLaC" magic.
	magic := make([]byte, 4)
	if _, err := io.ReadFull(rawFile, magic); err != nil {
		_ = rawFile.Close()
		return fmt.Errorf("failed to read fLaC magic: %w", err)
	}
	if string(magic) != "fLaC" {
		_ = rawFile.Close()
		return fmt.Errorf("not a valid FLAC file (bad magic): %s", filePath)
	}

	// Read the raw STREAMINFO block (header + body) verbatim.
	rawStreamInfo := make([]byte, 38)
	if _, err := io.ReadFull(rawFile, rawStreamInfo); err != nil {
		_ = rawFile.Close()
		return fmt.Errorf("failed to read STREAMINFO block: %w", err)
	}

	// We need rawFile open below to copy audio frames; keep it.

	// ── Step 2: parse metadata to find audio offset and existing blocks ──
	stream, err := flac.ParseFile(filePath)
	if err != nil {
		_ = rawFile.Close()
		return fmt.Errorf("failed to parse FLAC metadata: %w", err)
	}

	audioOffset := calcAudioOffset(stream)

	// Collect the SeekTable if present (preserve it).
	var seekTableBlock *meta.Block
	for _, b := range stream.Blocks {
		if b.Type == meta.TypeSeekTable {
			seekTableBlock = b
			break
		}
	}
	_ = stream.Close()

	// ── Step 3: build new metadata buffer ──
	// We will NOT set the last-block flag on STREAMINFO ourselves — instead we
	// mark the actual final block as last below.
	var metaBuf bytes.Buffer

	// 3a. STREAMINFO verbatim — but we must update the last-block flag (bit 7
	//     of the first byte). STREAMINFO is never last when more blocks follow,
	//     so clear that bit.
	siHeader := rawStreamInfo[0] & 0x7F // clear last-block flag
	metaBuf.WriteByte(siHeader)
	metaBuf.Write(rawStreamInfo[1:]) // remaining 37 bytes (3-byte length + 34-byte body)

	// Collect subsequent blocks in order.
	type rawBlock struct {
		blockType byte
		body      []byte
	}
	var blocks []rawBlock

	// 3b. SeekTable.
	if seekTableBlock != nil {
		body, err := encodeSeekTable(seekTableBlock.Body.(*meta.SeekTable))
		if err != nil {
			_ = rawFile.Close()
			return fmt.Errorf("failed to encode SeekTable: %w", err)
		}
		blocks = append(blocks, rawBlock{byte(meta.TypeSeekTable), body})
	}

	// 3c. VorbisComment.
	vc := newVorbisComment(track)
	vcBody, err := encodeVorbisComment(vc)
	if err != nil {
		_ = rawFile.Close()
		return fmt.Errorf("failed to encode VorbisComment: %w", err)
	}
	blocks = append(blocks, rawBlock{byte(meta.TypeVorbisComment), vcBody})

	// 3d. Picture (optional).
	if len(albumArtData) > 0 {
		picBody, err := encodePictureData(albumArtData)
		if err != nil {
			_ = rawFile.Close()
			return fmt.Errorf("failed to encode Picture block: %w", err)
		}
		blocks = append(blocks, rawBlock{byte(meta.TypePicture), picBody})
	}

	// Write blocks, setting the last-block flag only on the final one.
	for i, blk := range blocks {
		isLast := i == len(blocks)-1
		if err := writeRawBlock(&metaBuf, blk.blockType, blk.body, isLast); err != nil {
			_ = rawFile.Close()
			return err
		}
	}

	// Edge case: if there are no extra blocks (no seektable, no art, somehow no
	// vorbis comment — unlikely but guard it), STREAMINFO must be marked last.
	// We already wrote STREAMINFO without the last flag; if blocks is empty we
	// need to patch it. In practice this won't happen because we always add a
	// VorbisComment, but be safe.
	if len(blocks) == 0 {
		b := metaBuf.Bytes()
		b[0] |= 0x80 // set last-block flag on STREAMINFO
	}

	// ── Step 4: seek rawFile to audio start, then atomic write ──
	if _, err := rawFile.Seek(audioOffset, io.SeekStart); err != nil {
		_ = rawFile.Close()
		return fmt.Errorf("failed to seek to audio section: %w", err)
	}

	dir := filepath.Dir(filePath)
	tmpFile, err := os.CreateTemp(dir, "*.flac.tmp")
	if err != nil {
		_ = rawFile.Close()
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpPath)
		}
	}()

	// fLaC magic.
	if _, err := tmpFile.Write([]byte("fLaC")); err != nil {
		_ = tmpFile.Close()
		_ = rawFile.Close()
		return fmt.Errorf("failed to write fLaC magic: %w", err)
	}

	// New metadata.
	if _, err := tmpFile.Write(metaBuf.Bytes()); err != nil {
		_ = tmpFile.Close()
		_ = rawFile.Close()
		return fmt.Errorf("failed to write metadata blocks: %w", err)
	}

	// Verbatim audio frames.
	if _, err := io.Copy(tmpFile, rawFile); err != nil {
		_ = tmpFile.Close()
		_ = rawFile.Close()
		return fmt.Errorf("failed to copy audio data: %w", err)
	}

	_ = rawFile.Close()

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("failed to replace original FLAC file: %w", err)
	}
	success = true
	return nil
}

// writeRawBlock writes a single metadata block to w.
// [1-byte flags (last<<7 | type)] [3-byte big-endian body length] [body]
func writeRawBlock(w *bytes.Buffer, blockType byte, body []byte, isLast bool) error {
	length := len(body)
	if length > 0xFFFFFF {
		return fmt.Errorf("metadata block body too large: %d bytes", length)
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
	offset := int64(4) // "fLaC"
	offset += 4 + 34   // STREAMINFO header + body

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

	write32(3) // picture type: Cover (front)
	write32(uint32(len(mimeBytes)))
	buf.Write(mimeBytes)
	write32(uint32(len(desc)))
	buf.Write(desc)
	write32(0) // width  (0 = unknown)
	write32(0) // height (0 = unknown)
	write32(0) // color depth (0 = unknown)
	write32(0) // color count (0 = not indexed)
	write32(uint32(len(data)))
	buf.Write(data)

	return buf.Bytes(), nil
}

// newVorbisComment builds a populated VorbisComment from a Track.
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
