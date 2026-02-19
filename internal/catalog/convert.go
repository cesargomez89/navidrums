package catalog

import (
	"fmt"

	"github.com/cesargomez89/navidrums/internal/domain"
)

func (r APIArtist) ToDomain(p *HifiProvider) *domain.Artist {
	return nil
}

func (r APIArtistWithPicture) ToDomain(p *HifiProvider) *domain.Artist {
	return &domain.Artist{
		ID:         formatID(r.ID),
		Name:       r.Name,
		PictureURL: p.ensureAbsoluteURL(r.Picture, "320x320"),
	}
}

func (r APIArtistAggregationResponse) ToAlbums(artistName string, p *HifiProvider) []domain.Album {
	var albums []domain.Album
	for _, item := range r.Albums.Items {
		albums = append(albums, domain.Album{
			ID:          formatID(item.ID),
			Title:       item.Title,
			Artist:      artistName,
			AlbumArtURL: p.ensureAbsoluteURL(item.Cover, "640x640"),
		})
	}
	return albums
}

func (r APIArtistAggregationResponse) ToTopTracks(p *HifiProvider) []domain.CatalogTrack {
	var tracks []domain.CatalogTrack
	for _, item := range r.Tracks {
		tracks = append(tracks, domain.CatalogTrack{
			ID:          formatID(item.ID),
			Title:       item.Title,
			ArtistID:    formatID(item.Artist.ID),
			Artist:      item.Artist.Name,
			AlbumID:     formatID(item.Album.ID),
			Album:       item.Album.Title,
			TrackNumber: item.TrackNumber,
			Duration:    item.Duration,
			AlbumArtURL: p.ensureAbsoluteURL(item.Album.Cover, "640x640"),
		})
	}
	return tracks
}

func (r APIAlbumResponse) ToDomain(p *HifiProvider) *domain.Album {
	data := r.Data
	year := parseYear(data.ReleaseDate)

	albumArtURL := ""
	if len(data.Cover) > 0 {
		albumArtURL = p.ensureAbsoluteURL(data.Cover[0], "640x640")
	}

	album := &domain.Album{
		ID:           formatID(data.ID),
		Title:        data.Title,
		ArtistID:     formatID(data.Artist.ID),
		Artist:       data.Artist.Name,
		Artists:      []string{data.Artist.Name},
		ArtistIDs:    []string{formatID(data.Artist.ID)},
		Year:         year,
		ReleaseDate:  data.ReleaseDate,
		Copyright:    data.Copyright,
		TotalTracks:  data.NumberOfTracks,
		TotalDiscs:   data.NumberOfVolumes,
		AlbumArtURL:  albumArtURL,
		UPC:          data.UPC,
		AlbumType:    data.Type,
		URL:          data.URL,
		Explicit:     data.Explicit,
		AudioQuality: data.AudioQuality,
		Genre:        data.Genre,
		Label:        data.Label,
	}

	for _, wrapped := range data.Items {
		item := wrapped.Item
		track := item.ToDomain(album)
		album.Tracks = append(album.Tracks, track)
	}

	return album
}

func (r APIAlbumTrackItem) ToDomain(album *domain.Album) domain.CatalogTrack {
	tArtist := album.Artist
	tArtistID := album.ArtistID

	var artists []string
	var artistIDs []string
	for _, a := range r.Artists {
		artists = append(artists, a.Name)
		artistIDs = append(artistIDs, formatID(a.ID))
	}
	if len(artists) > 0 {
		tArtist = artists[0]
		tArtistID = artistIDs[0]
	}

	track := domain.CatalogTrack{
		ID:             formatID(r.ID),
		Title:          r.Title,
		ArtistID:       tArtistID,
		Artist:         tArtist,
		Artists:        artists,
		ArtistIDs:      artistIDs,
		AlbumID:        album.ID,
		AlbumArtist:    album.Artist,
		AlbumArtists:   album.Artists,
		AlbumArtistIDs: album.ArtistIDs,
		Album:          album.Title,
		TrackNumber:    r.TrackNumber,
		DiscNumber:     r.VolumeNumber,
		TotalTracks:    album.TotalTracks,
		TotalDiscs:     album.TotalDiscs,
		Duration:       r.Duration,
		Year:           album.Year,
		ReleaseDate:    album.ReleaseDate,
		Copyright:      album.Copyright,
		ISRC:           r.ISRC,
		AlbumArtURL:    album.AlbumArtURL,
		ExplicitLyrics: r.Explicit,
		BPM:            r.BPM,
		Key:            r.Key,
		KeyScale:       r.KeyScale,
		ReplayGain:     r.ReplayGain,
		Peak:           r.Peak,
		URL:            r.URL,
		AudioQuality:   r.AudioQuality,
		Genre:          album.Genre,
		Label:          album.Label,
	}
	return track
}

