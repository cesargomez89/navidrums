package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
)

type mockProvider struct {
	Provider
	searchCalled int
}

func (m *mockProvider) Search(ctx context.Context, query string, searchType string) (*domain.SearchResult, error) {
	m.searchCalled++
	return &domain.SearchResult{
		Artists: []domain.Artist{{Name: "Result"}},
	}, nil
}

type mockCache struct {
	data map[string][]byte
	err  error
}

func (m *mockCache) GetCache(key string) ([]byte, error) {
	return m.data[key], m.err
}

func (m *mockCache) SetCache(key string, data []byte, ttl time.Duration) error {
	m.data[key] = data
	return m.err
}

func (m *mockCache) ClearCache() error {
	m.data = make(map[string][]byte)
	return m.err
}

func TestCachedProvider_Search(t *testing.T) {
	inner := &mockProvider{}
	cache := &mockCache{data: make(map[string][]byte)}
	cp := NewCachedProvider(inner, cache, time.Hour)

	ctx := context.Background()

	// 1. First call - should call inner provider
	res, err := cp.Search(ctx, "query", "artist")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if res.Artists[0].Name != "Result" {
		t.Errorf("Unexpected result name")
	}
	if inner.searchCalled != 1 {
		t.Errorf("Expected inner provider to be called once, got %d", inner.searchCalled)
	}

	// 2. Second call - should hit cache
	res2, err := cp.Search(ctx, "query", "artist")
	if err != nil {
		t.Fatalf("Second Search failed: %v", err)
	}
	if res2.Artists[0].Name != "Result" {
		t.Errorf("Unexpected second result name")
	}
	if inner.searchCalled != 1 {
		t.Errorf("Expected inner provider to STILL be called once (cache hit), got %d", inner.searchCalled)
	}

	// 3. Clear cache - should call inner again
	_ = cp.ClearCache()
	_, _ = cp.Search(ctx, "query", "artist")
	if inner.searchCalled != 2 {
		t.Errorf("Expected inner provider to be called again after clear, got %d", inner.searchCalled)
	}
}

func TestCachedProvider_Error(t *testing.T) {
	inner := &mockProvider{}
	cache := &mockCache{err: errors.New("cache error")}
	cp := NewCachedProvider(inner, cache, time.Hour)

	_, err := cp.Search(context.Background(), "q", "a")
	if err == nil {
		t.Error("Expected error from cache to propagate")
	}
}
