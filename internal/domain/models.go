package domain

import (
	"database/sql"
	"strings"
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
	TrackStatusDownloaded  TrackStatus = "downloaded"
	TrackStatusProcessing  TrackStatus = "processing"
	TrackStatusCompleted   TrackStatus = "completed"
	TrackStatusFailed      TrackStatus = "failed"
)

// Track represents a track with full metadata for downloading
type Track struct { //nolint:govet // field ordering prioritizes readability over memory alignment
	ID             int            `json:"id" db:"id"`
	ProviderID     string         `json:"provider_id" db:"provider_id"`
	Title          string         `json:"title" db:"title"`
	Artist         string         `json:"artist" db:"artist"`
	Artists        StringSlice    `json:"artists" db:"artists"`
	Album          string         `json:"album" db:"album"`
	AlbumID        string         `json:"album_id,omitempty" db:"album_id"`
	AlbumArtist    string         `json:"album_artist" db:"album_artist"`
	AlbumArtists   StringSlice    `json:"album_artists" db:"album_artists"`
	TrackNumber    int            `json:"track_number" db:"track_number"`
	DiscNumber     int            `json:"disc_number" db:"disc_number"`
	TotalTracks    int            `json:"total_tracks" db:"total_tracks"`
	TotalDiscs     int            `json:"total_discs" db:"total_discs"`
	Year           int            `json:"year" db:"year"`
	Duration       int            `json:"duration" db:"duration"`
	Genre          string         `json:"genre" db:"genre"`
	SubGenre       sql.NullString `json:"-" db:"sub_genre"` // legacy DB column; absorbed to avoid scan errors
	Label          string         `json:"label" db:"label"`
	ISRC           string         `json:"isrc" db:"isrc"`
	Copyright      string         `json:"copyright" db:"copyright"`
	Composer       string         `json:"composer" db:"composer"`
	Explicit       bool           `json:"explicit" db:"explicit"`
	Compilation    bool           `json:"compilation" db:"compilation"`
	AlbumArtURL    string         `json:"album_art_url" db:"album_art_url"`
	Lyrics         string         `json:"lyrics" db:"lyrics"`
	Subtitles      string         `json:"subtitles" db:"subtitles"`
	BPM            int            `json:"bpm,omitempty" db:"bpm"`
	Key            string         `json:"key,omitempty" db:"key_name"`
	KeyScale       string         `json:"key_scale,omitempty" db:"key_scale"`
	ReplayGain     float64        `json:"replay_gain,omitempty" db:"replay_gain"`
	Peak           float64        `json:"peak,omitempty" db:"peak"`
	Version        string         `json:"version,omitempty" db:"version"`
	Description    string         `json:"description,omitempty" db:"description"`
	URL            string         `json:"url,omitempty" db:"url"`
	AudioQuality   string         `json:"audio_quality,omitempty" db:"audio_quality"`
	AudioModes     string         `json:"audio_modes,omitempty" db:"audio_modes"`
	ReleaseDate    string         `json:"release_date,omitempty" db:"release_date"`
	Barcode        string         `json:"barcode,omitempty" db:"barcode"`
	CatalogNumber  string         `json:"catalog_number,omitempty" db:"catalog_number"`
	ReleaseType    string         `json:"release_type,omitempty" db:"release_type"`
	ReleaseID      string         `json:"release_id,omitempty" db:"release_id"`
	RecordingID    *string        `json:"recording_id,omitempty" db:"recording_id"`
	Tags           StringSlice    `json:"tags,omitempty" db:"tags"`
	Status         TrackStatus    `json:"status" db:"status"`
	Error          string         `json:"error,omitempty" db:"error"`
	ParentJobID    string         `json:"parent_job_id" db:"parent_job_id"`
	FilePath       string         `json:"file_path" db:"file_path"`
	FileExtension  string         `json:"file_extension" db:"file_extension"`
	FileHash       string         `json:"file_hash,omitempty" db:"file_hash"`
	ETag           string         `json:"etag,omitempty" db:"etag"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" db:"updated_at"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty" db:"completed_at"`
	LastVerifiedAt *time.Time     `json:"last_verified_at,omitempty" db:"last_verified_at"`
	ArtistIDs      StringSlice    `json:"artist_ids,omitempty" db:"artist_ids"`
	AlbumArtistIDs StringSlice    `json:"album_artist_ids,omitempty" db:"album_artist_ids"`
}

// Normalize ensures the track data is consistent.
func (t *Track) Normalize() {
	if t.Genre != "" {
		t.Genre = strings.ToLower(t.Genre)
	}
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
