package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/httpclient"
)

var ErrQobuzNotSupported = errors.New("qobuz provider does not support this operation")

type QobuzProvider struct {
	client  *httpclient.Client
	BaseURL string
}

func NewQobuzProvider(baseURL string) *QobuzProvider {
	return &QobuzProvider{
		BaseURL: baseURL,
		client: httpclient.NewClient(&http.Client{
			Timeout: 20 * time.Second,
		}, 500*time.Millisecond),
	}
}

func (p *QobuzProvider) Search(ctx context.Context, query string, searchType string) (*domain.SearchResult, error) {
	searchURL := fmt.Sprintf("%s/get-music?q=%s&offset=0", p.BaseURL, url.QueryEscape(query))
	var resp QobuzSearchResponse
	if err := p.get(ctx, searchURL, &resp); err != nil {
		return nil, fmt.Errorf("qobuz search failed: %w", err)
	}

	result := resp.Data.ToDomain()

	if searchType != "" && searchType != "all" {
		switch searchType {
		case "artist":
			result.Albums = nil
			result.Tracks = nil
			result.Playlists = nil
		case "album":
			result.Artists = nil
			result.Tracks = nil
			result.Playlists = nil
		case "track":
			result.Albums = nil
			result.Artists = nil
			result.Playlists = nil
		case "playlist":
			result.Albums = nil
			result.Artists = nil
			result.Tracks = nil
		}
	}

	return result, nil
}

func (p *QobuzProvider) GetArtist(ctx context.Context, id string) (*domain.Artist, error) {
	artistID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid artist id: %w", err)
	}
	url := fmt.Sprintf("%s/get-artist?artist_id=%d", p.BaseURL, artistID)
	var resp QobuzArtistResponse
	if err := p.get(ctx, url, &resp); err != nil {
		return nil, fmt.Errorf("qobuz get artist failed: %w", err)
	}
	return resp.Data.ToDomain(), nil
}

func (p *QobuzProvider) GetAlbum(ctx context.Context, id string) (*domain.Album, error) {
	url := fmt.Sprintf("%s/get-album?album_id=%s", p.BaseURL, url.PathEscape(id))
	var resp QobuzAlbumResponse
	if err := p.get(ctx, url, &resp); err != nil {
		return nil, fmt.Errorf("qobuz get album failed: %w", err)
	}
	return resp.ToDomain(), nil
}

func (p *QobuzProvider) GetPlaylist(ctx context.Context, id string) (*domain.Playlist, error) {
	return nil, ErrQobuzNotSupported
}

func (p *QobuzProvider) GetTrack(ctx context.Context, id string) (*domain.CatalogTrack, error) {
	trackID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid track id: %w", err)
	}
	url := fmt.Sprintf("%s/get-track?isrc=%d", p.BaseURL, trackID)
	var resp QobuzTrackResponse
	if err := p.get(ctx, url, &resp); err != nil {
		return nil, fmt.Errorf("qobuz get track failed: %w", err)
	}
	track := resp.ToDomain()
	return &track, nil
}

func (p *QobuzProvider) GetStream(ctx context.Context, trackID string, quality string) (io.ReadCloser, string, error) {
	tid, err := strconv.Atoi(trackID)
	if err != nil {
		return nil, "", fmt.Errorf("invalid track id: %w", err)
	}
	q := qobuzQualityCode(quality)

	downloadURL := fmt.Sprintf("%s/download-music?track_id=%d&quality=%d", p.BaseURL, tid, q)
	var downloadResp QobuzDownloadResponse
	if downloadErr := p.get(ctx, downloadURL, &downloadResp); downloadErr != nil {
		return nil, "", fmt.Errorf("qobuz get stream failed: %w", downloadErr)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadResp.URL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create stream request: %w", err)
	}

	resp, err := p.client.GetUnderlyingClient().Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch stream: %w", err)
	}

	mime := resp.Header.Get("Content-Type")
	if mime == "" {
		mime = "audio/flac"
	}

	return resp.Body, mime, nil
}

func (p *QobuzProvider) GetSimilarAlbums(ctx context.Context, id string) ([]domain.Album, error) {
	return nil, ErrQobuzNotSupported
}

func (p *QobuzProvider) GetSimilarArtists(ctx context.Context, id string) ([]domain.Artist, error) {
	return nil, ErrQobuzNotSupported
}

func (p *QobuzProvider) GetLyrics(ctx context.Context, trackID string) (string, string, error) {
	return "", "", ErrQobuzNotSupported
}

func (p *QobuzProvider) GetRecommendations(ctx context.Context, id string) ([]domain.CatalogTrack, error) {
	return nil, ErrQobuzNotSupported
}

func (p *QobuzProvider) get(ctx context.Context, targetURL string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return err
	}
	p.setHeaders(req)

	resp, err := p.client.Do(ctx, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	decoder := json.NewDecoder(resp.Body)
	return decoder.Decode(result)
}

func (p *QobuzProvider) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
}

func qobuzQualityCode(quality string) int {
	switch quality {
	case "HI_RES_LOSSLESS":
		return 27
	case "LOSSLESS":
		return 6
	case "HIGH":
		return 5
	case "LOW":
		return 1
	default:
		return 6
	}
}

var _ Provider = (*QobuzProvider)(nil)
