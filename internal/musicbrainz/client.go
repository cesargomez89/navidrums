package musicbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultUserAgent   = "navidrums/1.0 (https://github.com/cesargomez89/navidrums)"
	requestTimeout     = 10 * time.Second
	minRequestInterval = 1050 * time.Millisecond
)

type Client struct {
	baseURL     string
	userAgent   string
	httpClient  *http.Client
	lastRequest time.Time
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		userAgent: DefaultUserAgent,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

func (c *Client) GetGenresByISRC(ctx context.Context, isrc string) ([]string, error) {
	if isrc == "" {
		return nil, nil
	}

	c.throttle()

	u := fmt.Sprintf("%s/recording?query=isrc:%s&inc=tags&fmt=json", c.baseURL, url.QueryEscape(isrc))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz returned status %d", resp.StatusCode)
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	genres := extractGenres(result.Recordings)
	if len(genres) == 0 {
		return nil, nil
	}

	return genres, nil
}

func (c *Client) throttle() {
	now := time.Now()
	elapsed := now.Sub(c.lastRequest)
	if elapsed < minRequestInterval {
		time.Sleep(minRequestInterval - elapsed)
	}
	c.lastRequest = time.Now()
}

func extractGenres(recordings []recording) []string {
	for _, rec := range recordings {
		if len(rec.Tags) > 0 {
			genres := make([]string, 0, len(rec.Tags))
			for _, tag := range rec.Tags {
				if tag.Count > 0 {
					genres = append(genres, tag.Name)
				}
			}
			if len(genres) > 0 {
				return genres
			}
		}
	}
	return nil
}

type searchResponse struct {
	Recordings []recording `json:"recordings"`
}

type recording struct {
	Tags []tag `json:"tags"`
}

type tag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
