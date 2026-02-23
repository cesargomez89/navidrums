package app

import (
	"context"
	"log/slog"

	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/musicbrainz"
)

type MetadataEnricher struct {
	mbClient musicbrainz.ClientInterface
}

func NewMetadataEnricher(mbClient musicbrainz.ClientInterface) *MetadataEnricher {
	return &MetadataEnricher{
		mbClient: mbClient,
	}
}

func (e *MetadataEnricher) EnrichTrack(ctx context.Context, track *domain.Track, logger *slog.Logger) error {
	recordingID := ""
	if track.RecordingID != nil {
		recordingID = *track.RecordingID
	}
	if track.ISRC == "" && recordingID == "" {
		return nil
	}

	meta, mbErr := e.mbClient.GetRecording(ctx, recordingID, track.ISRC, track.Album)
	if mbErr != nil {
		return mbErr
	}
	if meta == nil {
		return nil
	}

	if meta.RecordingID != "" && (track.RecordingID == nil || *track.RecordingID == "") {
		track.RecordingID = &meta.RecordingID
	}
	if track.Artist == "" && meta.Artist != "" {
		track.Artist = meta.Artist
	}
	if len(track.Artists) == 0 && len(meta.Artists) > 0 {
		track.Artists = meta.Artists
	}
	if track.Title == "" && meta.Title != "" {
		track.Title = meta.Title
	}
	if track.Duration == 0 && meta.Duration > 0 {
		track.Duration = meta.Duration
	}
	if track.Year == 0 && meta.Year > 0 {
		track.Year = meta.Year
	}
	if track.Barcode == "" && meta.Barcode != "" {
		track.Barcode = meta.Barcode
	}
	if track.CatalogNumber == "" && meta.CatalogNumber != "" {
		track.CatalogNumber = meta.CatalogNumber
	}
	if track.ReleaseType == "" && meta.ReleaseType != "" {
		track.ReleaseType = meta.ReleaseType
	}
	if meta.ReleaseID != "" {
		track.ReleaseID = meta.ReleaseID
	}
	if len(track.ArtistIDs) == 0 && len(meta.ArtistIDs) > 0 {
		track.ArtistIDs = meta.ArtistIDs
	}
	if len(track.AlbumArtistIDs) == 0 && len(meta.AlbumArtistIDs) > 0 {
		track.AlbumArtistIDs = meta.AlbumArtistIDs
	}
	if len(track.AlbumArtists) == 0 && len(meta.AlbumArtists) > 0 {
		track.AlbumArtists = meta.AlbumArtists
	}
	if track.Composer == "" && meta.Composer != "" {
		track.Composer = meta.Composer
	}

	if track.Genre == "" && meta.Genre != "" {
		track.Genre = meta.Genre
	}
	if track.SubGenre == "" && meta.SubGenre != "" {
		track.SubGenre = meta.SubGenre
	}

	if domain.IsSameGenre(track.Genre, track.SubGenre) {
		track.SubGenre = ""
	}

	return nil
}