func (r APIPlaylistResponse) ToDomain(p *HifiProvider) *domain.Playlist {
	pl := &domain.Playlist{
		ID:          r.Playlist.Uuid,
		Title:       r.Playlist.Title,
		Description: r.Playlist.Description,
		ImageURL:    p.ensureAbsoluteURL(r.Playlist.SquareImage, "640x640"),
	}

	for _, wrapped := range r.Items {
		item := wrapped.Item

		var artists []string
		var artistIDs []string
		for _, a := range item.Artists {
			artists = append(artists, a.Name)
			artistIDs = append(artistIDs, formatID(a.ID))
		}
		if len(artists) == 0 {
			artists = []string{"Unknown"}
			artistIDs = []string{""}
		}

		albumArtURL := ""
		if len(item.Album.Cover) > 0 {
			albumArtURL = p.ensureAbsoluteURL(item.Album.Cover[0], "640x640")
		}

		pl.Tracks = append(pl.Tracks, domain.CatalogTrack{
			ID:             formatID(item.ID),
			Title:          item.Title,
			ArtistID:       artistIDs[0],
			Artist:         artists[0],
			Artists:        artists,
			ArtistIDs:      artistIDs,
			AlbumID:        formatID(item.Album.ID),
			Album:          item.Album.Title,
			TrackNumber:    item.TrackNumber,
			Duration:       item.Duration,
			ISRC:           item.ISRC,
			AlbumArtURL:    albumArtURL,
			ExplicitLyrics: item.Explicit,
		})
	}

	return pl
}

func (r APITrackInfoResponse) ToDomain(p *HifiProvider) *domain.CatalogTrack {
	data := r.Data
	year := parseYear(data.Album.ReleaseDate)
	if year == 0 {
		year = parseYear(data.StreamStartDate)
	}

	albumArtURL := ""
	if len(data.Album.Cover) > 0 {
		albumArtURL = p.ensureAbsoluteURL(data.Album.Cover[0], "640x640")
	}

	albumArtist := data.Artist.Name

	audioModes := ""
	if len(data.AudioModes) > 0 {
		audioModes = data.AudioModes[0]
	}

	var artists []string
	var artistIDs []string
	for _, a := range data.Artists {
		artists = append(artists, a.Name)
		artistIDs = append(artistIDs, formatID(a.ID))
	}
	if len(artists) == 0 {
		artists = []string{data.Artist.Name}
		artistIDs = []string{formatID(data.Artist.ID)}
	}

	track := &domain.CatalogTrack{
		ID:             formatID(data.ID),
		Title:          data.Title,
		ArtistID:       artistIDs[0],
		Artist:         artists[0],
		Artists:        artists,
		ArtistIDs:      artistIDs,
		AlbumID:        formatID(data.Album.ID),
		AlbumArtist:    albumArtist,
		AlbumArtists:   []string{albumArtist},
		AlbumArtistIDs: []string{formatID(data.Artist.ID)},
		Album:          data.Album.Title,
		TrackNumber:    data.TrackNumber,
		DiscNumber:     data.VolumeNumber,
		TotalTracks:    data.Album.NumberOfTracks,
		TotalDiscs:     data.Album.NumberOfVolumes,
		Duration:       data.Duration,
		Year:           year,
		ReleaseDate:    data.Album.ReleaseDate,
		ISRC:           data.ISRC,
		Copyright:      data.Copyright,
		AlbumArtURL:    albumArtURL,
		ExplicitLyrics: data.Explicit,
		BPM:            data.BPM,
		Key:            data.Key,
		KeyScale:       data.KeyScale,
		ReplayGain:     data.ReplayGain,
		Peak:           data.Peak,
		URL:            data.URL,
		AudioQuality:   data.AudioQuality,
		AudioModes:     audioModes,
		Label:          data.Album.Label,
		Genre:          data.Album.Genre,
	}

	return track
}

