package catalog

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/httpclient"
)

type HifiProvider struct {
	Client  *httpclient.Client
	BaseURL string
}

func NewHifiProvider(baseURL string) *HifiProvider {
	return &HifiProvider{
		BaseURL: baseURL,
		Client: httpclient.NewClient(&http.Client{
			Timeout: 5 * time.Minute,
		}, 800*time.Millisecond), // ~1.2 requests per second
	}
}

func (p *HifiProvider) ensureAbsoluteURL(urlOrID string, size ...string) string {
	if urlOrID == "" {
		return ""
	}
	if strings.HasPrefix(urlOrID, "http://") || strings.HasPrefix(urlOrID, "https://") {
		return urlOrID
	}
	imgSize := "640x640"
	if len(size) > 0 {
		imgSize = size[0]
	}
	path := strings.ReplaceAll(urlOrID, "-", "/")
	return fmt.Sprintf("https://resources.tidal.com/images/%s/%s.jpg", path, imgSize)
}

func (p *HifiProvider) GetArtist(ctx context.Context, id string) (*domain.Artist, error) {
	u := fmt.Sprintf("%s/artist/?id=%s", p.BaseURL, id)
	var resp APIArtistResponse
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	artist := *resp.Artist.ToDomain(p)

	aggUrl := fmt.Sprintf("%s/artist/?f=%s&skip_tracks=true", p.BaseURL, id)
	var aggResp APIArtistAggregationResponse
	if err := p.get(ctx, aggUrl, &aggResp); err == nil {
		artist.Albums = aggResp.ToAlbums(artist.Name, p)
		artist.TopTracks = aggResp.ToTopTracks(p)
	}

	return &artist, nil
}

func (p *HifiProvider) GetAlbum(ctx context.Context, id string) (*domain.Album, error) {
	u := fmt.Sprintf("%s/album/?id=%s", p.BaseURL, id)
	var resp APIAlbumResponse
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	return resp.ToDomain(p), nil
}

func (p *HifiProvider) GetPlaylist(ctx context.Context, id string) (*domain.Playlist, error) {
	u := fmt.Sprintf("%s/playlist/?id=%s", p.BaseURL, id)
	var resp APIPlaylistResponse
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	return resp.ToDomain(p), nil
}

func (p *HifiProvider) GetTrack(ctx context.Context, id string) (*domain.CatalogTrack, error) {
	u := fmt.Sprintf("%s/info/?id=%s", p.BaseURL, id)
	var resp APITrackInfoResponse
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	return resp.ToDomain(p), nil
}

func (p *HifiProvider) GetStream(ctx context.Context, trackID string, quality string) (io.ReadCloser, string, error) {
	u := fmt.Sprintf("%s/track/?id=%s&quality=%s", p.BaseURL, trackID, quality)

	var resp APIStreamResponse
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, "", err
	}

	if resp.Data.Manifest == "" {
		return nil, "", fmt.Errorf("no manifest found")
	}

	decoded, err := base64.StdEncoding.DecodeString(resp.Data.Manifest)
	if err != nil {
		return nil, "", err
	}

	if resp.Data.ManifestMimeType == "application/vnd.tidal.bts" {
		var manifest struct {
			Urls []string `json:"urls"`
		}
		if err := json.Unmarshal(decoded, &manifest); err != nil {
			return nil, "", err
		}
		if len(manifest.Urls) == 0 {
			return nil, "", fmt.Errorf("no urls in manifest")
		}

		streamUrl := manifest.Urls[0]
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamUrl, nil)
		if err != nil {
			return nil, "", err
		}
		sResp, err := p.Client.Do(ctx, req)
		if err != nil {
			return nil, "", err
		}
		if sResp.StatusCode != http.StatusOK {
			_ = sResp.Body.Close()
			return nil, "", fmt.Errorf("stream fetch failed: %s", sResp.Status)
		}
		return sResp.Body, "audio/flac", nil
	}

	if resp.Data.ManifestMimeType == "application/dash+xml" {
		s := string(decoded)

		if strings.Contains(s, "<SegmentTemplate") {
			return p.handleSegmentedDash(ctx, s)
		}

		re := regexp.MustCompile(`(?is)<BaseURL[^>]*>(.*?)</BaseURL>`)
		match := re.FindStringSubmatch(s)
		streamUrl := ""
		if len(match) > 1 {
			streamUrl = strings.TrimSpace(match[1])
		}

		if streamUrl == "" {
			return nil, "", fmt.Errorf("no BaseURL found in DASH manifest")
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamUrl, nil)
		if err != nil {
			return nil, "", err
		}
		sResp, err := p.Client.Do(ctx, req)
		if err != nil {
			return nil, "", err
		}
		if sResp.StatusCode != http.StatusOK {
			_ = sResp.Body.Close()
			return nil, "", fmt.Errorf("stream fetch failed: %s", sResp.Status)
		}
		return sResp.Body, "audio/flac", nil
	}

	return nil, "", fmt.Errorf("unsupported manifest type: %s", resp.Data.ManifestMimeType)
}

