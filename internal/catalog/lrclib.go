package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type LrcLibProvider struct {
	baseURL string
	client  *http.Client
}

type lrclibResponse struct {
	PlainLyrics  string `json:"plainLyrics"`
	SyncedLyrics string `json:"syncedLyrics"`
}

func NewLrcLibProvider(baseURL string) *LrcLibProvider {
	return &LrcLibProvider{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *LrcLibProvider) Name() string {
	return "lrclib"
}

func (p *LrcLibProvider) GetLyrics(ctx context.Context, track, artist, album string, duration int) (string, string, error) {
	if track == "" {
		return "", "", fmt.Errorf("track name is required")
	}

	u := p.baseURL + "?" + url.Values{
		"track_name":  {track},
		"artist_name": {artist},
		"album_name":  {album},
		"duration":    {fmt.Sprintf("%d", duration)},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "navidrums/1.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch lyrics: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", "", fmt.Errorf("lyrics not found")
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("API returned status: %s", resp.Status)
	}

	var result lrclibResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.PlainLyrics == "" && result.SyncedLyrics == "" {
		return "", "", fmt.Errorf("no lyrics returned")
	}

	return result.PlainLyrics, result.SyncedLyrics, nil
}
