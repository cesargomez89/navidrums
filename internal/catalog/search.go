package catalog

import (
	"context"
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
	var resp APIArtistsSearchResponse
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	return resp.ToDomain(p), nil
}

func (p *HifiProvider) searchAlbums(ctx context.Context, query string) ([]domain.Album, error) {
	u := fmt.Sprintf("%s/search/?al=%s", p.BaseURL, url.QueryEscape(query))
	var resp APIAlbumsSearchResponse
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	return resp.ToDomain(p), nil
}

func (p *HifiProvider) searchTracks(ctx context.Context, query string) ([]domain.CatalogTrack, error) {
	u := fmt.Sprintf("%s/search/?s=%s", p.BaseURL, url.QueryEscape(query))
	var resp APITracksSearchResponse
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	return resp.ToDomain(p), nil
}

func (p *HifiProvider) searchPlaylists(ctx context.Context, query string) ([]domain.Playlist, error) {
	u := fmt.Sprintf("%s/search/?p=%s", p.BaseURL, url.QueryEscape(query))
	var resp APIPlaylistsSearchResponse
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	return resp.ToDomain(p), nil
}
