package catalog

import (
	"encoding/json"
	"sync"
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
