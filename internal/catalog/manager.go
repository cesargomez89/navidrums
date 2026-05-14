package catalog

import (
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
	logger    Logger
	providers *store.ProvidersRepo
	settings  *store.SettingsRepo
	cacheTTL  time.Duration
	db        *store.DB

	chains map[ProviderType]*CachedProvider
	mu     sync.RWMutex
}

func NewProviderManager(db *store.DB, settings *store.SettingsRepo, cacheTTL time.Duration, logger Logger) *ProviderManager {
	var providersRepo *store.ProvidersRepo
	if db != nil {
		providersRepo = store.NewProvidersRepo(db)
	}

	return &ProviderManager{
		logger:    logger,
		providers: providersRepo,
		settings:  settings,
		cacheTTL:  cacheTTL,
		db:        db,
	}
}

func (m *ProviderManager) readSetting(key string) ProviderType {
	if m.settings == nil {
		return ProviderTypeHifi
	}
	val, err := m.settings.Get(key)
	if err != nil || val == "" {
		return ProviderTypeHifi
	}
	pt := ProviderType(val)
	if pt != ProviderTypeHifi && pt != ProviderTypeQobuz {
		return ProviderTypeHifi
	}
	return pt
}

func (m *ProviderManager) buildChain(pt ProviderType) *CachedProvider {
	fb := &FallbackProvider{manager: m, providerType: pt}
	var cacheStore *storeCache
	if m.db != nil {
		cacheStore = &storeCache{store: m.db}
	}
	return NewCachedProvider(fb, cacheStore, m.cacheTTL)
}

func (m *ProviderManager) GetProvider(pt ProviderType) Provider {
	m.mu.RLock()
	chain := m.chains[pt]
	m.mu.RUnlock()

	if chain != nil {
		return chain
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.chains == nil {
		m.chains = make(map[ProviderType]*CachedProvider)
	}
	if m.chains[pt] == nil {
		m.chains[pt] = m.buildChain(pt)
	}
	return m.chains[pt]
}

func (m *ProviderManager) GetMetadataProvider() Provider {
	return m.GetProvider(m.readSetting(store.SettingActiveMetadataProvider))
}

func (m *ProviderManager) GetDownloadProvider() Provider {
	return m.GetProvider(m.readSetting(store.SettingActiveDownloadProvider))
}

func (m *ProviderManager) GetStreamingProvider() Provider {
	return m.GetProvider(m.readSetting(store.SettingActiveStreamingProvider))
}

func (m *ProviderManager) InvalidateAllCaches() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chains = nil
}

type CustomProvider struct {
	ID   int64  `json:"id,omitempty"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

func (m *ProviderManager) GetProvidersByType(providerType string) []CustomProvider {
	if m.providers == nil {
		return nil
	}
	providers, err := m.providers.ListByType(providerType)
	if err != nil {
		return nil
	}
	result := make([]CustomProvider, len(providers))
	for i, p := range providers {
		result[i] = CustomProvider{
			ID:   p.ID,
			Name: p.Name,
			URL:  p.URL,
			Type: p.Type,
		}
	}
	return result
}
