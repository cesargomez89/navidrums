package app

import (
	"context"
	"log/slog"

	"github.com/cesargomez89/navidrums/internal/catalog"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/musicbrainz"
)

type MetadataEnricher struct {
	mbClient        musicbrainz.ClientInterface
	providerManager *catalog.ProviderManager
}

func NewMetadataEnricher(mbClient musicbrainz.ClientInterface, pm *catalog.ProviderManager) *MetadataEnricher {
	return &MetadataEnricher{
		mbClient:        mbClient,
		providerManager: pm,
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
	if track.ISRC == "" && meta.ISRC != "" {
		track.ISRC = meta.ISRC
	}
	if track.Label == "" && meta.Label != "" {
		track.Label = meta.Label
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
		if meta.SubGenre != "" {
			track.Genre = meta.Genre + "; " + meta.SubGenre
		}
	}
	if len(track.Tags) == 0 && len(meta.Tags) > 0 {
		track.Tags = meta.Tags
	}

	return nil
}

func (e *MetadataEnricher) FetchLyrics(ctx context.Context, track *domain.Track, logger *slog.Logger) {
	if track.Lyrics != "" || track.Subtitles != "" {
		return
	}
	lyrics, subtitles, err := e.providerManager.GetProvider().GetLyrics(ctx, track.ProviderID)
	if err != nil {
		logger.Debug("Failed to fetch lyrics", "error", err)
		return
	}
	if track.Lyrics == "" && lyrics != "" {
		track.Lyrics = lyrics
	}
	if track.Subtitles == "" && subtitles != "" {
		track.Subtitles = subtitles
	}
}

func (e *MetadataEnricher) EnrichFromHiFi(ctx context.Context, track *domain.Track, logger *slog.Logger) error {
	catalogTrack, err := e.providerManager.GetProvider().GetTrack(ctx, track.ProviderID)
	if err != nil {
		logger.Warn("Failed to fetch Hi-Fi metadata for enrichment", "error", err)
		return err
	}

	oldTotalTracks := track.TotalTracks
	oldTotalDiscs := track.TotalDiscs
	oldReleaseDate := track.ReleaseDate
	oldGenre := track.Genre
	oldLabel := track.Label
	oldBarcode := track.Barcode

	e.updateTrackFromCatalog(track, catalogTrack)

	if track.TotalTracks == 0 && oldTotalTracks > 0 {
		track.TotalTracks = oldTotalTracks
	}
	if track.TotalDiscs == 0 && oldTotalDiscs > 0 {
		track.TotalDiscs = oldTotalDiscs
	}
	if track.ReleaseDate == "" && oldReleaseDate != "" {
		track.ReleaseDate = oldReleaseDate
	}
	if track.Genre == "" && oldGenre != "" {
		track.Genre = oldGenre
	}
	if track.Label == "" && oldLabel != "" {
		track.Label = oldLabel
	}
	if track.Barcode == "" && oldBarcode != "" {
		track.Barcode = oldBarcode
	}

	e.enrichWithAlbumMetadata(ctx, track, catalogTrack.AlbumID, logger)
	return nil
}

func (e *MetadataEnricher) EnrichComplete(ctx context.Context, track *domain.Track, logger *slog.Logger) {
	// 1. Hi-Fi metadata refresh
	if err := e.EnrichFromHiFi(ctx, track, logger); err != nil {
		logger.Warn("EnrichFromHiFi failed, proceeding with existing data", "error", err)
	}

	// 2. MusicBrainz Gap Fill
	if err := e.EnrichTrack(ctx, track, logger); err != nil {
		logger.Warn("MusicBrainz enrichment failed", "isrc", track.ISRC, "error", err)
	}

	// 3. Lyrics
	e.FetchLyrics(ctx, track, logger)
}

func (e *MetadataEnricher) updateTrackFromCatalog(track *domain.Track, ct *domain.CatalogTrack) {
	track.Title = ct.Title
	track.Artist = ct.Artist
	track.Artists = ct.Artists
	track.ArtistIDs = ct.ArtistIDs
	track.Album = ct.Album
	track.AlbumArtist = ct.AlbumArtist
	track.AlbumArtists = ct.AlbumArtists
	track.AlbumArtistIDs = ct.AlbumArtistIDs
	track.AlbumID = ct.AlbumID
	track.TrackNumber = ct.TrackNumber
	track.DiscNumber = ct.DiscNumber
	track.TotalTracks = ct.TotalTracks
	track.TotalDiscs = ct.TotalDiscs
	track.Year = ct.Year
	track.ReleaseDate = ct.ReleaseDate
	track.Genre = ct.Genre
	track.Label = ct.Label
	track.ISRC = ct.ISRC
	track.Copyright = ct.Copyright
	track.Composer = ct.Composer
	track.Duration = ct.Duration
	track.Explicit = ct.ExplicitLyrics
	track.Compilation = ct.Compilation
	track.AlbumArtURL = ct.AlbumArtURL
	track.BPM = ct.BPM
	track.Key = ct.Key
	track.KeyScale = ct.KeyScale
	track.ReplayGain = ct.ReplayGain
	track.Peak = ct.Peak
	track.Version = ct.Version
	track.Description = ct.Description
	track.URL = ct.URL
	track.AudioQuality = ct.AudioQuality
	track.AudioModes = ct.AudioModes
}

func (e *MetadataEnricher) enrichWithAlbumMetadata(ctx context.Context, track *domain.Track, albumID string, logger *slog.Logger) {
	if albumID == "" {
		return
	}

	hasAlbumMetadata := track.TotalTracks > 0 && track.TotalDiscs > 0 &&
		track.ReleaseDate != "" && track.Genre != "" && track.Label != ""
	if hasAlbumMetadata {
		logger.Debug("Track already has album metadata, skipping album fetch")
		return
	}

	album, err := e.providerManager.GetProvider().GetAlbum(ctx, albumID)
	if err != nil {
		logger.Debug("Failed to fetch album metadata", "album_id", albumID, "error", err)
		return
	}
	track.ReleaseDate = album.ReleaseDate
	track.Label = album.Label
	track.Genre = album.Genre
	track.TotalTracks = album.TotalTracks
	track.TotalDiscs = album.TotalDiscs
	track.Barcode = album.UPC
	if track.AlbumArtist == "" && album.Artist != "" {
		track.AlbumArtist = album.Artist
	}
	if len(track.AlbumArtists) == 0 && len(album.Artists) > 0 {
		track.AlbumArtists = album.Artists
	}
	if len(track.AlbumArtistIDs) == 0 && len(album.ArtistIDs) > 0 {
		track.AlbumArtistIDs = album.ArtistIDs
	}
	if album.AlbumArtURL != "" {
		track.AlbumArtURL = album.AlbumArtURL
	}
}
