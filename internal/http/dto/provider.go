package dto

import (
	"encoding/json"

	"github.com/cesargomez89/navidrums/internal/catalog"
)

type PredefinedProvider struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ProviderResponse struct {
	Active     string                   `json:"active"`
	Default    string                   `json:"default"`
	Predefined json.RawMessage          `json:"predefined"`
	Custom     []catalog.CustomProvider `json:"custom"`
}

func FromProviderData(active, defaultURL string, custom []catalog.CustomProvider) ProviderResponse {
	return ProviderResponse{
		Active:  active,
		Default: defaultURL,
		Custom:  custom,
	}
}
