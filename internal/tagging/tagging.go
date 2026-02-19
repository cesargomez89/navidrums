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
	"github.com/cesargomez89/navidrums/internal/constants"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/storage"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"
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

// tagFLAC writes Vorbis comments and an optional cover art picture block to a
// FLAC file using direct binary manipulation.
//
// We NEVER use the mewkiz/flac encoder to write audio frames — even with
// prediction analysis disabled it corrupts the audio on a passthrough rewrite.
// Instead we:
//  1. Use flac.Open to parse only the metadata blocks.
//  2. Measure where the audio section begins in the original file.
//  3. Encode only the new metadata blocks to a byte buffer.
//  4. Write: "fLaC" marker + new metadata bytes + verbatim raw audio bytes.
func tagFLAC(filePath string, track *domain.Track, albumArtData []byte) error {
	// Open once for lazy reading (reads metadata only, not audio frames).
	stream, err := flac.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open FLAC file: %w", err)
	}

	// Find the byte offset right after all metadata blocks, i.e., where the
	// raw audio frames start.
	audioOffset := calcAudioOffset(stream)

	// Build metadata block bytes from the already-parsed stream (no re-open).
	metaBytes, err := buildMetadataBytes(stream, track, albumArtData)
	stream.Close() // done with the reader; we re-open below as raw bytes
	if err != nil {
		return fmt.Errorf("failed to build metadata bytes: %w", err)
	}

	// Re-open as raw bytes so we can seek to the audio section.
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open FLAC file for reading: %w", err)
	}
	defer f.Close()

	// Seek to audio start.
	if _, err := f.Seek(audioOffset, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to audio section: %w", err)
	}

	// Write to a temp file in the same directory for an atomic rename.
	dir := filepath.Dir(filePath)
	tmpFile, err := os.CreateTemp(dir, "*.flac.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	// Write the fLaC magic.
	if _, err := tmpFile.Write([]byte("fLaC")); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write fLaC magic: %w", err)
	}

	// Write the new metadata blocks.
	if _, err := tmpFile.Write(metaBytes); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write metadata blocks: %w", err)
	}

	// Copy raw audio bytes verbatim.
	if _, err := io.Copy(tmpFile, f); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to copy audio data: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("failed to replace original FLAC file: %w", err)
	}
	success = true
	return nil
}

// calcAudioOffset returns the byte offset in the FLAC file where audio frames
// begin (i.e., right after the last metadata block).
// FLAC stream layout:
//
//	[4]  "fLaC" magic
//	For each metadata block:
//	  [1]  flags byte (last-metadata-block bit | block-type)
//	  [3]  24-bit big-endian length of block data
//	  [N]  block data (N = length above)
func calcAudioOffset(stream *flac.Stream) int64 {
	offset := int64(4) // "fLaC"
	for _, b := range stream.Blocks {
		// 4-byte block header + block body length
		offset += 4 + int64(b.Header.Length)
	}
	return offset
}

// buildMetadataBytes constructs the binary representation of the new metadata
// blocks (without the leading "fLaC" magic). It accepts an already-parsed
// stream to avoid a redundant file open. It:
//   - Keeps the original StreamInfo (mandatory, first block).
//   - Keeps any existing SeekTable.
//   - Adds a fresh VorbisComment (replaces any existing one).
//   - Optionally adds a Picture block for cover art.
func buildMetadataBytes(stream *flac.Stream, track *domain.Track, albumArtData []byte) ([]byte, error) {
	// Collect blocks we want to keep / add.
	var ordered []*meta.Block

	// 1. StreamInfo — always first.
	siBlock := &meta.Block{
		Header: meta.Header{Type: meta.TypeStreamInfo},
		Body:   stream.Info,
	}
	ordered = append(ordered, siBlock)

	// 2. SeekTable if present.
	for _, b := range stream.Blocks {
		if b.Type == meta.TypeSeekTable {
			ordered = append(ordered, b)
		}
	}

	// 3. New VorbisComment.
	vc := newVorbisComment(track)
	ordered = append(ordered, &meta.Block{
		Header: meta.Header{Type: meta.TypeVorbisComment},
		Body:   vc,
	})

	// 4. Optional cover art.
	if len(albumArtData) > 0 {
		if pb := buildPictureBlock(albumArtData); pb != nil {
			ordered = append(ordered, pb)
		}
	}

	// Encode each block to bytes.
	var buf bytes.Buffer
	for i, b := range ordered {
		isLast := i == len(ordered)-1
		blockBytes, err := encodeMetaBlock(b, isLast)
		if err != nil {
			return nil, fmt.Errorf("failed to encode block %v: %w", b.Type, err)
		}
		buf.Write(blockBytes)
	}

	return buf.Bytes(), nil
}

