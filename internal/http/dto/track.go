package dto

type TrackUpdate struct {
	Title         *string `form:"title"`
	Artist        *string `form:"artist"`
	Album         *string `form:"album"`
	AlbumArtist   *string `form:"album_artist"`
	Genre         *string `form:"genre"`
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
