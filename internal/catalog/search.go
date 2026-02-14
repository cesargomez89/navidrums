package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/cesargomez89/navidrums/internal/domain"
)

func (p *HifiProvider) Search(ctx context.Context, query string, searchType string) (*domain.SearchResult, error) {
	res := &domain.SearchResult{}

	if searchType == "" {
		searchType = "album"
	}

	switch searchType {
	case "artist":
		artists, err := p.searchArtists(ctx, query)
		if err == nil {
			res.Artists = artists
		}
	case "album":
		albums, err := p.searchAlbums(ctx, query)
		if err == nil {
			res.Albums = albums
		}
	case "track":
		tracks, err := p.searchTracks(ctx, query)
		if err == nil {
			res.Tracks = tracks
		}
	case "playlist":
		playlists, err := p.searchPlaylists(ctx, query)
		if err == nil {
			res.Playlists = playlists
		}
	default:
		albums, err := p.searchAlbums(ctx, query)
		if err == nil {
			res.Albums = albums
		}
	}

	return res, nil
}

func (p *HifiProvider) searchArtists(ctx context.Context, query string) ([]domain.Artist, error) {
	u := fmt.Sprintf("%s/search/?a=%s", p.BaseURL, url.QueryEscape(query))
	var resp struct {
		Data struct {
			Artists struct {
				Items []struct {
					ID      json.Number `json:"id"`
					Name    string      `json:"name"`
					Picture string      `json:"picture"`
				} `json:"items"`
			} `json:"artists"`
		} `json:"data"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	var artists []domain.Artist
	for _, item := range resp.Data.Artists.Items {
		artists = append(artists, domain.Artist{
			ID:         formatID(item.ID),
			Name:       item.Name,
			PictureURL: p.ensureAbsoluteURL(item.Picture, "320x320"),
		})
	}
	return artists, nil
}

func (p *HifiProvider) searchAlbums(ctx context.Context, query string) ([]domain.Album, error) {
	u := fmt.Sprintf("%s/search/?al=%s", p.BaseURL, url.QueryEscape(query))
	var resp struct {
		Data struct {
			Albums struct {
				Items []struct {
					ID      json.Number `json:"id"`
					Title   string      `json:"title"`
					Cover   string      `json:"cover"`
					Artists []struct {
						Name string `json:"name"`
					} `json:"artists"`
				} `json:"items"`
			} `json:"albums"`
		} `json:"data"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	var albums []domain.Album
	for _, item := range resp.Data.Albums.Items {
		artist := "Unknown"
		if len(item.Artists) > 0 {
			artist = item.Artists[0].Name
		}
		albums = append(albums, domain.Album{
			ID:          formatID(item.ID),
			Title:       item.Title,
			Artist:      artist,
			AlbumArtURL: p.ensureAbsoluteURL(item.Cover, "640x640"),
		})
	}
	return albums, nil
}

func (p *HifiProvider) searchTracks(ctx context.Context, query string) ([]domain.Track, error) {
	u := fmt.Sprintf("%s/search/?s=%s", p.BaseURL, url.QueryEscape(query))
	var resp struct {
		Data struct {
			Items []struct {
				ID          json.Number `json:"id"`
				Title       string      `json:"title"`
				Duration    int         `json:"duration"`
				TrackNumber int         `json:"trackNumber"`
				Album       struct {
					Title string `json:"title"`
					Cover string `json:"cover"`
				} `json:"album"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	var tracks []domain.Track
	for _, item := range resp.Data.Items {
		artist := "Unknown"
		if len(item.Artists) > 0 {
			artist = item.Artists[0].Name
		}
		tracks = append(tracks, domain.Track{
			ID:          formatID(item.ID),
			Title:       item.Title,
			Artist:      artist,
			Album:       item.Album.Title,
			TrackNumber: item.TrackNumber,
			Duration:    item.Duration,
			AlbumArtURL: p.ensureAbsoluteURL(item.Album.Cover, "640x640"),
		})
	}
	return tracks, nil
}

func (p *HifiProvider) searchPlaylists(ctx context.Context, query string) ([]domain.Playlist, error) {
	u := fmt.Sprintf("%s/search/?p=%s", p.BaseURL, url.QueryEscape(query))
	var resp struct {
		Data struct {
			Playlists struct {
				Items []struct {
					Uuid        string `json:"uuid"`
					Title       string `json:"title"`
					SquareImage string `json:"squareImage"`
				} `json:"items"`
			} `json:"playlists"`
		} `json:"data"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	var playlists []domain.Playlist
	for _, item := range resp.Data.Playlists.Items {
		playlists = append(playlists, domain.Playlist{
			ID:       item.Uuid,
			Title:    item.Title,
			ImageURL: p.ensureAbsoluteURL(item.SquareImage, "640x640"),
		})
	}
	return playlists, nil
}