// encodeMetaBlock serialises a single metadata block to its binary FLAC
// representation ([1-byte flags][3-byte length][N-byte body]).
func encodeMetaBlock(b *meta.Block, isLast bool) ([]byte, error) {
	// Encode the block body.
	body, err := encodeBlockBody(b)
	if err != nil {
		return nil, err
	}

	length := uint32(len(body))
	if length > 0xFFFFFF {
		return nil, fmt.Errorf("block body too large: %d bytes", length)
	}

	// Flags byte: bit 7 = isLast, bits 0-6 = block type.
	flags := byte(b.Type)
	if isLast {
		flags |= 0x80
	}

	out := make([]byte, 0, 4+length)
	out = append(out, flags)
	// 24-bit big-endian length.
	out = append(out, byte(length>>16), byte(length>>8), byte(length))
	out = append(out, body...)
	return out, nil
}

// encodeBlockBody encodes only the data portion of a metadata block.
func encodeBlockBody(b *meta.Block) ([]byte, error) {
	switch b.Type {
	case meta.TypeStreamInfo:
		return encodeStreamInfo(b.Body.(*meta.StreamInfo))
	case meta.TypeSeekTable:
		return encodeSeekTable(b.Body.(*meta.SeekTable))
	case meta.TypeVorbisComment:
		return encodeVorbisComment(b.Body.(*meta.VorbisComment))
	case meta.TypePicture:
		return encodePicture(b.Body.(*meta.Picture))
	default:
		return nil, fmt.Errorf("unsupported block type for manual encoding: %v", b.Type)
	}
}

// encodeStreamInfo encodes the 34-byte StreamInfo block body.
func encodeStreamInfo(si *meta.StreamInfo) ([]byte, error) {
	buf := make([]byte, 34)
	// Bytes 0-1: min block size (16 bits)
	binary.BigEndian.PutUint16(buf[0:2], si.BlockSizeMin)
	// Bytes 2-3: max block size (16 bits)
	binary.BigEndian.PutUint16(buf[2:4], si.BlockSizeMax)
	// Bytes 4-6: min frame size (24 bits)
	buf[4] = byte(si.FrameSizeMin >> 16)
	buf[5] = byte(si.FrameSizeMin >> 8)
	buf[6] = byte(si.FrameSizeMin)
	// Bytes 7-9: max frame size (24 bits)
	buf[7] = byte(si.FrameSizeMax >> 16)
	buf[8] = byte(si.FrameSizeMax >> 8)
	buf[9] = byte(si.FrameSizeMax)
	// Bytes 10-17: sample rate (20b) | channels-1 (3b) | bits/sample-1 (5b) | total samples (36b)
	// Pack into 8 bytes.
	sampleRate := uint64(si.SampleRate)
	nChannels := uint64(si.NChannels - 1)
	bps := uint64(si.BitsPerSample - 1)
	nSamples := si.NSamples
	packed := (sampleRate<<44 | nChannels<<41 | bps<<36) | nSamples
	binary.BigEndian.PutUint64(buf[10:18], packed)
	// Bytes 18-33: MD5 checksum (128 bits)
	copy(buf[18:34], si.MD5sum[:])
	return buf, nil
}

