package app

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

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

// -- Utility Merge Functions --

func coalesceString(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func coalesceInt(values ...int) int {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}

func coalesceFloat(values ...float64) float64 {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}

func coalesceStringSlice(values ...[]string) []string {
	for _, v := range values {
		if len(v) > 0 {
			return v
		}
	}
	return nil
}

// -- Main Enrichment Logic --

func (e *MetadataEnricher) EnrichComplete(ctx context.Context, track *domain.Track, logger *slog.Logger) {
	logger.Debug("EnrichComplete: starting", "track_year", track.Year, "track_provider_id", track.ProviderID)
	// 1. Hi-Fi metadata refresh
	if err := e.EnrichFromHiFi(ctx, track, logger); err != nil {
		logger.Warn("EnrichFromHiFi failed, proceeding with existing data", "error", err)
	}

	logger.Debug("EnrichComplete: after Hi-Fi", "track_year", track.Year)

	// 2. MusicBrainz Gap Fill
	if err := e.EnrichTrack(ctx, track, logger); err != nil {
		logger.Warn("MusicBrainz enrichment failed", "isrc", track.ISRC, "error", err)
	}

	logger.Debug("EnrichComplete: after MusicBrainz", "track_year", track.Year)

	// 3. Lyrics
	e.FetchLyrics(ctx, track, logger)
	logger.Debug("EnrichComplete: done", "track_year", track.Year)

	// 4. Final Fallbacks
	if track.AlbumArtist == "" && len(track.Artists) > 0 {
		track.AlbumArtist = track.Artists[0]
	}
	if len(track.AlbumArtists) == 0 && len(track.Artists) > 0 {
		track.AlbumArtists = track.Artists
	}
	if len(track.AlbumArtistIDs) == 0 && len(track.ArtistIDs) > 0 {
		track.AlbumArtistIDs = track.ArtistIDs
	}
	if track.PathArtist == "" {
		track.PathArtist = track.AlbumArtist
	}
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
	var ct *domain.CatalogTrack
	var album *domain.Album
	var err error

	if e.needsHiFiEnrichment(track) {
		ct, err = e.providerManager.GetProvider().GetTrack(ctx, track.ProviderID)
		if err != nil {
			logger.Warn("Failed to fetch Hi-Fi metadata for enrichment", "error", err)
		}
	}

	albumID := track.AlbumID
	if ct != nil && ct.AlbumID != "" {
		albumID = ct.AlbumID
	}

	// Determine if we need to fetch the Album metadata
	needsAlbumArtist := track.AlbumArtist == "" || len(track.AlbumArtists) == 0 || track.PathArtist == ""
	if ct != nil {
		needsAlbumArtist = ct.AlbumArtist == "" || len(ct.AlbumArtists) == 0
	}
	hasBasicMetadata := track.TotalTracks > 0 && track.TotalDiscs > 0 &&
		track.ReleaseDate != "" && track.Genre != "" && track.Label != ""
	if ct != nil {
		hasBasicMetadata = ct.TotalTracks > 0 && ct.TotalDiscs > 0 &&
			ct.ReleaseDate != "" && ct.Genre != "" && ct.Label != ""
	}

	if albumID != "" && (needsAlbumArtist || !hasBasicMetadata) {
		album, err = e.providerManager.GetProvider().GetAlbum(ctx, albumID)
		if err != nil {
			logger.Debug("Failed to fetch album metadata", "album_id", albumID, "error", err)
		}
	}

	// Merge Hi-Fi Data together
	e.mergeHiFi(track, ct, album)
	return nil
}

func (e *MetadataEnricher) UpdateTrackFromCatalog(track *domain.Track, ct *domain.CatalogTrack, logger *slog.Logger) {
	e.mergeHiFi(track, ct, nil)
}

func (e *MetadataEnricher) mergeHiFi(track *domain.Track, ct *domain.CatalogTrack, album *domain.Album) {
	if ct == nil {
		ct = &domain.CatalogTrack{}
	}
	if album == nil {
		album = &domain.Album{}
	}

	// Album-level fields Priority: Album > CatalogTrack > Track Existing
	track.AlbumArtist = coalesceString(album.Artist, ct.AlbumArtist, track.AlbumArtist)
	track.AlbumArtists = coalesceStringSlice(album.Artists, ct.AlbumArtists, track.AlbumArtists)
	if len(track.AlbumArtists) > 0 && track.AlbumArtist == "" {
		track.AlbumArtist = track.AlbumArtists[0]
	}

	track.AlbumArtistIDs = coalesceStringSlice(album.ArtistIDs, ct.AlbumArtistIDs, track.AlbumArtistIDs)
	track.Album = coalesceString(album.Title, ct.Album, track.Album)
	track.AlbumID = coalesceString(album.ID, ct.AlbumID, track.AlbumID)
	track.Genre = coalesceString(album.Genre, ct.Genre, track.Genre)
	track.Label = coalesceString(album.Label, ct.Label, track.Label)
	track.TotalTracks = coalesceInt(album.TotalTracks, ct.TotalTracks, track.TotalTracks)
	track.TotalDiscs = coalesceInt(album.TotalDiscs, ct.TotalDiscs, track.TotalDiscs)
	track.ReleaseDate = coalesceString(album.ReleaseDate, ct.ReleaseDate, track.ReleaseDate)
	track.AlbumArtURL = coalesceString(album.AlbumArtURL, ct.AlbumArtURL, track.AlbumArtURL)
	track.Barcode = coalesceString(album.UPC, track.Barcode)

	// Infer year from release date before checking explicit years
	e.setYearFromReleaseDate(track)
	track.Year = coalesceInt(album.Year, ct.Year, track.Year)

	// Track-level fields Priority: CatalogTrack > Track Existing
	title := coalesceString(ct.Title, track.Title)
	if ct.Title != "" && ct.Version != "" && !strings.Contains(strings.ToLower(title), strings.ToLower(ct.Version)) {
		title = fmt.Sprintf("%s (%s)", title, ct.Version)
	}
	track.Title = title

	track.Artist = coalesceString(ct.Artist, track.Artist)
	track.Artists = coalesceStringSlice(ct.Artists, track.Artists)
	track.ArtistIDs = coalesceStringSlice(ct.ArtistIDs, track.ArtistIDs)
	track.TrackNumber = coalesceInt(ct.TrackNumber, track.TrackNumber)
	track.DiscNumber = coalesceInt(ct.DiscNumber, track.DiscNumber)
	track.Duration = coalesceInt(ct.Duration, track.Duration)
	track.ISRC = coalesceString(ct.ISRC, track.ISRC)
	track.Copyright = coalesceString(ct.Copyright, track.Copyright)
	track.Composer = coalesceString(ct.Composer, track.Composer)
	track.BPM = coalesceInt(ct.BPM, track.BPM)
	track.Key = coalesceString(ct.Key, track.Key)
	track.KeyScale = coalesceString(ct.KeyScale, track.KeyScale)
	track.ReplayGain = coalesceFloat(ct.ReplayGain, track.ReplayGain)
	track.Peak = coalesceFloat(ct.Peak, track.Peak)
	track.Version = coalesceString(ct.Version, track.Version)
	track.Description = coalesceString(ct.Description, track.Description)
	track.URL = coalesceString(ct.URL, track.URL)
	track.AudioQuality = coalesceString(ct.AudioQuality, track.AudioQuality)

	if len(track.AudioModes) == 0 && len(ct.AudioModes) > 0 {
		track.AudioModes = ct.AudioModes
	}
	if !track.Explicit && ct.ExplicitLyrics {
		track.Explicit = true
	}
	if !track.Compilation && ct.Compilation {
		track.Compilation = true
	}

	// Update derived path artist
	track.PathArtist = track.AlbumArtist
	if len(track.AlbumArtists) > 0 {
		track.PathArtist = track.AlbumArtists[0]
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

	e.mergeMusicBrainz(track, meta, logger)
	return nil
}

func (e *MetadataEnricher) mergeMusicBrainz(track *domain.Track, mb *musicbrainz.RecordingMetadata, logger *slog.Logger) {
	if mb == nil {
		return
	}

	// MB provides Gap Fills (Priority: Track Existing > MB)
	if mb.RecordingID != "" && (track.RecordingID == nil || *track.RecordingID == "") {
		track.RecordingID = &mb.RecordingID
	}

	track.Artist = coalesceString(track.Artist, mb.Artist)
	track.Artists = coalesceStringSlice(track.Artists, mb.Artists)
	track.Title = coalesceString(track.Title, mb.Title)
	track.Duration = coalesceInt(track.Duration, mb.Duration)
	if track.Year == 0 && mb.Year > 0 {
		logger.Debug("Setting year from MusicBrainz", "old_year", track.Year, "new_year", mb.Year)
		track.Year = mb.Year
	}
	track.Barcode = coalesceString(track.Barcode, mb.Barcode)
	track.CatalogNumber = coalesceString(track.CatalogNumber, mb.CatalogNumber)
	track.ReleaseType = coalesceString(track.ReleaseType, mb.ReleaseType)
	track.ISRC = coalesceString(track.ISRC, mb.ISRC)
	track.Label = coalesceString(track.Label, mb.Label)
	track.ReleaseID = coalesceString(track.ReleaseID, mb.ReleaseID)
	track.ArtistIDs = coalesceStringSlice(track.ArtistIDs, mb.ArtistIDs)
	track.AlbumArtistIDs = coalesceStringSlice(track.AlbumArtistIDs, mb.AlbumArtistIDs)
	track.AlbumArtists = coalesceStringSlice(track.AlbumArtists, mb.AlbumArtists)
	track.AlbumArtist = coalesceString(track.AlbumArtist)
	if track.AlbumArtist == "" && len(mb.AlbumArtists) > 0 {
		track.AlbumArtist = mb.AlbumArtists[0]
	}
	if track.PathArtist == "" && len(mb.AlbumArtists) > 0 {
		track.PathArtist = mb.AlbumArtists[0]
	}
	track.Composer = coalesceString(track.Composer, mb.Composer)
	track.Genre = coalesceString(track.Genre, mb.Genre)
	if len(track.Tags) == 0 && len(mb.Tags) > 0 {
		track.Tags = mb.Tags
	}
}

func (e *MetadataEnricher) setYearFromReleaseDate(track *domain.Track) {
	if track.Year == 0 && track.ReleaseDate != "" && len(track.ReleaseDate) >= 4 {
		if y, err := strconv.Atoi(track.ReleaseDate[:4]); err == nil {
			track.Year = y
		}
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
