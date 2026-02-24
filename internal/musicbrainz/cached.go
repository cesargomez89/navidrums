package musicbrainz

import (
	"context"
	"encoding/json"
	"time"
)

type ClientInterface interface {
	SetGenreMap(m map[string]string)
	GetGenreMap() map[string]string
	GetRecording(ctx context.Context, recordingID, isrc, albumName string) (*RecordingMetadata, error)
	GetGenres(ctx context.Context, recordingID, isrc string) (GenreResult, error)
}

var _ ClientInterface = (*Client)(nil)
var _ ClientInterface = (*CachedClient)(nil)

type Cache interface {
	GetCache(key string) ([]byte, error)
	SetCache(key string, data []byte, ttl time.Duration) error
}

type CachedClient struct {
	client *Client
	cache  Cache
	ttl    time.Duration
}

func NewCachedClient(client *Client, cache Cache, ttl time.Duration) *CachedClient {
	return &CachedClient{
		client: client,
		cache:  cache,
		ttl:    ttl,
	}
}

func (c *CachedClient) SetGenreMap(m map[string]string) {
	c.client.SetGenreMap(m)
}

func (c *CachedClient) GetGenreMap() map[string]string {
	return c.client.GetGenreMap()
}

type cachedMetadata struct {
	Metadata *RecordingMetadata `json:"metadata"`
	NotFound bool               `json:"not_found"`
}

func (c *CachedClient) GetRecording(ctx context.Context, recordingID, isrc, albumName string) (*RecordingMetadata, error) {
	if recordingID != "" {
		return c.getRecordingByMBID(ctx, recordingID, albumName)
	}
	if isrc != "" {
		return c.getRecordingByISRC(ctx, isrc, albumName)
	}
	return nil, nil
}

func (c *CachedClient) getRecordingByMBID(ctx context.Context, mbid, albumName string) (*RecordingMetadata, error) {
	cacheKey := "mb:recording:" + mbid

	data, err := c.cache.GetCache(cacheKey)
	if err != nil {
		return nil, err
	}

	if data != nil {
		var cached cachedMetadata
		if unmarshalErr := json.Unmarshal(data, &cached); unmarshalErr == nil {
			return cached.Metadata, nil
		}
	}

	meta, err := c.client.GetRecordingByMBID(ctx, mbid, albumName)
	if err != nil {
		return nil, err
	}

	cached := cachedMetadata{Metadata: meta}
	if meta == nil {
		cached.NotFound = true
	}

	if data, marshalErr := json.Marshal(cached); marshalErr == nil {
		_ = c.cache.SetCache(cacheKey, data, c.ttl)
	}

	return meta, nil
}

func (c *CachedClient) getRecordingByISRC(ctx context.Context, isrc, albumName string) (*RecordingMetadata, error) {
	meta, err := c.client.GetRecordingByISRC(ctx, isrc, albumName)
	if err != nil {
		return nil, err
	}

	if meta != nil && meta.RecordingID != "" {
		cached := cachedMetadata{Metadata: meta}
		if data, marshalErr := json.Marshal(cached); marshalErr == nil {
			cacheKey := "mb:recording:" + meta.RecordingID
			_ = c.cache.SetCache(cacheKey, data, c.ttl)
		}
	}

	return meta, nil
}

type cachedGenre struct {
	Genre    GenreResult `json:"genre"`
	NotFound bool        `json:"not_found"`
}

func (c *CachedClient) GetGenres(ctx context.Context, recordingID, isrc string) (GenreResult, error) {
	if recordingID != "" {
		return c.getGenresByMBID(ctx, recordingID)
	}
	if isrc != "" {
		return c.getGenresByISRC(ctx, isrc)
	}
	return GenreResult{}, nil
}

func (c *CachedClient) getGenresByMBID(ctx context.Context, mbid string) (GenreResult, error) {
	cacheKey := "mb:genre:" + mbid

	data, err := c.cache.GetCache(cacheKey)
	if err != nil {
		return GenreResult{}, err
	}

	if data != nil {
		var cached cachedGenre
		if unmarshalErr := json.Unmarshal(data, &cached); unmarshalErr == nil {
			return cached.Genre, nil
		}
	}

	result, err := c.client.GetGenresByMBID(ctx, mbid)
	if err != nil {
		return GenreResult{}, err
	}

	cached := cachedGenre{Genre: result}
	if data, marshalErr := json.Marshal(cached); marshalErr == nil {
		_ = c.cache.SetCache(cacheKey, data, c.ttl)
	}

	return result, nil
}

func (c *CachedClient) getGenresByISRC(ctx context.Context, isrc string) (GenreResult, error) {
	return c.client.GetGenresByISRC(ctx, isrc)
}
