package providers

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/cesargomez89/navidrums/internal/models"
)

type ProviderManager struct {
	mu         sync.RWMutex
	provider   Provider
	baseURL    string
	defaultURL string
}

func NewProviderManager(baseURL string) *ProviderManager {
	return &ProviderManager{
		baseURL:    baseURL,
		defaultURL: baseURL,
		provider:   NewHifiProvider(baseURL),
	}
}

func (m *ProviderManager) GetDefaultURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultURL
}

func (m *ProviderManager) GetProvider() Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.provider
}

func (m *ProviderManager) SetProvider(baseURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.provider = NewHifiProvider(baseURL)
	m.baseURL = baseURL
}

func (m *ProviderManager) GetBaseURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.baseURL
}

func (m *ProviderManager) Search(ctx context.Context, query string, searchType string) (*models.SearchResult, error) {
	return m.GetProvider().Search(ctx, query, searchType)
}

func (m *ProviderManager) GetArtist(ctx context.Context, id string) (*models.Artist, error) {
	return m.GetProvider().GetArtist(ctx, id)
}

func (m *ProviderManager) GetAlbum(ctx context.Context, id string) (*models.Album, error) {
	return m.GetProvider().GetAlbum(ctx, id)
}

func (m *ProviderManager) GetPlaylist(ctx context.Context, id string) (*models.Playlist, error) {
	return m.GetProvider().GetPlaylist(ctx, id)
}

func (m *ProviderManager) GetTrack(ctx context.Context, id string) (*models.Track, error) {
	return m.GetProvider().GetTrack(ctx, id)
}

func (m *ProviderManager) GetStream(ctx context.Context, trackID string, quality string) (io.ReadCloser, string, error) {
	return m.GetProvider().GetStream(ctx, trackID, quality)
}

type CustomProvider struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ProviderSettings struct {
	ActiveProvider  string           `json:"active_provider"`
	CustomProviders []CustomProvider `json:"custom_providers"`
}

func (m *ProviderManager) GetSettingsJSON() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	settings := ProviderSettings{
		ActiveProvider:  m.baseURL,
		CustomProviders: []CustomProvider{},
	}

	data, _ := json.Marshal(settings)
	return string(data)
}

func GetPredefinedProvidersJSON() string {
	return "[]"
}
