package catalog

import (
	"encoding/json"
)

type APIMediaMetadata struct {
	Tags []string `json:"tags"`
}

func resolveAudioQuality(audioQuality string, tags []string) string {
	for _, tag := range tags {
		if tag == "HIRES_LOSSLESS" || tag == "HI_RES_LOSSLESS" {
			return "HI_RES_LOSSLESS"
		}
	}
	return audioQuality
}

type APIArtist struct {
	ID   json.Number `json:"id"`
	Name string      `json:"name"`
}

type APIAlbumStub struct {
	ID    json.Number `json:"id"`
	Title string      `json:"title"`
	Cover FlexCover   `json:"cover"`
}

type APIArtistWithPicture struct {
	ID      json.Number `json:"id"`
	Name    string      `json:"name"`
	Picture string      `json:"picture"`
}

type APITrackItem struct {
	ID          json.Number  `json:"id"`
	Title       string       `json:"title"`
	Artists     []APIArtist  `json:"artists"`
	Album       APIAlbumStub `json:"album"`
	TrackNumber int          `json:"trackNumber"`
	Duration    int          `json:"duration"`
	Explicit    bool         `json:"explicit"`
}

type APIAlbumWithTracks struct {
	Artist        APIArtist        `json:"artist"`
	Copyright     string           `json:"copyright"`
	UPC           string           `json:"upc"`
	ID            json.Number      `json:"id"`
	Title         string           `json:"title"`
	ReleaseDate   string           `json:"releaseDate"`
	Label         string           `json:"label"`
	Type          string           `json:"type"`
	Genre         string           `json:"genre"`
	URL           string           `json:"url"`
	AudioQuality  string           `json:"audioQuality"`
	MediaMetadata APIMediaMetadata `json:"mediaMetadata"`
	Cover         FlexCover        `json:"cover"`
	Items         []struct {
		Item APIAlbumTrackItem `json:"item"`
	} `json:"items"`
	NumberOfTracks  int  `json:"numberOfTracks"`
	NumberOfVolumes int  `json:"numberOfVolumes"`
	Explicit        bool `json:"explicit"`
}

type APIAlbumTrackItem struct {
	Version       *string          `json:"version"`
	URL           string           `json:"url"`
	Title         string           `json:"title"`
	AudioQuality  string           `json:"audioQuality"`
	MediaMetadata APIMediaMetadata `json:"mediaMetadata"`
	ISRC          string           `json:"isrc"`
	Key           string           `json:"key"`
	ID            json.Number      `json:"id"`
	KeyScale      string           `json:"keyScale"`
	Artists       []APIArtist      `json:"artists"`
	VolumeNumber  int              `json:"volumeNumber"`
	Peak          float64          `json:"peak"`
	ReplayGain    float64          `json:"replayGain"`
	Duration      int              `json:"duration"`
	BPM           int              `json:"bpm"`
	TrackNumber   int              `json:"trackNumber"`
	Explicit      bool             `json:"explicit"`
}

type APIPlaylistItem struct {
	Item struct {
		ID    json.Number `json:"id"`
		Title string      `json:"title"`
		ISRC  string      `json:"isrc"`
		Album struct {
			ID    json.Number `json:"id"`
			Title string      `json:"title"`
			Cover FlexCover   `json:"cover"`
		} `json:"album"`
		Artists     []APIArtist `json:"artists"`
		TrackNumber int         `json:"trackNumber"`
		Duration    int         `json:"duration"`
		Explicit    bool        `json:"explicit"`
	} `json:"item"`
}

type APIPlaylist struct {
	Uuid        string `json:"uuid"`
	Title       string `json:"title"`
	Description string `json:"description"`
	SquareImage string `json:"squareImage"`
}

type APIPlaylistResponse struct {
	Playlist APIPlaylist       `json:"playlist"`
	Items    []APIPlaylistItem `json:"items"`
}

type APIAlbumResponse struct {
	Data APIAlbumWithTracks `json:"data"`
}

type APIArtistResponse struct {
	Artist APIArtistWithPicture `json:"artist"`
}

type APIArtistAggregationResponse struct {
	Albums struct {
		Items []struct {
			ID            json.Number      `json:"id"`
			Title         string           `json:"title"`
			Cover         string           `json:"cover"`
			AudioQuality  string           `json:"audioQuality"`
			MediaMetadata APIMediaMetadata `json:"mediaMetadata"`
		} `json:"items"`
	} `json:"albums"`
	Tracks []struct {
		Album struct {
			ID    json.Number `json:"id"`
			Title string      `json:"title"`
			Cover string      `json:"cover"`
		} `json:"album"`
		Artist struct {
			ID   json.Number `json:"id"`
			Name string      `json:"name"`
		} `json:"artist"`
		ID            json.Number      `json:"id"`
		Title         string           `json:"title"`
		AudioQuality  string           `json:"audioQuality"`
		MediaMetadata APIMediaMetadata `json:"mediaMetadata"`
		TrackNumber   int              `json:"trackNumber"`
		Duration      int              `json:"duration"`
	} `json:"tracks"`
}

