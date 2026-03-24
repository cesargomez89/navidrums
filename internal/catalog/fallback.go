package catalog

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
)

type FallbackProvider struct {
	manager         *ProviderManager
	cachedProviders []Provider
	cacheExpiry     time.Time
	cacheMu         sync.Mutex
}

const providerCacheTTL = 30 * time.Second

func (f *FallbackProvider) getProviders() []Provider {
	f.cacheMu.Lock()
	defer f.cacheMu.Unlock()

	if f.cachedProviders != nil && time.Now().Before(f.cacheExpiry) {
		return f.cachedProviders
	}

	// Always start with the system default URL
	providers := []Provider{NewHifiProvider(f.manager.defaultURL)}

	if f.manager != nil && f.manager.providers != nil {
		storeProviders, _ := f.manager.providers.ListOrdered()
		for _, p := range storeProviders {
			// Skip if it's the same as default to avoid double-requesting
			if p.URL == f.manager.defaultURL {
				continue
			}
			providers = append(providers, NewHifiProvider(p.URL))
		}
	}

	f.cachedProviders = providers
	f.cacheExpiry = time.Now().Add(providerCacheTTL)
	return providers
}

func (f *FallbackProvider) invalidateCache() {
	f.cacheMu.Lock()
	defer f.cacheMu.Unlock()
	f.cachedProviders = nil
	f.cacheExpiry = time.Time{}
}

func fallbackWith[T any](f *FallbackProvider, opName string, op func(Provider) (T, error)) (T, error) {
	var lastErr error
	var zero T
	for _, provider := range f.getProviders() {
		result, err := op(provider)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return zero, fmt.Errorf("all providers failed for %s: %w", opName, lastErr)
	}
	return zero, fmt.Errorf("no providers available for %s", opName)
}

func (f *FallbackProvider) Search(ctx context.Context, query string, searchType string) (*domain.SearchResult, error) {
	return fallbackWith(f, "Search", func(p Provider) (*domain.SearchResult, error) { return p.Search(ctx, query, searchType) })
}

func (f *FallbackProvider) GetArtist(ctx context.Context, id string) (*domain.Artist, error) {
	return fallbackWith(f, "GetArtist", func(p Provider) (*domain.Artist, error) { return p.GetArtist(ctx, id) })
}

func (f *FallbackProvider) GetAlbum(ctx context.Context, id string) (*domain.Album, error) {
	return fallbackWith(f, "GetAlbum", func(p Provider) (*domain.Album, error) { return p.GetAlbum(ctx, id) })
}

func (f *FallbackProvider) GetPlaylist(ctx context.Context, id string) (*domain.Playlist, error) {
	return fallbackWith(f, "GetPlaylist", func(p Provider) (*domain.Playlist, error) { return p.GetPlaylist(ctx, id) })
}

func (f *FallbackProvider) GetTrack(ctx context.Context, id string) (*domain.CatalogTrack, error) {
	return fallbackWith(f, "GetTrack", func(p Provider) (*domain.CatalogTrack, error) { return p.GetTrack(ctx, id) })
}

func (f *FallbackProvider) GetStream(ctx context.Context, trackID string, quality string) (io.ReadCloser, string, error) {
	var lastErr error
	for _, provider := range f.getProviders() {
		stream, contentType, err := provider.GetStream(ctx, trackID, quality)
		if err == nil {
			return stream, contentType, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, "", fmt.Errorf("all providers failed for GetStream: %w", lastErr)
	}
	return nil, "", fmt.Errorf("no providers available for GetStream")
}

func (f *FallbackProvider) GetSimilarAlbums(ctx context.Context, id string) ([]domain.Album, error) {
	return fallbackWith(f, "GetSimilarAlbums", func(p Provider) ([]domain.Album, error) { return p.GetSimilarAlbums(ctx, id) })
}

func (f *FallbackProvider) GetSimilarArtists(ctx context.Context, id string) ([]domain.Artist, error) {
	return fallbackWith(f, "GetSimilarArtists", func(p Provider) ([]domain.Artist, error) { return p.GetSimilarArtists(ctx, id) })
}

func (f *FallbackProvider) GetLyrics(ctx context.Context, trackID string) (string, string, error) {
	var lastErr error
	for _, provider := range f.getProviders() {
		lyrics, source, err := provider.GetLyrics(ctx, trackID)
		if err == nil {
			return lyrics, source, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return "", "", fmt.Errorf("all providers failed for GetLyrics: %w", lastErr)
	}
	return "", "", fmt.Errorf("no providers available for GetLyrics")
}

func (f *FallbackProvider) GetRecommendations(ctx context.Context, id string) ([]domain.CatalogTrack, error) {
	return fallbackWith(f, "GetRecommendations", func(p Provider) ([]domain.CatalogTrack, error) { return p.GetRecommendations(ctx, id) })
}

var _ Provider = (*FallbackProvider)(nil)
