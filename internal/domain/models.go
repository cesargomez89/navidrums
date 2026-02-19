package domain

import (
	"time"
)

type JobType string

const (
	JobTypeTrack    JobType = "track"
	JobTypeAlbum    JobType = "album"
	JobTypePlaylist JobType = "playlist"
	JobTypeArtist   JobType = "artist"
)

type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// Job represents a work item in the queue
type Job struct {
	ID        string    `json:"id"`
	Type      JobType   `json:"type"`
	Status    JobStatus `json:"status"`
	Progress  float64   `json:"progress"` // 0-100
	SourceID  string    `json:"source_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Error     string    `json:"error,omitempty"`
}

// TrackStatus represents the download status of a track
type TrackStatus string

const (
	TrackStatusMissing     TrackStatus = "missing"
	TrackStatusQueued      TrackStatus = "queued"
	TrackStatusDownloading TrackStatus = "downloading"
	TrackStatusProcessing  TrackStatus = "processing"
	TrackStatusCompleted   TrackStatus = "completed"
	TrackStatusFailed      TrackStatus = "failed"
)

// Track represents a track with full metadata for downloading
type Track struct {
	// Identity
	ID         int    `json:"id"`
	ProviderID string `json:"provider_id"`

	// Metadata (all queryable)
	Title        string   `json:"title"`
	Artist       string   `json:"artist"`
	Artists      []string `json:"artists"`
	Album        string   `json:"album"`
	AlbumID      string   `json:"album_id,omitempty"`
	AlbumArtist  string   `json:"album_artist"`
	AlbumArtists []string `json:"album_artists"`
	TrackNumber  int      `json:"track_number"`
	DiscNumber   int      `json:"disc_number"`
	TotalTracks  int      `json:"total_tracks"`
	TotalDiscs   int      `json:"total_discs"`
	Year         int      `json:"year"`
	Genre        string   `json:"genre"`
	Label        string   `json:"label"`
	ISRC         string   `json:"isrc"`
	Copyright    string   `json:"copyright"`
	Composer     string   `json:"composer"`
	Duration     int      `json:"duration"`
	Explicit     bool     `json:"explicit"`
	Compilation  bool     `json:"compilation"`
	AlbumArtURL  string   `json:"album_art_url"`
	Lyrics       string   `json:"lyrics"`
	Subtitles    string   `json:"subtitles"`

	// Extended metadata for tagging
	ArtistIDs      []string `json:"artist_ids,omitempty"`
	AlbumArtistIDs []string `json:"album_artist_ids,omitempty"`
	BPM            int      `json:"bpm,omitempty"`
	Key            string   `json:"key,omitempty"`
	KeyScale       string   `json:"key_scale,omitempty"`
	ReplayGain     float64  `json:"replay_gain,omitempty"`
	Peak           float64  `json:"peak,omitempty"`
	Version        string   `json:"version,omitempty"`
	Description    string   `json:"description,omitempty"`
	URL            string   `json:"url,omitempty"`
	AudioQuality   string   `json:"audio_quality,omitempty"`
	AudioModes     string   `json:"audio_modes,omitempty"`
	ReleaseDate    string   `json:"release_date,omitempty"`
	Barcode        string   `json:"barcode,omitempty"`
	CatalogNumber  string   `json:"catalog_number,omitempty"`
	ReleaseType    string   `json:"release_type,omitempty"`
	ReleaseID      string   `json:"release_id,omitempty"`

	// Processing
	Status      TrackStatus `json:"status"`
	Error       string      `json:"error,omitempty"`
	ParentJobID string      `json:"parent_job_id"`

	// File
	FilePath      string `json:"file_path"`
	FileExtension string `json:"file_extension"`

	// Timestamps
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CompletedAt    time.Time `json:"completed_at,omitempty"`
	ETag           string    `json:"etag,omitempty"`
	FileHash       string    `json:"file_hash,omitempty"`
	LastVerifiedAt time.Time `json:"last_verified_at,omitempty"`
}

// CatalogTrack represents a track from the provider/catalog
type CatalogTrack struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	ArtistID       string   `json:"artist_id,omitempty"`
	Artist         string   `json:"artist"`
	Artists        []string `json:"artists,omitempty"`
	ArtistIDs      []string `json:"artist_ids,omitempty"`
	AlbumID        string   `json:"album_id,omitempty"`
	Album          string   `json:"album"`
	AlbumArtist    string   `json:"album_artist,omitempty"`
	AlbumArtists   []string `json:"album_artists,omitempty"`
	AlbumArtistIDs []string `json:"album_artist_ids,omitempty"`
	Compilation    bool     `json:"compilation,omitempty"`
	TrackNumber    int      `json:"track_number"`
	DiscNumber     int      `json:"disc_number,omitempty"`
	TotalTracks    int      `json:"total_tracks,omitempty"`
	TotalDiscs     int      `json:"total_discs,omitempty"`
	Duration       int      `json:"duration"`
	Year           int      `json:"year,omitempty"`
	Genre          string   `json:"genre,omitempty"`
	Label          string   `json:"label,omitempty"`
	ISRC           string   `json:"isrc,omitempty"`
	Copyright      string   `json:"copyright,omitempty"`
	Composer       string   `json:"composer,omitempty"`
	AlbumArtURL    string   `json:"album_art_url,omitempty"`
	ExplicitLyrics bool     `json:"explicit_lyrics,omitempty"`
	BPM            int      `json:"bpm,omitempty"`
	Key            string   `json:"key,omitempty"`
	KeyScale       string   `json:"key_scale,omitempty"`
	ReplayGain     float64  `json:"replay_gain,omitempty"`
	Peak           float64  `json:"peak,omitempty"`
	Version        string   `json:"version,omitempty"`
	Description    string   `json:"description,omitempty"`
	URL            string   `json:"url,omitempty"`
	AudioQuality   string   `json:"audio_quality,omitempty"`
	AudioModes     string   `json:"audio_modes,omitempty"`
	Lyrics         string   `json:"lyrics,omitempty"`
	Subtitles      string   `json:"subtitles,omitempty"`
	ReleaseDate    string   `json:"release_date,omitempty"`
}

type Album struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	ArtistID    string         `json:"artist_id,omitempty"`
	Artist      string         `json:"artist"`
	Artists     []string       `json:"artists,omitempty"`
	ArtistIDs   []string       `json:"artist_ids,omitempty"`
	Year        int            `json:"year,omitempty"`
	ReleaseDate string         `json:"release_date,omitempty"`
	Genre       string         `json:"genre,omitempty"`
	Label       string         `json:"label,omitempty"`
	Copyright   string         `json:"copyright,omitempty"`
	TotalTracks int            `json:"total_tracks,omitempty"`
	TotalDiscs  int            `json:"total_discs,omitempty"`
	AlbumArtURL string         `json:"album_art_url,omitempty"`
	Tracks      []CatalogTrack `json:"tracks"`
	UPC         string         `json:"upc,omitempty"`
	AlbumType   string         `json:"album_type,omitempty"`
	URL         string         `json:"url,omitempty"`
	Explicit    bool           `json:"explicit,omitempty"`
}

type Artist struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	PictureURL string         `json:"picture_url,omitempty"`
	Albums     []Album        `json:"albums,omitempty"`
	TopTracks  []CatalogTrack `json:"top_tracks,omitempty"`
}

type Playlist struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	ImageURL    string         `json:"image_url,omitempty"`
	Tracks      []CatalogTrack `json:"tracks"`
}

type SearchResult struct {
	Artists   []Artist       `json:"artists"`
	Albums    []Album        `json:"albums"`
	Playlists []Playlist     `json:"playlists"`
	Tracks    []CatalogTrack `json:"tracks"`
}
