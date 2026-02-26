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

	if !e.needsMusicBrainzEnrichment(track) {
		logger.Debug("Track already has all MusicBrainz fields, skipping enrichment", "isrc", track.ISRC)
		return nil
	}

	meta, mbErr := e.mbClient.GetRecording(ctx, recordingID, track.ISRC, track.Album)
	if mbErr != nil {
		return mbErr
	}
	if meta == nil {
		return nil
	}

	e.fillTrackFromMusicBrainz(track, meta)

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
	if !e.needsHiFiEnrichment(track) {
		logger.Debug("Track already has all Hi-Fi fields, skipping enrichment", "provider_id", track.ProviderID)
		e.enrichWithAlbumMetadata(ctx, track, track.AlbumID, logger)
		return nil
	}

	catalogTrack, err := e.providerManager.GetProvider().GetTrack(ctx, track.ProviderID)
	if err != nil {
		logger.Warn("Failed to fetch Hi-Fi metadata for enrichment", "error", err)
		return err
	}

	e.UpdateTrackFromCatalog(track, catalogTrack)

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

func (e *MetadataEnricher) fillTrackFromMusicBrainz(track *domain.Track, meta *musicbrainz.RecordingMetadata) {
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
	if track.ReleaseID == "" && meta.ReleaseID != "" {
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
	if len(track.Tags) == 0 && len(meta.Tags) > 0 {
		track.Tags = meta.Tags
	}
}

func (e *MetadataEnricher) UpdateTrackFromCatalog(track *domain.Track, ct *domain.CatalogTrack) {
	if track.Title == "" && ct.Title != "" {
		track.Title = ct.Title
	}
	if track.Artist == "" && ct.Artist != "" {
		track.Artist = ct.Artist
	}
	if len(track.Artists) == 0 && len(ct.Artists) > 0 {
		track.Artists = ct.Artists
	}
	if len(track.ArtistIDs) == 0 && len(ct.ArtistIDs) > 0 {
		track.ArtistIDs = ct.ArtistIDs
	}
	if track.Album == "" && ct.Album != "" {
		track.Album = ct.Album
	}
	if track.AlbumArtist == "" && ct.AlbumArtist != "" {
		track.AlbumArtist = ct.AlbumArtist
	}
	if len(track.AlbumArtists) == 0 && len(ct.AlbumArtists) > 0 {
		track.AlbumArtists = ct.AlbumArtists
	}
	if len(track.AlbumArtistIDs) == 0 && len(ct.AlbumArtistIDs) > 0 {
		track.AlbumArtistIDs = ct.AlbumArtistIDs
	}
	if track.AlbumID == "" && ct.AlbumID != "" {
		track.AlbumID = ct.AlbumID
	}
	if track.TrackNumber == 0 && ct.TrackNumber > 0 {
		track.TrackNumber = ct.TrackNumber
	}
	if track.DiscNumber == 0 && ct.DiscNumber > 0 {
		track.DiscNumber = ct.DiscNumber
	}
	if track.TotalTracks == 0 && ct.TotalTracks > 0 {
		track.TotalTracks = ct.TotalTracks
	}
	if track.TotalDiscs == 0 && ct.TotalDiscs > 0 {
		track.TotalDiscs = ct.TotalDiscs
	}
	if track.Year == 0 && ct.Year > 0 {
		track.Year = ct.Year
	}
	if track.ReleaseDate == "" && ct.ReleaseDate != "" {
		track.ReleaseDate = ct.ReleaseDate
	}
	if track.Genre == "" && ct.Genre != "" {
		track.Genre = ct.Genre
	}
	if track.Label == "" && ct.Label != "" {
		track.Label = ct.Label
	}
	if track.ISRC == "" && ct.ISRC != "" {
		track.ISRC = ct.ISRC
	}
	if track.Copyright == "" && ct.Copyright != "" {
		track.Copyright = ct.Copyright
	}
	if track.Composer == "" && ct.Composer != "" {
		track.Composer = ct.Composer
	}
	if track.Duration == 0 && ct.Duration > 0 {
		track.Duration = ct.Duration
	}
	if !track.Explicit && ct.ExplicitLyrics {
		track.Explicit = ct.ExplicitLyrics
	}
	if !track.Compilation && ct.Compilation {
		track.Compilation = ct.Compilation
	}
	if track.AlbumArtURL == "" && ct.AlbumArtURL != "" {
		track.AlbumArtURL = ct.AlbumArtURL
	}
	if track.BPM == 0 && ct.BPM > 0 {
		track.BPM = ct.BPM
	}
	if track.Key == "" && ct.Key != "" {
		track.Key = ct.Key
	}
	if track.KeyScale == "" && ct.KeyScale != "" {
		track.KeyScale = ct.KeyScale
	}
	if track.ReplayGain == 0 && ct.ReplayGain != 0 {
		track.ReplayGain = ct.ReplayGain
	}
	if track.Peak == 0 && ct.Peak != 0 {
		track.Peak = ct.Peak
	}
	if track.Version == "" && ct.Version != "" {
		track.Version = ct.Version
	}
	if track.Description == "" && ct.Description != "" {
		track.Description = ct.Description
	}
	if track.URL == "" && ct.URL != "" {
		track.URL = ct.URL
	}
	if track.AudioQuality == "" && ct.AudioQuality != "" {
		track.AudioQuality = ct.AudioQuality
	}
	if len(track.AudioModes) == 0 && len(ct.AudioModes) > 0 {
		track.AudioModes = ct.AudioModes
	}
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
	if track.ReleaseDate == "" && album.ReleaseDate != "" {
		track.ReleaseDate = album.ReleaseDate
	}
	if track.Label == "" && album.Label != "" {
		track.Label = album.Label
	}
	if track.Genre == "" && album.Genre != "" {
		track.Genre = album.Genre
	}
	if track.TotalTracks == 0 && album.TotalTracks > 0 {
		track.TotalTracks = album.TotalTracks
	}
	if track.TotalDiscs == 0 && album.TotalDiscs > 0 {
		track.TotalDiscs = album.TotalDiscs
	}
	if track.Barcode == "" && album.UPC != "" {
		track.Barcode = album.UPC
	}
	if track.AlbumArtist == "" && album.Artist != "" {
		track.AlbumArtist = album.Artist
	}
	if len(track.AlbumArtists) == 0 && len(album.Artists) > 0 {
		track.AlbumArtists = album.Artists
	}
	if len(track.AlbumArtistIDs) == 0 && len(album.ArtistIDs) > 0 {
		track.AlbumArtistIDs = album.ArtistIDs
	}
	if track.AlbumArtURL == "" && album.AlbumArtURL != "" {
		track.AlbumArtURL = album.AlbumArtURL
	}
}

func (e *MetadataEnricher) missingCommonMetadata(track *domain.Track) bool {
	return track.Artist == "" || len(track.Artists) == 0 || track.Title == "" ||
		track.Duration == 0 || track.Year == 0 || track.ISRC == "" || track.Label == "" ||
		len(track.ArtistIDs) == 0 || len(track.AlbumArtistIDs) == 0 ||
		len(track.AlbumArtists) == 0 || track.Composer == "" || track.Genre == ""
}

func (e *MetadataEnricher) needsMusicBrainzEnrichment(track *domain.Track) bool {
	return track.RecordingID == nil || *track.RecordingID == "" ||
		track.Barcode == "" || track.CatalogNumber == "" || track.ReleaseType == "" ||
		track.ReleaseID == "" || len(track.Tags) == 0 || e.missingCommonMetadata(track)
}

func (e *MetadataEnricher) needsHiFiEnrichment(track *domain.Track) bool {
	return track.Album == "" || track.AlbumArtist == "" ||
		track.AlbumID == "" || track.TrackNumber == 0 || track.DiscNumber == 0 ||
		track.TotalTracks == 0 || track.TotalDiscs == 0 || track.ReleaseDate == "" ||
		track.Copyright == "" || track.AlbumArtURL == "" || track.BPM == 0 ||
		track.Key == "" || track.KeyScale == "" || track.ReplayGain == 0 ||
		track.Peak == 0 || track.Version == "" || track.Description == "" ||
		track.URL == "" || track.AudioQuality == "" || len(track.AudioModes) == 0 || e.missingCommonMetadata(track)
}
