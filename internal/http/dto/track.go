package dto

import (
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
)

type TrackUpdateRequest struct {
	Title         *string `form:"title"`
	Artist        *string `form:"artist"`
	Album         *string `form:"album"`
	AlbumArtist   *string `form:"album_artist"`
	Genre         *string `form:"genre"`
	SubGenre      *string `form:"sub_genre"`
	Label         *string `form:"label"`
	Composer      *string `form:"composer"`
	Copyright     *string `form:"copyright"`
	ISRC          *string `form:"isrc"`
	Version       *string `form:"version"`
	Description   *string `form:"description"`
	URL           *string `form:"url"`
	AudioQuality  *string `form:"audio_quality"`
	AudioModes    *string `form:"audio_modes"`
	Lyrics        *string `form:"lyrics"`
	Subtitles     *string `form:"subtitles"`
	Barcode       *string `form:"barcode"`
	CatalogNumber *string `form:"catalog_number"`
	ReleaseType   *string `form:"release_type"`
	ReleaseDate   *string `form:"release_date"`
	Key           *string `form:"key"`
	KeyScale      *string `form:"key_scale"`

	TrackNumber *int     `form:"track_number"`
	DiscNumber  *int     `form:"disc_number"`
	TotalTracks *int     `form:"total_tracks"`
	TotalDiscs  *int     `form:"total_discs"`
	Year        *int     `form:"year"`
	BPM         *int     `form:"bpm"`
	ReplayGain  *float64 `form:"replay_gain"`
	Peak        *float64 `form:"peak"`
	Compilation *bool    `form:"compilation"`
	Explicit    *bool    `form:"explicit"`
}

func (r *TrackUpdateRequest) Validate() []ValidationError {
	var errs []ValidationError

	errs = append(errs, validateYear(r.Year)...)
	errs = append(errs, validateBPM(r.BPM)...)
	errs = append(errs, validateTrackNumber(r.TrackNumber)...)
	errs = append(errs, validateDiscNumber(r.DiscNumber)...)
	errs = append(errs, validateTotalTracks(r.TotalTracks)...)
	errs = append(errs, validateTotalDiscs(r.TotalDiscs)...)
	errs = append(errs, validateReplayGain(r.ReplayGain)...)
	errs = append(errs, validatePeak(r.Peak)...)
	errs = append(errs, validateISRC(r.ISRC)...)
	errs = append(errs, validateReleaseDate(r.ReleaseDate)...)
	errs = append(errs, validateURL(r.URL)...)
	errs = append(errs, validateKeyScale(r.KeyScale)...)

	return errs
}

func (r *TrackUpdateRequest) ToUpdates() map[string]interface{} {
	updates := make(map[string]interface{})

	if r.Genre != nil && *r.Genre != "" {
		updates["genre"] = *r.Genre
	}
	if r.SubGenre != nil && *r.SubGenre != "" {
		updates["sub_genre"] = *r.SubGenre
	}
	if r.Title != nil && *r.Title != "" {
		updates["title"] = *r.Title
	}
	if r.Artist != nil && *r.Artist != "" {
		updates["artist"] = *r.Artist
	}
	if r.Album != nil && *r.Album != "" {
		updates["album"] = *r.Album
	}
	if r.AlbumArtist != nil && *r.AlbumArtist != "" {
		updates["album_artist"] = *r.AlbumArtist
	}
	if r.Label != nil && *r.Label != "" {
		updates["label"] = *r.Label
	}
	if r.Composer != nil && *r.Composer != "" {
		updates["composer"] = *r.Composer
	}
	if r.Copyright != nil && *r.Copyright != "" {
		updates["copyright"] = *r.Copyright
	}
	if r.ISRC != nil && *r.ISRC != "" {
		updates["isrc"] = *r.ISRC
	}
	if r.Version != nil && *r.Version != "" {
		updates["version"] = *r.Version
	}
	if r.Description != nil && *r.Description != "" {
		updates["description"] = *r.Description
	}
	if r.URL != nil && *r.URL != "" {
		updates["url"] = *r.URL
	}
	if r.AudioQuality != nil && *r.AudioQuality != "" {
		updates["audio_quality"] = *r.AudioQuality
	}
	if r.AudioModes != nil && *r.AudioModes != "" {
		updates["audio_modes"] = *r.AudioModes
	}
	if r.Lyrics != nil && *r.Lyrics != "" {
		updates["lyrics"] = *r.Lyrics
	}
	if r.Subtitles != nil && *r.Subtitles != "" {
		updates["subtitles"] = *r.Subtitles
	}
	if r.Barcode != nil && *r.Barcode != "" {
		updates["barcode"] = *r.Barcode
	}
	if r.CatalogNumber != nil && *r.CatalogNumber != "" {
		updates["catalog_number"] = *r.CatalogNumber
	}
	if r.ReleaseType != nil && *r.ReleaseType != "" {
		updates["release_type"] = *r.ReleaseType
	}
	if r.ReleaseDate != nil && *r.ReleaseDate != "" {
		updates["release_date"] = *r.ReleaseDate
	}
	if r.Key != nil && *r.Key != "" {
		updates["key_name"] = *r.Key
	}
	if r.KeyScale != nil && *r.KeyScale != "" {
		updates["key_scale"] = *r.KeyScale
	}

	if r.TrackNumber != nil {
		updates["track_number"] = *r.TrackNumber
	}
	if r.DiscNumber != nil {
		updates["disc_number"] = *r.DiscNumber
	}
	if r.TotalTracks != nil {
		updates["total_tracks"] = *r.TotalTracks
	}
	if r.TotalDiscs != nil {
		updates["total_discs"] = *r.TotalDiscs
	}
	if r.Year != nil {
		updates["year"] = *r.Year
	}
	if r.BPM != nil {
		updates["bpm"] = *r.BPM
	}
	if r.ReplayGain != nil {
		updates["replay_gain"] = *r.ReplayGain
	}
	if r.Peak != nil {
		updates["peak"] = *r.Peak
	}
	if r.Compilation != nil {
		updates["compilation"] = *r.Compilation
	}
	if r.Explicit != nil {
		updates["explicit"] = *r.Explicit
	}

	return updates
}