func (p *HifiProvider) handleSegmentedDash(ctx context.Context, manifest string) (io.ReadCloser, string, error) {
	initRe := regexp.MustCompile(`initialization="([^"]+)"`)
	mediaRe := regexp.MustCompile(`media="([^"]+)"`)
	startNumRe := regexp.MustCompile(`startNumber="(\d+)"`)

	initMatch := initRe.FindStringSubmatch(manifest)
	mediaMatch := mediaRe.FindStringSubmatch(manifest)
	startNumMatch := startNumRe.FindStringSubmatch(manifest)

	if len(initMatch) < 2 || len(mediaMatch) < 2 {
		return nil, "", fmt.Errorf("failed to parse SegmentTemplate URLs")
	}

	initUrl := strings.ReplaceAll(initMatch[1], "&amp;", "&")
	mediaTemplate := strings.ReplaceAll(mediaMatch[1], "&amp;", "&")
	startNum := 1
	if len(startNumMatch) > 1 {
		_, _ = fmt.Sscanf(startNumMatch[1], "%d", &startNum)
	}

	count := 0
	fullSRe := regexp.MustCompile(`<S\s+([^>]*?)/>`)
	sMatches := fullSRe.FindAllStringSubmatch(manifest, -1)
	for _, sm := range sMatches {
		attrs := sm[1]
		rMatch := regexp.MustCompile(`r="(\d+)"`).FindStringSubmatch(attrs)
		if len(rMatch) > 1 {
			r := 0
			_, _ = fmt.Sscanf(rMatch[1], "%d", &r)
			count += 1 + r
		} else {
			count += 1
		}
	}

	urls := []string{initUrl}
	for i := 0; i < count; i++ {
		num := startNum + i
		segUrl := strings.ReplaceAll(mediaTemplate, "$Number$", fmt.Sprintf("%d", num))
		urls = append(urls, segUrl)
	}

	return &multiSegmentReader{
		urls:   urls,
		client: p.Client.GetUnderlyingClient(),
		ctx:    ctx,
	}, "audio/mp4", nil
}

func (p *HifiProvider) GetSimilarAlbums(ctx context.Context, id string) ([]domain.Album, error) {
	u := fmt.Sprintf("%s/album/similar/?id=%s&limit=8", p.BaseURL, id)

	var resp APISimilarAlbumsResponse
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	return resp.ToDomain(p), nil
}

func (p *HifiProvider) GetLyrics(ctx context.Context, trackID string) (string, string, error) {
	u := fmt.Sprintf("%s/lyrics/?id=%s", p.BaseURL, trackID)
	var resp APILyricsResponse
	if err := p.get(ctx, u, &resp); err != nil {
		return "", "", err
	}
	return resp.Lyrics.Lyrics, resp.Lyrics.Subtitles, nil
}

func (p *HifiProvider) get(ctx context.Context, url string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := p.Client.Do(ctx, req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed: %s", resp.Status)
	}

	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	err = decoder.Decode(target)
	return err
}
