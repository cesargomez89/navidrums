package catalog

import (
	"encoding/json"
	"testing"
)

func TestAPIArtistWithPicture_ToDomain(t *testing.T) {
	p := &HifiProvider{}
	apiArtist := APIArtistWithPicture{
		ID:      json.Number("123"),
		Name:    "Artist Name",
		Picture: "img-id",
	}

	result := apiArtist.ToDomain(p)

	if result.ID != "123" {
		t.Errorf("Expected ID 123, got %s", result.ID)
	}
	if result.Name != "Artist Name" {
		t.Errorf("Expected Name 'Artist Name', got %s", result.Name)
	}
	expectedURL := "https://resources.tidal.com/images/img/id/320x320.jpg"
	if result.PictureURL != expectedURL {
		t.Errorf("Expected PictureURL %s, got %s", expectedURL, result.PictureURL)
	}
}

func TestAPIArtistAggregationResponse_ToAlbums(t *testing.T) {
	p := &HifiProvider{}
	resp := APIArtistAggregationResponse{}
	resp.Albums.Items = []struct {
		ID           json.Number "json:\"id\""
		Title        string      "json:\"title\""
		Cover        string      "json:\"cover\""
		AudioQuality string      "json:\"audioQuality\""
	}{
		{ID: json.Number("1"), Title: "Album 1", Cover: "cover-1", AudioQuality: "LOSSLESS"},
	}

	albums := resp.ToAlbums("ArtistName", p)

	if len(albums) != 1 {
		t.Fatalf("Expected 1 album, got %d", len(albums))
	}
	if albums[0].Title != "Album 1" {
		t.Errorf("Expected Title 'Album 1', got %s", albums[0].Title)
	}
	if albums[0].Artist != "ArtistName" {
		t.Errorf("Expected Artist 'ArtistName', got %s", albums[0].Artist)
	}
}

func TestAPIArtistAggregationResponse_ToTopTracks(t *testing.T) {
	p := &HifiProvider{}
	resp := APIArtistAggregationResponse{}
	resp.Tracks = []struct {
		Album struct {
			ID    json.Number "json:\"id\""
			Title string      "json:\"title\""
			Cover string      "json:\"cover\""
		} "json:\"album\""
		Artist struct {
			ID   json.Number "json:\"id\""
			Name string      "json:\"name\""
		} "json:\"artist\""
		ID           json.Number "json:\"id\""
		Title        string      "json:\"title\""
		AudioQuality string      "json:\"audioQuality\""
		TrackNumber  int         "json:\"trackNumber\""
		Duration     int         "json:\"duration\""
	}{
		{
			ID:           json.Number("101"),
			Title:        "Track 1",
			TrackNumber:  1,
			Duration:     200,
			AudioQuality: "HIGH",
			Artist: struct {
				ID   json.Number "json:\"id\""
				Name string      "json:\"name\""
			}{ID: json.Number("1"), Name: "Artist"},
			Album: struct {
				ID    json.Number "json:\"id\""
				Title string      "json:\"title\""
				Cover string      "json:\"cover\""
			}{ID: json.Number("201"), Title: "Album", Cover: "cover-id"},
		},
	}

	tracks := resp.ToTopTracks(p)

	if len(tracks) != 1 {
		t.Fatalf("Expected 1 track, got %d", len(tracks))
	}
	if tracks[0].Title != "Track 1" {
		t.Errorf("Expected Title 'Track 1', got %s", tracks[0].Title)
	}
	if tracks[0].Artist != "Artist" {
		t.Errorf("Expected Artist 'Artist', got %s", tracks[0].Artist)
	}
	if tracks[0].Album != "Album" {
		t.Errorf("Expected Album 'Album', got %s", tracks[0].Album)
	}
}

func TestAPIAlbumResponse_ToDomain(t *testing.T) {
	p := &HifiProvider{}
	resp := APIAlbumResponse{
		Data: APIAlbumWithTracks{
			ID:          json.Number("1"),
			Title:       "Album Title",
			ReleaseDate: "2023-01-01",
			Artist: APIArtist{
				ID:   json.Number("10"),
				Name: "Artist Name",
			},
			Cover: FlexCover{"cover-id"},
			Items: []struct {
				Item APIAlbumTrackItem "json:\"item\""
			}{
				{
					Item: APIAlbumTrackItem{
						ID:          json.Number("101"),
						Title:       "Track 1",
						TrackNumber: 1,
						Duration:    180,
					},
				},
			},
			NumberOfTracks: 1,
		},
	}

	album := resp.ToDomain(p)

	if album.Title != "Album Title" {
		t.Errorf("Expected Title 'Album Title', got %s", album.Title)
	}
	if album.Year != 2023 {
		t.Errorf("Expected Year 2023, got %d", album.Year)
	}
	if len(album.Tracks) != 1 {
		t.Fatalf("Expected 1 track, got %d", len(album.Tracks))
	}
	if album.Tracks[0].Title != "Track 1" {
		t.Errorf("Expected Track Title 'Track 1', got %s", album.Tracks[0].Title)
	}
}

