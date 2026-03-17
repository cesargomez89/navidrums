package catalog

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/cesargomez89/navidrums/internal/store"
)

type Logger interface {
	With(keyValues ...any) *slog.Logger
	Info(msg string, keyValues ...any)
	Error(msg string, keyValues ...any)
}

type ProviderManager struct {
	defaultURL       string
	logger           Logger
	providers        *store.ProvidersRepo
	cacheTTL         time.Duration
	cache            *CachedProvider
	fallbackProvider *FallbackProvider
	mu               sync.RWMutex
}

func NewProviderManager(defaultURL string, db *store.DB, cacheTTL time.Duration, logger Logger) *ProviderManager {
	var providersRepo *store.ProvidersRepo
	if db != nil {
		providersRepo = store.NewProvidersRepo(db)
	}

	pm := &ProviderManager{
		defaultURL: defaultURL,
		logger:     logger,
		providers:  providersRepo,
		cacheTTL:   cacheTTL,
	}

	pm.fallbackProvider = &FallbackProvider{manager: pm}

	if db != nil {
		pm.cache = NewCachedProvider(pm.fallbackProvider, &storeCache{store: db}, cacheTTL)
	}

	return pm
}

func (m *ProviderManager) GetProvider() Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.cache != nil {
		return m.cache
	}
	return m.fallbackProvider
}

func (m *ProviderManager) InvalidateProviderCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cache != nil {
		if fb, ok := m.cache.provider.(*FallbackProvider); ok {
			fb.invalidateCache()
		}
	}
	if m.fallbackProvider != nil {
		m.fallbackProvider.invalidateCache()
	}
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
	if m.cache != nil {
		if fb, ok := m.cache.provider.(*FallbackProvider); ok {
			fb.invalidateCache()
		}
	}
}

func (m *ProviderManager) GetBaseURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultURL
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
