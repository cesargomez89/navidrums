package domain

import (
	"time"
)

type JobType string

const (
	JobTypeTrack           JobType = "track"
	JobTypeAlbum           JobType = "album"
	JobTypePlaylist        JobType = "playlist"
	JobTypeArtist          JobType = "artist"
	JobTypeSyncFile        JobType = "sync_file"
	JobTypeSyncMusicBrainz JobType = "sync_musicbrainz"
	JobTypeSyncHiFi        JobType = "sync_hifi"
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
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Error     *string   `json:"error,omitempty" db:"error"`
	ID        string    `json:"id" db:"id"`
	Type      JobType   `json:"type" db:"type"`
	Status    JobStatus `json:"status" db:"status"`
	SourceID  string    `json:"source_id" db:"source_id"`
	Progress  float64   `json:"progress" db:"progress"`
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
	LastVerifiedAt *time.Time  `json:"last_verified_at,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	CompletedAt    *time.Time  `json:"completed_at,omitempty"`
	Composer       string      `json:"composer"`
	Error          string      `json:"error,omitempty"`
	AlbumID        string      `json:"album_id,omitempty"`
	AlbumArtist    string      `json:"album_artist"`
	FileHash       string      `json:"file_hash,omitempty"`
	ETag           string      `json:"etag,omitempty"`
	Artist         string      `json:"artist"`
	Title          string      `json:"title"`
	ProviderID     string      `json:"provider_id"`
	FileExtension  string      `json:"file_extension"`
	Genre          string      `json:"genre"`
	SubGenre       string      `json:"sub_genre"`
	Label          string      `json:"label"`
	ISRC           string      `json:"isrc"`
	Copyright      string      `json:"copyright"`
	ReleaseDate    string      `json:"release_date,omitempty"`
	FilePath       string      `json:"file_path"`
	ParentJobID    string      `json:"parent_job_id"`
	Album          string      `json:"album"`
	AlbumArtURL    string      `json:"album_art_url"`
	Lyrics         string      `json:"lyrics"`
	Subtitles      string      `json:"subtitles"`
	Status         TrackStatus `json:"status"`
	ReleaseID      string      `json:"release_id,omitempty"`
	ReleaseType    string      `json:"release_type,omitempty"`
	Key            string      `json:"key,omitempty"`
	KeyScale       string      `json:"key_scale,omitempty"`
	CatalogNumber  string      `json:"catalog_number,omitempty"`
	Barcode        string      `json:"barcode,omitempty"`
	Version        string      `json:"version,omitempty"`
	Description    string      `json:"description,omitempty"`
	URL            string      `json:"url,omitempty"`
	AudioQuality   string      `json:"audio_quality,omitempty"`
	AudioModes     string      `json:"audio_modes,omitempty"`
	AlbumArtistIDs []string    `json:"album_artist_ids,omitempty"`
	ArtistIDs      []string    `json:"artist_ids,omitempty"`
	Artists        []string    `json:"artists"`
	AlbumArtists   []string    `json:"album_artists"`
	TotalDiscs     int         `json:"total_discs"`
	ID             int         `json:"id"`
	Duration       int         `json:"duration"`
	Year           int         `json:"year"`
	Peak           float64     `json:"peak,omitempty"`
	TotalTracks    int         `json:"total_tracks"`
	DiscNumber     int         `json:"disc_number"`
	TrackNumber    int         `json:"track_number"`
	BPM            int         `json:"bpm,omitempty"`
	ReplayGain     float64     `json:"replay_gain,omitempty"`
	Compilation    bool        `json:"compilation"`
	Explicit       bool        `json:"explicit"`
}

// CatalogTrack represents a track from the provider/catalog
type CatalogTrack struct {
	KeyScale       string   `json:"key_scale,omitempty"`
	Lyrics         string   `json:"lyrics,omitempty"`
	ArtistID       string   `json:"artist_id,omitempty"`
	Artist         string   `json:"artist"`
	ReleaseDate    string   `json:"release_date,omitempty"`
	Subtitles      string   `json:"subtitles,omitempty"`
	AlbumID        string   `json:"album_id,omitempty"`
	Album          string   `json:"album"`
	AlbumArtist    string   `json:"album_artist,omitempty"`
	ISRC           string   `json:"isrc,omitempty"`
	Genre          string   `json:"genre,omitempty"`
	AlbumArtURL    string   `json:"album_art_url,omitempty"`
	AudioModes     string   `json:"audio_modes,omitempty"`
	AudioQuality   string   `json:"audio_quality,omitempty"`
	URL            string   `json:"url,omitempty"`
	Description    string   `json:"description,omitempty"`
	Version        string   `json:"version,omitempty"`
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Label          string   `json:"label,omitempty"`
	Key            string   `json:"key,omitempty"`
	Copyright      string   `json:"copyright,omitempty"`
	Composer       string   `json:"composer,omitempty"`
	AlbumArtistIDs []string `json:"album_artist_ids,omitempty"`
	ArtistIDs      []string `json:"artist_ids,omitempty"`
	Artists        []string `json:"artists,omitempty"`
	AlbumArtists   []string `json:"album_artists,omitempty"`
	Duration       int      `json:"duration"`
	ReplayGain     float64  `json:"replay_gain,omitempty"`
	Peak           float64  `json:"peak,omitempty"`
	TotalDiscs     int      `json:"total_discs,omitempty"`
	TotalTracks    int      `json:"total_tracks,omitempty"`
	DiscNumber     int      `json:"disc_number,omitempty"`
	TrackNumber    int      `json:"track_number"`
	Year           int      `json:"year,omitempty"`
	BPM            int      `json:"bpm,omitempty"`
	ExplicitLyrics bool     `json:"explicit_lyrics,omitempty"`
	Compilation    bool     `json:"compilation,omitempty"`
}

type Album struct {
	AlbumArtURL string         `json:"album_art_url,omitempty"`
	Title       string         `json:"title"`
	ArtistID    string         `json:"artist_id,omitempty"`
	Artist      string         `json:"artist"`
	ID          string         `json:"id"`
	URL         string         `json:"url,omitempty"`
	AlbumType   string         `json:"album_type,omitempty"`
	ReleaseDate string         `json:"release_date,omitempty"`
	Genre       string         `json:"genre,omitempty"`
	Label       string         `json:"label,omitempty"`
	Copyright   string         `json:"copyright,omitempty"`
	UPC         string         `json:"upc,omitempty"`
	Artists     []string       `json:"artists,omitempty"`
	Tracks      []CatalogTrack `json:"tracks"`
	ArtistIDs   []string       `json:"artist_ids,omitempty"`
	TotalDiscs  int            `json:"total_discs,omitempty"`
	TotalTracks int            `json:"total_tracks,omitempty"`
	Year        int            `json:"year,omitempty"`
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
