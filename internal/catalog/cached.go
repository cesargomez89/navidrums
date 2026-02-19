package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/store"
)

type Cache interface {
	GetCache(key string) ([]byte, error)
	SetCache(key string, data []byte, ttl time.Duration) error
	ClearCache() error
}

type CachedProvider struct {
	provider Provider
	cache    Cache
	cacheTTL time.Duration
}

func NewCachedProvider(provider Provider, cache Cache, cacheTTL time.Duration) *CachedProvider {
	return &CachedProvider{
		provider: provider,
		cache:    cache,
		cacheTTL: cacheTTL,
	}
}

func (c *CachedProvider) Search(ctx context.Context, query string, searchType string) (*domain.SearchResult, error) {
	cacheKey := fmt.Sprintf("search:%s:%s", searchType, query)

	data, err := c.cache.GetCache(cacheKey)
	if err != nil {
		return nil, err
	}
	if data != nil {
		var result domain.SearchResult
		if err := json.Unmarshal(data, &result); err == nil {
			return &result, nil
		}
	}

	result, err := c.provider.Search(ctx, query, searchType)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(result); err == nil {
		c.cache.SetCache(cacheKey, data, c.cacheTTL)
	}

	return result, nil
}

func (c *CachedProvider) GetArtist(ctx context.Context, id string) (*domain.Artist, error) {
	cacheKey := fmt.Sprintf("artist:%s", id)

	data, err := c.cache.GetCache(cacheKey)
	if err != nil {
		return nil, err
	}
	if data != nil {
		var artist domain.Artist
		if err := json.Unmarshal(data, &artist); err == nil {
			return &artist, nil
		}
	}

	artist, err := c.provider.GetArtist(ctx, id)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(artist); err == nil {
		c.cache.SetCache(cacheKey, data, c.cacheTTL)
	}

	return artist, nil
}

func (c *CachedProvider) GetAlbum(ctx context.Context, id string) (*domain.Album, error) {
	cacheKey := fmt.Sprintf("album:%s", id)

	data, err := c.cache.GetCache(cacheKey)
	if err != nil {
		return nil, err
	}
	if data != nil {
		var album domain.Album
		if err := json.Unmarshal(data, &album); err == nil {
			return &album, nil
		}
	}

	album, err := c.provider.GetAlbum(ctx, id)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(album); err == nil {
		c.cache.SetCache(cacheKey, data, c.cacheTTL)
	}

	return album, nil
}

func (c *CachedProvider) GetPlaylist(ctx context.Context, id string) (*domain.Playlist, error) {
	cacheKey := fmt.Sprintf("playlist:%s", id)

	data, err := c.cache.GetCache(cacheKey)
	if err != nil {
		return nil, err
	}
	if data != nil {
		var playlist domain.Playlist
		if err := json.Unmarshal(data, &playlist); err == nil {
			return &playlist, nil
		}
	}

	playlist, err := c.provider.GetPlaylist(ctx, id)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(playlist); err == nil {
		c.cache.SetCache(cacheKey, data, c.cacheTTL)
	}

	return playlist, nil
}

func (c *CachedProvider) GetTrack(ctx context.Context, id string) (*domain.CatalogTrack, error) {
	cacheKey := fmt.Sprintf("track:%s", id)

	data, err := c.cache.GetCache(cacheKey)
	if err != nil {
		return nil, err
	}
	if data != nil {
		var track domain.CatalogTrack
		if err := json.Unmarshal(data, &track); err == nil {
			return &track, nil
		}
	}

	track, err := c.provider.GetTrack(ctx, id)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(track); err == nil {
		c.cache.SetCache(cacheKey, data, c.cacheTTL)
	}

	return track, nil
}

func (c *CachedProvider) GetStream(ctx context.Context, trackID string, quality string) (io.ReadCloser, string, error) {
	return c.provider.GetStream(ctx, trackID, quality)
}

func (c *CachedProvider) GetSimilarAlbums(ctx context.Context, id string) ([]domain.Album, error) {
	return c.provider.GetSimilarAlbums(ctx, id)
}

func (c *CachedProvider) GetLyrics(ctx context.Context, trackID string) (string, string, error) {
	return c.provider.GetLyrics(ctx, trackID)
}

func (c *CachedProvider) ClearCache() error {
	return c.cache.ClearCache()
}

var _ Provider = (*CachedProvider)(nil)

type storeCache struct {
	store *store.DB
}

func (s *storeCache) GetCache(key string) ([]byte, error) {
	return s.store.GetCache(key)
}

func (s *storeCache) SetCache(key string, data []byte, ttl time.Duration) error {
	return s.store.SetCache(key, data, ttl)
}

func (s *storeCache) ClearCache() error {
	return s.store.ClearCache()
}

var _ Cache = (*storeCache)(nil)
