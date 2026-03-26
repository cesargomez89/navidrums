package tagging

import (
	"fmt"
	"strings"

	"github.com/bogem/id3v2/v2"
)

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
		genres := strings.Split(tags.Genre, GenreSeparator)
		for _, g := range genres {
			g = strings.TrimSpace(g)
			if g != "" {
				tag.AddTextFrame("TCON", tag.DefaultEncoding(), g)
			}
		}
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
	if tags.Language != "" {
		tag.AddTextFrame("TLAN", tag.DefaultEncoding(), tags.Language)
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
		case "COUNTRY":
			tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
				Encoding:    id3v2.EncodingUTF8,
				Description: "COUNTRY",
				Value:       v,
			})
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
