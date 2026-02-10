package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/cesargomez89/navidrums/internal/models"
)

type HifiProvider struct {
	BaseURL string
	Client  *http.Client
}

func NewHifiProvider(baseURL string) *HifiProvider {
	return &HifiProvider{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

// Helper structs for JSON parsing
type apiResponse struct {
	Data  json.RawMessage   `json:"data"`
	Items []json.RawMessage `json:"items"` // sometimes top level?
}

type FlexCover []string

func (f *FlexCover) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*f = []string{s}
		return nil
	}
	if data[0] == '[' {
		var items []struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
		var urls []string
		for _, item := range items {
			urls = append(urls, item.URL)
		}
		*f = urls
		return nil
	}
	if data[0] == '{' {
		var item struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(data, &item); err != nil {
			return err
		}
		*f = []string{item.URL}
		return nil
	}
	return nil
}

func formatID(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	case json.Number:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (p *HifiProvider) ensureAbsoluteURL(urlOrID string) string {
	if urlOrID == "" {
		return ""
	}
	if strings.HasPrefix(urlOrID, "http://") || strings.HasPrefix(urlOrID, "https://") {
		return urlOrID
	}
	// It's a Tidal ID. Use Tidal's official CDN for high-quality images.
	// We replace dashes with slashes if they are present, as many Tidal clients do.
	path := strings.ReplaceAll(urlOrID, "-", "/")
	return fmt.Sprintf("https://resources.tidal.com/images/%s/640x640.jpg", path)
}

// ... more helpers inside methods ...

func (p *HifiProvider) Search(ctx context.Context, query string, searchType string) (*models.SearchResult, error) {
	res := &models.SearchResult{}

	if searchType == "" {
		searchType = "album"
	}

	switch searchType {
	case "artist":
		artists, err := p.searchArtists(ctx, query)
		if err == nil {
			res.Artists = artists
		}
	case "album":
		albums, err := p.searchAlbums(ctx, query)
		if err == nil {
			res.Albums = albums
		}
	case "track":
		tracks, err := p.searchTracks(ctx, query)
		if err == nil {
			res.Tracks = tracks
		}
	case "playlist":
		playlists, err := p.searchPlaylists(ctx, query)
		if err == nil {
			res.Playlists = playlists
		}
	default:
		// Default to album if type is unknown
		albums, err := p.searchAlbums(ctx, query)
		if err == nil {
			res.Albums = albums
		}
	}

	return res, nil
}

func (p *HifiProvider) searchArtists(ctx context.Context, query string) ([]models.Artist, error) {
	u := fmt.Sprintf("%s/search/?a=%s", p.BaseURL, url.QueryEscape(query))
	var resp struct {
		Data struct {
			Artists struct {
				Items []struct {
					ID   json.Number `json:"id"`
					Name string      `json:"name"`
				} `json:"items"`
			} `json:"artists"`
		} `json:"data"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	var artists []models.Artist
	for _, item := range resp.Data.Artists.Items {
		artists = append(artists, models.Artist{
			ID:   formatID(item.ID),
			Name: item.Name,
		})
	}
	return artists, nil
}

func (p *HifiProvider) searchAlbums(ctx context.Context, query string) ([]models.Album, error) {
	u := fmt.Sprintf("%s/search/?al=%s", p.BaseURL, url.QueryEscape(query))
	var resp struct {
		Data struct {
			Albums struct {
				Items []struct {
					ID      json.Number `json:"id"`
					Title   string      `json:"title"`
					Artists []struct {
						Name string `json:"name"`
					} `json:"artists"`
				} `json:"items"`
			} `json:"albums"`
		} `json:"data"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	var albums []models.Album
	for _, item := range resp.Data.Albums.Items {
		artist := "Unknown"
		if len(item.Artists) > 0 {
			artist = item.Artists[0].Name
		}
		albums = append(albums, models.Album{
			ID:     formatID(item.ID),
			Title:  item.Title,
			Artist: artist,
		})
	}
	return albums, nil
}

func (p *HifiProvider) searchTracks(ctx context.Context, query string) ([]models.Track, error) {
	u := fmt.Sprintf("%s/search/?s=%s", p.BaseURL, url.QueryEscape(query))
	var resp struct {
		Data struct {
			Items []struct {
				ID          json.Number `json:"id"`
				Title       string      `json:"title"`
				Duration    int         `json:"duration"`
				TrackNumber int         `json:"trackNumber"`
				Album       struct {
					Title string `json:"title"`
				} `json:"album"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	var tracks []models.Track
	for _, item := range resp.Data.Items {
		artist := "Unknown"
		if len(item.Artists) > 0 {
			artist = item.Artists[0].Name
		}
		tracks = append(tracks, models.Track{
			ID:          formatID(item.ID),
			Title:       item.Title,
			Artist:      artist,
			Album:       item.Album.Title,
			TrackNumber: item.TrackNumber,
			Duration:    item.Duration,
		})
	}
	return tracks, nil
}

func (p *HifiProvider) searchPlaylists(ctx context.Context, query string) ([]models.Playlist, error) {
	u := fmt.Sprintf("%s/search/?p=%s", p.BaseURL, url.QueryEscape(query))
	var resp struct {
		Data struct {
			Playlists struct {
				Items []struct {
					Uuid  string `json:"uuid"`
					Title string `json:"title"`
				} `json:"items"`
			} `json:"playlists"`
		} `json:"data"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	var playlists []models.Playlist
	for _, item := range resp.Data.Playlists.Items {
		playlists = append(playlists, models.Playlist{
			ID:    item.Uuid,
			Title: item.Title,
		})
	}
	return playlists, nil
}

func (p *HifiProvider) GetArtist(ctx context.Context, id string) (*models.Artist, error) {
	u := fmt.Sprintf("%s/artist/?id=%s", p.BaseURL, id)
	var resp struct {
		Artist struct {
			ID   json.Number `json:"id"`
			Name string      `json:"name"`
		} `json:"artist"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	artist := &models.Artist{
		ID:   formatID(resp.Artist.ID),
		Name: resp.Artist.Name,
	}

	// Fetch Albums and Top Tracks using the aggregate endpoint
	aggUrl := fmt.Sprintf("%s/artist/?f=%s&skip_tracks=true", p.BaseURL, id)
	var aggResp struct {
		Albums struct {
			Items []struct {
				ID    json.Number `json:"id"`
				Title string      `json:"title"`
			} `json:"items"`
		} `json:"albums"`
		Tracks []struct {
			ID          json.Number `json:"id"`
			Title       string      `json:"title"`
			TrackNumber int         `json:"trackNumber"`
			Duration    int         `json:"duration"`
			Album       struct {
				Title string `json:"title"`
			} `json:"album"`
		} `json:"tracks"`
	}

	if err := p.get(ctx, aggUrl, &aggResp); err == nil {
		for _, item := range aggResp.Albums.Items {
			artist.Albums = append(artist.Albums, models.Album{
				ID:     formatID(item.ID),
				Title:  item.Title,
				Artist: artist.Name,
			})
		}
		for _, item := range aggResp.Tracks {
			artist.TopTracks = append(artist.TopTracks, models.Track{
				ID:          formatID(item.ID),
				Title:       item.Title,
				Artist:      artist.Name,
				Album:       item.Album.Title,
				TrackNumber: item.TrackNumber,
				Duration:    item.Duration,
			})
		}
	}

	return artist, nil
}

func (p *HifiProvider) GetAlbum(ctx context.Context, id string) (*models.Album, error) {
	u := fmt.Sprintf("%s/album/?id=%s", p.BaseURL, id)
	var resp struct {
		Data struct {
			ID              json.Number `json:"id"`
			Title           string      `json:"title"`
			ReleaseDate     string      `json:"releaseDate"`
			Copyright       string      `json:"copyright"`
			NumberOfTracks  int         `json:"numberOfTracks"`
			NumberOfVolumes int         `json:"numberOfVolumes"`
			Artist          struct {
				Name string `json:"name"`
			} `json:"artist"`
			Cover FlexCover `json:"cover"`
			Items []struct {
				Item struct {
					Title        string      `json:"title"`
					Duration     int         `json:"duration"`
					TrackNumber  int         `json:"trackNumber"`
					VolumeNumber int         `json:"volumeNumber"`
					ID           json.Number `json:"id"`
					ISRC         string      `json:"isrc"`
					Explicit     bool        `json:"explicit"`
					Artists      []struct {
						Name string `json:"name"`
					} `json:"artists"`
				} `json:"item"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	// Extract year from release date
	year := 0
	if len(resp.Data.ReleaseDate) >= 4 {
		fmt.Sscanf(resp.Data.ReleaseDate[:4], "%d", &year)
	}

	// Get album art URL
	albumArtURL := ""
	if len(resp.Data.Cover) > 0 {
		albumArtURL = p.ensureAbsoluteURL(resp.Data.Cover[0])
	}

	album := &models.Album{
		ID:          formatID(resp.Data.ID),
		Title:       resp.Data.Title,
		Artist:      resp.Data.Artist.Name,
		Year:        year,
		Copyright:   resp.Data.Copyright,
		TotalTracks: resp.Data.NumberOfTracks,
		TotalDiscs:  resp.Data.NumberOfVolumes,
		AlbumArtURL: albumArtURL,
	}

	for _, wrapped := range resp.Data.Items {
		item := wrapped.Item
		tArtist := album.Artist
		if len(item.Artists) > 0 {
			tArtist = item.Artists[0].Name
		}

		album.Tracks = append(album.Tracks, models.Track{
			ID:             formatID(item.ID),
			Title:          item.Title,
			Artist:         tArtist,
			AlbumArtist:    album.Artist,
			Album:          album.Title,
			TrackNumber:    item.TrackNumber,
			DiscNumber:     item.VolumeNumber,
			TotalTracks:    album.TotalTracks,
			TotalDiscs:     album.TotalDiscs,
			Duration:       item.Duration,
			Year:           album.Year,
			Copyright:      album.Copyright,
			ISRC:           item.ISRC,
			AlbumArtURL:    album.AlbumArtURL,
			ExplicitLyrics: item.Explicit,
		})
	}
	return album, nil
}

func (p *HifiProvider) GetPlaylist(ctx context.Context, id string) (*models.Playlist, error) {
	u := fmt.Sprintf("%s/playlist/?id=%s", p.BaseURL, id)
	var resp struct {
		Playlist struct {
			Uuid        string `json:"uuid"`
			Title       string `json:"title"`
			Description string `json:"description"`
			SquareImage string `json:"squareImage"`
		} `json:"playlist"`
		Items []struct {
			Item struct {
				ID          json.Number `json:"id"`
				Title       string      `json:"title"`
				TrackNumber int         `json:"trackNumber"`
				Duration    int         `json:"duration"`
				ISRC        string      `json:"isrc"`
				Explicit    bool        `json:"explicit"`
				Album       struct {
					Title string    `json:"title"`
					Cover FlexCover `json:"cover"`
				} `json:"album"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"item"`
		} `json:"items"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	pl := &models.Playlist{
		ID:          resp.Playlist.Uuid,
		Title:       resp.Playlist.Title,
		Description: resp.Playlist.Description,
		ImageURL:    p.ensureAbsoluteURL(resp.Playlist.SquareImage),
	}

	for _, wrapped := range resp.Items {
		item := wrapped.Item
		artist := "Unknown"
		if len(item.Artists) > 0 {
			artist = item.Artists[0].Name
		}

		albumArtURL := ""
		if len(item.Album.Cover) > 0 {
			albumArtURL = p.ensureAbsoluteURL(item.Album.Cover[0])
		}

		pl.Tracks = append(pl.Tracks, models.Track{
			ID:             formatID(item.ID),
			Title:          item.Title,
			Artist:         artist,
			Album:          item.Album.Title,
			TrackNumber:    item.TrackNumber,
			Duration:       item.Duration,
			ISRC:           item.ISRC,
			AlbumArtURL:    albumArtURL,
			ExplicitLyrics: item.Explicit,
		})
	}

	return pl, nil
}

func (p *HifiProvider) GetTrack(ctx context.Context, id string) (*models.Track, error) {
	u := fmt.Sprintf("%s/info/?id=%s", p.BaseURL, id)
	var resp struct {
		Data struct {
			ID           json.Number `json:"id"`
			Title        string      `json:"title"`
			Duration     int         `json:"duration"`
			TrackNumber  int         `json:"trackNumber"`
			VolumeNumber int         `json:"volumeNumber"`
			ISRC         string      `json:"isrc"`
			Explicit     bool        `json:"explicit"`
			Copyright    string      `json:"copyright"`
			Album        struct {
				Title           string    `json:"title"`
				ReleaseDate     string    `json:"releaseDate"`
				NumberOfTracks  int       `json:"numberOfTracks"`
				NumberOfVolumes int       `json:"numberOfVolumes"`
				Cover           FlexCover `json:"cover"`
			} `json:"album"`
			Artist struct {
				Name string `json:"name"`
			} `json:"artist"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"data"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	// Extract year from release date
	year := 0
	if len(resp.Data.Album.ReleaseDate) >= 4 {
		fmt.Sscanf(resp.Data.Album.ReleaseDate[:4], "%d", &year)
	}

	// Get album art URL
	albumArtURL := ""
	if len(resp.Data.Album.Cover) > 0 {
		albumArtURL = p.ensureAbsoluteURL(resp.Data.Album.Cover[0])
	}

	// Get album artist (use first artist or main artist)
	albumArtist := resp.Data.Artist.Name

	return &models.Track{
		ID:             formatID(resp.Data.ID),
		Title:          resp.Data.Title,
		Artist:         resp.Data.Artist.Name,
		AlbumArtist:    albumArtist,
		Album:          resp.Data.Album.Title,
		TrackNumber:    resp.Data.TrackNumber,
		DiscNumber:     resp.Data.VolumeNumber,
		TotalTracks:    resp.Data.Album.NumberOfTracks,
		TotalDiscs:     resp.Data.Album.NumberOfVolumes,
		Duration:       resp.Data.Duration,
		Year:           year,
		ISRC:           resp.Data.ISRC,
		Copyright:      resp.Data.Copyright,
		AlbumArtURL:    albumArtURL,
		ExplicitLyrics: resp.Data.Explicit,
	}, nil
}

func (p *HifiProvider) GetStream(ctx context.Context, trackID string, quality string) (io.ReadCloser, string, error) {
	u := fmt.Sprintf("%s/track/?id=%s&quality=%s", p.BaseURL, trackID, quality)

	var resp struct {
		Data struct {
			Manifest         string `json:"manifest"`
			ManifestMimeType string `json:"manifestMimeType"`
		} `json:"data"`
	}

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

		// Fetch stream
		streamUrl := manifest.Urls[0]
		sResp, err := p.Client.Get(streamUrl)
		if err != nil {
			return nil, "", err
		}
		if sResp.StatusCode != 200 {
			sResp.Body.Close()
			return nil, "", fmt.Errorf("stream fetch failed: %s", sResp.Status)
		}
		return sResp.Body, "audio/flac", nil // Assuming FLAC for now
	}

	if resp.Data.ManifestMimeType == "application/dash+xml" {
		s := string(decoded)

		// Check for SegmentTemplate (segmented DASH)
		if strings.Contains(s, "<SegmentTemplate") {
			return p.handleSegmentedDash(ctx, s)
		}

		// Regex to find BaseURL content regardless of namespaces or attributes
		re := regexp.MustCompile(`(?is)<BaseURL[^>]*>(.*?)</BaseURL>`)
		match := re.FindStringSubmatch(s)
		streamUrl := ""
		if len(match) > 1 {
			streamUrl = strings.TrimSpace(match[1])
		}

		if streamUrl == "" {
			fmt.Printf("[DEBUG] FAILED DASH MANIFEST: %s\n", s)
			return nil, "", fmt.Errorf("no BaseURL found in DASH manifest")
		}

		// Fetch stream
		sResp, err := p.Client.Get(streamUrl)
		if err != nil {
			return nil, "", err
		}
		if sResp.StatusCode != 200 {
			sResp.Body.Close()
			return nil, "", fmt.Errorf("stream fetch failed: %s", sResp.Status)
		}
		return sResp.Body, "audio/flac", nil
	}

	return nil, "", fmt.Errorf("unsupported manifest type: %s", resp.Data.ManifestMimeType)
}

func (p *HifiProvider) handleSegmentedDash(ctx context.Context, manifest string) (io.ReadCloser, string, error) {
	// Simple regex/string scanning for SegmentTemplate attributes
	// <SegmentTemplate timescale="48000" initialization=".../0.mp4?..." media=".../$Number$.mp4?..." startNumber="1">
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
		fmt.Sscanf(startNumMatch[1], "%d", &startNum)
	}

	// Parse SegmentTimeline to get total count
	// <S d="188416" r="38"/> -> 1 + 38 = 39 segments
	// <S d="118208"/> -> 1 segment
	count := 0
	sRe := regexp.MustCompile(`<S\s+[^>]*?r="(\d+)"[^>]*/>`)
	matches := sRe.FindAllStringSubmatch(manifest, -1)
	for _, m := range matches {
		r := 0
		fmt.Sscanf(m[1], "%d", &r)
		count += 1 + r
	}
	// Also count segments without 'r'
	sSimpleRe := regexp.MustCompile(`<S\s+[^>]*?d="\d+"(?:\s+[^>]*?)?(?:\s+r="\d+")?[^>]*?/>`)
	_ = sSimpleRe.FindAllString(manifest, -1)
	// Actually, the logic above covers 'r' ones. Let's just count total <S> tags minus the ones with 'r'
	// No, easier: iterate through all <S> tags and parse 'r' if present.

	count = 0
	fullSRe := regexp.MustCompile(`<S\s+([^>]*?)/>`)
	sMatches := fullSRe.FindAllStringSubmatch(manifest, -1)
	for _, sm := range sMatches {
		attrs := sm[1]
		rMatch := regexp.MustCompile(`r="(\d+)"`).FindStringSubmatch(attrs)
		if len(rMatch) > 1 {
			r := 0
			fmt.Sscanf(rMatch[1], "%d", &r)
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
		client: p.Client,
		ctx:    ctx,
	}, "audio/mp4", nil
}

type multiSegmentReader struct {
	urls     []string
	client   *http.Client
	ctx      context.Context
	currIdx  int
	currBody io.ReadCloser
}

func (r *multiSegmentReader) Read(p []byte) (n int, err error) {
	if r.currBody == nil {
		if r.currIdx >= len(r.urls) {
			return 0, io.EOF
		}
		// Fetch next segment
		req, err := http.NewRequestWithContext(r.ctx, "GET", r.urls[r.currIdx], nil)
		if err != nil {
			return 0, err
		}
		resp, err := r.client.Do(req)
		if err != nil {
			return 0, err
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			return 0, fmt.Errorf("segment fetch failed (%d): %s", r.currIdx, resp.Status)
		}
		r.currBody = resp.Body
		r.currIdx++
	}

	n, err = r.currBody.Read(p)
	if err == io.EOF {
		r.currBody.Close()
		r.currBody = nil
		return r.Read(p) // recursive call to next segment
	}
	return n, err
}

func (r *multiSegmentReader) Close() error {
	if r.currBody != nil {
		return r.currBody.Close()
	}
	return nil
}

func (p *HifiProvider) get(ctx context.Context, url string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("API request failed: %s", resp.Status)
	}

	// For debugging, peek at the body if it fails to decode?
	// or just decode.
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	err = decoder.Decode(target)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to decode response from %s: %v\n", url, err)
	}
	return err
}
