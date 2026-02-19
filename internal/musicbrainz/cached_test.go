package musicbrainz

import (
	"context"
	"testing"
	"time"
)

// mockMBClient is a simple mock for musicbrainz.Client (internal)
// wait, musicbrainz.Client is a struct, not an interface.
// But CachedClient stores it as *Client.
// I might need to make Client an interface or use a different approach.
// Looking at musicbrainz/cached.go:
/*
type CachedClient struct {
	client *Client
	cache  Cache
	ttl    time.Duration
}
*/
// It uses *Client directly. This makes it hard to mock the MB API calls without a server.
// However, I can test the logic in GetRecording which coordinates between MBID/ISRC calls.

type mockCache struct {
	data map[string][]byte
}

func (m *mockCache) GetCache(key string) ([]byte, error) {
	return m.data[key], nil
}

func (m *mockCache) SetCache(key string, data []byte, ttl time.Duration) error {
	m.data[key] = data
	return nil
}

func TestCachedClient_GetRecording_MBIDCacheHit(t *testing.T) {
	// We can't easily mock *Client without an injector or a httptest server.
	// But we can verify it checks cache.
	cache := &mockCache{data: make(map[string][]byte)}
	// For this test, we'll manually populate the cache and see if it returns it without calling client
	// (we'll pass a nil client and expect no panic if hit)

	cc := &CachedClient{
		client: nil, // Should not be called if hit
		cache:  cache,
		ttl:    time.Hour,
	}

	mbid := "test-mbid"
	cacheKey := "mb:recording:" + mbid
	cache.data[cacheKey] = []byte(`{"metadata":{"RecordingID":"test-mbid","Title":"Cached Title"},"not_found":false}`)

	meta, err := cc.GetRecording(context.Background(), mbid, "", "")
	if err != nil {
		t.Fatalf("GetRecording failed: %v", err)
	}
	if meta == nil || meta.Title != "Cached Title" {
		t.Errorf("Expected cached title, got %+v", meta)
	}
}

func TestCachedClient_GetGenres_MBIDCacheHit(t *testing.T) {
	cache := &mockCache{data: make(map[string][]byte)}
	cc := &CachedClient{
		client: nil,
		cache:  cache,
		ttl:    time.Hour,
	}

	mbid := "test-mbid"
	cacheKey := "mb:genre:" + mbid
	cache.data[cacheKey] = []byte(`{"genre":{"MainGenre":"Rock","SubGenre":"Indie"},"not_found":false}`)

	res, err := cc.GetGenres(context.Background(), mbid, "")
	if err != nil {
		t.Fatalf("GetGenres failed: %v", err)
	}
	if res.MainGenre != "Rock" {
		t.Errorf("Expected cached genre 'Rock', got %s", res.MainGenre)
	}
}