// encodeSeekTable encodes the seek table block body.
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

// encodeVorbisComment encodes the VorbisComment block body using the
// Vorbis comment framing format (little-endian lengths, UTF-8 strings).
func encodeVorbisComment(vc *meta.VorbisComment) ([]byte, error) {
	var buf bytes.Buffer
	writeLE32 := func(n uint32) {
		b := [4]byte{}
		binary.LittleEndian.PutUint32(b[:], n)
		buf.Write(b[:])
	}

	vendorBytes := []byte(vc.Vendor)
	writeLE32(uint32(len(vendorBytes)))
	buf.Write(vendorBytes)

	writeLE32(uint32(len(vc.Tags)))
	for _, tag := range vc.Tags {
		entry := tag[0] + "=" + tag[1]
		writeLE32(uint32(len(entry)))
		buf.WriteString(entry)
	}
	return buf.Bytes(), nil
}

// encodePicture encodes the Picture metadata block body.
func encodePicture(pic *meta.Picture) ([]byte, error) {
	mimeBytes := []byte(pic.MIME)
	descBytes := []byte(pic.Desc)

	size := 4 + 4 + len(mimeBytes) + 4 + len(descBytes) + 4 + 4 + 4 + 4 + 4 + len(pic.Data)
	buf := make([]byte, 0, size)
	write32 := func(v uint32) { buf = append(buf, byte(v>>24), byte(v>>16), byte(v>>8), byte(v)) }

	write32(uint32(pic.Type))
	write32(uint32(len(mimeBytes)))
	buf = append(buf, mimeBytes...)
	write32(uint32(len(descBytes)))
	buf = append(buf, descBytes...)
	write32(pic.Width)
	write32(pic.Height)
	write32(pic.Depth)
	write32(pic.NPalColors)
	write32(uint32(len(pic.Data)))
	buf = append(buf, pic.Data...)
	return buf, nil
}

func newVorbisComment(track *domain.Track) *meta.VorbisComment {
	vc := &meta.VorbisComment{
		Vendor: "navidrums",
	}

	addTag := func(name, value string) {
		if value != "" {
			vc.Tags = append(vc.Tags, [2]string{name, value})
		}
	}

	addTag("TITLE", track.Title)

	// Multiple artists get individual ARTIST tags (recommended by Vorbis spec).
	if len(track.Artists) > 0 {
		for _, a := range track.Artists {
			addTag("ARTIST", a)
		}
	} else {
		addTag("ARTIST", track.Artist)
	}

	if len(track.AlbumArtists) > 0 {
		for _, a := range track.AlbumArtists {
			addTag("ALBUMARTIST", a)
		}
	} else {
		addTag("ALBUMARTIST", track.AlbumArtist)
	}

	addTag("ALBUM", track.Album)

	if track.TrackNumber > 0 {
		addTag("TRACKNUMBER", fmt.Sprintf("%d", track.TrackNumber))
	}
	if track.TotalTracks > 0 {
		addTag("TRACKTOTAL", fmt.Sprintf("%d", track.TotalTracks))
	}
	if track.DiscNumber > 0 {
		addTag("DISCNUMBER", fmt.Sprintf("%d", track.DiscNumber))
	}
	if track.TotalDiscs > 0 {
		addTag("DISCTOTAL", fmt.Sprintf("%d", track.TotalDiscs))
	}
	if track.Year > 0 {
		addTag("DATE", fmt.Sprintf("%d", track.Year))
	}

	addTag("RELEASEDATE", track.ReleaseDate)
	addTag("GENRE", track.Genre)
	addTag("LABEL", track.Label)
	addTag("ISRC", track.ISRC)
	addTag("COPYRIGHT", track.Copyright)
	addTag("COMPOSER", track.Composer)

	if track.BPM > 0 {
		addTag("BPM", fmt.Sprintf("%d", track.BPM))
	}
	addTag("KEY", track.Key)
	addTag("KEY_SCALE", track.KeyScale)

	if track.ReplayGain != 0 {
		addTag("REPLAYGAIN_TRACK_GAIN", fmt.Sprintf("%.2f dB", track.ReplayGain))
	}
	if track.Peak != 0 {
		addTag("REPLAYGAIN_TRACK_PEAK", fmt.Sprintf("%.6f", track.Peak))
	}

	addTag("VERSION", track.Version)
	addTag("DESCRIPTION", track.Description)
	addTag("URL", track.URL)
	addTag("AUDIO_QUALITY", track.AudioQuality)
	addTag("AUDIO_MODE", track.AudioModes)

	addTag("UNSYNCEDLYRICS", track.Lyrics)
	if track.Subtitles != "" {
		addTag("LYRICS", formatToLRC(track.Subtitles))
	}

	if track.Compilation {
		addTag("COMPILATION", "1")
	}

	for _, id := range track.ArtistIDs {
		addTag("MUSICBRAINZ_ARTISTID", id)
	}
	for _, id := range track.AlbumArtistIDs {
		addTag("MUSICBRAINZ_ALBUMARTISTID", id)
	}

	addTag("BARCODE", track.Barcode)
	addTag("CATALOGNUMBER", track.CatalogNumber)
	addTag("RELEASETYPE", track.ReleaseType)
	addTag("MUSICBRAINZ_RELEASEGROUPID", track.ReleaseID)

	return vc
}