func (r APISimilarAlbumsResponse) ToDomain(p *HifiProvider) []domain.Album {
	var albums []domain.Album
	for _, item := range r.Albums {
		artistName := ""
		if len(item.Artists) > 0 {
			artistName = item.Artists[0].Name
		}

		albums = append(albums, domain.Album{
			ID:          formatID(item.ID),
			Title:       item.Title,
			Artist:      artistName,
			AlbumArtURL: p.ensureAbsoluteURL(item.Cover, "640x640"),
		})
	}
	return albums
}

func (r APIArtistsSearchResponse) ToDomain(p *HifiProvider) []domain.Artist {
	var artists []domain.Artist
	for _, item := range r.Data.Artists.Items {
		artists = append(artists, domain.Artist{
			ID:         formatID(item.ID),
			Name:       item.Name,
			PictureURL: p.ensureAbsoluteURL(item.Picture, "320x320"),
		})
	}
	return artists
}

func (r APIAlbumsSearchResponse) ToDomain(p *HifiProvider) []domain.Album {
	var albums []domain.Album
	for _, item := range r.Data.Albums.Items {
		artist := "Unknown"
		if len(item.Artists) > 0 {
			artist = item.Artists[0].Name
		}
		albums = append(albums, domain.Album{
			ID:           formatID(item.ID),
			Title:        item.Title,
			Artist:       artist,
			AudioQuality: item.AudioQuality,
			AlbumArtURL:  p.ensureAbsoluteURL(item.Cover, "640x640"),
		})
	}
	return albums
}

func (r APITracksSearchResponse) ToDomain(p *HifiProvider) []domain.CatalogTrack {
	var tracks []domain.CatalogTrack
	for _, item := range r.Data.Items {
		var artists []string
		var artistIDs []string
		for _, a := range item.Artists {
			artists = append(artists, a.Name)
			artistIDs = append(artistIDs, formatID(a.ID))
		}
		if len(artists) == 0 {
			artists = []string{"Unknown"}
			artistIDs = []string{""}
		}
		tracks = append(tracks, domain.CatalogTrack{
			ID:           formatID(item.ID),
			Title:        item.Title,
			ArtistID:     artistIDs[0],
			Artist:       artists[0],
			Artists:      artists,
			ArtistIDs:    artistIDs,
			AlbumID:      formatID(item.Album.ID),
			Album:        item.Album.Title,
			TrackNumber:  item.TrackNumber,
			Duration:     item.Duration,
			AudioQuality: item.AudioQuality,
			AlbumArtURL:  p.ensureAbsoluteURL(item.Album.Cover, "640x640"),
		})
	}
	return tracks
}

func (r APIPlaylistsSearchResponse) ToDomain(p *HifiProvider) []domain.Playlist {
	var playlists []domain.Playlist
	for _, item := range r.Data.Playlists.Items {
		playlists = append(playlists, domain.Playlist{
			ID:       item.Uuid,
			Title:    item.Title,
			ImageURL: p.ensureAbsoluteURL(item.SquareImage, "640x640"),
		})
	}
	return playlists
}

func parseYear(date string) int {
	if len(date) < 4 {
		return 0
	}
	var year int
	if _, err := fmt.Sscanf(date[:4], "%d", &year); err != nil {
		return 0
	}
	return year
}
