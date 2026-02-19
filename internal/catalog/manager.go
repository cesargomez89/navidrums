package catalog

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/cesargomez89/navidrums/internal/store"
)

type Logger interface {
	With(keyValues ...interface{}) *slog.Logger
	Info(msg string, keyValues ...interface{})
	Error(msg string, keyValues ...interface{})
}

type ProviderManager struct {
	provider   Provider
	logger     Logger
	cached     *CachedProvider
	baseURL    string
	defaultURL string
	mu         sync.RWMutex
}

func NewProviderManager(baseURL string, db *store.DB, cacheTTL time.Duration, logger Logger) *ProviderManager {
	hifi := NewHifiProvider(baseURL)
	var cached *CachedProvider
	if db != nil {
		cached = NewCachedProvider(hifi, &storeCache{store: db}, cacheTTL)
	}
	return &ProviderManager{
		baseURL:    baseURL,
		defaultURL: baseURL,
		provider:   hifi,
		cached:     cached,
		logger:     logger,
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
	if m.cached != nil {
		return m.cached
	}
	return m.provider
}

func (m *ProviderManager) SetProvider(baseURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.logger != nil {
		m.logger.Info("Setting provider", "url", baseURL)
	}
	m.provider = NewHifiProvider(baseURL)
	if m.cached != nil {
		m.cached.provider = m.provider
		_ = m.cached.ClearCache()
	}
	m.baseURL = baseURL
}

func (m *ProviderManager) GetBaseURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.baseURL
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
