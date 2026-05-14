package catalog

// Response wrappers
type QobuzSearchResponse struct {
	Success bool            `json:"success"`
	Data    QobuzSearchData `json:"data"`
}

type QobuzArtistResponse struct {
	Success bool            `json:"success"`
	Data    QobuzArtistData `json:"data"`
}

type QobuzAlbumDataResponse struct {
	Success bool                `json:"success"`
	Data    *QobuzAlbumResponse `json:"data"`
}

type QobuzAlbumResponse struct {
	ID                  string               `json:"id"`
	Title               string               `json:"title"`
	QobuzID             int                  `json:"qobuz_id"`
	UPC                 string               `json:"upc"`
	TracksCount         int                  `json:"tracks_count"`
	MediaCount          int                  `json:"media_count"`
	ParentalWarning     bool                 `json:"parental_warning"`
	Copyright           string               `json:"copyright"`
	ReleaseDateOriginal string               `json:"release_date_original"`
	Image               QobuzImage           `json:"image"`
	Artist              QobuzArtistRef       `json:"artist"`
	Artists             []QobuzAlbumArtist   `json:"artists"`
	Label               QobuzLabel           `json:"label"`
	Genre               QobuzGenre           `json:"genre"`
	Tracks              QobuzTracksContainer `json:"tracks"`
}

type QobuzTrackDataResponse struct {
	Success bool                `json:"success"`
	Data    *QobuzTrackResponse `json:"data"`
}

type QobuzTrackResponse struct {
	MaximumBitDepth     int              `json:"maximum_bit_depth"`
	Copyright           string           `json:"copyright"`
	Performers          string           `json:"performers"`
	AudioInfo           *QobuzReplayGain `json:"audio_info"`
	Performer           QobuzPerformer   `json:"performer"`
	Album               *QobuzTrackAlbum `json:"album"`
	Work                interface{}      `json:"work"`
	Composer            *QobuzComposer   `json:"composer"`
	ISRC                string           `json:"isrc"`
	Title               string           `json:"title"`
	Version             *string          `json:"version"`
	Duration            int              `json:"duration"`
	ParentalWarning     bool             `json:"parental_warning"`
	TrackNumber         int              `json:"track_number"`
	MaximumChannelCount int              `json:"maximum_channel_count"`
	ID                  int              `json:"id"`
	MediaNumber         int              `json:"media_number"`
	MaximumSamplingRate float64          `json:"maximum_sampling_rate"`
	ReleaseDateOriginal string           `json:"release_date_original"`
	Purchasable         bool             `json:"purchasable"`
	Streamable          bool             `json:"streamable"`
	Previewable         bool             `json:"previewable"`
	Sampleable          bool             `json:"sampleable"`
	Downloadable        bool             `json:"downloadable"`
	Displayable         bool             `json:"displayable"`
	Hires               bool             `json:"hires"`
	HiresStreamable     bool             `json:"hires_streamable"`
}

type QobuzTrackLookupData struct {
	ID int `json:"id"`
}

type QobuzTrackLookupResponse struct {
	Success bool                  `json:"success"`
	Data    *QobuzTrackLookupData `json:"data"`
}

type QobuzDownloadData struct {
	URL string `json:"url"`
}

type QobuzDownloadResponse struct {
	Success bool               `json:"success"`
	Data    *QobuzDownloadData `json:"data"`
}

// Shared types
type QobuzImage struct {
	Small     string `json:"small"`
	Thumbnail string `json:"thumbnail"`
	Large     string `json:"large"`
}

type QobuzLabel struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type QobuzGenre struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
	Path []int  `json:"path"`
}

type QobuzComposer struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type QobuzArtistRef struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
	Slug string `json:"slug"`
}