func TestAPITrackInfoResponse_ToDomain(t *testing.T) {
	p := &HifiProvider{}
	resp := APITrackInfoResponse{
		Data: APITrackInfoData{
			ID:    json.Number("101"),
			Title: "Track Title",
			Artist: APIArtist{
				ID:   json.Number("1"),
				Name: "Artist Name",
			},
			Album: struct {
				ID              json.Number "json:\"id\""
				Title           string      "json:\"title\""
				ReleaseDate     string      "json:\"releaseDate\""
				UPC             string      "json:\"upc\""
				Label           string      "json:\"label\""
				Genre           string      "json:\"genre\""
				Cover           FlexCover   "json:\"cover\""
				NumberOfTracks  int         "json:\"numberOfTracks\""
				NumberOfVolumes int         "json:\"numberOfVolumes\""
			}{
				ID:          json.Number("201"),
				Title:       "Album Title",
				ReleaseDate: "2023-05-15",
			},
			Duration:    210,
			TrackNumber: 2,
			AudioModes:  []string{"STEREO"},
		},
	}

	track := resp.ToDomain(p)

	if track.Title != "Track Title" {
		t.Errorf("Expected Title 'Track Title', got %s", track.Title)
	}
	if track.Artist != "Artist Name" {
		t.Errorf("Expected Artist 'Artist Name', got %s", track.Artist)
	}
	if track.Album != "Album Title" {
		t.Errorf("Expected Album 'Album Title', got %s", track.Album)
	}
	if track.Year != 2023 {
		t.Errorf("Expected Year 2023, got %d", track.Year)
	}
	if track.AudioModes != "STEREO" {
		t.Errorf("Expected AudioModes 'STEREO', got %s", track.AudioModes)
	}
}

func TestAPIPlaylistResponse_ToDomain(t *testing.T) {
	p := &HifiProvider{}
	resp := APIPlaylistResponse{
		Playlist: APIPlaylist{
			Uuid:  "uuid-123",
			Title: "My Playlist",
		},
		Items: []APIPlaylistItem{
			{
				Item: struct {
					ID    json.Number "json:\"id\""
					Title string      "json:\"title\""
					ISRC  string      "json:\"isrc\""
					Album struct {
						ID    json.Number "json:\"id\""
						Title string      "json:\"title\""
						Cover FlexCover   "json:\"cover\""
					} "json:\"album\""
					Artists     []APIArtist "json:\"artists\""
					TrackNumber int         "json:\"trackNumber\""
					Duration    int         "json:\"duration\""
					Explicit    bool        "json:\"explicit\""
				}{
					ID:    json.Number("101"),
					Title: "Playlist Track",
					Artists: []APIArtist{
						{ID: json.Number("1"), Name: "Artist"},
					},
					Duration: 180,
				},
			},
		},
	}

	playlist := resp.ToDomain(p)

	if playlist.ID != "uuid-123" {
		t.Errorf("Expected ID 'uuid-123', got %s", playlist.ID)
	}
	if len(playlist.Tracks) != 1 {
		t.Fatalf("Expected 1 track, got %d", len(playlist.Tracks))
	}
	if playlist.Tracks[0].Title != "Playlist Track" {
		t.Errorf("Expected Track Title 'Playlist Track', got %s", playlist.Tracks[0].Title)
	}
}

func TestAPISearchResponses_ToDomain(t *testing.T) {
	p := &HifiProvider{}

	t.Run("Artists", func(t *testing.T) {
		resp := APIArtistsSearchResponse{}
		resp.Data.Artists.Items = []APISearchArtistItem{
			{ID: json.Number("1"), Name: "Artist 1", Picture: "pic"},
		}
		result := resp.ToDomain(p)
		if len(result) != 1 || result[0].Name != "Artist 1" {
			t.Errorf("Artist search conversion failed")
		}
	})

	t.Run("Albums", func(t *testing.T) {
		resp := APIAlbumsSearchResponse{}
		resp.Data.Albums.Items = []APISearchAlbumItem{
			{ID: json.Number("1"), Title: "Album 1", Cover: "cover"},
		}
		result := resp.ToDomain(p)
		if len(result) != 1 || result[0].Title != "Album 1" {
			t.Errorf("Album search conversion failed")
		}
	})

	t.Run("Tracks", func(t *testing.T) {
		resp := APITracksSearchResponse{}
		resp.Data.Items = []APISearchTrackItem{
			{ID: json.Number("1"), Title: "Track 1"},
		}
		result := resp.ToDomain(p)
		if len(result) != 1 || result[0].Title != "Track 1" {
			t.Errorf("Track search conversion failed")
		}
	})
}
