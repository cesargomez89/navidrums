package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/store"
)

type Logger interface {
	With(keyValues ...interface{}) *slog.Logger
	Info(msg string, keyValues ...interface{})
	Error(msg string, keyValues ...interface{})
}

type ProviderManager struct {
	defaultURL string
	logger     Logger
	providers  *store.ProvidersRepo
	cacheTTL   time.Duration
	cache      *CachedProvider
	mu         sync.RWMutex
}

func NewProviderManager(defaultURL string, db *store.DB, cacheTTL time.Duration, logger Logger) *ProviderManager {
	var providersRepo *store.ProvidersRepo
	if db != nil {
		providersRepo = store.NewProvidersRepo(db)
	}

	var cache *CachedProvider
	if db != nil {
		cache = NewCachedProvider(&FallbackProvider{}, &storeCache{store: db}, cacheTTL)
	}

	return &ProviderManager{
		defaultURL: defaultURL,
		logger:     logger,
		providers:  providersRepo,
		cacheTTL:   cacheTTL,
		cache:      cache,
	}
}

func (m *ProviderManager) GetProvider() Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cache != nil {
		m.cache.provider = &FallbackProvider{manager: m}
		return m.cache
	}
	return &FallbackProvider{manager: m}
}

func (m *ProviderManager) SetProvider(url string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.logger != nil {
		m.logger.Info("Setting primary provider", "url", url)
	}
	if m.providers != nil {
		if !m.providers.Exists(url) {
			m.providers.Create(url, "")
		}
		providers, _ := m.providers.ListOrdered()
		for _, p := range providers {
			if p.URL == url {
				if p.Position != 0 {
					ids := make([]int64, len(providers))
					for i, prov := range providers {
						ids[i] = prov.ID
						if prov.URL == url {
							ids[0], ids[i] = ids[i], ids[0]
						}
					}
					_ = m.providers.Reorder(ids)
				}
				break
			}
		}
	}
}

func (m *ProviderManager) GetBaseURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultURL
}

func (m *ProviderManager) GetDownloadProvider() Provider {
	return m.GetProvider()
}

func (m *ProviderManager) GetDownloadURL() string {
	return m.GetBaseURL()
}

func (m *ProviderManager) GetDefaultURL() string {
	return m.GetBaseURL()
}

func (m *ProviderManager) GetDefaultDownloadURL() string {
	return m.GetBaseURL()
}

func (m *ProviderManager) SetMetadataProvider(url string) {
	m.SetProvider(url)
}

func (m *ProviderManager) SetDownloadProvider(url string) {
	m.SetProvider(url)
}

func (m *ProviderManager) GetSettingsJSON() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	settings := ProviderSettings{
		Providers:       []CustomProvider{},
		DefaultProvider: m.defaultURL,
	}

	if m.providers != nil {
		providers, _ := m.providers.ListOrdered()
		for _, p := range providers {
			settings.Providers = append(settings.Providers, CustomProvider{
				ID:   p.ID,
				Name: p.Name,
				URL:  p.URL,
			})
		}
	}

	data, _ := json.Marshal(settings)
	return string(data)
}

type CustomProvider struct {
	ID   int64  `json:"id,omitempty"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ProviderSettings struct {
	Providers       []CustomProvider `json:"providers"`
	DefaultProvider string           `json:"default_provider"`
}

func GetPredefinedProvidersJSON() string {
	return "[]"
}

type FallbackProvider struct {
	manager *ProviderManager
}

func (f *FallbackProvider) getProviders() []Provider {
	if f.manager == nil || f.manager.providers == nil {
		return []Provider{NewHifiProvider(f.manager.defaultURL)}
	}
	providers, err := f.manager.providers.ListOrdered()
	if err != nil || len(providers) == 0 {
		return []Provider{NewHifiProvider(f.manager.defaultURL)}
	}
	result := make([]Provider, len(providers))
	for i, p := range providers {
		result[i] = NewHifiProvider(p.URL)
	}
	return result
}

func (f *FallbackProvider) Search(ctx context.Context, query string, searchType string) (*domain.SearchResult, error) {
	var lastErr error
	for _, provider := range f.getProviders() {
		result, err := provider.Search(ctx, query, searchType)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all providers failed for Search: %w", lastErr)
	}
	return nil, fmt.Errorf("no providers available for Search")
}

func (f *FallbackProvider) GetArtist(ctx context.Context, id string) (*domain.Artist, error) {
	var lastErr error
	for _, provider := range f.getProviders() {
		artist, err := provider.GetArtist(ctx, id)
		if err == nil {
			return artist, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all providers failed for GetArtist: %w", lastErr)
	}
	return nil, fmt.Errorf("no providers available for GetArtist")
}

func (f *FallbackProvider) GetAlbum(ctx context.Context, id string) (*domain.Album, error) {
	var lastErr error
	for _, provider := range f.getProviders() {
		album, err := provider.GetAlbum(ctx, id)
		if err == nil {
			return album, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all providers failed for GetAlbum: %w", lastErr)
	}
	return nil, fmt.Errorf("no providers available for GetAlbum")
}

func (f *FallbackProvider) GetPlaylist(ctx context.Context, id string) (*domain.Playlist, error) {
	var lastErr error
	for _, provider := range f.getProviders() {
		playlist, err := provider.GetPlaylist(ctx, id)
		if err == nil {
			return playlist, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all providers failed for GetPlaylist: %w", lastErr)
	}
	return nil, fmt.Errorf("no providers available for GetPlaylist")
}

func (f *FallbackProvider) GetTrack(ctx context.Context, id string) (*domain.CatalogTrack, error) {
	var lastErr error
	for _, provider := range f.getProviders() {
		track, err := provider.GetTrack(ctx, id)
		if err == nil {
			return track, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all providers failed for GetTrack: %w", lastErr)
	}
	return nil, fmt.Errorf("no providers available for GetTrack")
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
	var lastErr error
	for _, provider := range f.getProviders() {
		albums, err := provider.GetSimilarAlbums(ctx, id)
		if err == nil {
			return albums, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all providers failed for GetSimilarAlbums: %w", lastErr)
	}
	return nil, fmt.Errorf("no providers available for GetSimilarAlbums")
}

func (f *FallbackProvider) GetSimilarArtists(ctx context.Context, id string) ([]domain.Artist, error) {
	var lastErr error
	for _, provider := range f.getProviders() {
		artists, err := provider.GetSimilarArtists(ctx, id)
		if err == nil {
			return artists, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all providers failed for GetSimilarArtists: %w", lastErr)
	}
	return nil, fmt.Errorf("no providers available for GetSimilarArtists")
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
	var lastErr error
	for _, provider := range f.getProviders() {
		tracks, err := provider.GetRecommendations(ctx, id)
		if err == nil {
			return tracks, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all providers failed for GetRecommendations: %w", lastErr)
	}
	return nil, fmt.Errorf("no providers available for GetRecommendations")
}

var _ Provider = (*FallbackProvider)(nil)