type QobuzAlbumArtist struct {
	ID    int      `json:"id"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

type QobuzPerformer struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type QobuzReplayGain struct {
	ReplayGainTrackPeak float64 `json:"replaygain_track_peak"`
	ReplayGainTrackGain float64 `json:"replaygain_track_gain"`
}

type QobuzTracksContainer struct {
	Total  int              `json:"total"`
	Offset int              `json:"offset"`
	Limit  int              `json:"limit"`
	Items  []QobuzTrackItem `json:"items"`
}

type QobuzTrackItem struct {
	MaximumBitDepth     int              `json:"maximum_bit_depth"`
	Copyright           string           `json:"copyright"`
	Performers          string           `json:"performers"`
	AudioInfo           *QobuzReplayGain `json:"audio_info"`
	Performer           QobuzPerformer   `json:"performer"`
	Album               *QobuzTrackAlbum `json:"album"`
	Work                interface{}      `json:"work"`
	Composer            *QobuzComposer   `json:"composer"`
	ISRC                string           `json:"isrc"`
	Title               string           `json:"title"`
	Version             *string          `json:"version"`
	Duration            int              `json:"duration"`
	ParentalWarning     bool             `json:"parental_warning"`
	TrackNumber         int              `json:"track_number"`
	MaximumChannelCount int              `json:"maximum_channel_count"`
	ID                  int              `json:"id"`
	MediaNumber         int              `json:"media_number"`
	MaximumSamplingRate float64          `json:"maximum_sampling_rate"`
	ReleaseDateOriginal string           `json:"release_date_original"`
	Purchasable         bool             `json:"purchasable"`
	Streamable          bool             `json:"streamable"`
	Previewable         bool             `json:"previewable"`
	Sampleable          bool             `json:"sampleable"`
	Downloadable        bool             `json:"downloadable"`
	Displayable         bool             `json:"displayable"`
	Hires               bool             `json:"hires"`
	HiresStreamable     bool             `json:"hires_streamable"`
}

type QobuzTrackAlbum struct {
	ID                  string         `json:"id"`
	Title               string         `json:"title"`
	QobuzID             int            `json:"qobuz_id"`
	Artist              QobuzArtistRef `json:"artist"`
	Genre               QobuzGenre     `json:"genre"`
	Image               QobuzImage     `json:"image"`
	MaximumBitDepth     int            `json:"maximum_bit_depth"`
	MaximumSamplingRate float64        `json:"maximum_sampling_rate"`
}

// Artist response types
type QobuzArtistData struct {
	Artist QobuzArtistFull `json:"artist"`
}

type QobuzArtistFull struct {
	ID             int                 `json:"id"`
	Name           QobuzNameObject     `json:"name"`
	Biography      *QobuzBiography     `json:"biography"`
	Images         QobuzArtistImages   `json:"images"`
	Albums         QobuzArtistAlbums   `json:"albums"`
	TopTracks      []QobuzTopTrackItem `json:"top_tracks"`
	SimilarArtists QobuzSimilarArtists `json:"similar_artists"`
}

type QobuzNameObject struct {
	Display string `json:"display"`
}

type QobuzBiography struct {
	Content  string `json:"content"`
	Language string `json:"language"`
}

type QobuzArtistImages struct {
	Portrait *QobuzImageHash `json:"portrait"`
}

type QobuzImageHash struct {
	Hash   string `json:"hash"`
	Format string `json:"format"`
}

type QobuzArtistAlbums struct {
	Items []QobuzArtistAlbumItem `json:"items"`
}

type QobuzArtistAlbumItem struct {
	ID                  string     `json:"id"`
	Title               string     `json:"title"`
	Image               QobuzImage `json:"image"`
	Genre               QobuzGenre `json:"genre"`
	ReleaseDateOriginal string     `json:"release_date_original"`
	MaximumBitDepth     int        `json:"maximum_bit_depth"`
}

type QobuzTopTrackItem struct {
	ID              int                  `json:"id"`
	ISRC            string               `json:"isrc"`
	Title           string               `json:"title"`
	Work            interface{}          `json:"work"`
	Version         *string              `json:"version"`
	Duration        int                  `json:"duration"`
	ParentalWarning bool                 `json:"parental_warning"`
	Composer        QobuzNameObject      `json:"composer"`
	Artist          QobuzNameObject      `json:"artist"`
	Artists         []interface{}        `json:"artists"`
	AudioInfo       QobuzTechAudioInfo   `json:"audio_info"`
	Rights          QobuzTrackRights     `json:"rights"`
	PhysicalSupport QobuzPhysicalSupport `json:"physical_support"`
	Album           *QobuzTopTrackAlbum  `json:"album"`
}

type QobuzPhysicalSupport struct {
	MediaNumber int `json:"media_number"`
	TrackNumber int `json:"track_number"`
}

type QobuzTrackRights struct {
	Streamable       bool `json:"streamable"`
	HiresStreamable  bool `json:"hires_streamable"`
	HiresPurchasable bool `json:"hires_purchasable"`
	Purchasable      bool `json:"purchasable"`
	Downloadable     bool `json:"downloadable"`
	Previewable      bool `json:"previewable"`
	Sampleable       bool `json:"sampleable"`
}

type QobuzTechAudioInfo struct {
	MaximumBitDepth     int     `json:"maximum_bit_depth"`
	MaximumChannelCount int     `json:"maximum_channel_count"`
	MaximumSamplingRate float64 `json:"maximum_sampling_rate"`
}

type QobuzTopTrackAlbum struct {
	ID    string     `json:"id"`
	Title string     `json:"title"`
	Image QobuzImage `json:"image"`
	Label QobuzLabel `json:"label"`
	Genre QobuzGenre `json:"genre"`
}

type QobuzSimilarArtists struct {
	HasMore bool                     `json:"has_more"`
	Items   []QobuzSimilarArtistItem `json:"items"`
}

type QobuzSimilarArtistItem struct {
	ID     int               `json:"id"`
	Name   QobuzNameObject   `json:"name"`
	Images QobuzArtistImages `json:"images"`
}

// Search response types
type QobuzSearchData struct {
	Query     string               `json:"query"`
	Albums    QobuzSearchAlbums    `json:"albums"`
	Tracks    QobuzSearchTracks    `json:"tracks"`
	Artists   QobuzSearchArtists   `json:"artists"`
	Playlists QobuzSearchPlaylists `json:"playlists"`
}

type QobuzSearchAlbums struct {
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
	Total  int                    `json:"total"`
	Items  []QobuzSearchAlbumItem `json:"items"`
}

type QobuzSearchAlbumItem struct {
	MaximumBitDepth     int                `json:"maximum_bit_depth"`
	Image               QobuzImage         `json:"image"`
	MediaCount          int                `json:"media_count"`
	Artist              QobuzArtistRef     `json:"artist"`
	Artists             []QobuzAlbumArtist `json:"artists"`
	UPC                 string             `json:"upc"`
	ReleasedAt          int                `json:"released_at"`
	Label               QobuzLabel         `json:"label"`
	Title               string             `json:"title"`
	QobuzID             int                `json:"qobuz_id"`
	Version             *string            `json:"version"`
	URL                 string             `json:"url"`
	Slug                string             `json:"slug"`
	Duration            int                `json:"duration"`
	ParentalWarning     bool               `json:"parental_warning"`
	Popularity          int                `json:"popularity"`
	TracksCount         int                `json:"tracks_count"`
	Genre               QobuzGenre         `json:"genre"`
	MaximumChannelCount int                `json:"maximum_channel_count"`
	ID                  string             `json:"id"`
	MaximumSamplingRate float64            `json:"maximum_sampling_rate"`
	ReleaseDateOriginal string             `json:"release_date_original"`
	Streamable          bool               `json:"streamable"`
	Hires               bool               `json:"hires"`
	HiresStreamable     bool               `json:"hires_streamable"`
}

type QobuzSearchTracks struct {
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
	Total  int              `json:"total"`
	Items  []QobuzTrackItem `json:"items"`
}

type QobuzSearchArtists struct {
	Limit  int                     `json:"limit"`
	Offset int                     `json:"offset"`
	Total  int                     `json:"total"`
	Items  []QobuzSearchArtistItem `json:"items"`
}

type QobuzSearchArtistItem struct {
	ID    int             `json:"id"`
	Name  string          `json:"name"`
	Image *QobuzImageHash `json:"image"`
}

type QobuzSearchPlaylists struct {
	Limit  int                       `json:"limit"`
	Offset int                       `json:"offset"`
	Total  int                       `json:"total"`
	Items  []QobuzSearchPlaylistItem `json:"items"`
}

type QobuzSearchPlaylistItem struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Image       QobuzImage `json:"image"`
}