type TrackResponse struct {
	CreatedAt      time.Time  `json:"created_at"`
	LastVerifiedAt *time.Time `json:"last_verified_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Lyrics         string     `json:"lyrics"`
	Version        string     `json:"version"`
	Label          string     `json:"label"`
	Title          string     `json:"title"`
	Artist         string     `json:"artist"`
	Album          string     `json:"album"`
	AlbumArtist    string     `json:"album_artist"`
	FilePath       string     `json:"file_path"`
	Status         string     `json:"status"`
	ProviderID     string     `json:"provider_id"`
	AlbumID        string     `json:"album_id"`
	ReleaseID      string     `json:"release_id"`
	Composer       string     `json:"composer"`
	ReleaseType    string     `json:"release_type"`
	ISRC           string     `json:"isrc"`
	CatalogNumber  string     `json:"catalog_number"`
	Description    string     `json:"description"`
	URL            string     `json:"url"`
	AudioQuality   string     `json:"audio_quality"`
	AudioModes     string     `json:"audio_modes"`
	Error          string     `json:"error,omitempty"`
	Subtitles      string     `json:"subtitles"`
	Genre          string     `json:"genre"`
	SubGenre       string     `json:"sub_genre"`
	Barcode        string     `json:"barcode"`
	Copyright      string     `json:"copyright"`
	ReleaseDate    string     `json:"release_date"`
	Key            string     `json:"key"`
	KeyScale       string     `json:"key_scale"`
	ParentJobID    string     `json:"parent_job_id"`
	FileExtension  string     `json:"file_extension"`
	AlbumArtURL    string     `json:"album_art_url"`
	AlbumArtists   []string   `json:"album_artists"`
	Artists        []string   `json:"artists"`
	TotalDiscs     int        `json:"total_discs"`
	ID             int        `json:"id"`
	Peak           float64    `json:"peak"`
	TotalTracks    int        `json:"total_tracks"`
	ReplayGain     float64    `json:"replay_gain"`
	BPM            int        `json:"bpm"`
	Duration       int        `json:"duration"`
	Year           int        `json:"year"`
	DiscNumber     int        `json:"disc_number"`
	TrackNumber    int        `json:"track_number"`
	Compilation    bool       `json:"compilation"`
	Explicit       bool       `json:"explicit"`
}

func NewTrackResponse(t *domain.Track) TrackResponse {
	return TrackResponse{
		ID:             t.ID,
		Title:          t.Title,
		Artist:         t.Artist,
		Album:          t.Album,
		AlbumArtist:    t.AlbumArtist,
		Genre:          t.Genre,
		SubGenre:       t.SubGenre,
		Label:          t.Label,
		TrackNumber:    t.TrackNumber,
		DiscNumber:     t.DiscNumber,
		Year:           t.Year,
		Duration:       t.Duration,
		FilePath:       t.FilePath,
		Status:         string(t.Status),
		ProviderID:     t.ProviderID,
		AlbumID:        t.AlbumID,
		ReleaseID:      t.ReleaseID,
		Composer:       t.Composer,
		Copyright:      t.Copyright,
		ISRC:           t.ISRC,
		Version:        t.Version,
		Description:    t.Description,
		URL:            t.URL,
		AudioQuality:   t.AudioQuality,
		AudioModes:     t.AudioModes,
		Lyrics:         t.Lyrics,
		Subtitles:      t.Subtitles,
		Barcode:        t.Barcode,
		CatalogNumber:  t.CatalogNumber,
		ReleaseType:    t.ReleaseType,
		ReleaseDate:    t.ReleaseDate,
		Key:            t.Key,
		KeyScale:       t.KeyScale,
		BPM:            t.BPM,
		ReplayGain:     t.ReplayGain,
		Peak:           t.Peak,
		Compilation:    t.Compilation,
		Explicit:       t.Explicit,
		TotalTracks:    t.TotalTracks,
		TotalDiscs:     t.TotalDiscs,
		AlbumArtURL:    t.AlbumArtURL,
		FileExtension:  t.FileExtension,
		Artists:        t.Artists,
		AlbumArtists:   t.AlbumArtists,
		Error:          t.Error,
		ParentJobID:    t.ParentJobID,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
		CompletedAt:    t.CompletedAt,
		LastVerifiedAt: t.LastVerifiedAt,
	}
}
