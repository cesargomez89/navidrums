package domain

import (
	"time"
)

type JobType string

const (
	JobTypeTrack    JobType = "track"
	JobTypeAlbum    JobType = "album"
	JobTypePlaylist JobType = "playlist"
	JobTypeArtist   JobType = "artist" // effectively playlist of top tracks
)

type JobStatus string

const (
	JobStatusQueued      JobStatus = "queued"
	JobStatusResolve     JobStatus = "resolving_tracks"
	JobStatusDownloading JobStatus = "downloading"
	JobStatusCompleted   JobStatus = "completed"
	JobStatusFailed      JobStatus = "failed"
	JobStatusCancelled   JobStatus = "cancelled"
)

type Job struct {
	ID        string    `json:"id"`
	Type      JobType   `json:"type"`
	Status    JobStatus `json:"status"`
	Title     string    `json:"title"`
	Artist    string    `json:"artist"`
	Progress  float64   `json:"progress"` // 0-100
	SourceID  string    `json:"source_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Error     string    `json:"error,omitempty"`
}

type Download struct {
	ProviderID    string    `json:"provider_id"`
	FilePath      string    `json:"file_path"`      // Absolute path
	FileExtension string    `json:"file_extension"` // e.g., ".flac", ".mp3", ".m4a"
	CompletedAt   time.Time `json:"completed_at"`
}

// Normalized structures for provider response
type Track struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	ArtistID       string `json:"artist_id,omitempty"`
	Artist         string `json:"artist"`
	AlbumID        string `json:"album_id,omitempty"`
	Album          string `json:"album"`
	AlbumArtist    string `json:"album_artist,omitempty"`
	TrackNumber    int    `json:"track_number"`
	DiscNumber     int    `json:"disc_number,omitempty"`
	TotalTracks    int    `json:"total_tracks,omitempty"`
	TotalDiscs     int    `json:"total_discs,omitempty"`
	Duration       int    `json:"duration"` // seconds
	Year           int    `json:"year,omitempty"`
	Genre          string `json:"genre,omitempty"`
	Label          string `json:"label,omitempty"`
	ISRC           string `json:"isrc,omitempty"`
	Copyright      string `json:"copyright,omitempty"`
	Composer       string `json:"composer,omitempty"`
	AlbumArtURL    string `json:"album_art_url,omitempty"`
	ExplicitLyrics bool   `json:"explicit_lyrics,omitempty"`
	// Additional metadata fields
	BPM          int     `json:"bpm,omitempty"`
	Key          string  `json:"key,omitempty"`
	KeyScale     string  `json:"key_scale,omitempty"`
	ReplayGain   float64 `json:"replay_gain,omitempty"`
	Peak         float64 `json:"peak,omitempty"`
	Version      string  `json:"version,omitempty"`
	Description  string  `json:"description,omitempty"`
	URL          string  `json:"url,omitempty"`
	AudioQuality string  `json:"audio_quality,omitempty"`
	AudioModes   string  `json:"audio_modes,omitempty"`
	Lyrics       string  `json:"lyrics,omitempty"`
	Subtitles    string  `json:"subtitles,omitempty"`
	ReleaseDate  string  `json:"release_date,omitempty"` // Full date YYYY-MM-DD
}

type Album struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	ArtistID    string  `json:"artist_id,omitempty"`
	Artist      string  `json:"artist"`
	Year        int     `json:"year,omitempty"`
	ReleaseDate string  `json:"release_date,omitempty"` // Full date YYYY-MM-DD
	Genre       string  `json:"genre,omitempty"`
	Label       string  `json:"label,omitempty"`
	Copyright   string  `json:"copyright,omitempty"`
	TotalTracks int     `json:"total_tracks,omitempty"`
	TotalDiscs  int     `json:"total_discs,omitempty"`
	AlbumArtURL string  `json:"album_art_url,omitempty"`
	Tracks      []Track `json:"tracks"`
	// Additional metadata fields
	UPC       string `json:"upc,omitempty"`
	AlbumType string `json:"album_type,omitempty"` // ALBUM, EP, SINGLE
	URL       string `json:"url,omitempty"`
	Explicit  bool   `json:"explicit,omitempty"`
}

type Artist struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	PictureURL string  `json:"picture_url,omitempty"`
	Albums     []Album `json:"albums,omitempty"`
	TopTracks  []Track `json:"top_tracks,omitempty"`
}

type Playlist struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	ImageURL    string  `json:"image_url,omitempty"`
	Tracks      []Track `json:"tracks"`
}

type SearchResult struct {
	Artists   []Artist   `json:"artists"`
	Albums    []Album    `json:"albums"`
	Playlists []Playlist `json:"playlists"`
	Tracks    []Track    `json:"tracks"`
}
