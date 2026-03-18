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
	provider           Provider
	metadataProvider   *HifiProvider
	downloadProvider   *HifiProvider
	logger             Logger
	cached             *CachedProvider
	settingsRepo       *store.SettingsRepo
	metadataURL        string
	downloadURL        string
	defaultMetadataURL string
	defaultDownloadURL string
	mu                 sync.RWMutex
}

func NewProviderManager(metURL, dlURL string, db *store.DB, cacheTTL time.Duration, logger Logger) *ProviderManager {
	if metURL == "" {
		metURL = dlURL
	}
	if dlURL == "" {
		dlURL = metURL
	}

	var settingsRepo *store.SettingsRepo
	if db != nil {
		settingsRepo = store.NewSettingsRepo(db)
	}

	hifi := NewHifiProviderDual(metURL, dlURL)
	var cached *CachedProvider
	if db != nil {
		cached = NewCachedProvider(hifi, &storeCache{store: db}, cacheTTL)
	}
	return &ProviderManager{
		metadataURL:        metURL,
		downloadURL:        dlURL,
		defaultMetadataURL: metURL,
		defaultDownloadURL: dlURL,
		metadataProvider:   hifi,
		downloadProvider:   hifi,
		provider:           hifi,
		cached:             cached,
		logger:             logger,
		settingsRepo:       settingsRepo,
	}
}

func (m *ProviderManager) GetDefaultURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultMetadataURL
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
	m.metadataProvider = NewHifiProviderDual(baseURL, baseURL)
	m.downloadProvider = m.metadataProvider
	m.provider = m.metadataProvider
	if m.cached != nil {
		m.cached.provider = m.metadataProvider
		_ = m.cached.ClearCache()
	}
	m.metadataURL = baseURL
	m.downloadURL = baseURL
	if m.settingsRepo != nil {
		_ = m.settingsRepo.Set(store.SettingActiveMetadataProvider, baseURL)
		_ = m.settingsRepo.Set(store.SettingActiveDownloadProvider, baseURL)
	}
}

func (m *ProviderManager) GetBaseURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metadataURL
}

func (m *ProviderManager) GetDownloadProvider() Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.downloadProvider
}

func (m *ProviderManager) GetDownloadURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.downloadURL
}

func (m *ProviderManager) GetDefaultDownloadURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultDownloadURL
}

func (m *ProviderManager) SetMetadataProvider(url string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.logger != nil {
		m.logger.Info("Setting metadata provider", "url", url)
	}
	m.metadataProvider = NewHifiProviderDual(url, m.downloadURL)
	m.provider = m.metadataProvider
	if m.cached != nil {
		m.cached.provider = m.metadataProvider
		_ = m.cached.ClearCache()
	}
	m.metadataURL = url
	if m.settingsRepo != nil {
		_ = m.settingsRepo.Set(store.SettingActiveMetadataProvider, url)
	}
}

func (m *ProviderManager) SetDownloadProvider(url string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.logger != nil {
		m.logger.Info("Setting download provider", "url", url)
	}
	m.downloadProvider = NewHifiProviderDual(m.metadataURL, url)
	m.downloadURL = url
	if m.settingsRepo != nil {
		_ = m.settingsRepo.Set(store.SettingActiveDownloadProvider, url)
	}
}

type CustomProvider struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ProviderSettings struct {
	MetadataURL        string           `json:"metadata_url"`
	DownloadURL        string           `json:"download_url"`
	DefaultMetadataURL string           `json:"default_metadata_url"`
	DefaultDownloadURL string           `json:"default_download_url"`
	CustomProviders    []CustomProvider `json:"custom_providers"`
}

func (m *ProviderManager) GetSettingsJSON() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	settings := ProviderSettings{
		MetadataURL:        m.metadataURL,
		DownloadURL:        m.downloadURL,
		DefaultMetadataURL: m.defaultMetadataURL,
		DefaultDownloadURL: m.defaultDownloadURL,
		CustomProviders:    []CustomProvider{},
	}

	data, _ := json.Marshal(settings)
	return string(data)
}

func GetPredefinedProvidersJSON() string {
	return "[]"
}
