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
	lastRequest time.Time
	httpClient  *http.Client
	baseURL     string
	userAgent   string
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
	defer func() {
		_ = resp.Body.Close()
	}()

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

func (c *Client) GetRecordingByISRC(ctx context.Context, isrc string) (*RecordingMetadata, error) {
	if isrc == "" {
		return nil, nil
	}

	c.throttle()

	u := fmt.Sprintf("%s/recording?query=isrc:%s&inc=artists+releases+release-artists&fmt=json&limit=1", c.baseURL, url.QueryEscape(isrc))

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
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz returned status %d", resp.StatusCode)
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Recordings) == 0 {
		return nil, nil
	}

	rec := result.Recordings[0]
	meta := &RecordingMetadata{
		Title:    rec.Title,
		Duration: rec.Length,
	}

	if len(rec.ArtistCredit) > 0 {
		meta.Artist = rec.ArtistCredit[0].Artist.Name
		meta.Artists = make([]string, len(rec.ArtistCredit))
		meta.ArtistIDs = make([]string, len(rec.ArtistCredit))
		for i, ac := range rec.ArtistCredit {
			meta.Artists[i] = ac.Artist.Name
			meta.ArtistIDs[i] = ac.Artist.ID
			if ac.Type == "composer" && meta.Composer == "" {
				meta.Composer = ac.Artist.Name
			}
		}
	}

	if len(rec.Releases) > 0 {
		rel := rec.Releases[0]
		meta.Album = rel.Title
		meta.ReleaseDate = rel.Date
		meta.ReleaseID = rel.ReleaseGroup.ID
		meta.Barcode = rel.Barcode
		meta.CatalogNumber = rel.CatalogNumber
		meta.ReleaseType = rel.ReleaseGroup.PrimaryType
		if rel.Date != "" && len(rel.Date) >= 4 {
			_, _ = fmt.Sscanf(rel.Date, "%d", &meta.Year)
		}
		if len(rel.ArtistCredit) > 0 {
			meta.AlbumArtists = make([]string, len(rel.ArtistCredit))
			meta.AlbumArtistIDs = make([]string, len(rel.ArtistCredit))
			for i, ac := range rel.ArtistCredit {
				meta.AlbumArtists[i] = ac.Artist.Name
				meta.AlbumArtistIDs[i] = ac.Artist.ID
			}
			if len(meta.AlbumArtists) > 0 {
				meta.AlbumArtist = meta.AlbumArtists[0]
			}
		}
	}

	return meta, nil
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
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Tags         []tag          `json:"tags"`
	Releases     []release      `json:"releases"`
	ArtistCredit []artistCredit `json:"artist-credit"`
	Length       int            `json:"length"`
}

type release struct {
	ID            string         `json:"id"`
	Title         string         `json:"title"`
	Status        string         `json:"status"`
	Date          string         `json:"date"`
	Country       string         `json:"country"`
	Barcode       string         `json:"barcode"`
	CatalogNumber string         `json:"catalognumber"`
	Label         string         `json:"label"`
	ReleaseGroup  releaseGroup   `json:"release-group"`
	Media         []media        `json:"media"`
	ArtistCredit  []artistCredit `json:"artist-credit"`
}

type artistCredit struct {
	Name       string `json:"name"`
	Artist     artist `json:"artist"`
	JoinPhrase string `json:"joinphrase"`
	Type       string `json:"type"`
}

type releaseGroup struct {
	ID          string `json:"id"`
	PrimaryType string `json:"primary-type"`
}

type media struct {
	TrackCount int `json:"trackCount"`
}

type artist struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	SortName string `json:"sort-name"`
}

type tag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type RecordingMetadata struct {
	ReleaseID      string
	Composer       string
	Album          string
	AlbumArtist    string
	ReleaseDate    string
	Barcode        string
	Artist         string
	CatalogNumber  string
	Title          string
	ReleaseType    string
	Artists        []string
	ArtistIDs      []string
	AlbumArtists   []string
	AlbumArtistIDs []string
	Year           int
	Duration       int
}
