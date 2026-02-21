// Package constants contains application-wide constants to avoid magic numbers and strings.
package constants

import "time"

// Application defaults
const (
	DefaultPort           = "8080"
	DefaultDBPath         = "navidrums.db"
	DefaultQuality        = "LOSSLESS"
	DefaultProviderURL    = "http://127.0.0.1:8000"
	DefaultConcurrency    = 2
	DefaultPollInterval   = 2 * time.Second
	DefaultHTTPTimeout    = 5 * time.Minute
	ImageHTTPTimeout      = 30 * time.Second
	DefaultRetryCount     = 3
	DefaultRetryBase      = 1 * time.Second
	DefaultUsername       = "navidrums"
	DefaultSubdirTemplate = "{{.AlbumArtist}}/{{.OriginalYear}} - {{.Album}}/{{.Disc}}-{{.Track}} {{.Title}}"
	DefaultCacheTTL       = 12 * time.Hour
)

// Quality levels
const (
	QualityLossless      = "LOSSLESS"
	QualityHiResLossless = "HI_RES_LOSSLESS"
	QualityHigh          = "HIGH"
	QualityLow           = "LOW"
)

// Image sizes
const (
	ImageSizeSmall  = "320x320"
	ImageSizeMedium = "640x640"
	ImageSizeLarge  = "1280x1280"
)

// Tidal CDN URLs
const (
	TidalImageExt = ".jpg"
)

// MIME Types
const (
	MimeTypeBTS     = "application/vnd.tidal.bts"
	MimeTypeDashXML = "application/dash+xml"
	MimeTypeFLAC    = "audio/flac"
	MimeTypeMP3     = "audio/mpeg"
	MimeTypeMP4     = "audio/mp4"
	MimeTypeJPEG    = "image/jpeg"
)

// Database
const (
	JobsTable      = "jobs"
	DownloadsTable = "downloads"
	CacheTable     = "cache"
)

// File Extensions
const (
	ExtFLAC = ".flac"
	ExtMP3  = ".mp3"
	ExtMP4  = ".mp4"
	ExtM4A  = ".m4a"
	ExtM3U  = ".m3u"
	ExtJPG  = ".jpg"
)

// File Names
const (
	PlaylistsDir = "playlists"
)

// File Permissions
const (
	DirPermissions  = 0755
	FilePermissions = 0644
)

// HTTP Status Codes
const (
	StatusOK                 = 200
	StatusBadRequest         = 400
	StatusNotFound           = 404
	StatusInternalError      = 500
	StatusServiceUnavailable = 503
)

// UI/UX
const (
	MaxHistoryItems     = 20
	MaxSearchResults    = 50
	ProgressUpdateFreq  = 2 * time.Second
	ProgressUpdateBytes = 1024 * 1024 // 1MB
)

// Characters to sanitize from filesystem paths
const InvalidPathChars = "<>:\"/\\|?*"