// buildPictureBlock creates a meta.Block of type Picture for the given image
// data. Returns nil if the data is empty.
func buildPictureBlock(data []byte) *meta.Block {
	if len(data) == 0 {
		return nil
	}

	// Detect MIME type from the image header bytes.
	mime := http.DetectContentType(data)
	// Trim any parameters such as charset.
	if idx := strings.Index(mime, ";"); idx != -1 {
		mime = strings.TrimSpace(mime[:idx])
	}

	pic := &meta.Picture{
		Type: 3, // Cover (front)
		MIME: mime,
		Desc: "Front Cover",
		Data: data,
	}

	return &meta.Block{
		Header: meta.Header{Type: meta.TypePicture},
		Body:   pic,
	}
}

// formatToLRC converts subtitles to LRC format.
// Input lines look like "[MM:SS.mm] Lyrics text" or "[M:SS.mm] Lyrics text".
func formatToLRC(subtitles string) string {
	var result strings.Builder
	lines := strings.Split(subtitles, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Find the closing bracket of the timestamp tag.
		if line[0] == '[' {
			if close := strings.IndexByte(line, ']'); close != -1 {
				result.WriteString(line[:close+1])
				result.WriteString(line[close+1:])
			} else {
				result.WriteString(line)
			}
		} else {
			result.WriteString(line)
		}
		result.WriteString("\n")
	}
	return result.String()
}

// tagMP3 writes ID3v2 tags to an MP3 file.
func tagMP3(filePath string, track *domain.Track, albumArtData []byte) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer tag.Close()

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
		// TXXX frames require "Description\x00Value" — use UserDefinedTextFrame
		// via the TXXX id with a proper description/value split.
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
		// Detect MIME type so PNG covers aren't labelled as image/jpeg.
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

// tagMP4 writes metadata to an MP4/M4A file.
// Full atom manipulation is not yet implemented.
func tagMP4(filePath string, track *domain.Track, albumArtData []byte) error {
	return fmt.Errorf("MP4 tagging not yet implemented")
}

// DownloadImage downloads an image from a URL and returns the raw bytes.
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

// SaveImageToFile persists image bytes to the given file path,
// creating parent directories as needed.
func SaveImageToFile(imageData []byte, filePath string) error {
	if len(imageData) == 0 {
		return nil
	}

	dir := filepath.Dir(filePath)
	if err := storage.EnsureDir(dir); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := storage.WriteFile(filePath, imageData); err != nil {
		return fmt.Errorf("failed to write image file: %w", err)
	}

	return nil
}
