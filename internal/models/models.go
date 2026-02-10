package models

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

type JobItemStatus string

const (
	JobItemStatusPending     JobItemStatus = "pending"
	JobItemStatusDownloading JobItemStatus = "downloading"
	JobItemStatusCompelted   JobItemStatus = "completed"
	JobItemStatusFailed      JobItemStatus = "failed"
)

type JobItem struct {
	ID       int64         `json:"id"` // SQLite ID
	JobID    string        `json:"job_id"`
	TrackID  string        `json:"track_id"`
	Title    string        `json:"title"`
	Status   JobItemStatus `json:"status"`
	Progress float64       `json:"progress"`
	FilePath string        `json:"file_path,omitempty"`
}

type Download struct {
	ProviderID  string    `json:"provider_id"`
	FilePath    string    `json:"file_path"` // Absolute path
	CompletedAt time.Time `json:"completed_at"`
}

// Normalized structures for provider response
type Track struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	TrackNumber int    `json:"track_number"`
	Duration    int    `json:"duration"` // seconds
}

type Album struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Artist string  `json:"artist"`
	Tracks []Track `json:"tracks"`
}

type Artist struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Albums    []Album `json:"albums,omitempty"`
	TopTracks []Track `json:"top_tracks,omitempty"`
}

type Playlist struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Tracks []Track `json:"tracks"`
}

type SearchResult struct {
	Artists   []Artist   `json:"artists"`
	Albums    []Album    `json:"albums"`
	Playlists []Playlist `json:"playlists"`
	Tracks    []Track    `json:"tracks"`
}