type APITrackInfoResponse struct {
	Data APITrackInfoData `json:"data"`
}

type APITrackInfoData struct {
	Version         *string          `json:"version"`
	Artist          APIArtist        `json:"artist"`
	Copyright       string           `json:"copyright"`
	StreamStartDate string           `json:"streamStartDate"`
	URL             string           `json:"url"`
	KeyScale        string           `json:"keyScale"`
	ID              json.Number      `json:"id"`
	Title           string           `json:"title"`
	ISRC            string           `json:"isrc"`
	AudioQuality    string           `json:"audioQuality"`
	MediaMetadata   APIMediaMetadata `json:"mediaMetadata"`
	Key             string           `json:"key"`
	Artists         []APIArtist      `json:"artists"`
	AudioModes      []string         `json:"audioModes"`
	Album           struct {
		ID              json.Number `json:"id"`
		Title           string      `json:"title"`
		ReleaseDate     string      `json:"releaseDate"`
		UPC             string      `json:"upc"`
		Label           string      `json:"label"`
		Genre           string      `json:"genre"`
		Cover           FlexCover   `json:"cover"`
		NumberOfTracks  int         `json:"numberOfTracks"`
		NumberOfVolumes int         `json:"numberOfVolumes"`
	} `json:"album"`
	BPM          int     `json:"bpm"`
	Peak         float64 `json:"peak"`
	ReplayGain   float64 `json:"replayGain"`
	Duration     int     `json:"duration"`
	VolumeNumber int     `json:"volumeNumber"`
	TrackNumber  int     `json:"trackNumber"`
	Explicit     bool    `json:"explicit"`
}

type APISimilarAlbum struct {
	Cover     string   `json:"cover"`
	Title     string   `json:"title"`
	MediaTags []string `json:"mediaTags"`
	Artists   []struct {
		Name string `json:"name"`
		ID   int    `json:"id"`
	} `json:"artists"`
	ID int `json:"id"`
}

type APISimilarAlbumsResponse struct {
	Albums []APISimilarAlbum `json:"albums"`
}

type APIStreamResponse struct {
	Data struct {
		Manifest         string `json:"manifest"`
		ManifestMimeType string `json:"manifestMimeType"`
	} `json:"data"`
}

type APILyricsResponse struct {
	Lyrics struct {
		Lyrics    string `json:"lyrics"`
		Subtitles string `json:"subtitles"`
		Provider  string `json:"lyricsProvider"`
	} `json:"lyrics"`
}

// Search responses
type APISearchArtistItem struct {
	ID      json.Number `json:"id"`
	Name    string      `json:"name"`
	Picture string      `json:"picture"`
}

type APISearchAlbumItem struct {
	ID            json.Number      `json:"id"`
	Title         string           `json:"title"`
	Cover         string           `json:"cover"`
	AudioQuality  string           `json:"audioQuality"`
	MediaMetadata APIMediaMetadata `json:"mediaMetadata"`
	Artists       []struct {
		Name string `json:"name"`
	} `json:"artists"`
}

type APISearchTrackItem struct {
	Album struct {
		ID    json.Number `json:"id"`
		Title string      `json:"title"`
		Cover string      `json:"cover"`
	} `json:"album"`
	ID            json.Number      `json:"id"`
	Title         string           `json:"title"`
	AudioQuality  string           `json:"audioQuality"`
	MediaMetadata APIMediaMetadata `json:"mediaMetadata"`
	Artists       []APIArtist      `json:"artists"`
	Duration      int              `json:"duration"`
	TrackNumber   int              `json:"trackNumber"`
}

type APISearchPlaylistItem struct {
	Uuid        string `json:"uuid"`
	Title       string `json:"title"`
	SquareImage string `json:"squareImage"`
}

type APISearchResponse struct {
	Data struct {
		Artists struct {
			Items []APISearchArtistItem `json:"items"`
		} `json:"artists"`
		Albums struct {
			Items []APISearchAlbumItem `json:"items"`
		} `json:"albums"`
		Items     []APISearchTrackItem `json:"items"`
		Playlists struct {
			Items []APISearchPlaylistItem `json:"items"`
		} `json:"playlists"`
	} `json:"data"`
}

type APIArtistsSearchResponse struct {
	Data struct {
		Artists struct {
			Items []APISearchArtistItem `json:"items"`
		} `json:"artists"`
	} `json:"data"`
}

type APIAlbumsSearchResponse struct {
	Data struct {
		Albums struct {
			Items []APISearchAlbumItem `json:"items"`
		} `json:"albums"`
	} `json:"data"`
}

type APITracksSearchResponse struct {
	Data struct {
		Items []APISearchTrackItem `json:"items"`
	} `json:"data"`
}

type APIPlaylistsSearchResponse struct {
	Data struct {
		Playlists struct {
			Items []APISearchPlaylistItem `json:"items"`
		} `json:"playlists"`
	} `json:"data"`
}
